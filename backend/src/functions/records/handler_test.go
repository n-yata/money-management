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

func insertChild(t *testing.T, ctx context.Context, db *mongo.Database, userID bson.ObjectID) models.Child {
	t.Helper()
	now := time.Now()
	child := models.Child{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		Name:      "テスト子ども",
		Age:       8,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := db.Collection(models.CollectionChildren).InsertOne(ctx, child); err != nil {
		t.Fatalf("テスト子ども作成失敗: %v", err)
	}
	return child
}

func insertAllowanceType(t *testing.T, ctx context.Context, db *mongo.Database, userID bson.ObjectID) models.AllowanceType {
	t.Helper()
	now := time.Now()
	at := models.AllowanceType{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		Name:      "お皿洗い",
		Amount:    50,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := db.Collection(models.CollectionAllowanceTypes).InsertOne(ctx, at); err != nil {
		t.Fatalf("テスト種類作成失敗: %v", err)
	}
	return at
}

func insertRecord(t *testing.T, ctx context.Context, db *mongo.Database, childID bson.ObjectID, recType string, amount int64, date time.Time) models.Record {
	t.Helper()
	record := models.Record{
		ID:          bson.NewObjectID(),
		ChildID:     childID,
		Type:        recType,
		Amount:      amount,
		Description: "テスト",
		Date:        date,
		CreatedAt:   time.Now(),
	}
	if _, err := db.Collection(models.CollectionRecords).InsertOne(ctx, record); err != nil {
		t.Fatalf("テスト収支記録作成失敗: %v", err)
	}
	return record
}

// ─── listRecords のテスト ────────────────────────────────────

func TestListRecords(t *testing.T) {
	ctx := context.Background()

	t.Run("指定月の記録が日付降順で返される", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		mar01 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		mar15 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 500, mar01)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeExpense, 200, mar15)

		resp, err := listRecords(ctx, db, user, child.ID.Hex(), map[string]string{"year": "2026", "month": "3"})
		if err != nil {
			t.Fatalf("listRecords() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var body map[string][]models.Record
		json.Unmarshal([]byte(resp.Body), &body)
		records := body["data"]
		if len(records) != 2 {
			t.Fatalf("len(records) = %d, want 2", len(records))
		}
		// 日付降順: mar15 が先
		if records[0].Date.Before(records[1].Date) {
			t.Errorf("日付降順になっていない")
		}
	})

	t.Run("他月の記録は含まれない", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 500, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 300, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))

		resp, err := listRecords(ctx, db, user, child.ID.Hex(), map[string]string{"year": "2026", "month": "3"})
		if err != nil {
			t.Fatalf("listRecords() error = %v", err)
		}

		var body map[string][]models.Record
		json.Unmarshal([]byte(resp.Body), &body)
		if len(body["data"]) != 1 {
			t.Errorf("len(records) = %d, want 1（3月のみ）", len(body["data"]))
		}
	})

	t.Run("year・monthなしはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		resp, err := listRecords(ctx, db, user, child.ID.Hex(), map[string]string{})
		if err != nil {
			t.Fatalf("listRecords() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("他ユーザーの子どもは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID)

		resp, err := listRecords(ctx, db, user1, child.ID.Hex(), map[string]string{"year": "2026", "month": "3"})
		if err != nil {
			t.Fatalf("listRecords() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ─── createRecord のテスト ───────────────────────────────────

func TestCreateRecord(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に収支記録を登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		body := `{"type":"income","amount":500,"description":"お皿洗い","date":"2026-03-01"}`

		resp, err := createRecord(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		var result map[string]models.Record
		json.Unmarshal([]byte(resp.Body), &result)
		got := result["data"]
		if got.Type != "income" {
			t.Errorf("Type = %q, want %q", got.Type, "income")
		}
		if got.Amount != 500 {
			t.Errorf("Amount = %d, want 500", got.Amount)
		}
		if got.AllowanceTypeID != nil {
			t.Errorf("AllowanceTypeID = %v, want nil", got.AllowanceTypeID)
		}
	})

	t.Run("allowance_type_idが自分の種類なら登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		at := insertAllowanceType(t, ctx, db, user.ID)
		body := fmt.Sprintf(`{"type":"income","amount":50,"description":"お皿洗い","date":"2026-03-01","allowance_type_id":"%s"}`, at.ID.Hex())

		resp, err := createRecord(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		var result map[string]models.Record
		json.Unmarshal([]byte(resp.Body), &result)
		if result["data"].AllowanceTypeID == nil {
			t.Error("AllowanceTypeID = nil, want non-nil")
		}
	})

	t.Run("他ユーザーのallowance_type_idはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user1.ID)
		at := insertAllowanceType(t, ctx, db, user2.ID) // user2の種類
		body := fmt.Sprintf(`{"type":"income","amount":50,"description":"テスト","date":"2026-03-01","allowance_type_id":"%s"}`, at.ID.Hex())

		resp, err := createRecord(ctx, db, user1, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("バリデーションエラー（typeが不正）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		resp, err := createRecord(ctx, db, user, child.ID.Hex(), `{"type":"invalid","amount":100,"description":"テスト","date":"2026-03-01"}`)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("他ユーザーの子どもは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID)

		resp, err := createRecord(ctx, db, user1, child.ID.Hex(), `{"type":"income","amount":100,"description":"テスト","date":"2026-03-01"}`)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("同じ日・同じ種類のお手伝いは重複登録できない（409 Conflict）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		at := insertAllowanceType(t, ctx, db, user.ID)
		body := fmt.Sprintf(`{"type":"income","amount":50,"description":"お皿洗い","date":"2026-03-01","allowance_type_id":"%s"}`, at.ID.Hex())

		// 1回目は成功
		resp, err := createRecord(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("1回目 StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		// 2回目は重複エラー
		resp, err = createRecord(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("2回目 StatusCode = %d, want %d", resp.StatusCode, http.StatusConflict)
		}
	})

	t.Run("別の日なら同じ種類でも登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		at := insertAllowanceType(t, ctx, db, user.ID)

		body1 := fmt.Sprintf(`{"type":"income","amount":50,"description":"お皿洗い","date":"2026-03-01","allowance_type_id":"%s"}`, at.ID.Hex())
		resp, err := createRecord(ctx, db, user, child.ID.Hex(), body1)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("1日目 StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		body2 := fmt.Sprintf(`{"type":"income","amount":50,"description":"お皿洗い","date":"2026-03-02","allowance_type_id":"%s"}`, at.ID.Hex())
		resp, err = createRecord(ctx, db, user, child.ID.Hex(), body2)
		if err != nil {
			t.Fatalf("createRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("翌日 StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}
	})

	t.Run("allowance_type_idなしの記録は同日複数登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		body := `{"type":"income","amount":500,"description":"手動記録","date":"2026-03-01"}`

		for i := 0; i < 2; i++ {
			resp, err := createRecord(ctx, db, user, child.ID.Hex(), body)
			if err != nil {
				t.Fatalf("createRecord() error = %v", err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("登録%d回目 StatusCode = %d, want %d", i+1, resp.StatusCode, http.StatusCreated)
			}
		}
	})
}

// ─── deleteRecord のテスト ───────────────────────────────────

func TestDeleteRecord(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に収支記録を削除できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)
		record := insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 500, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

		resp, err := deleteRecord(ctx, db, user, child.ID.Hex(), record.ID.Hex())
		if err != nil {
			t.Fatalf("deleteRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		count, _ := db.Collection(models.CollectionRecords).CountDocuments(ctx, bson.M{"_id": record.ID})
		if count != 0 {
			t.Errorf("削除後のrecord件数 = %d, want 0", count)
		}
	})

	t.Run("他ユーザーの子どもに紐づくrecordは削除できない（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID)
		record := insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 500, time.Now())

		resp, err := deleteRecord(ctx, db, user1, child.ID.Hex(), record.ID.Hex())
		if err != nil {
			t.Fatalf("deleteRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("存在しないrecordIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		resp, err := deleteRecord(ctx, db, user, child.ID.Hex(), bson.NewObjectID().Hex())
		if err != nil {
			t.Fatalf("deleteRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("不正なrecordIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID)

		resp, err := deleteRecord(ctx, db, user, child.ID.Hex(), "invalid-id")
		if err != nil {
			t.Fatalf("deleteRecord() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}
