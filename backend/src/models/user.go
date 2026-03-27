package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User は親ユーザーを表す。Auth0のsub IDで一意に識別される。
type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Auth0Sub  string             `bson:"auth0_sub"     json:"auth0_sub"`
	Email     string             `bson:"email"         json:"email"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"    json:"updated_at"`
}
