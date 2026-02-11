package embed

import (
	"embed"
	_ "embed"
	"io/fs"
	"log/slog"
)

//go:embed res/dbinit.sql
var InitSql string

//go:embed res/migration_v1.sql
var MigrationV1Sql string

//go:embed www
var webContent embed.FS

// WebContent 是 frontend 构建产物（已剥离 www 前缀）
var WebContent fs.FS

func init() {
	var err error
	WebContent, err = fs.Sub(webContent, "www")
	if err != nil {
		slog.Error("failed to create sub fs", "error", err)
	}
}
