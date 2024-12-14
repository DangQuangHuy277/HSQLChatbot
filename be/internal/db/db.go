package db

import (
	"database/sql"
)

var ddlLoaders = make(map[string]DDLLoader)

type HDb struct {
	*sql.DB
	DDLLoader
}

func NewHDb(driverName, dataSourceUrl string) (*HDb, error) {
	db, err := sql.Open(driverName, dataSourceUrl)
	if err != nil {
		return nil, err
	}
	ddlLoader := ddlLoaders[driverName]

	return &HDb{db, ddlLoader}, nil
}

func (hdb *HDb) LoadDDL() (string, error) {
	return hdb.DDLLoader.LoadDDL(hdb.DB)
}

type DDLLoader interface {
	LoadDDL(*sql.DB) (string, error)
}
