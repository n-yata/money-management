package main

import (
	"context"
	"encoding/json"
	"log"
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
)

// ─── 初期化 ──────────────────────────────────────────────────

func init() {
	ctx := context.Background()
	if err := lib.EnsureIndexes(ctx); err != nil {
		log.Printf("EnsureIndexes warning: %v", err)
	}
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

// ─── 残高計算ヘルパー ────────────────────────────────────────

// childWithBalance はAggregation Pipelineの結果を受け取るための内部型。
type childWithBalance struct {
	ID            bson.ObjectID `bson:"_id"`
	Name          string        `bson:"name"`
	Age           int           `bson:"age"`
	BaseAllowance int64         `bson:"base_allowance"`
	Balance       int64         `bson:"balance"`
	CreatedAt     time.Time     `bson:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at"`
}

// calcBalanceForChild は指定した childID の残高を Aggregation で計算して返す。
// 全件ロードを避けることでメモリ効率を高める。
func calcBalanceForChild(ctx context.Context, db *mongo.Database, childID bson.ObjectID) (int64, error) {
	col := db.Collection(models.CollectionRecords)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"child_id": childID}}},
		{{Key: "$group", Value: bson.M{
			"_id": nil,
			"balance": bson.M{"$sum": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$type", "income"}},
				"$amount",
				bson.M{"$multiply": []any{"$amount", -1}},
			}}},
		}}},
	}

	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		Balance int64 `bson:"balance"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0].Balance, nil
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

	method := req.HTTPMethod
	childID, hasID := req.PathParameters["id"]
	hasID = hasID && childID != ""

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
		return lib.ErrorResponse(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "許可されていないメソッドです"), nil
	}
}

// ─── GET /api/v1/children ───────────────────────────────────

func listChildren(ctx context.Context, db *mongo.Database, user models.User) (events.APIGatewayProxyResponse, error) {
	col := db.Collection(models.CollectionChildren)
	// $lookup で records を結合し、残高を集計することで N+1 クエリを解消する
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": user.ID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         models.CollectionRecords,
			"localField":   "_id",
			"foreignField": "child_id",
			"as":           "records",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"balance": bson.M{"$sum": bson.M{"$map": bson.M{
				"input": "$records",
				"as":    "r",
				"in": bson.M{"$cond": []any{
					bson.M{"$eq": []any{"$$r.type", "income"}},
					"$$r.amount",
					bson.M{"$multiply": []any{"$$r.amount", -1}},
				}},
			}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: 1}}}},
		{{Key: "$project", Value: bson.M{"records": 0}}},
	}

	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども一覧の取得に失敗しました"), nil
	}
	defer cursor.Close(ctx)

	var rows []childWithBalance
	if err := cursor.All(ctx, &rows); err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども一覧の取得に失敗しました"), nil
	}

	result := make([]models.ChildResponse, 0, len(rows))
	for _, r := range rows {
		result = append(result, models.ChildResponse{
			ID:            r.ID,
			Name:          r.Name,
			Age:           r.Age,
			BaseAllowance: r.BaseAllowance,
			Balance:       r.Balance,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		})
	}
	return lib.JSONResponse(http.StatusOK, map[string]any{"data": result}), nil
}

// ─── POST /api/v1/children ──────────────────────────────────

func createChild(ctx context.Context, db *mongo.Database, user models.User, body string) (events.APIGatewayProxyResponse, error) {
	var input childInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
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
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子どもの登録に失敗しました"), nil
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
	return lib.JSONResponse(http.StatusCreated, map[string]any{"data": resp}), nil
}

// ─── GET /api/v1/children/:id ───────────────────────────────

func getChild(ctx context.Context, db *mongo.Database, user models.User, idStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	balance, err := calcBalanceForChild(ctx, db, child.ID)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "残高計算に失敗しました"), nil
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
	return lib.JSONResponse(http.StatusOK, map[string]any{"data": resp}), nil
}

// ─── PUT /api/v1/children/:id ───────────────────────────────

func updateChild(ctx context.Context, db *mongo.Database, user models.User, idStr string, body string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	var input childInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", "リクエストボディが不正です"), nil
	}
	if msg := input.validate(); msg != "" {
		return lib.ErrorResponse(http.StatusBadRequest, "VALIDATION_ERROR", msg), nil
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
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の更新に失敗しました"), nil
	}

	child.Name = strings.TrimSpace(input.Name)
	child.Age = input.Age
	child.BaseAllowance = input.BaseAllowance
	child.UpdatedAt = now

	balance, err := calcBalanceForChild(ctx, db, child.ID)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "残高計算に失敗しました"), nil
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
	return lib.JSONResponse(http.StatusOK, map[string]any{"data": resp}), nil
}

// ─── DELETE /api/v1/children/:id ────────────────────────────

func deleteChild(ctx context.Context, db *mongo.Database, user models.User, idStr string) (events.APIGatewayProxyResponse, error) {
	child, found, err := lib.FindOwnedChild(ctx, db, user.ID, idStr)
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子ども情報の取得に失敗しました"), nil
	}
	if !found {
		return lib.ErrorResponse(http.StatusNotFound, "NOT_FOUND", "指定された子どもが見つかりません"), nil
	}

	// records の cascade 削除と child 削除をトランザクションでアトミックに実行する
	session, err := db.Client().StartSession()
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子どもの削除に失敗しました"), nil
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		if _, err := db.Collection(models.CollectionRecords).DeleteMany(sc, bson.M{"child_id": child.ID}); err != nil {
			return nil, err
		}
		return db.Collection(models.CollectionChildren).DeleteOne(sc, bson.M{"_id": child.ID})
	})
	if err != nil {
		return lib.ErrorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "子どもの削除に失敗しました"), nil
	}

	return lib.JSONResponse(http.StatusOK, map[string]any{"data": nil}), nil
}

func main() {
	lambda.Start(handler)
}
