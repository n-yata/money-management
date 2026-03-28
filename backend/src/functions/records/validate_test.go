package main

import (
	"strings"
	"testing"
)

func TestRecordInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   recordInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常な入力（income）",
			input:   recordInput{Type: "income", Amount: 100, Description: "お皿洗い", Date: "2026-03-01"},
			wantErr: false,
		},
		{
			name:    "正常な入力（expense）",
			input:   recordInput{Type: "expense", Amount: 1, Description: "おかし", Date: "2026-03-15"},
			wantErr: false,
		},
		{
			name:    "説明が50文字（境界値OK）",
			input:   recordInput{Type: "income", Amount: 100, Description: strings.Repeat("あ", 50), Date: "2026-03-01"},
			wantErr: false,
		},
		{
			name:    "typeが不正",
			input:   recordInput{Type: "invalid", Amount: 100, Description: "テスト", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "typeはincomeまたはexpenseで入力してください",
		},
		{
			name:    "typeが空",
			input:   recordInput{Type: "", Amount: 100, Description: "テスト", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "typeはincomeまたはexpenseで入力してください",
		},
		{
			name:    "金額がゼロ",
			input:   recordInput{Type: "income", Amount: 0, Description: "テスト", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "金額は1円以上で入力してください",
		},
		{
			name:    "金額がマイナス",
			input:   recordInput{Type: "income", Amount: -1, Description: "テスト", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "金額は1円以上で入力してください",
		},
		{
			name:    "金額が上限超過（10,000,001円）",
			input:   recordInput{Type: "income", Amount: 10_000_001, Description: "テスト", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "金額は10,000,000円以下で入力してください",
		},
		{
			name:    "説明が空",
			input:   recordInput{Type: "income", Amount: 100, Description: "", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "説明は必須です",
		},
		{
			name:    "説明がスペースのみ",
			input:   recordInput{Type: "income", Amount: 100, Description: "   ", Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "説明は必須です",
		},
		{
			name:    "説明が51文字（境界値NG）",
			input:   recordInput{Type: "income", Amount: 100, Description: strings.Repeat("あ", 51), Date: "2026-03-01"},
			wantErr: true,
			errMsg:  "説明は50文字以内で入力してください",
		},
		{
			name:    "日付が不正な形式",
			input:   recordInput{Type: "income", Amount: 100, Description: "テスト", Date: "20260301"},
			wantErr: true,
			errMsg:  "日付はYYYY-MM-DD形式で入力してください",
		},
		{
			name:    "日付が空",
			input:   recordInput{Type: "income", Amount: 100, Description: "テスト", Date: ""},
			wantErr: true,
			errMsg:  "日付はYYYY-MM-DD形式で入力してください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.validate()
			if tt.wantErr {
				if got == "" {
					t.Errorf("validate() = \"\", want error message %q", tt.errMsg)
				} else if tt.errMsg != "" && got != tt.errMsg {
					t.Errorf("validate() = %q, want %q", got, tt.errMsg)
				}
			} else {
				if got != "" {
					t.Errorf("validate() = %q, want \"\"", got)
				}
			}
		})
	}
}
