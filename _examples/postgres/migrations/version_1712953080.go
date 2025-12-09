package migrations

import (
    "database/sql"
)

type Migration1712953080 struct {
    Db *sql.DB
}

func (migration *Migration1712953080) Version() uint64 {
    return 1712953080
}

func (migration *Migration1712953080) Up() error {
    _, err := migration.Db.Exec("ALTER TABLE users RENAME COLUMN phone TO phone_num")
    return err
}

func (migration *Migration1712953080) Down() error {
    _, err := migration.Db.Exec("ALTER TABLE users RENAME COLUMN phone_num TO phone")
    return err
}
