package auth

import (
	"strings"
	"testing"
)

func TestLoginPageThemeAccessibilityAndEscaping(t *testing.T) {
	page := renderLoginPage(`/auth/start?next=" onmouseover="unsafe`)
	for _, required := range []string{`<main class="login-card">`, `<h1 class="login-brand">`, `localStorage.getItem('theme')`, `prefers-color-scheme: dark`, `focus-visible`, `min-height: 100dvh`} {
		if !strings.Contains(page, required) {
			t.Errorf("login page missing %q", required)
		}
	}
	if strings.Contains(page, `href="/auth/start?next=" onmouseover=`) || !strings.Contains(page, `&#34;`) {
		t.Fatal("login URL is not escaped")
	}
}
