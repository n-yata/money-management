package lib

import "github.com/n-yata/money-management/backend/src/models"

// CalcBalance はrecordsスライスから残高（収入合計 - 支出合計）を計算して返す。
func CalcBalance(records []models.Record) int64 {
	var balance int64
	for _, r := range records {
		switch r.Type {
		case models.RecordTypeIncome:
			balance += r.Amount
		case models.RecordTypeExpense:
			balance -= r.Amount
		}
	}
	return balance
}
