// Package models はMongoDBコレクションに対応するGoの構造体を定義する。
// 各構造体はbsonタグでシリアライズ/デシリアライズを行う。
package models

// コレクション名定数
const (
	CollectionUsers         = "users"
	CollectionChildren      = "children"
	CollectionAllowanceTypes = "allowance_types"
	CollectionRecords       = "records"
)
