package parser

import (
	"testing"
)

func TestEvalIfCondition_Basic(t *testing.T) {
	tests := []struct {
		name     string
		cond     string
		data     map[string]interface{}
		expected bool
	}{
		// 基本數值與字串比對
		{"相等數值", "1 == 1", nil, true},
		{"不相等數值", "1 == 2", nil, false},
		{"字串相等", "'news' == 'news'", nil, true},
		{"字串不相等", "'news' == 'product'", nil, false},
		{"數值大於", "10 > 5", nil, true},
		{"數值小於", "3 < 2", nil, false},
		{"大於等於", "5 >= 5", nil, true},
		{"小於等於", "4 <= 4", nil, true},
		{"不等於成立", "1 != 2", nil, true},
		{"不等於不成立", "1 != 1", nil, false},

		// 邏輯運算
		{"邏輯 AND 均為真", "1 == 1 && 2 == 2", nil, true},
		{"邏輯 AND 部分為假", "1 == 1 && 2 == 3", nil, false},
		{"邏輯 OR 一真一假", "1 == 2 || 3 == 3", nil, true},
		{"邏輯 OR 均為假", "1 == 2 || 3 == 4", nil, false},

		// 變量替換測試 [content:xxx], [sort:xxx], {sort:xxx}, {gboot:xxx}
		{
			name:     "content 變量相等",
			cond:     "[content:istop] == 1",
			data:     map[string]interface{}{"istop": 1},
			expected: true,
		},
		{
			name:     "content 變量不相等",
			cond:     "[content:istop] == 1",
			data:     map[string]interface{}{"istop": 0},
			expected: false,
		},
		{
			name:     "sort 變量大於",
			cond:     "[sort:sorting] > 10",
			data:     map[string]interface{}{"sorting": 20},
			expected: true,
		},
		{
			name:     "花括號 gboot 變量",
			cond:     "{gboot:islogin} == 1",
			data:     map[string]interface{}{"islogin": 1},
			expected: true,
		},

		// 安全防護：惡意 PHP/Shell 注入攔截
		{"攔截 eval", "eval('echo 1')", nil, false},
		{"攔截 system", "system('ls')", nil, false},
		{"攔截 $_GET", "$_GET['cmd'] == 1", nil, false},
		{"攔截 base64_decode", "base64_decode('xxx') == 1", nil, false},

		// 真值判斷
		{"單一非零數值", "1", nil, true},
		{"單一零數值", "0", nil, false},
		{"空白表達式", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvalIfCondition(tt.cond, tt.data)
			if result != tt.expected {
				t.Errorf("EvalIfCondition(%q) = %v, expected %v", tt.cond, result, tt.expected)
			}
		})
	}
}

func BenchmarkEvalIfCondition(b *testing.B) {
	data := map[string]interface{}{
		"istop":   1,
		"islogin": 1,
		"sorting": 15,
	}
	cond := "[content:istop] == 1 && [sort:sorting] > 10"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EvalIfCondition(cond, data)
	}
}
