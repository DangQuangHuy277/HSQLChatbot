package db

import (
	"github.com/jmoiron/sqlx"
	"os"
)

type PgDDLLoader struct {
}

func init() {
	ddlLoaders["postgres"] = &PgDDLLoader{} // Pg driver isn't have constant for driver name. So we also hardcode here
}

func (d *PgDDLLoader) LoadDDL(db *sqlx.DB) (string, error) {
	// At this time, we will load the DDL from the init file
	// TODO: Load from the running database instead
	content, err := os.ReadFile("/home/huy/Code/Personal/KLTN/be/db/init/1-schema.sql")
	if err != nil {
		return "", err
	}
	return string(content), nil
	//cmd := exec.Command("pg_dump", "--schema-only", os.Getenv("DATABASE_URL"))
	//out, err := cmd.Output()
	//if err != nil {
	//	return "", err
	//}
	//return string(out), nil
}
