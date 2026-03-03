package plugins

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"ywwzwb/imagespider/embed"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
	"ywwzwb/imagespider/models/config"

	"github.com/lib/pq"
)


type DB struct {
	app    interfaces.IApplication
	config config.DatabaseConfig
	db     *sql.DB
}
// DBError 数据库错误类型
type DBError string

func (e DBError) Error() string {
	return string(e)
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
	return interfaces.DBPluginID
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

	// 执行数据库迁移
	if err := s.runMigrations(); err != nil {
		slog.Error("database migration failed", "error", err)
		return err
	}

	return nil
}

// runMigrations 检查并执行数据库迁移
func (s *DB) runMigrations() error {
	logger := slog.With("migration", "v1")

	// 检查是否需要执行 migration v1 (添加 integrity_status 列)
	needsMigration, err := s.checkIfNeedsMigrationV1()
	if err != nil {
		logger.Error("failed to check migration status", "error", err)
		return err
	}

	if needsMigration {
		logger.Info("executing migration v1: adding integrity_status column")

		// 执行迁移脚本
		_, err = s.db.Exec(embed.MigrationV1Sql)
		if err != nil {
			logger.Error("migration v1 failed", "error", err)
			return fmt.Errorf("migration v1 failed: %w", err)
		}

		logger.Info("migration v1 completed successfully")
	} else {
		logger.Debug("migration v1 not needed or already applied")
	}

	return nil
}

// checkIfNeedsMigrationV1 检查是否需要执行 migration v1
func (s *DB) checkIfNeedsMigrationV1() (bool, error) {
	logger := slog.With("migration", "v1")

	// 首先检查 schema_migrations 表是否存在
	var tableExists bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'schema_migrations'
		)
	`).Scan(&tableExists)

	if err != nil {
		return false, err
	}

	if !tableExists {
		logger.Info("schema_migrations table does not exist, migration needed")
		return true, nil
	}

	// 检查 migration v1 是否已经执行
	var migrationExists bool
	err = s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = 'v1'
		)
	`).Scan(&migrationExists)

	if err != nil {
		return false, err
	}

	if migrationExists {
		logger.Debug("migration v1 already applied")
		return false, nil
	}

	logger.Info("migration v1 not applied yet")
	return true, nil
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
	rows, err := s.db.Query("SELECT id, tags, image_url, local_path, post_time, source_id, integrity_status FROM images WHERE id = $1 AND source_id= $2", id, source)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, false
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false
	}
	meta := models.ImageMeta{}
	var status sql.NullInt16
	err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.LocalPath, &meta.PostTime, &meta.SourceID, &status)
	if err != nil {
		slog.Error("scan failed", "error", err)
		return nil, false
	}
	// 处理NULL值，NULL也视为unknown
	if status.Valid {
		meta.IntegrityStatus = models.ImageIntegrityStatus(status.Int16)
	} else {
		meta.IntegrityStatus = models.ImageIntegrityUnknown
	}
	return &meta, true
}
func (s *DB) InsertMeta(meta models.ImageMeta) error {
	// 设置integrity_status的默认值
	if meta.IntegrityStatus == 0 {
		meta.IntegrityStatus = models.ImageIntegrityUnknown
	}

	_, err := s.db.Exec("INSERT INTO images (id, source_id, tags, image_url, local_path, post_time, integrity_status) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		meta.ID, meta.SourceID, pq.Array(meta.Tags), meta.ImageURL, meta.LocalPath, meta.PostTime, int16(meta.IntegrityStatus))
	if err != nil {
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
		_, err = s.db.Exec("INSERT INTO images (id, source_id, tags, image_url, local_path, post_time, integrity_status) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			meta.ID, meta.SourceID, pq.Array(meta.Tags), meta.ImageURL, meta.LocalPath, meta.PostTime, int16(meta.IntegrityStatus))
		if err != nil {
			slog.Error("insert meta failed", "error", err)
			return err
		}
	}
	for _, tag := range meta.Tags {
		// 插入 tag 信息
		s.db.Exec("INSERT INTO tags (tag, source_id, count) VALUES ($1, $2, 0)", tag, meta.SourceID)
		s.db.Exec("UPDATE tags SET count = count + 1 WHERE tag = $1 AND source_id = $2", tag, meta.SourceID)

		// 刷新标签封面（如果还没有封面）
		if meta.LocalPath != nil && *meta.LocalPath != "" {
			s.refreshTagCover(meta.SourceID, tag)
		}
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
		// 设置默认值为unknown
		meta.IntegrityStatus = models.ImageIntegrityUnknown
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
		for _, tag := range meta.Tags {
			// 插入 cover 信息
			s.db.Exec("UPDATE tags SET cover = $1 WHERE cover IS NULL AND tag = $2 AND source_id = $3", meta.ID, tag, meta.SourceID)
			// 刷新标签封面（如果还没有封面）
			s.refreshTagCover(meta.SourceID, tag)
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
			coverID sql.NullString
		)
		err := rows.Scan(
			&tagInfo.Tag,
			&tagInfo.Count,
			&coverID,
			pq.Array(&cover.Tags),
			&cover.LocalPath,
			&cover.ImageURL,
			&cover.PostTime,
			&cover.SourceID,
		)
		if err != nil {
			return nil, err
		}
		if coverID.Valid {
			cover.ID = coverID.String
			tagInfo.Cover = cover
		}
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

func (s *DB) ListDownloadedImage(source string, tags []string, status []models.ImageIntegrityStatus, offset, limit int64) (*models.ImageList, error) {
	// 构建动态SQL查询
	var conditions []string
	var args []interface{}
	argIndex := 1

	// 基础条件：source_id 和 local_path 不为空
	conditions = append(conditions, fmt.Sprintf("source_id = $%d", argIndex))
	args = append(args, source)
	argIndex++

	conditions = append(conditions, "local_path IS NOT NULL")
	conditions = append(conditions, "local_path != ''")

	// 如果指定了tags，添加tags筛选条件
	if len(tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argIndex))
		args = append(args, pq.Array(tags))
		argIndex++
	}

	// 如果指定了status，添加status筛选条件
	if len(status) > 0 {
		// 将status转换为int16数组
		statusInts := make([]int16, len(status))
		for i, s := range status {
			statusInts[i] = int16(s)
		}
		conditions = append(conditions, fmt.Sprintf("integrity_status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusInts))
		argIndex++
	}

	// 构建完整的SQL查询
	sqlQuery := fmt.Sprintf(`WITH filtered_images AS (
		SELECT id, tags, image_url, post_time, source_id, local_path, integrity_status
		FROM images
		WHERE %s
	), total_count AS (
		SELECT COUNT(*) AS total_items
		FROM filtered_images
	)
	SELECT i.id, i.tags, i.image_url, i.post_time, i.source_id, i.local_path, i.integrity_status, t.total_items
	FROM filtered_images i
	CROSS JOIN total_count t
	ORDER BY i.post_time DESC
	LIMIT $%d OFFSET $%d`, strings.Join(conditions, " AND "), argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		slog.Error("query failed", "error", err, "sql", sqlQuery)
		return nil, err
	}
	defer rows.Close()

	imageList := &models.ImageList{
		ImageList:  make([]models.ImageMeta, 0),
		TotalCount: 0,
	}
	for rows.Next() {
		meta := models.ImageMeta{}
		var status sql.NullInt16
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID, &meta.LocalPath, &status, &imageList.TotalCount)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		// 处理NULL值，NULL也视为unknown
		if status.Valid {
			meta.IntegrityStatus = models.ImageIntegrityStatus(status.Int16)
		} else {
			meta.IntegrityStatus = models.ImageIntegrityUnknown
		}
		imageList.ImageList = append(imageList.ImageList, meta)
	}
	return imageList, nil
}

// ListDownloadedImagesWithUnknownStatus 只查询未检测过的图片（integrity_status为unknown或NULL）
func (s *DB) ListDownloadedImagesWithUnknownStatus(source string, maxSize int) (*models.ImageList, error) {
	rows, err := s.db.Query(`WITH filtered_images AS (
		SELECT id, tags, image_url, post_time, source_id, local_path, integrity_status
		FROM images
		WHERE source_id = $1
		AND local_path IS NOT NULL
		AND local_path != ''
		AND (integrity_status = 0 OR integrity_status IS NULL)
		ORDER BY post_time DESC
		LIMIT $2
	)
	SELECT i.id, i.tags, i.image_url, i.post_time, i.source_id, i.local_path, i.integrity_status,
		(SELECT COUNT(*) FROM filtered_images) as total_items
	FROM filtered_images i
	ORDER BY i.post_time DESC;`, source, maxSize)

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
		var status sql.NullInt16
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID, &meta.LocalPath, &status, &imageList.TotalCount)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		// 处理NULL值，NULL也视为unknown
		if status.Valid {
			meta.IntegrityStatus = models.ImageIntegrityStatus(status.Int16)
		} else {
			meta.IntegrityStatus = models.ImageIntegrityUnknown
		}
		imageList.ImageList = append(imageList.ImageList, meta)
	}
	return imageList, nil
}
func (s *DB) GetImageMeta(source string, id string) (*models.ImageMeta, error) {
	rows, err := s.db.Query(`
	SELECT id, tags, image_url, post_time, source_id, local_path, integrity_status
	FROM images
	WHERE source_id = $1
	AND id = $2;`, source, id)
	if err != nil {
		slog.Error("query failed", "error", err)
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var meta models.ImageMeta
		var status sql.NullInt16
		err = rows.Scan(&meta.ID, pq.Array(&meta.Tags), &meta.ImageURL, &meta.PostTime, &meta.SourceID, &meta.LocalPath, &status)
		if err != nil {
			slog.Error("scan failed", "error", err)
			return nil, err
		}
		// 处理NULL值，NULL也视为unknown
		if status.Valid {
			meta.IntegrityStatus = models.ImageIntegrityStatus(status.Int16)
		} else {
			meta.IntegrityStatus = models.ImageIntegrityUnknown
		}
		return &meta, nil
	}
	return nil, interfaces.ErrNotFound
}

// DeleteImageFile 删除图片文件（包括缩略图），并将local_path设置为空
func (s *DB) DeleteImageFile(source string, id string) error {
	logger := slog.With("source", source, "id", id)

	// 获取图片元数据
	meta, err := s.GetImageMeta(source, id)
	if err != nil {
		logger.Error("failed to get image meta", "error", err)
		return fmt.Errorf("failed to get image meta: %w", err)
	}

	// 如果local_path为空或nil，直接返回
	if meta.LocalPath == nil || *meta.LocalPath == "" {
		logger.Warn("local_path is empty, nothing to delete")
		return nil
	}

	// 拼接完整的文件路径
	fullPath := filepath.Join(s.app.GetAppConfig().ImageDir, *meta.LocalPath)

	// 删除主文件和所有匹配的缩略图
	if err := s.deleteImageFiles(fullPath); err != nil {
		logger.Error("failed to delete image files", "error", err)
		return fmt.Errorf("failed to delete image files: %w", err)
	}

	// 检查并刷新相关标签的封面（如果当前图片是封面）
	if err := s.resetTagCoversOfImageID(source, id); err != nil {
		logger.Error("failed to refresh tag covers", "error", err)
		// 不返回错误，继续执行
	}

	// 将local_path设置为空
	empty := ""
	meta.LocalPath = &empty
	if err := s.UpdateLocalPathForMeta(*meta); err != nil {
		logger.Error("failed to update local_path in database", "error", err)
		return fmt.Errorf("failed to update local_path: %w", err)
	}

	logger.Info("image file deleted successfully")
	return nil
}

// DeleteImageRecord 删除图片文件（包括缩略图）和数据库记录
func (s *DB) DeleteImageRecord(source string, id string) error {
	logger := slog.With("source", source, "id", id)

	// 获取图片元数据
	meta, err := s.GetImageMeta(source, id)
	if err != nil {
		logger.Error("failed to get image meta", "error", err)
		return fmt.Errorf("failed to get image meta: %w", err)
	}
	for _, tag := range meta.Tags {
		// 更新相关标签的计数（减1）
		if _, err := s.db.Exec("UPDATE tags SET count = count - 1 WHERE tag = $1 AND source_id = $2", tag, meta.SourceID); err != nil {
			logger.Error("failed to update tag counts", "error", err)
		}
		// 删除没有图片的tag
		if _, err := s.db.Exec("DELETE FROM tags WHERE count = 0"); err != nil {
			logger.Error("failed to clear empty tag", "error", err)
		}
	}
	// 如果local_path不为空，删除相关文件
	if meta.LocalPath != nil && *meta.LocalPath != "" {
		// 检查并刷新相关标签的封面（如果当前图片是封面）
		if err := s.resetTagCoversOfImageID(source, id); err != nil {
			logger.Error("failed to refresh tag covers", "error", err)
			// 不返回错误，继续执行
		}
		// 拼接完整的文件路径
		fullPath := filepath.Join(s.app.GetAppConfig().ImageDir, *meta.LocalPath)

		// 删除主文件和所有匹配的缩略图
		if err := s.deleteImageFiles(fullPath); err != nil {
			logger.Error("failed to delete image files", "error", err)
			return fmt.Errorf("failed to delete image files: %w", err)
		}
	}
	// 删除数据库记录
	_, err = s.db.Exec("DELETE FROM images WHERE source_id = $1 AND id = $2", source, id)
	if err != nil {
		logger.Error("failed to delete database record", "error", err)
		return fmt.Errorf("failed to delete database record: %w", err)
	}

	logger.Info("image record deleted successfully")
	return nil
}

// refreshTagCover 刷新标签封面，如果标签还没有封面，则从该标签的图片中选择一张作为封面
func (s *DB) refreshTagCover(source string, tag string) error {
	logger := slog.With("source", source, "tag", tag)

	// 检查标签是否已经有封面
	var existingCover sql.NullString
	err := s.db.QueryRow("SELECT cover FROM tags WHERE source_id = $1 AND tag = $2", source, tag).Scan(&existingCover)
	if err != nil {
		logger.Error("failed to query tag cover", "error", err)
		return err
	}

	// 如果已经有封面，不需要刷新
	if existingCover.Valid && existingCover.String != "" {
		logger.Debug("tag already has cover, skipping refresh")
		return nil
	}

	// 从该标签的图片中选择一张作为封面（优先选择有本地路径的图片）
	var coverImageID string
	err = s.db.QueryRow(`
		SELECT id FROM images
		WHERE source_id = $1
		AND tags @> ARRAY[$2]
		AND local_path IS NOT NULL
		AND local_path != ''
		ORDER BY post_time DESC
		LIMIT 1
	`, source, tag).Scan(&coverImageID)

	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug("no suitable image found for tag cover")
			return nil
		}
		logger.Error("failed to query cover image", "error", err)
		return err
	}

	// 更新标签封面
	_, err = s.db.Exec("UPDATE tags SET cover = $1 WHERE source_id = $2 AND tag = $3", coverImageID, source, tag)
	if err != nil {
		logger.Error("failed to update tag cover", "error", err)
		return err
	}

	logger.Info("updated tag cover", "cover_image_id", coverImageID)
	return nil
}

// resetTagCoversOfImageID 检查图片相关的所有标签，刷新封面（如果当前图片是封面）
func (s *DB) resetTagCoversOfImageID(source string, imageID string) error {
	logger := slog.With("source", source, "imageID", imageID)

	// 查询哪些标签引用了这张图片作为封面
	rows, err := s.db.Query("SELECT tag FROM tags WHERE source_id = $1 AND cover = $2", source, imageID)
	if err != nil {
		logger.Error("failed to query tags using image as cover", "error", err)
		return err
	}
	defer rows.Close()

	var tagsToRefresh []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			logger.Error("failed to scan tag", "error", err)
			continue
		}
		tagsToRefresh = append(tagsToRefresh, tag)
	}

	// 为每个需要刷新的标签选择新的封面
	for _, tag := range tagsToRefresh {
		if _, err = s.db.Exec("UPDATE tags SET cover = '' WHERE source_id = $1 AND tag = $2", source, tag); err != nil {
			logger.Error("failed to clear old tag cover", "tag", tag, "error", err)
			continue
		}
		if err := s.refreshTagCover(source, tag); err != nil {
			logger.Error("failed to refresh tag cover", "tag", tag, "error", err)
			// 继续处理其他标签，不返回错误
		}
	}

	logger.Info("refreshed tag covers", "count", len(tagsToRefresh))
	return nil
}

// SetTagCover 手动设置标签封面
func (s *DB) SetTagCover(source string, tag string, imageID string) error {
	logger := slog.With("source", source, "tag", tag, "imageID", imageID)

	// 检查图片是否存在且有本地路径
	meta, err := s.GetImageMeta(source, imageID)
	if err != nil {
		logger.Error("failed to get image meta", "error", err)
		return fmt.Errorf("failed to get image meta: %w", err)
	}

	// 检查图片是否已下载
	if meta.LocalPath == nil || *meta.LocalPath == "" {
		logger.Error("image has not been downloaded yet")
		return fmt.Errorf("image has not been downloaded yet")
	}

	// 检查图片是否包含该标签
	if !slices.Contains(meta.Tags, tag) {
		logger.Error("image does not belong to the specified tag")
		return fmt.Errorf("image does not belong to the specified tag")
	}
	// 更新标签封面（存储图片ID）
	_, err = s.db.Exec("UPDATE tags SET cover = $1 WHERE source_id = $2 AND tag = $3", imageID, source, tag)
	if err != nil {
		logger.Error("failed to set tag cover", "error", err)
		return fmt.Errorf("failed to set tag cover: %w", err)
	}

	logger.Info("tag cover set successfully")
	return nil
}

// UpdateImageIntegrityStatus 更新图片完整性状态
func (s *DB) UpdateImageIntegrityStatus(source string, id string, status models.ImageIntegrityStatus) error {
	logger := slog.With("source", source, "id", id, "status", status)

	_, err := s.db.Exec("UPDATE images SET integrity_status = $1 WHERE source_id = $2 AND id = $3",
		int16(status), source, id)
	if err != nil {
		logger.Error("failed to update integrity status", "error", err)
		return fmt.Errorf("failed to update integrity status: %w", err)
	}

	logger.Info("integrity status updated successfully")
	return nil
}

// deleteImageFiles 删除主文件和所有匹配的缩略图
func (s *DB) deleteImageFiles(fullPath string) error {
	if fullPath == "" {
		return nil
	}

	// 提取目录和文件名（不带扩展名）
	dir := filepath.Dir(fullPath)
	baseName := strings.TrimSuffix(filepath.Base(fullPath), filepath.Ext(filepath.Base(fullPath)))

	// 搜索所有匹配的文件（baseName*.*）
	pattern := filepath.Join(dir, baseName+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		slog.Error("failed to glob pattern", "pattern", pattern, "error", err)
		return err
	}

	// 删除所有匹配的文件
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			// 如果文件不存在或无法删除，记录警告但不返回错误
			if !os.IsNotExist(err) {
				slog.Warn("failed to delete file", "file", file, "error", err)
			}
		} else {
			slog.Debug("deleted file", "file", file)
		}
	}

	return nil
}
