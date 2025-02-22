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
	initImagesSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS images_source_%s PARTITION OF images FOR VALUES IN ('%s') PARTITION BY RANGE (post_time)",
		id, id)
	logger := slog.With("sql", initImagesSql)
	res, err := s.db.Exec(initImagesSql)
	if err != nil {
		logger.Error("init image source failed", "error", err)
		return err
	}
	logger.Info("init image source success", "result", res)
	initTagsSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS tags_source_%s PARTITION OF tags FOR VALUES IN ('%s')",
		id, id)
	logger = slog.With("sql", initTagsSql)
	res, err = s.db.Exec(initTagsSql)
	if err != nil {
		logger.Error("init tag source failed", "error", err)
		return err
	}
	logger.Info("init tag source success", "result", res)
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
	for tag := range meta.Tags {
		// 插入 tag 信息
		s.db.Exec("INSERT INTO tags (tag, source_id, count) VALUES ($1, $2, 0)", tag, meta.SourceID)
		s.db.Exec("UPDATE tags SET count = count + 1 WHERE tag = $1 AND source_id = $2", tag, meta.SourceID)
	}
	if err == nil {
		return nil
	}
	slog.Warn("insert failed, maybe partition not exist, create now")
	// insert partition
	partitionName := fmt.Sprintf("%04d%02d", meta.PostTime.UTC().Year(), meta.PostTime.UTC().Month())
	begin := fmt.Sprintf("%04d-%02d-01", meta.PostTime.UTC().Year(), meta.PostTime.UTC().Month())
	end := fmt.Sprintf("%04d-%02d-01", meta.PostTime.AddDate(0, 1, 0).UTC().Year(), meta.PostTime.AddDate(0, 1, 0).UTC().Month())
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS images_source_%s_%s PARTITION OF images_source_%s FOR VALUES FROM ('%s') TO ('%s');",
		meta.SourceID, partitionName, meta.SourceID, begin, end)
	_, err = s.db.Exec(sql)
	if err != nil {
		slog.Error("create partition failed", "error", err, "sql", sql)
		return err
	}
	slog.Info("create partition succeed, retry insert", "sql", sql)
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
	if meta.LocalPath != nil && len(*meta.LocalPath) != 0 {
		for tag := range meta.Tags {
			// 插入 cover 信息
			s.db.Exec("UPDATE tags SET cover = $1 WHERE cover IS NULL AND tag = $2 AND source_id = $3", meta.ID, tag, meta.SourceID)
		}
	}
	return nil
}
func (s *DB) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.DBServiceID:
		return s, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (s *DB) ListNotGroupTags(source string, offset, limit int64) (*models.TagList, error) {
	// 分页查询核心SQL（带封面信息）
	pageQuery := `
    SELECT 
        t.tag,
        t.count,
        i.id AS cover_id,
        i.tags AS cover_tags,
        i.local_path AS cover_local_path,
        i.image_url AS cover_image_url,
        i.post_time AS cover_post_time,
        i.source_id AS cover_source_id
    FROM tags t
    LEFT JOIN images i 
        ON t.cover = i.id
        AND i.source_id = t.source_id
    WHERE 
        t.source_id = $1
        AND t.cover IS NOT NULL
        AND t.tag NOT LIKE 'group_%'
    ORDER BY t.count DESC
    LIMIT $2 OFFSET $3
    `

	// 执行分页查询
	rows, err := s.db.Query(pageQuery, source, limit, offset)
	if err != nil {
		slog.Error("分页查询失败", "error", err)
		return nil, err
	}
	defer rows.Close()

	var tagList []models.TagInfo
	for rows.Next() {
		var (
			tagInfo models.TagInfo
			cover   models.ImageMeta
		)
		err := rows.Scan(
			&tagInfo.Tag,
			&tagInfo.Count,
			&cover.ID,
			pq.Array(&cover.Tags),
			&cover.LocalPath,
			&cover.ImageURL,
			&cover.PostTime,
			&cover.SourceID,
		)
		if err != nil {
			return nil, err
		}
		tagInfo.Cover = cover
		tagList = append(tagList, tagInfo)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// 获取总记录数（从tags表直接统计）
	countQuery := `
        SELECT COUNT(*) 
        FROM tags 
        WHERE 
            source_id = $1
            AND tag NOT LIKE 'group_%'
            AND cover IS NOT NULL
    `
	var totalCount int
	err = s.db.QueryRow(countQuery, source).Scan(&totalCount)
	if err != nil {
		slog.Error("总数查询失败", "error", err)
		return nil, err
	}

	return &models.TagList{
		TagList:    tagList,
		TotalCount: totalCount,
	}, nil
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
