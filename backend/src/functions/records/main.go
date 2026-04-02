package main

import (
	"context"
	"encoding/json"
	"log"
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

// ─── 初期化 ──────────────────────────────────────────────────

func init() {
	ctx := context.Background()
	if err := lib.EnsureIndexes(ctx); err != nil {
		log.Printf("EnsureIndexes warning: %v", err)
	}
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
	if i.Amount > 10_000_000 {
		return "金額は10,000,000円以下で入力してください"
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

// ─── ハンドラー ─────────────────────────────────────────────

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	auth0Sub, ok := middleware.GetAuthSub(req)
	if !ok {
		return lib.ErrorResponse(http.StatusUnauthorized, "UNAUTHORIZED", "認証情報が取得できません"), nil
	}

	db, err := lib.GetDB()
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "DB接続に失敗しました"), nil
	}

	user, err := lib.ResolveUser(ctx, db, auth0Sub)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "ユーザー情報の取得に失敗しました"), nil
	}

	childID := req.PathParameters["id"]
	recordID, hasRecordID := req.PathParameters["recordId"]
	hasRecordID = hasRecordID && recordID != ""
	method := req.HTTPMethod

	switch {
	case method == http.MethodGet && !hasRecordID:
		return listRecords(ctx, db, user, childID, req.QueryStringParameters)
	case method == http.MethodPost && !hasRecordID:
		return createRecord(ctx, db, user, childID, req.Body)
	case method == http.MethodDelete && hasRecordID:
		return deleteRecord(ctx, db, user, childID, recordID)
	default:
		return lib.ErrorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"), nil
	}
}

// ─── GET /api/v1/children/:id/records ───────────────────────

func listRecords(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, params map[string]string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	// year・month は必須クエリパラメータ
	yearStr, monthStr := params["year"], params["month"]
	if yearStr == "" || monthStr == "" {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "yearとmonthは必須です"), nil
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1 || year > 2100 {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "yearが不正です"), nil
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "monthは1〜12で指定してください"), nil
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
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の取得に失敗しました"), nil
	}
	defer cursor.Close(ctx)

	var records []models.Record
	if err := cursor.All(ctx, &records); err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の取得に失敗しました"), nil
	}

	if records == nil {
		records = []models.Record{}
	}
	return lib.JSONResponse(http.StatusOK, map[string]any{"data": records}), nil
}

// ─── POST /api/v1/children/:id/records ──────────────────────

func createRecord(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, body string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	var input recordInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
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
			return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "allowance_type_idが不正です"), nil
		}
		count, err := db.Collection(models.CollectionAllowanceTypes).CountDocuments(ctx,
			bson.M{"_id": atOID, "user_id": user.ID},
		)
		if err != nil {
			return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "種類の確認に失敗しました"), nil
		}
		if count == 0 {
			return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "指定されたallowance_type_idが見つかりません"), nil
		}
		record.AllowanceTypeID = &atOID
	}

	// 同じ日・同じ子ども・同じ種類のお手伝いは1日1回まで
	if record.AllowanceTypeID != nil {
		dupCount, err := db.Collection(models.CollectionRecords).CountDocuments(ctx,
			bson.M{
				"child_id":          child.ID,
				"allowance_type_id": record.AllowanceTypeID,
				"date":              date,
			},
		)
		if err != nil {
			return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "重複チェックに失敗しました"), nil
		}
		if dupCount > 0 {
			return lib.ErrorResponse(http.StatusConflict, "DUPLICATE_CHORE", "このお手伝いは今日すでに登録されています"), nil
		}
	}

	if _, err := db.Collection(models.CollectionRecords).InsertOne(ctx, record); err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の登録に失敗しました"), nil
	}

	return lib.JSONResponse(http.StatusCreated, map[string]any{"data": record}), nil
}

// ─── DELETE /api/v1/children/:id/records/:recordId ──────────

func deleteRecord(ctx context.Context, db *mongo.Database, user models.User, childIDStr string, recordIDStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, childIDStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	recordOID, err := bson.ObjectIDFromHex(recordIDStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された収支記録が見つかりません"), nil
	}

	result, err := db.Collection(models.CollectionRecords).DeleteOne(ctx,
		bson.M{"_id": recordOID, "child_id": child.ID},
	)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "収支記録の削除に失敗しました"), nil
	}
	if result.DeletedCount == 0 {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された収支記録が見つかりません"), nil
	}

	return lib.JSONResponse(http.StatusOK, map[string]any{"data": nil}), nil
}

func main() {
	lambda.Start(handler)
}
