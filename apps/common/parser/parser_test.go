package parser

import (
	"strings"
	"testing"
)

func TestNormalizePbootPrefix(t *testing.T) {
	input := `<html><title>{pboot:sitetitle}</title>{pboot:nav num=10}[nav:name]{/pboot:nav}</html>`
	expected := `<html><title>{gboot:sitetitle}</title>{gboot:nav num=10}[nav:name]{/gboot:nav}</html>`

	got := NormalizePbootPrefix(input)
	if got != expected {
		t.Errorf("NormalizePbootPrefix() =\n%s\nwant:\n%s", got, expected)
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input: `num=10 scode="1,2" order='sorting asc'`,
			expected: map[string]string{
				"num":   "10",
				"scode": "1,2",
				"order": "sorting asc",
			},
		},
		{
			input: `start=1 pagesize=20 fuzzy=0`,
			expected: map[string]string{
				"start":    "1",
				"pagesize": "20",
				"fuzzy":    "0",
			},
		},
	}

	for _, tt := range tests {
		got := ParseParams(tt.input)
		for k, v := range tt.expected {
			if got[k] != v {
				t.Errorf("ParseParams(%q)[%s] = %q, want %q", tt.input, k, got[k], v)
			}
		}
	}
}

func TestTagParser_IfTag(t *testing.T) {
	p := New()
	registerIfProvider(p, &Context{})

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "If 條件為真",
			template: "{pboot:if(1 == 1)}顯示真{else}顯示假{/pboot:if}",
			expected: "顯示真",
		},
		{
			name:     "If 條件為假",
			template: "{pboot:if(1 == 2)}顯示真{else}顯示假{/pboot:if}",
			expected: "顯示假",
		},
		{
			name:     "If 無 else 為真",
			template: "前綴-{pboot:if(5 > 3)}內容{/pboot:if}-後綴",
			expected: "前綴-內容-後綴",
		},
		{
			name:     "If 無 else 為假",
			template: "前綴-{pboot:if(2 > 5)}內容{/pboot:if}-後綴",
			expected: "前綴--後綴",
		},
		{
			name:     "字串比對為真",
			template: "{pboot:if('apple' == 'apple')}蘋果{else}其它{/pboot:if}",
			expected: "蘋果",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Render(tt.template)
			if got != tt.expected {
				t.Errorf("Render(%q) = %q, want %q", tt.template, got, tt.expected)
			}
		})
	}
}

func TestTagParser_PreBlockProtection(t *testing.T) {
	p := New()
	registerIfProvider(p, &Context{})

	// {gboot:pre} 區塊內的標籤應該原樣保留，不被解析執行
	template := `代碼示例：{gboot:pre}{gboot:if(1 == 1)}原樣保留代碼{/gboot:if}{/gboot:pre}`
	got := p.Render(template)

	if !strings.Contains(got, "{gboot:if(1 == 1)}原樣保留代碼{/gboot:if}") {
		t.Errorf("PreBlockProtection failed, pre block was corrupted or evaluated: %s", got)
	}
}
