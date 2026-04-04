//go:build integration

package lib_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var integrationMongoURI string

func TestMain(m *testing.M) {
	ctx := context.Background()
	// WithReplicaSet を使用してシングルノードレプリカセットを起動する（トランザクション対応）
	container, err := mongodb.Run(ctx, "mongo:6.0", mongodb.WithReplicaSet("rs0"))
	if err != nil {
		log.Fatalf("MongoDBコンテナ起動失敗: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("コンテナ終了失敗: %v", err)
		}
	}()

	integrationMongoURI, err = container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("接続URI取得失敗: %v", err)
	}
	// Docker Desktop（Windows）ではレプリカセットの内部IPに接続できないため
	// directConnection=true でトポロジー探索をスキップする
	integrationMongoURI += "&directConnection=true"

	// EnsureIndexes のテストで GetDB() が機能するよう環境変数をセットする
	if err := os.Setenv("MONGODB_URI", integrationMongoURI); err != nil {
		log.Fatalf("MONGODB_URI の環境変数セット失敗: %v", err)
	}

	os.Exit(m.Run())
}

// newIntegrationDB はテストごとに独立したDBを返す。
func newIntegrationDB(t *testing.T) *mongo.Database {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(integrationMongoURI))
	if err != nil {
		t.Fatalf("MongoDB接続失敗: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	// テストごとに一意なDB名で独立性を確保
	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())
	return client.Database(dbName)
}
