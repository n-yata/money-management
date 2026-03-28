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

	"github.com/n-yata/money-management/backend/src/lib"
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

// newTestDB はテストごとに独立したDBを返す。
func newTestDB(t *testing.T) *mongo.Database {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
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

// newTestUser はテスト用ユーザーをDBに挿入して返す。
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

// insertChild はテスト用子どもをDBに挿入して返す。
func insertChild(t *testing.T, ctx context.Context, db *mongo.Database, userID bson.ObjectID, name string, age int, baseAllowance int64) models.Child {
	t.Helper()
	now := time.Now()
	child := models.Child{
		ID:            bson.NewObjectID(),
		UserID:        userID,
		Name:          name,
		Age:           age,
		BaseAllowance: baseAllowance,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := db.Collection(models.CollectionChildren).InsertOne(ctx, child); err != nil {
		t.Fatalf("テスト子ども作成失敗: %v", err)
	}
	return child
}

// insertRecord はテスト用収支記録をDBに挿入する。
func insertRecord(t *testing.T, ctx context.Context, db *mongo.Database, childID bson.ObjectID, recType string, amount int64) {
	t.Helper()
	record := models.Record{
		ID:        bson.NewObjectID(),
		ChildID:   childID,
		Type:      recType,
		Amount:    amount,
		Date:      time.Now(),
		CreatedAt: time.Now(),
	}
	if _, err := db.Collection(models.CollectionRecords).InsertOne(ctx, record); err != nil {
		t.Fatalf("テスト収支記録作成失敗: %v", err)
	}
}

// ─── resolveUser のテスト ────────────────────────────────────

func TestResolveUser(t *testing.T) {
	ctx := context.Background()

	t.Run("新規ユーザーが自動作成される", func(t *testing.T) {
		db := newTestDB(t)
		auth0Sub := "auth0|new-user"

		user, err := lib.ResolveUser(ctx, db, auth0Sub)
		if err != nil {
			t.Fatalf("lib.ResolveUser() error = %v", err)
		}
		if user.Auth0Sub != auth0Sub {
			t.Errorf("user.Auth0Sub = %q, want %q", user.Auth0Sub, auth0Sub)
		}
		if user.ID.IsZero() {
			t.Error("user.ID should not be zero")
		}
	})

	t.Run("既存ユーザーが返される", func(t *testing.T) {
		db := newTestDB(t)
		existing := newTestUser(t, ctx, db)

		user, err := lib.ResolveUser(ctx, db, existing.Auth0Sub)
		if err != nil {
			t.Fatalf("lib.ResolveUser() error = %v", err)
		}
		if user.ID != existing.ID {
			t.Errorf("user.ID = %v, want %v", user.ID, existing.ID)
		}
	})

	t.Run("同じsubで2回呼んでも重複作成されない", func(t *testing.T) {
		db := newTestDB(t)
		auth0Sub := "auth0|idempotent-user"

		user1, err := lib.ResolveUser(ctx, db, auth0Sub)
		if err != nil {
			t.Fatalf("1回目: lib.ResolveUser() error = %v", err)
		}
		user2, err := lib.ResolveUser(ctx, db, auth0Sub)
		if err != nil {
			t.Fatalf("2回目: lib.ResolveUser() error = %v", err)
		}
		if user1.ID != user2.ID {
			t.Errorf("IDが一致しない: user1.ID = %v, user2.ID = %v", user1.ID, user2.ID)
		}
	})
}

// ─── listChildren のテスト ───────────────────────────────────

func TestListChildren(t *testing.T) {
	ctx := context.Background()

	t.Run("子どもがいない場合は空配列を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := listChildren(ctx, db, user)
		if err != nil {
			t.Fatalf("listChildren() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		var children []models.ChildResponse
		if err := json.Unmarshal(body["data"], &children); err != nil {
			t.Fatalf("dataフィールドのパース失敗: %v", err)
		}
		if len(children) != 0 {
			t.Errorf("len(children) = %d, want 0", len(children))
		}
	})

	t.Run("自分の子どもが残高付きで返される", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "たろう", 8, 1000)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 500)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeExpense, 200)

		resp, err := listChildren(ctx, db, user)
		if err != nil {
			t.Fatalf("listChildren() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var body map[string][]models.ChildResponse
		if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		children := body["data"]
		if len(children) != 1 {
			t.Fatalf("len(children) = %d, want 1", len(children))
		}
		if children[0].Name != "たろう" {
			t.Errorf("Name = %q, want %q", children[0].Name, "たろう")
		}
		if children[0].Balance != 300 {
			t.Errorf("Balance = %d, want 300", children[0].Balance)
		}
	})

	t.Run("他のユーザーの子どもは含まれない", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		insertChild(t, ctx, db, user2.ID, "他人の子", 10, 2000)

		resp, err := listChildren(ctx, db, user1)
		if err != nil {
			t.Fatalf("listChildren() error = %v", err)
		}

		var body map[string][]models.ChildResponse
		if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		if len(body["data"]) != 0 {
			t.Errorf("他ユーザーの子どもが含まれている: len = %d", len(body["data"]))
		}
	})
}

// ─── createChild のテスト ────────────────────────────────────

func TestCreateChild(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に子どもを登録できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"はなこ","age":10,"base_allowance":2000}`

		resp, err := createChild(ctx, db, user, body)
		if err != nil {
			t.Fatalf("createChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
		}

		var result map[string]models.ChildResponse
		if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
			t.Fatalf("レスポンスのパース失敗: %v", err)
		}
		child := result["data"]
		if child.Name != "はなこ" {
			t.Errorf("Name = %q, want %q", child.Name, "はなこ")
		}
		if child.Age != 10 {
			t.Errorf("Age = %d, want 10", child.Age)
		}
		if child.BaseAllowance != 2000 {
			t.Errorf("BaseAllowance = %d, want 2000", child.BaseAllowance)
		}
		if child.Balance != 0 {
			t.Errorf("Balance = %d, want 0（新規登録時）", child.Balance)
		}
	})

	t.Run("名前の前後スペースがトリミングされる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"  たろう  ","age":8,"base_allowance":1000}`

		resp, err := createChild(ctx, db, user, body)
		if err != nil {
			t.Fatalf("createChild() error = %v", err)
		}

		var result map[string]models.ChildResponse
		json.Unmarshal([]byte(resp.Body), &result)
		if result["data"].Name != "たろう" {
			t.Errorf("Name = %q, want %q", result["data"].Name, "たろう")
		}
	})

	t.Run("バリデーションエラー（名前なし）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"","age":8,"base_allowance":1000}`

		resp, err := createChild(ctx, db, user, body)
		if err != nil {
			t.Fatalf("createChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("不正なJSONはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := createChild(ctx, db, user, "invalid json")
		if err != nil {
			t.Fatalf("createChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

// ─── getChild のテスト ───────────────────────────────────────

func TestGetChild(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に子どもを取得できる（残高含む）", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "じろう", 12, 1500)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 1000)

		resp, err := getChild(ctx, db, user, child.ID.Hex())
		if err != nil {
			t.Fatalf("getChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]models.ChildResponse
		json.Unmarshal([]byte(resp.Body), &result)
		if result["data"].Balance != 1000 {
			t.Errorf("Balance = %d, want 1000", result["data"].Balance)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := getChild(ctx, db, user, bson.NewObjectID().Hex())
		if err != nil {
			t.Fatalf("getChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("不正なID形式は404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := getChild(ctx, db, user, "invalid-id")
		if err != nil {
			t.Fatalf("getChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("他ユーザーの子どもは404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID, "他人の子", 10, 2000)

		resp, err := getChild(ctx, db, user1, child.ID.Hex())
		if err != nil {
			t.Fatalf("getChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d（他ユーザーのリソースは404）", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ─── updateChild のテスト ────────────────────────────────────

func TestUpdateChild(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に子ども情報を更新できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "たろう", 8, 1000)
		body := `{"name":"たろうくん","age":9,"base_allowance":1500}`

		resp, err := updateChild(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("updateChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]models.ChildResponse
		json.Unmarshal([]byte(resp.Body), &result)
		got := result["data"]
		if got.Name != "たろうくん" {
			t.Errorf("Name = %q, want %q", got.Name, "たろうくん")
		}
		if got.Age != 9 {
			t.Errorf("Age = %d, want 9", got.Age)
		}
		if got.BaseAllowance != 1500 {
			t.Errorf("BaseAllowance = %d, want 1500", got.BaseAllowance)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		body := `{"name":"たろう","age":8,"base_allowance":1000}`

		resp, err := updateChild(ctx, db, user, bson.NewObjectID().Hex(), body)
		if err != nil {
			t.Fatalf("updateChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("バリデーションエラーはBadRequestを返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "たろう", 8, 1000)
		body := `{"name":"","age":8,"base_allowance":1000}`

		resp, err := updateChild(ctx, db, user, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("updateChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("他ユーザーの子どもは404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID, "他人の子", 10, 2000)
		body := `{"name":"改ざん","age":10,"base_allowance":2000}`

		resp, err := updateChild(ctx, db, user1, child.ID.Hex(), body)
		if err != nil {
			t.Fatalf("updateChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// ─── deleteChild のテスト ────────────────────────────────────

func TestDeleteChild(t *testing.T) {
	ctx := context.Background()

	t.Run("正常に子どもを削除できる", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "さぶろう", 6, 500)

		resp, err := deleteChild(ctx, db, user, child.ID.Hex())
		if err != nil {
			t.Fatalf("deleteChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// 削除後に取得しても404
		getResp, _ := getChild(ctx, db, user, child.ID.Hex())
		if getResp.StatusCode != http.StatusNotFound {
			t.Errorf("削除後の取得: StatusCode = %d, want %d", getResp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("関連する収支記録がカスケード削除される", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user.ID, "しろう", 7, 800)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeIncome, 1000)
		insertRecord(t, ctx, db, child.ID, models.RecordTypeExpense, 300)

		resp, err := deleteChild(ctx, db, user, child.ID.Hex())
		if err != nil {
			t.Fatalf("deleteChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		// 関連する records が削除されているか確認
		count, err := db.Collection(models.CollectionRecords).CountDocuments(ctx, bson.M{"child_id": child.ID})
		if err != nil {
			t.Fatalf("records件数確認失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("カスケード削除後のrecords件数 = %d, want 0", count)
		}
	})

	t.Run("存在しないIDは404を返す", func(t *testing.T) {
		db := newTestDB(t)
		user := newTestUser(t, ctx, db)

		resp, err := deleteChild(ctx, db, user, bson.NewObjectID().Hex())
		if err != nil {
			t.Fatalf("deleteChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("他ユーザーの子どもは404を返す（所有権チェック）", func(t *testing.T) {
		db := newTestDB(t)
		user1 := newTestUser(t, ctx, db)
		user2 := newTestUser(t, ctx, db)
		child := insertChild(t, ctx, db, user2.ID, "他人の子", 10, 2000)

		resp, err := deleteChild(ctx, db, user1, child.ID.Hex())
		if err != nil {
			t.Fatalf("deleteChild() error = %v", err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}

		// 他ユーザーの子どもが削除されていないこと
		getResp, _ := getChild(ctx, db, user2, child.ID.Hex())
		if getResp.StatusCode != http.StatusOK {
			t.Errorf("他ユーザーの子どもが誤削除された")
		}
	})
}
