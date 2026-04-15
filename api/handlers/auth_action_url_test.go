package handlers

import "testing"

func TestBuildActionURL(t *testing.T) {
	tests := []struct {
		name       string
		base       string
		fallback   string
		token      string
		expect     string
	}{
		{
			name:     "uses fallback when base missing",
			base:     "",
			fallback: "/reset-password",
			token:    "tok123",
			expect:   "/reset-password?token=tok123",
		},
		{
			name:     "appends token to absolute url",
			base:     "https://app.example.com/reset-password",
			fallback: "/reset-password",
			token:    "tok123",
			expect:   "https://app.example.com/reset-password?token=tok123",
		},
		{
			name:     "replaces existing token-like query keys",
			base:     "https://app.example.com/reset-password?foo=1&token=old&reset_token=old2",
			fallback: "/reset-password",
			token:    "newtok",
			expect:   "https://app.example.com/reset-password?foo=1&token=newtok",
		},
		{
			name:     "escapes token in fallback string path",
			base:     "not a url",
			fallback: "/reset-password",
			token:    "a+b c",
			expect:   "not%20a%20url?token=a%2Bb+c",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildActionURL(tc.base, tc.fallback, tc.token)
			if got != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestRedactTokenForLog(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		expect string
	}{
		{name: "empty token", token: "", expect: ""},
		{name: "short token", token: "abcd", expect: "***"},
		{name: "long token", token: "abcdefgh12345678", expect: "abcd...5678"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := redactTokenForLog(tc.token)
			if got != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestRedactActionURLForLog(t *testing.T) {
	raw := "https://app.example.com/reset-password?foo=1&token=abcdefghijklmnopqrstuvwxyz"
	got := redactActionURLForLog(raw)
	expect := "https://app.example.com/reset-password?foo=1&token=abcd...wxyz"
	if got != expect {
		t.Fatalf("expected %q, got %q", expect, got)
	}
}
