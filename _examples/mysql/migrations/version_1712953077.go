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
		ctx,
		"create table if not exists `users` (`id` integer unsigned auto_increment not null, `name` varchar(128) not null, `phone` varchar(32) not null, primary key (`id`))",
	)
	return err
}

func (migration *Migration1712953077) Down(ctx context.Context, db any) error {
	_, err := db.(*sql.DB).ExecContext(ctx, "drop table if exists `users`")
	return err
}
