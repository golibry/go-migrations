package migrations

import (
	"context"
	"github.com/golibry/go-migrations/migration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	mongoDb := db.(*mongo.Database)
	collection := mongoDb.Collection("users")
	_, err := collection.UpdateMany(
		ctx, bson.D{}, bson.D{
			{"$rename", bson.D{{"phone", "phoneNumber"}}},
		},
	)
	return err
}

func (migration *Migration1712953080) Down(ctx context.Context, db any) error {
	mongoDb := db.(*mongo.Database)
	collection := mongoDb.Collection("users")
	_, err := collection.UpdateMany(
		ctx, bson.D{}, bson.D{
			{"$rename", bson.D{{"phoneNumber", "phone"}}},
		},
	)
	return err
}
