package embed

import _ "embed"

//go:embed res/dbinit.sql
var InitSql string
