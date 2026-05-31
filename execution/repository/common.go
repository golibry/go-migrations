package repository

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
)

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

func quoteSQLTableName(tableName string, quoteIdentifier func(string) string) (string, error) {
	parts := strings.Split(tableName, ".")
	quotedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || strings.ContainsRune(part, 0) {
			return "", fmt.Errorf("invalid table name %q", tableName)
		}
		quotedParts = append(quotedParts, quoteIdentifier(part))
	}

	return strings.Join(quotedParts, "."), nil
}

func advisoryLockID(lockName string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(lockName))
	return int64(hash.Sum64())
}
