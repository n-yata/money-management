package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Child は子どもを表す。1ユーザーに複数紐づく。
type Child struct {
	ID            bson.ObjectID `bson:"_id,omitempty"   json:"id"`
	UserID        bson.ObjectID `bson:"user_id"         json:"-"` // 内部フィールドのためJSONレスポンスから除外
	Name          string             `bson:"name"            json:"name"`
	Age           int                `bson:"age"             json:"age"`
	BaseAllowance int64              `bson:"base_allowance"  json:"base_allowance"`
	CreatedAt     time.Time          `bson:"created_at"      json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"      json:"updated_at"`
}

// ChildResponse はAPIレスポンス用。残高（Balance）を含む。
type ChildResponse struct {
	ID            bson.ObjectID `json:"id"`
	Name          string             `json:"name"`
	Age           int                `json:"age"`
	BaseAllowance int64              `json:"base_allowance"`
	Balance       int64              `json:"balance"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}
