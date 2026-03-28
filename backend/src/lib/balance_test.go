package lib_test

import (
	"testing"

	"github.com/n-yata/money-management/backend/src/lib"
	"github.com/n-yata/money-management/backend/src/models"
)

func TestCalcBalance(t *testing.T) {
	tests := []struct {
		name    string
		records []models.Record
		want    int64
	}{
		{
			name:    "空のレコード",
			records: []models.Record{},
			want:    0,
		},
		{
			name: "収入のみ",
			records: []models.Record{
				{Type: models.RecordTypeIncome, Amount: 1000},
			},
			want: 1000,
		},
		{
			name: "支出のみ",
			records: []models.Record{
				{Type: models.RecordTypeExpense, Amount: 300},
			},
			want: -300,
		},
		{
			name: "収入と支出の組み合わせ",
			records: []models.Record{
				{Type: models.RecordTypeIncome, Amount: 1000},
				{Type: models.RecordTypeExpense, Amount: 300},
			},
			want: 700,
		},
		{
			name: "複数の収入と支出",
			records: []models.Record{
				{Type: models.RecordTypeIncome, Amount: 1000},
				{Type: models.RecordTypeIncome, Amount: 500},
				{Type: models.RecordTypeExpense, Amount: 200},
				{Type: models.RecordTypeExpense, Amount: 100},
			},
			want: 1200,
		},
		{
			name: "残高ゼロ（収支均衡）",
			records: []models.Record{
				{Type: models.RecordTypeIncome, Amount: 1000},
				{Type: models.RecordTypeExpense, Amount: 1000},
			},
			want: 0,
		},
		{
			name: "支出が収入を超えてマイナス残高",
			records: []models.Record{
				{Type: models.RecordTypeIncome, Amount: 500},
				{Type: models.RecordTypeExpense, Amount: 800},
			},
			want: -300,
		},
		{
			name: "不明なtypeは無視される",
			records: []models.Record{
				{Type: "unknown", Amount: 999},
				{Type: models.RecordTypeIncome, Amount: 500},
			},
			want: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lib.CalcBalance(tt.records)
			if got != tt.want {
				t.Errorf("CalcBalance() = %d, want %d", got, tt.want)
			}
		})
	}
}
