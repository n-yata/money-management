package lib

import (
	"context"
	"time"

	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ResolveUser は auth0_sub に紐づくユーザーをアトミックに取得または作成する。
// FindOneAndUpdate + upsert を使用することで、同時リクエスト時の重複ユーザー作成（race condition）を防ぐ。
func ResolveUser(ctx context.Context, db *mongo.Database, auth0Sub string) (models.User, error) {
	col := db.Collection(models.CollectionUsers)
	now := time.Now()
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var user models.User
	err := col.FindOneAndUpdate(ctx,
		bson.M{"auth0_sub": auth0Sub},
		bson.M{
			"$setOnInsert": bson.M{
				"_id":        bson.NewObjectID(),
				"auth0_sub":  auth0Sub,
				"created_at": now,
				"updated_at": now,
			},
		},
		opts,
	).Decode(&user)
	return user, err
}
