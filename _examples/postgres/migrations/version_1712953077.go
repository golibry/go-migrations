package migrations

import (
	"context"
	"database/sql"
	"github.com/golibry/go-migrations/migration"
)

func init() {
	migration.Register(&Migration1712953077{})
}

type Migration1712953077 struct {
}

func (migration *Migration1712953077) Version() uint64 {
	return 1712953077
}

func (migration *Migration1712953077) Up(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(
		ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(128) NOT NULL,
            phone VARCHAR(32) NOT NULL
        )
    `,
	)
	return err
}

func (migration *Migration1712953077) Down(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(ctx, "DROP TABLE IF EXISTS users")
	return err
}
