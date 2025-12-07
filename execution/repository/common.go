package repository

import "database/sql"

func newDbHandle(dsn, driverName string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dsn)

	if db == nil {
		return nil, err
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
	return db, err
}
