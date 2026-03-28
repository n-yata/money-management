package lib

import (
	"fmt"
	"os"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const DBName = "money-management"

var (
	clientOnce  sync.Once
	mongoClient *mongo.Client
	mongoErr    error
)

// GetClient はMongoDBクライアントをシングルトンで返す。
// Lambdaのグローバルスコープでコネクションを再利用し、コールドスタートコストを最小化する。
// 接続エラーが発生した場合、同一Lambdaコンテナ内では回復しない（sync.Once の制約）。
// 一時的な接続エラーは Lambda コンテナの自然な再起動（デフォルト数分〜数時間）で回復することを期待する。
func GetClient() (*mongo.Client, error) {
	clientOnce.Do(func() {
		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			mongoErr = fmt.Errorf("MONGODB_URI is not set")
			return
		}
		mongoClient, mongoErr = mongo.Connect(options.Client().ApplyURI(uri))
	})
	return mongoClient, mongoErr
}

// GetDB はMongoDBデータベースを返す。
func GetDB() (*mongo.Database, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return client.Database(DBName), nil
}
