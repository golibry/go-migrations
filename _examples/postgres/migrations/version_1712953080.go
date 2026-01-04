package migrations

import (
	"context"
	"database/sql"
	"github.com/golibry/go-migrations/migration"
)

func init() {
	migration.Register(&Migration1712953080{})
}

type Migration1712953080 struct {
}

func (migration *Migration1712953080) Version() uint64 {
	return 1712953080
}

func (migration *Migration1712953080) Up(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(ctx, "ALTER TABLE users RENAME COLUMN phone TO phone_num")
	return err
}

func (migration *Migration1712953080) Down(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(ctx, "ALTER TABLE users RENAME COLUMN phone_num TO phone")
	return err
}
