package embed

import (
	"embed"
	_ "embed"
)

//go:embed res/dbinit.sql
var InitSql string

//go:embed res/migration_v1.sql
var MigrationV1Sql string

//go:embed www
var WebContent embed.FS
