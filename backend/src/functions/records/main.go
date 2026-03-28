package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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

type recordInput struct {
	Type            string  `json:"type"`
	Amount          int64   `json:"amount"`
	Description     string  `json:"description"`
	Date            string  `json:"date"`
	AllowanceTypeID *string `json:"allowance_type_id"`
}

func (i recordInput) validate() string {
	if i.Type != models.RecordTypeIncome && i.Type != models.RecordTypeExpense {
		return "typeはincomeまたはexpenseで入力してください"
	}
	if i.Amount < 1 {
		return "金額は1円以上で入力してください"
	}
	descLen := utf8.RuneCountInString(strings.TrimSpace(i.Description))
	if descLen < 1 {
		return "説明は必須です"
	}
	if descLen > 50 {
		return "説明は50文字以内で入力してください"
	}
	if _, err := time.Parse("2006-01-02", i.Date); err != nil {
		return "日付はYYYY-MM-DD形式で入力してください"
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

	childID := req.PathParameters["id"]
	recordID, hasRecordID := req.PathParameters["recordId"]
	method := req.HTTPMethod

	switch {
	case method == http.MethodGet && !hasRecordID:
		return listRecords(ctx, db, user, childID, req.QueryStringParameters)
	case method == http.MethodPost && !hasRecordID:
		return createRecord(ctx, db, user, childID, req.Body)
	case method == http.MethodDelete && hasRecordID:
		return deleteRecord(ctx, db, user, childID, recordID)
	default:
		return errorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"), nil
	}
}

// ─── GET /api/v1/children/:id/records ───────────────────────

func listRecords(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, params map[string]string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	// year・month は必須クエリパラメータ
	yearStr, monthStr := params["year"], params["month"]
	if yearStr == "" || monthStr == "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "yearとmonthは必須です"), nil
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1 {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "yearが不正です"), nil
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "monthは1〜12で指定してください"), nil
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	col := db.Collection(models.CollectionRecords)
	cursor, err := col.Find(ctx,
		bson.M{
			"child_id": child.ID,
			"date":     bson.M{"$gte": start, "$lt": end},
		},
		options.Find().SetSort(bson.D{{Key: "date", Value: -1}}),
	)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の取得に失敗しました"), nil
	}
	defer cursor.Close(ctx)

	var records []models.Record
	if err := cursor.All(ctx, &records); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の取得に失敗しました"), nil
	}

	if records == nil {
		records = []models.Record{}
	}
	return jsonResponse(http.StatusOK, map[string]any{"data": records}), nil
}

// ─── POST /api/v1/children/:id/records ──────────────────────

func createRecord(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, body string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	var input recordInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
	}

	date, _ := time.Parse("2006-01-02", input.Date)
	now := time.Now()

	record := models.Record{
		ID:          bson.NewObjectID(),
		ChildID:     child.ID,
		Type:        input.Type,
		Amount:      input.Amount,
		Description: strings.TrimSpace(input.Description),
		Date:        date,
		CreatedAt:   now,
	}

	// allowance_type_id が指定された場合、自分のユーザーに紐づく種類か確認
	if input.AllowanceTypeID != nil && *input.AllowanceTypeID != "" {
		atOID, err := bson.ObjectIDFromHex(*input.AllowanceTypeID)
		if err != nil {
			return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "allowance_type_idが不正です"), nil
		}
		count, err := db.Collection(models.CollectionAllowanceTypes).CountDocuments(ctx,
			bson.M{"_id": atOID, "user_id": user.ID},
		)
		if err != nil {
			return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類の確認に失敗しました"), nil
		}
		if count == 0 {
			return errorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "指定されたallowance_type_idが見つかりません"), nil
		}
		record.AllowanceTypeID = &atOID
	}

	if _, err := db.Collection(models.CollectionRecords).InsertOne(ctx, record); err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の登録に失敗しました"), nil
	}

	return jsonResponse(http.StatusCreated, map[string]any{"data": record}), nil
}

// ─── DELETE /api/v1/children/:id/records/:recordId ──────────

func deleteRecord(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, recordIDStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := findOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	recordOID, err := bson.ObjectIDFromHex(recordIDStr)
	if err != nil {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された収支記録が見つかりません"), nil
	}

	result, err := db.Collection(models.CollectionRecords).DeleteOne(ctx,
		bson.M{"_id": recordOID, "child_id": child.ID},
	)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の削除に失敗しました"), nil
	}
	if result.DeletedCount == 0 {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "指定された収支記録が見つかりません"), nil
	}

	return jsonResponse(http.StatusOK, map[string]any{"data": nil}), nil
}

// ─── 共通ヘルパー ────────────────────────────────────────────

func findOwnedChild(ctx context.Context, db *mongo.Database, userID bson.ObjectID, idStr string) (models.Child, bool, error) {
	oid, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		return models.Child{}, false, nil
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
