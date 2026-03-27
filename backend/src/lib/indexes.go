package lib

import (
	"context"

	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes は全コレクションに必要なインデックスを作成する。
// 冪等なので複数回呼び出しても安全。
func EnsureIndexes(ctx context.Context) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	if err := ensureUsersIndexes(ctx, db); err != nil {
		return err
	}
	if err := ensureChildrenIndexes(ctx, db); err != nil {
		return err
	}
	if err := ensureAllowanceTypesIndexes(ctx, db); err != nil {
		return err
	}
	return ensureRecordsIndexes(ctx, db)
}

// users: auth0_sub に unique index
func ensureUsersIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(models.CollectionUsers)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "auth0_sub", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// children: user_id に index
func ensureChildrenIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(models.CollectionChildren)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
	return err
}

// allowance_types: user_id に index
func ensureAllowanceTypesIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(models.CollectionAllowanceTypes)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
	return err
}

// records: child_id + date に複合 index
func ensureRecordsIndexes(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(models.CollectionRecords)
	_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "child_id", Value: 1},
			{Key: "date", Value: -1},
		},
	})
	return err
}
