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

type allowanceTypeInput struct {
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

func (i allowanceTypeInput) validate() string {
	name := strings.TrimSpace(i.Name)
	if name == "" {
		return "種類名は必須です"
	}
	if utf8.RuneCountInString(name) > 30 {
		return "種類名は30文字以内で入力してください"
	}
	if i.Amount < 1 {
		return "報酬金額は1円以上で入力してください"
	}
	return ""
}

// ─── ユーザー解決 ────────────────────────────────────────────

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
	typeID, hasID := req.PathParameters["id"]

	switch {
	case method == http.MethodGet && !hasID:
		return listAllowanceTypes(ctx, db, user)
	case method == http.MethodPost && !hasID:
		return createAllowanceType(ctx, db, user, req.Body)
	case method == http.MethodPut && hasID:
		return updateAllowanceType(ctx, db, user, typeID, req.Body)
	case method == http.MethodDelete && hasID:
		return deleteAllowanceType(ctx, db, user, typeID)
	default:
		return errorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"), nil
	}
}

// ─── GET /api/v1/allowance-types ────────────────────────────

func listAllowanceTypes(ctx context.Context, db *mongo.Database, user models.User) (events.APIGatewayProxyResponse, error) {
	col := db.Collection(models.CollectionAllowanceTypes)
	cursor, err := col.Find(ctx, bson.M{"user_id": user.ID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類一覧の取得に失敗しました"), nil
	}
	defer cursor.Close(ctx)

	var types []models.AllowanceType
	if err := cursor.All(ctx, &types); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類一覧の取得に失敗しました"), nil
	}

	if types == nil {
		types = []models.AllowanceType{}
	}
	return jsonResponse(http.StatusOK, map[string]any{"data": types}), nil
}

// ─── POST /api/v1/allowance-types ───────────────────────────

func createAllowanceType(ctx context.Context, db *mongo.Database, user models.User, body string) (events.APIGatewayProxyResponse, error) {
	var input allowanceTypeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
	}

	now := time.Now()
	at := models.AllowanceType{
		ID:        bson.NewObjectID(),
		UserID:    user.ID,
		Name:      strings.TrimSpace(input.Name),
		Amount:    input.Amount,
		CreatedAt: now,
		UpdatedAt: now,
	}

	col := db.Collection(models.CollectionAllowanceTypes)
	if _, err := col.InsertOne(ctx, at); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類の登録に失敗しました"), nil
	}

	return jsonResponse(http.StatusCreated, map[string]any{"data": at}), nil
}

// ─── PUT /api/v1/allowance-types/:id ────────────────────────

func updateAllowanceType(ctx context.Context, db *mongo.Database, user models.User, idStr string, body string) (events.APIGatewayProxyResponse, error) {
	at, found, err := findOwnedAllowanceType(ctx, db, user.ID, idStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された種類が見つかりません"), nil
	}

	var input allowanceTypeInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
	}

	now := time.Now()
	col := db.Collection(models.CollectionAllowanceTypes)
	_, err = col.UpdateOne(ctx,
		bson.M{"_id": at.ID},
		bson.M{"$set": bson.M{
			"name":       strings.TrimSpace(input.Name),
			"amount":     input.Amount,
			"updated_at": now,
		}},
	)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類の更新に失敗しました"), nil
	}

	at.Name = strings.TrimSpace(input.Name)
	at.Amount = input.Amount
	at.UpdatedAt = now

	return jsonResponse(http.StatusOK, map[string]any{"data": at}), nil
}

// ─── DELETE /api/v1/allowance-types/:id ─────────────────────

func deleteAllowanceType(ctx context.Context, db *mongo.Database, user models.User, idStr string) (events.APIGatewayProxyResponse, error) {
	at, found, err := findOwnedAllowanceType(ctx, db, user.ID, idStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された種類が見つかりません"), nil
	}

	// 関連する records の allowance_type_id を null にする
	recordsCol := db.Collection(models.CollectionRecords)
	_, err = recordsCol.UpdateMany(ctx,
		bson.M{"allowance_type_id": at.ID},
		bson.M{"$unset": bson.M{"allowance_type_id": ""}},
	)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の更新に失敗しました"), nil
	}

	typesCol := db.Collection(models.CollectionAllowanceTypes)
	if _, err := typesCol.DeleteOne(ctx, bson.M{"_id": at.ID}); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類の削除に失敗しました"), nil
	}

	return jsonResponse(http.StatusOK, map[string]any{"data": nil}), nil
}

// ─── 共通ヘルパー ────────────────────────────────────────────

func findOwnedAllowanceType(ctx context.Context, db *mongo.Database, userID bson.ObjectID, idStr string) (models.AllowanceType, bool, error) {
	oid, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		return models.AllowanceType{}, false, nil
	}

	col := db.Collection(models.CollectionAllowanceTypes)
	var at models.AllowanceType
	err = col.FindOne(ctx, bson.M{"_id": oid, "user_id": userID}).Decode(&at)
	if err == mongo.ErrNoDocuments {
		return models.AllowanceType{}, false, nil
	}
	if err != nil {
		return models.AllowanceType{}, false, err
	}
	return at, true, nil
}

func main() {
	lambda.Start(handler)
}
