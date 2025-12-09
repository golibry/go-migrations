package migrations

import (
    "database/sql"
)

type Migration1712953077 struct {
    Db *sql.DB
}

func (migration *Migration1712953077) Version() uint64 {
    return 1712953077
}

func (migration *Migration1712953077) Up() error {
    _, err := migration.Db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(128) NOT NULL,
            phone VARCHAR(32) NOT NULL
        )
    `)
    return err
}

func (migration *Migration1712953077) Down() error {
    _, err := migration.Db.Exec("DROP TABLE IF EXISTS users")
    return err
}
