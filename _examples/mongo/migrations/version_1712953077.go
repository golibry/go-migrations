package migrations

import (
	"context"
	"github.com/golibry/go-migrations/migration"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	migration.Register(&Migration1712953077{})
}

type Migration1712953077 struct {
}

func (migration *Migration1712953077) Version() uint64 {
	return 1712953077
}

type user struct {
	Email    string `bson:"email"`
	Phone    string `bson:"phone"`
	FullName string `bson:"fullName"`
}

func (migration *Migration1712953077) Up(ctx context.Context, db any) error {
	var users []interface{}

	for _, u := range []user{
		{"test@test12345.com", "123456", "John Doe"},
		{"test@test123456.com", "123456", "Jane Doe"},
		{"test@test1234567.com", "123456", "Clark Kent"},
		{"test@test12345678.com", "123456", "Mia Khan"},
		{"test@test123456789.com", "123456", "Alberta Buz"},
	} {
		users = append(users, u)
	}

	mongoDb := db.(*mongo.Database)
	collection := mongoDb.Collection("users")
	_, err := collection.InsertMany(ctx, users)
	return err
}

func (migration *Migration1712953077) Down(ctx context.Context, db any) error {
	mongoDb := db.(*mongo.Database)
	collection := mongoDb.Collection("users")
	return collection.Drop(ctx)
}
