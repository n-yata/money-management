package lib

import (
	"context"

	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// FindOwnedChild は指定IDの子どもを取得し、ログインユーザーの所有であることを確認する。
// 存在しない、他ユーザーのリソース、不正なID形式の場合は (_, false, nil) を返す（404扱い）。
func FindOwnedChild(ctx context.Context, db *mongo.Database, userID bson.ObjectID, idStr string) (models.Child, bool, error) {
	if idStr == "" {
		return models.Child{}, false, nil
	}
	oid, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		return models.Child{}, false, nil
	}

	col := db.Collection(models.CollectionChildren)
	var child models.Child
	err = col.FindOne(ctx, bson.M{"_id": oid, "user_id": userID}).Decode(&child)
	if err == mongo.ErrNoDocuments {
		return models.Child{}, false, nil
	}
	if err != nil {
		return models.Child{}, false, err
	}
	return child, true, nil
}
