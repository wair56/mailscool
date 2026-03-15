package middleware

import (
	"testing"
)

func TestIsIPAllowed_ExactMatch(t *testing.T) {
	tests := []struct {
		clientIP  string
		whitelist string
		expected  bool
	}{
		{"192.168.1.1", "192.168.1.1", true},
		{"192.168.1.2", "192.168.1.1", false},
		{"10.0.0.1", "10.0.0.1,10.0.0.2", true},
		{"10.0.0.2", "10.0.0.1,10.0.0.2", true},
		{"10.0.0.3", "10.0.0.1,10.0.0.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.clientIP+"_"+tt.whitelist, func(t *testing.T) {
			got := isIPAllowed(tt.clientIP, tt.whitelist)
			if got != tt.expected {
				t.Errorf("isIPAllowed(%q, %q) = %v, want %v", tt.clientIP, tt.whitelist, got, tt.expected)
			}
		})
	}
}

func TestIsIPAllowed_CIDR(t *testing.T) {
	tests := []struct {
		clientIP  string
		whitelist string
		expected  bool
	}{
		{"192.168.1.100", "192.168.1.0/24", true},
		{"192.168.2.1", "192.168.1.0/24", false},
		{"10.0.0.5", "10.0.0.0/8", true},
		{"172.16.0.1", "10.0.0.0/8", false},
	}

	for _, tt := range tests {
		t.Run(tt.clientIP+"_"+tt.whitelist, func(t *testing.T) {
			got := isIPAllowed(tt.clientIP, tt.whitelist)
			if got != tt.expected {
				t.Errorf("isIPAllowed(%q, %q) = %v, want %v", tt.clientIP, tt.whitelist, got, tt.expected)
			}
		})
	}
}

func TestIsIPAllowed_JSONFormat(t *testing.T) {
	// IP 白名单也支持 JSON 数组格式
	got := isIPAllowed("192.168.1.1", `["192.168.1.1","10.0.0.1"]`)
	if !got {
		t.Error("Expected true for JSON array whitelist match")
	}

	got = isIPAllowed("172.16.0.1", `["192.168.1.1","10.0.0.1"]`)
	if got {
		t.Error("Expected false for JSON array whitelist non-match")
	}
}

func TestIsIPAllowed_Empty(t *testing.T) {
	// 空白名单应不允许
	got := isIPAllowed("192.168.1.1", "")
	if got {
		t.Error("Expected false for empty whitelist")
	}
}

func TestIsIPAllowed_MixedCIDRAndExact(t *testing.T) {
	whitelist := "10.0.0.1,192.168.0.0/16"
	
	if !isIPAllowed("10.0.0.1", whitelist) {
		t.Error("Should allow exact IP match")
	}
	if !isIPAllowed("192.168.5.5", whitelist) {
		t.Error("Should allow CIDR match")
	}
	if isIPAllowed("172.16.0.1", whitelist) {
		t.Error("Should deny non-matching IP")
	}
}
