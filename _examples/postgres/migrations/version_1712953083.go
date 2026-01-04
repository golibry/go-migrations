package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/golibry/go-migrations/migration"
)

func init() {
	migration.Register(&Migration1712953083{})
}

type Migration1712953083 struct {
}

func (migration *Migration1712953083) Version() uint64 {
	return 1712953083
}

func (migration *Migration1712953083) Up(ctx context.Context, db any) error {
	sqlDb := db.(*sql.DB)
	tx, err := sqlDb.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO users (name, phone_num) VALUES ('Alex', '1234'), ('Jada', '4567'), ('Tia', '7890')",
	)

	if err != nil {
		errRollback := tx.Rollback()

		if errRollback != nil {
			return fmt.Errorf("%w, with rollback error: %w", err, errRollback)
		}
		return err
	}

	return tx.Commit()
}

func (migration *Migration1712953083) Down(ctx context.Context, db any) error {
	sqlDb := db.(*sql.DB)
	tx, err := sqlDb.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		"DELETE FROM users WHERE name IN ('Alex', 'Jada', 'Tia')",
	)

	if err != nil {
		errRollback := tx.Rollback()

		if errRollback != nil {
			return fmt.Errorf("%w, with rollback error: %w", err, errRollback)
		}
		return err
	}

	return tx.Commit()
}
