package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc"

	"github.com/angoo/agentfoundry-ui/internal/config"
)

type UserInfo struct {
	Subject     string   `json:"sub"`
	Username    string   `json:"preferred_username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Groups      []string `json:"groups"`
	IsAdmin     bool     `json:"is_admin"`
	IsTeamAdmin bool     `json:"is_team_admin"`
	Teams       []string `json:"teams"`
}

type sessionData struct {
	AccessToken string
	UserInfo    UserInfo
	ExpiresAt   time.Time
}

type Manager struct {
	cfg      *config.Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier

	mu       sync.RWMutex
	sessions map[string]*sessionData
}

func NewManager(ctx context.Context, cfg *config.Config) (*Manager, error) {
	if !cfg.AuthEnabled() {
		return &Manager{cfg: cfg, sessions: make(map[string]*sessionData)}, nil
	}

	issuer := strings.TrimRight(cfg.KeycloakURL, "/") + "/realms/" + cfg.KeycloakRealm
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("init OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.KeycloakClientID,
	})

	m := &Manager{
		cfg:      cfg,
		provider: provider,
		verifier: verifier,
		sessions: make(map[string]*sessionData),
	}

	go m.cleanupExpired()

	return m, nil
}

func (m *Manager) Enabled() bool {
	return m.cfg.AuthEnabled()
}

func (m *Manager) LoginURL(state, redirectURL string) string {
	base := strings.TrimRight(m.cfg.KeycloakURL, "/")
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+profile+email&state=%s",
		base, m.cfg.KeycloakRealm, m.cfg.KeycloakClientID, redirectURL, state,
	)
}

func (m *Manager) ExchangeCode(ctx context.Context, code, redirectURL string) (*sessionData, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimRight(m.cfg.KeycloakURL, "/"), m.cfg.KeycloakRealm)

	data := "grant_type=authorization_code" +
		"&code=" + code +
		"&redirect_uri=" + redirectURL +
		"&client_id=" + m.cfg.KeycloakClientID

	if m.cfg.KeycloakClientSecret != "" {
		data += "&client_secret=" + m.cfg.KeycloakClientSecret
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange: %s: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	rawIDToken := tokenResp.AccessToken
	idToken, err := m.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	sub, _ := claims["sub"].(string)
	username, _ := claims["preferred_username"].(string)
	email, _ := claims["email"].(string)

	roles := extractClaim(claims, "realm_access.roles")
	groups := extractClaim(claims, "groups")

	isAdmin := false
	for _, r := range roles {
		if r == m.cfg.AdminRole {
			isAdmin = true
			break
		}
	}

	isTeamAdmin := false
	for _, r := range roles {
		if r == m.cfg.TeamAdminRole {
			isTeamAdmin = true
			break
		}
	}

	teams := make([]string, 0, len(groups))
	for _, g := range groups {
		t := strings.TrimPrefix(g, "/")
		if t != "" && !strings.Contains(t, "/") {
			teams = append(teams, t)
		}
	}

	userInfo := UserInfo{
		Subject:     sub,
		Username:    username,
		Email:       email,
		Roles:       roles,
		Groups:      groups,
		IsAdmin:     isAdmin,
		IsTeamAdmin: isTeamAdmin,
		Teams:       teams,
	}

	sd := &sessionData{
		AccessToken: tokenResp.AccessToken,
		UserInfo:    userInfo,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return sd, nil
}

func (m *Manager) CreateSession(sd *sessionData) string {
	b := make([]byte, 32)
	rand.Read(b)
	sessionID := base64.URLEncoding.EncodeToString(b)

	m.mu.Lock()
	m.sessions[sessionID] = sd
	m.mu.Unlock()

	return sessionID
}

func (m *Manager) GetSession(sessionID string) (*sessionData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sd, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if time.Now().After(sd.ExpiresAt) {
		return nil, false
	}
	return sd, true
}

func (m *Manager) DeleteSession(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

func (m *Manager) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for id, sd := range m.sessions {
			if now.After(sd.ExpiresAt) {
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
	}
}

func schemeForRequest(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return "https"
	}
	return "http"
}

func extractClaim(claims map[string]any, path string) []string {
	parts := strings.Split(path, ".")
	var current any = claims
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}
	switch v := current.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{v}
	default:
		return nil
	}
}

func GenerateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type contextKey struct{}

type accessTokenKey struct{}

func UserInfoFromRequest(r *http.Request) *UserInfo {
	if v, ok := r.Context().Value(contextKey{}).(*UserInfo); ok {
		return v
	}
	return nil
}

func ContextWithUserInfo(ctx context.Context, ui *UserInfo) context.Context {
	return context.WithValue(ctx, contextKey{}, ui)
}

func ContextWithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, token)
}

func AccessTokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(accessTokenKey{}).(string); ok {
		return v
	}
	return ""
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.Enabled() {
			next.ServeHTTP(w, r)
			return
		}

		switch r.URL.Path {
		case "/auth/login", "/auth/callback", "/auth/logout", "/auth/me":
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			m.redirectToLogin(w, r)
			return
		}

		sd, ok := m.GetSession(cookie.Value)
		if !ok {
			http.SetCookie(w, &http.Cookie{
				Name:   "session",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			m.redirectToLogin(w, r)
			return
		}

		ctx := ContextWithUserInfo(r.Context(), &sd.UserInfo)
		ctx = ContextWithAccessToken(ctx, sd.AccessToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Manager) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

func renderLoginPage(loginURL string) string {
	return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sign in — agentfoundry</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap">
<style>
  *, *::before, *::after { box-sizing: border-box; }
  .dark {
    --bg-base: #0f0f11;
    --bg-card: #18181c;
    --border: #27272a;
    --text-base: #e4e4e7;
    --text-muted: #71717a;
    --purple: oklch(59.1% 0.249 292.7);
    --purple-dim: oklch(59.1% 0.249 292.7 / 0.15);
    --purple-solid: #7c3aed;
  }
  html, body {
    margin: 0; height: 100%;
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    background: var(--bg-base); color: var(--text-base);
  }
  body {
    display: flex; align-items: center; justify-content: center;
  }
  .login-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 40px;
    width: 380px;
    max-width: 90vw;
    text-align: center;
  }
  .login-icon {
    width: 56px; height: 56px; border-radius: 14px;
    background: var(--purple-dim);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.4);
    display: flex; align-items: center; justify-content: center;
    margin: 0 auto 20px;
  }
  .login-icon svg { width: 28px; height: 28px; color: var(--purple); }
  .login-brand {
    font-size: 1.1rem; font-weight: 700;
    margin-bottom: 8px;
  }
  .login-brand-agent { color: var(--text-base); }
  .login-brand-file { color: var(--purple); }
  .login-sub {
    font-size: 0.85rem; color: var(--text-muted);
    margin-bottom: 28px; line-height: 1.5;
  }
  .login-btn {
    display: inline-flex; align-items: center; justify-content: center; gap: 8px;
    width: 100%; padding: 12px;
    background: var(--purple-solid); color: #fff;
    border: none; border-radius: 10px;
    font-family: inherit; font-size: 0.9rem; font-weight: 600;
    cursor: pointer; text-decoration: none;
    transition: opacity 0.15s;
  }
  .login-btn:hover { opacity: 0.88; }
  .login-btn svg { width: 16px; height: 16px; }
</style>
</head>
<body>
<div class="login-card">
  <div class="login-icon">
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
    </svg>
  </div>
  <div class="login-brand">
    <span class="login-brand-agent">agent</span><span class="login-brand-file">foundry</span>
  </div>
  <p class="login-sub">Sign in to continue to agentfoundry</p>
  <a class="login-btn" href="` + loginURL + `">
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"></path>
      <polyline points="10 17 15 12 10 7"></polyline>
      <line x1="15" y1="12" x2="3" y2="12"></line>
    </svg>
    Sign in with Keycloak
  </a>
</div>
</body>
</html>`
}
