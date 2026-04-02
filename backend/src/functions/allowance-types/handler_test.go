//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/n-yata/money-management/backend/src/models"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ─── テストセットアップ ──────────────────────────────────────

var mongoURI string

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

	mongoURI, err = container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("接続URI取得失敗: %v", err)
	}
	// Docker Desktop（Windows）ではレプリカセットの内部IPに接続できないため
	// directConnection=true でトポロジー探索をスキップする
	mongoURI += "&directConnection=true"

	os.Exit(m.Run())
}

func newTestDB(t *testing.T) *mongo.Database {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("MongoDB接続失敗: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())
	return client.Database(dbName)
}

func newTestUser(t *testing.T, ctx context.Context, db *mongo.Database) models.User {
	t.Helper()
	now := time.Now()
	user := models.User{
		ID:        bson.NewObjectID(),
		Auth0Sub:  "auth0|" + bson.NewObjectID().Hex(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := db.Collection(models.CollectionUsers).InsertOne(ctx, user); err != nil {
		t.Fatalf("テストユーザー作成失敗: %v", err)
	}
	return user
}

func insertAllowanceType(t *testing.T, ctx context.Context, db *mongo.Database, userID bson.ObjectID, name string, amount int64) models.AllowanceType {
	t.Helper()
	now := time.Now()
	at := models.AllowanceType{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		Name:      name,
		Amount:    amount,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := db.Collection(models.CollectionAllowanceTypes).InsertOne(ctx, at); err != nil {
		t.Fatalf("テスト種類作成失敗: %v", err)
	}
	return at
}

func insertRecord(t *testing.T, ctx context.Context, db *mongo.Database, childID bson.ObjectID, allowanceTypeID *bson.ObjectID) models.Record {
	t.Helper()
	record := models.Record{
		ID:              bson.NewObjectID(),
		ChildID:         childID,
		AllowanceTypeID: allowanceTypeID,
		Type:            models.RecordTypeIncome,
		Amount:          100,
		Date:            time.Now(),
		CreatedAt:       time.Now(),
	}
	if _, err := db.Collection(models.CollectionRecords).InsertOne(ctx, record); err != nil {
		t.Fatalf("テスト収支記録作成失敗: %v", err)
	}
	return record
}

// ─── listAllowanceTypes のテスト ─────────────────────────────

func TestListAllowanceTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("種類がない場合は空配列を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := listAllowanceTypes(ctx, db, user)
		if err != nil {
			t.Fatalf("listAllowanceTypes() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var body map[string][]models.AllowanceType
		if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		if len(body["data"]) != 0 {
			t.Errorf("len(data) = %d, want 0", len(body["data"]))
		}
	})

	t.Run("自分の種類一覧が返される", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)
		insertAllowanceType(t, ctx, db, user.ID, "掃除機", 100)

		resp, err := listAllowanceTypes(ctx, db, user)
		if err != nil {
			t.Fatalf("listAllowanceTypes() error = %v", err)
		}

		var body map[string][]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &body)
		if len(body["data"]) != 2 {
			t.Errorf("len(data) = %d, want 2", len(body["data"]))
		}
	})

	t.Run("他のユーザーの種類は含まれない", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		insertAllowanceType(t, ctx, db, user2.ID, "他人の種類", 50)

		resp, err := listAllowanceTypes(ctx, db, user1)
		if err != nil {
			t.Fatalf("listAllowanceTypes() error = %v", err)
		}

		var body map[string][]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &body)
		if len(body["data"]) != 0 {
			t.Errorf("他ユーザーの種類が含まれている: len = %d", len(body["data"]))
		}
	})
}

// ─── getAllowanceType のテスト ────────────────────────────────

func TestGetAllowanceType(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に種類を1件取得できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)

		resp, err := getAllowanceType(ctx, db, user, at.ID.Hex())
		if err != nil {
			t.Fatalf("getAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &result)
		got := result["data"]
		if got.Name != "お皿洗い" {
			t.Errorf("Name = %q, want %q", got.Name, "お皿洗い")
		}
		if got.Amount != 50 {
			t.Errorf("Amount = %d, want 50", got.Amount)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := getAllowanceType(ctx, db, user, bson.NewObjectID().Hex())
		if err != nil {
			t.Fatalf("getAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("他ユーザーの種類は404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user2.ID, "他人の種類", 100)

		resp, err := getAllowanceType(ctx, db, user1, at.ID.Hex())
		if err != nil {
			t.Fatalf("getAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ─── createAllowanceType のテスト ────────────────────────────

func TestCreateAllowanceType(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に種類を登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"お皿洗い","amount":50}`

		resp, err := createAllowanceType(ctx, db, user, body)
		if err != nil {
			t.Fatalf("createAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		var result map[string]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &result)
		got := result["data"]
		if got.Name != "お皿洗い" {
			t.Errorf("Name = %q, want %q", got.Name, "お皿洗い")
		}
		if got.Amount != 50 {
			t.Errorf("Amount = %d, want 50", got.Amount)
		}
	})

	t.Run("種類名の前後スペースがトリミングされる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"  掃除機  ","amount":100}`

		resp, err := createAllowanceType(ctx, db, user, body)
		if err != nil {
			t.Fatalf("createAllowanceType() error = %v", err)
		}

		var result map[string]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &result)
		if result["data"].Name != "掃除機" {
			t.Errorf("Name = %q, want %q", result["data"].Name, "掃除機")
		}
	})

	t.Run("バリデーションエラー（種類名なし）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := createAllowanceType(ctx, db, user, `{"name":"","amount":50}`)
		if err != nil {
			t.Fatalf("createAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("バリデーションエラー（金額ゼロ）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := createAllowanceType(ctx, db, user, `{"name":"お皿洗い","amount":0}`)
		if err != nil {
			t.Fatalf("createAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("不正なJSONはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := createAllowanceType(ctx, db, user, "invalid json")
		if err != nil {
			t.Fatalf("createAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

// ─── updateAllowanceType のテスト ────────────────────────────

func TestUpdateAllowanceType(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に種類を更新できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)

		resp, err := updateAllowanceType(ctx, db, user, at.ID.Hex(), `{"name":"食器洗い","amount":80}`)
		if err != nil {
			t.Fatalf("updateAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]models.AllowanceType
		json.Unmarshal([]byte(resp.Body), &result)
		got := result["data"]
		if got.Name != "食器洗い" {
			t.Errorf("Name = %q, want %q", got.Name, "食器洗い")
		}
		if got.Amount != 80 {
			t.Errorf("Amount = %d, want 80", got.Amount)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := updateAllowanceType(ctx, db, user, bson.NewObjectID().Hex(), `{"name":"お皿洗い","amount":50}`)
		if err != nil {
			t.Fatalf("updateAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("バリデーションエラーはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)

		resp, err := updateAllowanceType(ctx, db, user, at.ID.Hex(), `{"name":"","amount":50}`)
		if err != nil {
			t.Fatalf("updateAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("他ユーザーの種類は404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user2.ID, "他人の種類", 50)

		resp, err := updateAllowanceType(ctx, db, user1, at.ID.Hex(), `{"name":"改ざん","amount":50}`)
		if err != nil {
			t.Fatalf("updateAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ─── deleteAllowanceType のテスト ────────────────────────────

func TestDeleteAllowanceType(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に種類を削除できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)

		resp, err := deleteAllowanceType(ctx, db, user, at.ID.Hex())
		if err != nil {
			t.Fatalf("deleteAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// 削除後は取得できない
		count, _ := db.Collection(models.CollectionAllowanceTypes).CountDocuments(ctx, bson.M{"_id": at.ID})
		if count != 0 {
			t.Errorf("削除後のドキュメント件数 = %d, want 0", count)
		}
	})

	t.Run("関連recordsのallowance_type_idがnullになる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user.ID, "お皿洗い", 50)
		childID := bson.NewObjectID()
		atID := at.ID
		record := insertRecord(t, ctx, db, childID, &atID)

		resp, err := deleteAllowanceType(ctx, db, user, at.ID.Hex())
		if err != nil {
			t.Fatalf("deleteAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// recordのallowance_type_idがnullになっているか確認
		var updated models.Record
		err = db.Collection(models.CollectionRecords).FindOne(ctx, bson.M{"_id": record.ID}).Decode(&updated)
		if err != nil {
			t.Fatalf("record取得失敗: %v", err)
		}
		if updated.AllowanceTypeID != nil {
			t.Errorf("AllowanceTypeID = %v, want nil", updated.AllowanceTypeID)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := deleteAllowanceType(ctx, db, user, bson.NewObjectID().Hex())
		if err != nil {
			t.Fatalf("deleteAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("他ユーザーの種類は404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		at := insertAllowanceType(t, ctx, db, user2.ID, "他人の種類", 50)

		resp, err := deleteAllowanceType(ctx, db, user1, at.ID.Hex())
		if err != nil {
			t.Fatalf("deleteAllowanceType() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}

		// 他ユーザーの種類が削除されていないこと
		count, _ := db.Collection(models.CollectionAllowanceTypes).CountDocuments(ctx, bson.M{"_id": at.ID})
		if count != 1 {
			t.Errorf("他ユーザーの種類が誤削除された")
		}
	})
}
