package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AllowanceType はおこづかいの種類と報酬金額を表す。ユーザーが自由に定義する。
type AllowanceType struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"user_id"       json:"user_id"`
	Name      string             `bson:"name"          json:"name"`
	Amount    int64              `bson:"amount"        json:"amount"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"    json:"updated_at"`
}
