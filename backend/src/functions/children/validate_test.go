package main

import (
	"strings"
	"testing"
)

func TestChildInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   childInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常な入力",
			input:   childInput{Name: "たろう", Age: 8, BaseAllowance: 1000},
			wantErr: false,
		},
		{
			name:    "年齢下限（1歳）",
			input:   childInput{Name: "たろう", Age: 1, BaseAllowance: 0},
			wantErr: false,
		},
		{
			name:    "年齢上限（18歳）",
			input:   childInput{Name: "たろう", Age: 18, BaseAllowance: 0},
			wantErr: false,
		},
		{
			name:    "基本おこずかいゼロ",
			input:   childInput{Name: "たろう", Age: 10, BaseAllowance: 0},
			wantErr: false,
		},
		{
			name:    "名前が空",
			input:   childInput{Name: "", Age: 8, BaseAllowance: 1000},
			wantErr: true,
			errMsg:  "名前は必須です",
		},
		{
			name:    "名前がスペースのみ",
			input:   childInput{Name: "   ", Age: 8, BaseAllowance: 1000},
			wantErr: true,
			errMsg:  "名前は必須です",
		},
		{
			name:    "名前が20文字（境界値OK）",
			input:   childInput{Name: strings.Repeat("あ", 20), Age: 8, BaseAllowance: 1000},
			wantErr: false,
		},
		{
			name:    "名前が21文字（境界値NG）",
			input:   childInput{Name: strings.Repeat("あ", 21), Age: 8, BaseAllowance: 1000},
			wantErr: true,
			errMsg:  "名前は20文字以内で入力してください",
		},
		{
			name:    "年齢が0（下限未満）",
			input:   childInput{Name: "たろう", Age: 0, BaseAllowance: 1000},
			wantErr: true,
			errMsg:  "年齢は1〜18の整数で入力してください",
		},
		{
			name:    "年齢が19（上限超え）",
			input:   childInput{Name: "たろう", Age: 19, BaseAllowance: 1000},
			wantErr: true,
			errMsg:  "年齢は1〜18の整数で入力してください",
		},
		{
			name:    "基本おこずかいがマイナス",
			input:   childInput{Name: "たろう", Age: 8, BaseAllowance: -1},
			wantErr: true,
			errMsg:  "基本おこずかい額は0以上で入力してください",
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
