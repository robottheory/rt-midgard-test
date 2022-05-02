package db

import _ "embed"

//go:embed ddl.sql
var dataDDL string

func Ddl() string {
	return dataDDL
}
