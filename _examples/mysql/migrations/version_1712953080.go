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
	_, err := db.(*sql.DB).ExecContext(
		ctx,
		"alter table `users` change `phone` `phone_num` varchar(64) not null",
	)
	return err
}

func (migration *Migration1712953080) Down(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(
		ctx,
		"alter table `users` change `phone_num` `phone` varchar(32) not null",
	)
	return err
}
