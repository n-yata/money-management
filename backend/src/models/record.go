package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Record は収支記録を表す。type は "income" または "expense"。
type Record struct {
	ID              bson.ObjectID  `bson:"_id,omitempty"               json:"id"`
	ChildID         bson.ObjectID  `bson:"child_id"                    json:"child_id"`
	AllowanceTypeID *bson.ObjectID `bson:"allowance_type_id,omitempty"  json:"allowance_type_id,omitempty"`
	Type            string              `bson:"type"                       json:"type"`
	Amount          int64               `bson:"amount"                     json:"amount"`
	Description     string              `bson:"description"                json:"description"`
	Date            time.Time           `bson:"date"                       json:"date"`
	CreatedAt       time.Time           `bson:"created_at"                 json:"created_at"`
}

const (
	RecordTypeIncome  = "income"
	RecordTypeExpense = "expense"
)
