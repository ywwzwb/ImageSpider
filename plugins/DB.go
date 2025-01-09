package plugins

import (
	"fmt"
	"log/slog"
	"ywwzwb/imagespider/embed"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
	"ywwzwb/imagespider/models/config"

	"database/sql"

	"github.com/lib/pq"
)

const DBPluginID string = "DB"

type DB struct {
	app    interfaces.IApplication
	config config.DatabaseConfig
	db     *sql.DB
}
type DBCommonError int

const NotFound DBCommonError = 1

func (e DBCommonError) Error() string {
	switch e {
	case NotFound:
		return "not found"
	default:
		return "unknown error"
	}
}
func newDB() *DB {
	DB := DB{}
	return &DB
}

func init() {
	DB := newDB()
	interfaces.Plugins[DB.ID()] = DB
}

func (s *DB) Name() string {
	return "DB"
}
func (s *DB) ID() string {
	return DBPluginID
}
func (s *DB) Load(app interfaces.IApplication) error {
	s.app = app
	s.config = app.GetAppConfig().DatabaseConfig
	db, err := sql.Open("postgres", s.config.Connection)
	if err != nil {
		slog.Error("open database failed", "error", err)
		return err
	}
	s.db = db
	res, err := db.Exec(embed.InitSql)
	if err != nil {
		slog.Error("init database failed", "error", err)
		return err
	}
	slog.Info("init database success", "result", res)
	return nil
}
func (s *DB) Unload() {
	s.db.Close()
}
func (s *DB) InitSource(id string) error {
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS images_source_%s PARTITION OF images FOR VALUES IN ('%s') PARTITION BY RANGE (post_time)",
		id, id)
	logger := slog.With("sql", sql)
	res, err := s.db.Exec(sql)
	if err != nil {
		logger.Error("init source failed", "error", err)
		return err
	}
	logger.Info("init source success", "result", res)
	return nil

}
func (s *DB) GetMeta(id, source string) (*models.ImageMeta, bool) {
	rows, err := s.db.Query("SELECT id, tags, image_url, local_path, post_time, source_id FROM images WHERE id = $1 AND source_id= $2", id, source)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, false
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false
	}
	meta := models.ImageMeta{}
	err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.LocalPath, &meta.PostTime, &meta.SourceID)
	if err != nil {
		slog.Error("scan failed", "error", err)
		return nil, false
	}
	return &meta, true
}
func (s *DB) InsertMeta(meta models.ImageMeta) error {
	_, err := s.db.Exec("INSERT INTO images (id, source_id, tags, image_url, local_path, post_time) VALUES ($1, $2, $3, $4, $5, $6)",
		meta.ID, meta.SourceID, pq.Array(meta.Tags), meta.ImageURL, meta.LocalPath, meta.PostTime)
	if err == nil {
		return nil
	}
	slog.Warn("insert failed, maybe parition not exist, create now")
	// insert partion
	partionName := fmt.Sprintf("%04d%02d", meta.PostTime.UTC().Year(), meta.PostTime.UTC().Month())
	begin := fmt.Sprintf("%04d-%02d-01", meta.PostTime.UTC().Year(), meta.PostTime.UTC().Month())
	end := fmt.Sprintf("%04d-%02d-01", meta.PostTime.AddDate(0, 1, 0).UTC().Year(), meta.PostTime.AddDate(0, 1, 0).UTC().Month())
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS images_source_%s_%s PARTITION OF images_source_%s FOR VALUES FROM ('%s') TO ('%s');",
		meta.SourceID, partionName, meta.SourceID, begin, end)
	_, err = s.db.Exec(sql)
	if err != nil {
		slog.Error("create parition failed", "error", err, "sql", sql)
		return err
	}
	slog.Info("create partion succeed, retry insert", "sql", sql)
	_, err = s.db.Exec("INSERT INTO images (id, source_id, tags, image_url, local_path, post_time) VALUES ($1, $2, $3, $4, $5, $6)",
		meta.ID, meta.SourceID, pq.Array(meta.Tags), meta.ImageURL, meta.LocalPath, meta.PostTime)
	if err != nil {
		slog.Error("insert meta failed", "error", err)
		return err
	}
	return nil
}
func (s *DB) GetMetaLocalPathNULL(source string, maxSize int) []models.ImageMeta {
	// 读取没有本地路径的图片, 最多返回maxSize条数据, 使用post_time 倒序排列
	rows, err := s.db.Query(
		`SELECT id, tags, image_url, post_time, source_id 
			FROM images 
			WHERE source_id = $1 
				AND local_path IS NULL
			ORDER BY post_time 
			DESC LIMIT $2`, source, maxSize)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil
	}
	defer rows.Close()
	var metas []models.ImageMeta
	for rows.Next() {
		meta := models.ImageMeta{}
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil
		}
		metas = append(metas, meta)
	}
	return metas
}
func (s *DB) UpdateLocalPathForMeta(meta models.ImageMeta) error {
	_, err := s.db.Exec("UPDATE images SET local_path = $1 WHERE id = $2 AND source_id = $3 and post_time=$4", meta.LocalPath, meta.ID, meta.SourceID, meta.PostTime)
	if err != nil {
		slog.Error("update local path failed", "error", err)
		return err
	}
	return nil
}
func (s *DB) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.IDBServiceID:
		return s, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (s *DB) ListNotGroupTags(source string, offset, limit int64) (*models.TagList, error) {
	rows, err := s.db.Query(`WITH tag_list AS (
				SELECT unnest(tags) AS tag
				FROM images
				WHERE source_id = $1
			), filtered_tags AS (
				SELECT tag
				FROM tag_list
				WHERE tag NOT LIKE 'group_%'
			), tag_count AS (
				SELECT tag, COUNT(*) AS count
				FROM filtered_tags
				GROUP BY tag
				ORDER BY count DESC, tag
			)
			SELECT tag, count, total_tags
			FROM (
				SELECT tag, count, COUNT(*) OVER() AS total_tags
				FROM tag_count
			) subquery
			LIMIT $2 OFFSET $3;`, source, limit, offset)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, err
	}
	defer rows.Close()
	tagList := &models.TagList{
		TagList:    make([]models.TagInfo, 0),
		TotalCount: 0,
	}
	for rows.Next() {
		var tag models.TagInfo
		err = rows.Scan(&tag.Tag, &tag.Count, &tagList.TotalCount)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		tagList.TagList = append(tagList.TagList, tag)
	}
	return tagList, nil
}

func (s *DB) ListDownloadedImageOfTags(source string, tags []string, offset, limit int64) (*models.ImageList, error) {
	var rows *sql.Rows
	var err error
	if len(tags) == 0 {
		rows, err = s.db.Query(`WITH filtered_images AS (
			SELECT id, tags, image_url, post_time, source_id, local_path
			FROM images
			WHERE source_id = $1
			AND local_path IS NOT NULL
			AND local_path != ''
		), total_count AS (
			SELECT COUNT(*) AS total_items
			FROM filtered_images
		)
		SELECT i.id, i.tags, i.image_url, i.post_time, i.source_id, i.local_path, t.total_items
		FROM filtered_images i
		CROSS JOIN total_count t
		ORDER BY i.post_time DESC
		LIMIT $2 OFFSET $3;`, source, limit, offset)
	} else {
		rows, err = s.db.Query(`WITH filtered_images AS (
			SELECT id, tags, image_url, post_time, source_id, local_path
			FROM images
			WHERE source_id = $1
			AND local_path IS NOT NULL
			AND local_path != ''
			AND tags @> $2
		), total_count AS (
			SELECT COUNT(*) AS total_items
			FROM filtered_images
		)
		SELECT i.id, i.tags, i.image_url, i.post_time, i.source_id, i.local_path, t.total_items
		FROM filtered_images i
		CROSS JOIN total_count t
		ORDER BY i.post_time DESC
		LIMIT $3 OFFSET $4;`, source, pq.Array(tags), limit, offset)
	}
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, err
	}
	defer rows.Close()
	imageList := &models.ImageList{
		ImageList:  make([]models.ImageMeta, 0),
		TotalCount: 0,
	}
	for rows.Next() {
		meta := models.ImageMeta{}
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID, &meta.LocalPath, &imageList.TotalCount)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		imageList.ImageList = append(imageList.ImageList, meta)
	}
	return imageList, nil
}
func (s *DB) GetImageMeta(source string, id string) (*models.ImageMeta, error) {
	rows, err := s.db.Query(`
	SELECT id, tags, image_url, post_time, source_id, local_path
	images
	WHERE source_id = $1
	AND id = $2;`, source, id)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var meta models.ImageMeta
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID, &meta.LocalPath)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		return &meta, nil
	}
	return nil, NotFound
}
