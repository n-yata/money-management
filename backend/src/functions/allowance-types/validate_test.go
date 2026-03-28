package main

import (
	"strings"
	"testing"
)

func TestAllowanceTypeInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   allowanceTypeInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常な入力",
			input:   allowanceTypeInput{Name: "お皿洗い", Amount: 50},
			wantErr: false,
		},
		{
			name:    "報酬金額1円（下限）",
			input:   allowanceTypeInput{Name: "お皿洗い", Amount: 1},
			wantErr: false,
		},
		{
			name:    "種類名が30文字（境界値OK）",
			input:   allowanceTypeInput{Name: strings.Repeat("あ", 30), Amount: 100},
			wantErr: false,
		},
		{
			name:    "種類名が空",
			input:   allowanceTypeInput{Name: "", Amount: 50},
			wantErr: true,
			errMsg:  "種類名は必須です",
		},
		{
			name:    "種類名がスペースのみ",
			input:   allowanceTypeInput{Name: "   ", Amount: 50},
			wantErr: true,
			errMsg:  "種類名は必須です",
		},
		{
			name:    "種類名が31文字（境界値NG）",
			input:   allowanceTypeInput{Name: strings.Repeat("あ", 31), Amount: 100},
			wantErr: true,
			errMsg:  "種類名は30文字以内で入力してください",
		},
		{
			name:    "報酬金額がゼロ",
			input:   allowanceTypeInput{Name: "お皿洗い", Amount: 0},
			wantErr: true,
			errMsg:  "報酬金額は1円以上で入力してください",
		},
		{
			name:    "報酬金額がマイナス",
			input:   allowanceTypeInput{Name: "お皿洗い", Amount: -1},
			wantErr: true,
			errMsg:  "報酬金額は1円以上で入力してください",
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
