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
