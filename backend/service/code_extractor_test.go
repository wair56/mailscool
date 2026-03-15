package service

import "testing"

func TestExtractCode_OneTimeCodeIsPattern(t *testing.T) {
	text := "Sign in to Cursor\n\nYou requested to sign in to Cursor. Your one-time code is:\n\n880730\n\nThis code expires in 10 minutes."

	got := ExtractCode(text)
	if got != "880730" {
		t.Fatalf("expected code 880730, got %q", got)
	}
}

func TestExtractCode_VerificationCodeIsPattern(t *testing.T) {
	text := "Your verification code is 123456"

	got := ExtractCode(text)
	if got != "123456" {
		t.Fatalf("expected code 123456, got %q", got)
	}
}
