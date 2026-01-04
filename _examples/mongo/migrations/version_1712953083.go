package migrations

import (
	"context"
	"github.com/golibry/go-migrations/migration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
)

func init() {
	migration.Register(&Migration1712953083{})
}

const FullNameSplitLock = "full-name-split-lock"

type userWithNewPhone struct {
	Email    string `bson:"email"`
	Phone    string `bson:"phoneNumber"`
	FullName string `bson:"fullName"`
}

type userWithFullNameSplit struct {
	Email     string `bson:"email"`
	Phone     string `bson:"phoneNumber"`
	FirstName string `bson:"firstName"`
	LastName  string `bson:"lastName"`
}

type Migration1712953083 struct {
}

func (migration *Migration1712953083) Version() uint64 {
	return 1712953083
}

type runChange func(session mongo.Session) error

func (migration *Migration1712953083) lockAndRunChange(
	ctx context.Context,
	db *mongo.Database,
	run runChange,
) error {
	client := db.Client()
	session, err := client.StartSession()

	if err != nil {
		return err
	}

	err = session.StartTransaction()

	if err != nil {
		return err
	}

	locksCollection := db.Collection("locks")

	// Obtain collection lock for update
	locksCollection.FindOneAndUpdate(
		ctx,
		bson.D{{"lockName", FullNameSplitLock}},
		bson.D{
			{"$set", bson.D{{"randVal", primitive.NewObjectID()}}},
		},
	)

	err = run(session)
	_, _ = locksCollection.DeleteOne(ctx, bson.D{{"lockName", FullNameSplitLock}})

	if err != nil {
		_ = session.AbortTransaction(ctx)
		return err
	}

	if err = session.CommitTransaction(ctx); err != nil {
		_ = session.AbortTransaction(ctx)
		return err
	}

	return nil
}

func (migration *Migration1712953083) Up(ctx context.Context, db any) error {
	mongoDb := db.(*mongo.Database)
	return migration.lockAndRunChange(
		ctx, mongoDb,
		func(session mongo.Session) error {
			usersCollection := mongoDb.Collection("users")
			usersCursor, err := usersCollection.Find(
				ctx,
				bson.D{},
			)

			if usersCursor == nil {
				return nil
			}

			var results []userWithNewPhone
			err = usersCursor.All(ctx, &results)

			if err != nil {
				return err
			}

			for _, userToChange := range results {
				nameSplit := strings.Split(userToChange.FullName, " ")
				changedUser := userWithFullNameSplit{
					Email:     userToChange.Email,
					Phone:     userToChange.Phone,
					FirstName: nameSplit[0],
					LastName:  nameSplit[1],
				}
				_, err = usersCollection.ReplaceOne(
					ctx,
					bson.D{{"email", userToChange.Email}},
					changedUser,
				)

				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (migration *Migration1712953083) Down(ctx context.Context, db any) error {
	mongoDb := db.(*mongo.Database)
	return migration.lockAndRunChange(
		ctx, mongoDb,
		func(session mongo.Session) error {
			usersCollection := mongoDb.Collection("users")
			usersCursor, err := usersCollection.Find(
				ctx,
				bson.D{},
			)

			if usersCursor == nil {
				return nil
			}

			var results []userWithFullNameSplit
			err = usersCursor.All(ctx, &results)

			if err != nil {
				return err
			}

			for _, userToChange := range results {
				fullName := userToChange.FirstName + " " + userToChange.LastName
				changedUser := user{
					Email:    userToChange.Email,
					Phone:    userToChange.Phone,
					FullName: fullName,
				}
				_, err = usersCollection.ReplaceOne(
					ctx,
					bson.D{{"email", userToChange.Email}},
					changedUser,
				)

				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}
