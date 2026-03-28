package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/n-yata/money-management/backend/src/lib"
	"github.com/n-yata/money-management/backend/src/middleware"
	"github.com/n-yata/money-management/backend/src/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ─── レスポンスヘルパー ──────────────────────────────────────

func jsonResponse(status int, body any) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func errorResponse(status int, code, message string) events.APIGatewayProxyResponse {
	return jsonResponse(status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// ─── リクエストボディ ────────────────────────────────────────

type childInput struct {
	Name          string `json:"name"`
	Age           int    `json:"age"`
	BaseAllowance int64  `json:"base_allowance"`
}

func (i childInput) validate() string {
	name := strings.TrimSpace(i.Name)
	if name == "" {
		return "名前は必須です"
	}
	if utf8.RuneCountInString(name) > 20 {
		return "名前は20文字以内で入力してください"
	}
	if i.Age < 1 || i.Age > 18 {
		return "年齢は1〜18の整数で入力してください"
	}
	if i.BaseAllowance < 0 {
		return "基本おこずかい額は0以上で入力してください"
	}
	return ""
}

// ─── ユーザー解決 ────────────────────────────────────────────

// resolveUser は auth0_sub に紐づくユーザーを取得する。存在しない場合は新規作成する。
func resolveUser(ctx context.Context, db *mongo.Database, auth0Sub string) (models.User, error) {
	col := db.Collection(models.CollectionUsers)
	var user models.User
	err := col.FindOne(ctx, bson.M{"auth0_sub": auth0Sub}).Decode(&user)
	if err == nil {
		return user, nil
	}
	if err != mongo.ErrNoDocuments {
		return user, err
	}
	// 新規ユーザー作成
	now := time.Now()
	user = models.User{
		ID:        bson.NewObjectID(),
		Auth0Sub:  auth0Sub,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := col.InsertOne(ctx, user); err != nil {
		return user, err
	}
	return user, nil
}

// ─── 残高計算ヘルパー ────────────────────────────────────────

// calcBalanceForChild は指定した childID の残高を計算して返す。
func calcBalanceForChild(ctx context.Context, db *mongo.Database, childID bson.ObjectID) (int64, error) {
	col := db.Collection(models.CollectionRecords)
	cursor, err := col.Find(ctx, bson.M{"child_id": childID})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var records []models.Record
	if err := cursor.All(ctx, &records); err != nil {
		return 0, err
	}
	return lib.CalcBalance(records), nil
}

// ─── ハンドラー ─────────────────────────────────────────────

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	auth0Sub, ok := middleware.GetAuthSub(req)
	if !ok {
		return errorResponse(http.StatusUnauthorized, "UNAUTHORIZED", "認証情報が取得できません"), nil
	}

	db, err := lib.GetDB()
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "DB接続に失敗しました"), nil
	}

	user, err := resolveUser(ctx, db, auth0Sub)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "ユーザー情報の取得に失敗しました"), nil
	}

	method := req.HTTPMethod
	childID, hasID := req.PathParameters["id"]

	switch {
	case method == http.MethodGet && !hasID:
		return listChildren(ctx, db, user)
	case method == http.MethodPost && !hasID:
		return createChild(ctx, db, user, req.Body)
	case method == http.MethodGet && hasID:
		return getChild(ctx, db, user, childID)
	case method == http.MethodPut && hasID:
		return updateChild(ctx, db, user, childID, req.Body)
	case method == http.MethodDelete && hasID:
		return deleteChild(ctx, db, user, childID)
	default:
		return errorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"), nil
	}
}

// ─── GET /api/v1/children ───────────────────────────────────

func listChildren(ctx context.Context, db *mongo.Database, user models.User) (events.APIGatewayProxyResponse, error) {
	col := db.Collection(models.CollectionChildren)
	cursor, err := col.Find(ctx, bson.M{"user_id": user.ID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども一覧の取得に失敗しました"), nil
	}
	defer cursor.Close(ctx)

	var children []models.Child
	if err := cursor.All(ctx, &children); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども一覧の取得に失敗しました"), nil
	}

	result := make([]models.ChildResponse, 0, len(children))
	for _, c := range children {
		balance, err := calcBalanceForChild(ctx, db, c.ID)
		if err != nil {
			return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "残高計算に失敗しました"), nil
		}
		result = append(result, models.ChildResponse{
			ID:            c.ID,
			Name:          c.Name,
			Age:           c.Age,
			BaseAllowance: c.BaseAllowance,
			Balance:       balance,
			CreatedAt:     c.CreatedAt,
			UpdatedAt:     c.UpdatedAt,
		})
	}
	return jsonResponse(http.StatusOK, map[string]any{"data": result}), nil
}

// ─── POST /api/v1/children ──────────────────────────────────

func createChild(ctx context.Context, db *mongo.Database, user models.User, body string) (events.APIGatewayProxyResponse, error) {
	var input childInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
	}

	now := time.Now()
	child := models.Child{
		ID:            bson.NewObjectID(),
		UserID:        user.ID,
		Name:          strings.TrimSpace(input.Name),
		Age:           input.Age,
		BaseAllowance: input.BaseAllowance,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	col := db.Collection(models.CollectionChildren)
	if _, err := col.InsertOne(ctx, child); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子どもの登録に失敗しました"), nil
	}

	resp := models.ChildResponse{
		ID:            child.ID,
		Name:          child.Name,
		Age:           child.Age,
		BaseAllowance: child.BaseAllowance,
		Balance:       0,
		CreatedAt:     child.CreatedAt,
		UpdatedAt:     child.UpdatedAt,
	}
	return jsonResponse(http.StatusCreated, map[string]any{"data": resp}), nil
}

// ─── GET /api/v1/children/:id ───────────────────────────────

func getChild(ctx context.Context, db *mongo.Database, user models.User, idStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	balance, err := calcBalanceForChild(ctx, db, child.ID)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "残高計算に失敗しました"), nil
	}

	resp := models.ChildResponse{
		ID:            child.ID,
		Name:          child.Name,
		Age:           child.Age,
		BaseAllowance: child.BaseAllowance,
		Balance:       balance,
		CreatedAt:     child.CreatedAt,
		UpdatedAt:     child.UpdatedAt,
	}
	return jsonResponse(http.StatusOK, map[string]any{"data": resp}), nil
}

// ─── PUT /api/v1/children/:id ───────────────────────────────

func updateChild(ctx context.Context, db *mongo.Database, user models.User, idStr string, body string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	var input childInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
	}

	now := time.Now()
	col := db.Collection(models.CollectionChildren)
	_, err = col.UpdateOne(ctx,
		bson.M{"_id": child.ID},
		bson.M{"$set": bson.M{
			"name":           strings.TrimSpace(input.Name),
			"age":            input.Age,
			"base_allowance": input.BaseAllowance,
			"updated_at":     now,
		}},
	)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の更新に失敗しました"), nil
	}

	child.Name = strings.TrimSpace(input.Name)
	child.Age = input.Age
	child.BaseAllowance = input.BaseAllowance
	child.UpdatedAt = now

	balance, err := calcBalanceForChild(ctx, db, child.ID)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "残高計算に失敗しました"), nil
	}

	resp := models.ChildResponse{
		ID:            child.ID,
		Name:          child.Name,
		Age:           child.Age,
		BaseAllowance: child.BaseAllowance,
		Balance:       balance,
		CreatedAt:     child.CreatedAt,
		UpdatedAt:     child.UpdatedAt,
	}
	return jsonResponse(http.StatusOK, map[string]any{"data": resp}), nil
}

// ─── DELETE /api/v1/children/:id ────────────────────────────

func deleteChild(ctx context.Context, db *mongo.Database, user models.User, idStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	// 関連する records をカスケード削除
	recordsCol := db.Collection(models.CollectionRecords)
	if _, err := recordsCol.DeleteMany(ctx, bson.M{"child_id": child.ID}); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の削除に失敗しました"), nil
	}

	childrenCol := db.Collection(models.CollectionChildren)
	if _, err := childrenCol.DeleteOne(ctx, bson.M{"_id": child.ID}); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子どもの削除に失敗しました"), nil
	}

	return jsonResponse(http.StatusOK, map[string]any{"data": nil}), nil
}

// ─── 共通ヘルパー ────────────────────────────────────────────

// findOwnedChild は指定IDの子どもを取得し、ログインユーザーの所有であることを確認する。
// 存在しない or 他ユーザーのリソースの場合は (_, false, nil) を返す（404扱い）。
func findOwnedChild(ctx context.Context, db *mongo.Database, userID bson.ObjectID, idStr string) (models.Child, bool, error) {
	oid, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		return models.Child{}, false, nil // 不正なID形式は404扱い
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

func main() {
	lambda.Start(handler)
}
