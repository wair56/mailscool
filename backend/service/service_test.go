package service

import (
	"testing"
)

// === ExtractCode 扩展测试 ===

func TestExtractCode_ChinesePatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"验证码", "您的验证码是：654321，请在5分钟内使用。", "654321"},
		{"校验码", "您的校验码为 123456", "123456"},
		{"确认码换行", "确认码：\n789012", "789012"},
		{"安全码冒号", "安全码:456789", "456789"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCode(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractCode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractCode_EnglishPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"security code", "Your security code is: 987654", "987654"},
		{"OTP", "Your OTP is 123456", "123456"},
		{"auth code", "auth code: 5678", "5678"},
		{"passcode colon", "passcode: 9876", "9876"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCode(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractCode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractCode_NoCode(t *testing.T) {
	inputs := []string{
		"Hello, this is a regular email without any codes.",
		"Please visit our website for more information.",
		"",
	}
	for _, input := range inputs {
		if got := ExtractCode(input); got != "" {
			t.Errorf("ExtractCode(%q) = %q, want empty", input, got)
		}
	}
}

func TestExtractCode_AlphanumericCode(t *testing.T) {
	got := ExtractCode("Your verification code is: AB1234")
	if got != "AB1234" {
		t.Errorf("ExtractCode alphanumeric = %q, want AB1234", got)
	}
}

// === ExtractLinks 测试 ===

func TestExtractLinks_Basic(t *testing.T) {
	text := "Click here: https://example.com/verify?token=abc123 to verify your email."
	links := ExtractLinks(text)
	if len(links) == 0 {
		t.Fatal("Expected at least one link")
	}
	if links[0] != "https://example.com/verify?token=abc123" {
		t.Errorf("Unexpected link: %q", links[0])
	}
}

func TestExtractLinks_Multiple(t *testing.T) {
	text := "Visit http://a.com and https://b.com/path for more info."
	links := ExtractLinks(text)
	if len(links) != 2 {
		t.Fatalf("Expected 2 links, got %d", len(links))
	}
}

func TestExtractLinks_Dedup(t *testing.T) {
	text := "Click https://example.com here or https://example.com there."
	links := ExtractLinks(text)
	if len(links) != 1 {
		t.Fatalf("Expected 1 deduplicated link, got %d", len(links))
	}
}

func TestExtractLinks_NoLinks(t *testing.T) {
	text := "No links in this email at all."
	links := ExtractLinks(text)
	if len(links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(links))
	}
}

func TestExtractLinks_TrailingPunctuation(t *testing.T) {
	text := "Go to https://example.com/page. And also https://example.com/other!"
	links := ExtractLinks(text)
	for _, l := range links {
		if l[len(l)-1] == '.' || l[len(l)-1] == '!' {
			t.Errorf("Link should not have trailing punctuation: %q", l)
		}
	}
}
