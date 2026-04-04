//go:build integration

package lib_test

import (
	"context"
	"testing"

	"github.com/n-yata/money-management/backend/src/lib"
	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// indexExists は指定コレクションに指定フィールドのインデックスが存在するか確認する。
func indexExists(t *testing.T, ctx context.Context, db *mongo.Database, collection string, fieldName string) bool {
	t.Helper()
	cursor, err := db.Collection(collection).Indexes().List(ctx)
	if err != nil {
		t.Fatalf("インデックス一覧取得失敗: %v", err)
	}
	defer cursor.Close(ctx)

	type indexInfo struct {
		Key map[string]interface{} `bson:"key"`
	}
	var indexes []indexInfo
	if err := cursor.All(ctx, &indexes); err != nil {
		t.Fatalf("インデックスのデコード失敗: %v", err)
	}
	for _, idx := range indexes {
		if _, ok := idx.Key[fieldName]; ok {
			return true
		}
	}
	return false
}

func TestEnsureIndexes(t *testing.T) {
	ctx := context.Background()

	// EnsureIndexes が使う DB（lib.DBName = "money-management"）に直接接続する
	client, err := mongo.Connect(options.Client().ApplyURI(integrationMongoURI))
	if err != nil {
		t.Fatalf("MongoDB接続失敗: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	db := client.Database(lib.DBName)

	t.Run("EnsureIndexes呼び出し後にchildrenコレクションのuser_idインデックスが存在する", func(t *testing.T) {
		if err := lib.EnsureIndexes(ctx); err != nil {
			t.Fatalf("EnsureIndexes() error = %v", err)
		}

		if !indexExists(t, ctx, db, models.CollectionChildren, "user_id") {
			t.Errorf("childrenコレクションに user_id インデックスが存在しない")
		}
	})

	t.Run("EnsureIndexesを複数回呼んでもエラーにならない（冪等性）", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			if err := lib.EnsureIndexes(ctx); err != nil {
				t.Errorf("EnsureIndexes() %d回目 error = %v", i+1, err)
			}
		}
	})
}
