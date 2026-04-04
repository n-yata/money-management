//go:build integration

package lib_test

import (
	"context"
	"testing"
	"time"

	"github.com/n-yata/money-management/backend/src/lib"
	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestFindOwnedChild(t *testing.T) {
	ctx := context.Background()

	t.Run("空文字IDは見つからずエラーなし", func(t *testing.T) {
		db := newIntegrationDB(t)
		userID := bson.NewObjectID()

		child, found, err := lib.FindOwnedChild(ctx, db, userID, "")

		if err != nil {
			t.Errorf("FindOwnedChild() error = %v, want nil", err)
		}
		if found {
			t.Errorf("FindOwnedChild() found = true, want false")
		}
		if !child.ID.IsZero() {
			t.Errorf("FindOwnedChild() child.ID = %v, want zero value", child.ID)
		}
	})

	t.Run("不正なObjectID形式は見つからずエラーなし", func(t *testing.T) {
		db := newIntegrationDB(t)
		userID := bson.NewObjectID()

		child, found, err := lib.FindOwnedChild(ctx, db, userID, "not-a-valid-objectid")

		if err != nil {
			t.Errorf("FindOwnedChild() error = %v, want nil", err)
		}
		if found {
			t.Errorf("FindOwnedChild() found = true, want false")
		}
		if !child.ID.IsZero() {
			t.Errorf("FindOwnedChild() child.ID = %v, want zero value", child.ID)
		}
	})

	t.Run("他ユーザーの子どもIDは所有権NGで見つからない", func(t *testing.T) {
		db := newIntegrationDB(t)
		ownerID := bson.NewObjectID()
		requesterID := bson.NewObjectID()

		// 別ユーザーの子どもを登録
		now := time.Now()
		otherChild := models.Child{
			ID:        bson.NewObjectID(),
			UserID:    ownerID,
			Name:      "他人の子",
			Age:       8,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := db.Collection(models.CollectionChildren).InsertOne(ctx, otherChild); err != nil {
			t.Fatalf("テストデータ挿入失敗: %v", err)
		}

		child, found, err := lib.FindOwnedChild(ctx, db, requesterID, otherChild.ID.Hex())

		if err != nil {
			t.Errorf("FindOwnedChild() error = %v, want nil", err)
		}
		if found {
			t.Errorf("FindOwnedChild() found = true, want false（他ユーザーの子どもは取得不可）")
		}
		if !child.ID.IsZero() {
			t.Errorf("FindOwnedChild() child.ID = %v, want zero value", child.ID)
		}
	})

	t.Run("自ユーザーの子どもIDは正常に取得できる", func(t *testing.T) {
		db := newIntegrationDB(t)
		userID := bson.NewObjectID()

		now := time.Now()
		ownChild := models.Child{
			ID:            bson.NewObjectID(),
			UserID:        userID,
			Name:          "たろう",
			Age:           10,
			BaseAllowance: 1000,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if _, err := db.Collection(models.CollectionChildren).InsertOne(ctx, ownChild); err != nil {
			t.Fatalf("テストデータ挿入失敗: %v", err)
		}

		child, found, err := lib.FindOwnedChild(ctx, db, userID, ownChild.ID.Hex())

		if err != nil {
			t.Errorf("FindOwnedChild() error = %v, want nil", err)
		}
		if !found {
			t.Errorf("FindOwnedChild() found = false, want true")
		}
		if child.ID != ownChild.ID {
			t.Errorf("FindOwnedChild() child.ID = %v, want %v", child.ID, ownChild.ID)
		}
		if child.Name != ownChild.Name {
			t.Errorf("FindOwnedChild() child.Name = %q, want %q", child.Name, ownChild.Name)
		}
		if child.UserID != userID {
			t.Errorf("FindOwnedChild() child.UserID = %v, want %v", child.UserID, userID)
		}
	})
}
