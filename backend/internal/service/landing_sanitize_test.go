package service

import (
	"strings"
	"testing"
)

// containsCase 大小写不敏感包含检查，防止绕过型断言遗漏。
func containsCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func TestSanitizeCustomHTML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		notContains []string // 输出中不允许出现（大小写不敏感）
		contains    []string // 输出中必须保留
	}{
		{
			name:        "script 标签被转义",
			input:       `<p>hello</p><script>alert(1)</script>`,
			notContains: []string{"<script", "</script"},
			contains:    []string{"<p>hello</p>"},
		},
		{
			name:        "script 大小写混合绕过被拦截",
			input:       `<ScRiPt>alert(1)</sCrIpT><SCRIPT SRC=//evil.com></SCRIPT>`,
			notContains: []string{"<script", "</script"},
		},
		{
			name:        "javascript 协议被移除",
			input:       `<a href="javascript:alert(1)">click</a>`,
			notContains: []string{"javascript:"},
			contains:    []string{`<a href="alert(1)">click</a>`},
		},
		{
			name:        "javascript 协议大小写混合绕过被拦截",
			input:       `<a href="JaVaScRiPt:alert(1)">click</a>`,
			notContains: []string{"javascript:"},
		},
		{
			name:        "onerror 事件属性被移除",
			input:       `<img src=x onerror=alert(1)>`,
			notContains: []string{"onerror"},
			contains:    []string{"<img src=x"},
		},
		{
			name:        "事件属性大小写混合绕过被拦截",
			input:       `<img src=x OnErRoR=alert(1)>`,
			notContains: []string{"onerror"},
		},
		{
			name:        "引号包裹的事件属性被移除",
			input:       `<div onclick='steal()' class="box">内容</div>`,
			notContains: []string{"onclick", "steal"},
			contains:    []string{`class="box"`, "内容"},
		},
		{
			name:        "正常 href 保留且 onclick 移除",
			input:       `<a href="https://example.com" onclick="track()">链接</a>`,
			notContains: []string{"onclick"},
			contains:    []string{`href="https://example.com"`, "链接"},
		},
		{
			name:        "多个事件属性同时移除",
			input:       `<img src=x onerror=a() onload=b() onmouseover=c()>`,
			notContains: []string{"onerror", "onload", "onmouseover"},
		},
		{
			name:        "正常 HTML 原样保留",
			input:       `<div class="hero"><h1>标题</h1><p>说明文字</p><a href="https://example.com">入口</a></div>`,
			notContains: []string{},
			contains:    []string{`<div class="hero">`, "<h1>标题</h1>", `href="https://example.com"`},
		},
		{
			name:        "复合攻击向量全部拦截",
			input:       `<script>var a=1</script><a href="javascript:x()" onclick="y()">t</a><img src=x onerror=z()>`,
			notContains: []string{"<script", "javascript:", "onclick", "onerror"},
		},
		{
			name:        "空字符串与纯文本不变",
			input:       `plain text 这里是中文`,
			notContains: []string{},
			contains:    []string{"plain text 这里是中文"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCustomHTML(tt.input)
			for _, needle := range tt.notContains {
				if containsCase(got, needle) {
					t.Errorf("sanitize output should not contain %q, got: %s", needle, got)
				}
			}
			for _, needle := range tt.contains {
				if !strings.Contains(got, needle) {
					t.Errorf("sanitize output should contain %q, got: %s", needle, got)
				}
			}
		})
	}
}

func TestStripEventHandlersInTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"无事件属性", `<div class="a">`, `<div class="a">`},
		{"无引号事件", `<img src=x onerror=alert(1)>`, `<img src=x>`},
		{"双引号事件", `<img src=x onerror="alert(1)">`, `<img src=x>`},
		{"单引号事件", `<img src=x onerror='alert(1)'>`, `<img src=x>`},
		{"含数字连字符的事件名", `<div onkeyup-2="x()">t</div>`, `<div>t</div>`},
		{"非事件 on 开头词保留", `<div class="btn on">t</div>`, `<div class="btn on">t</div>`},
		{"值中含 on 字样的非事件保留", `<span title="strong on top">t</span>`, `<span title="strong on top">t</span>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripEventHandlersInTag(tt.tag); got != tt.want {
				t.Errorf("stripEventHandlersInTag() = %q, want %q", got, tt.want)
			}
		})
	}
}
