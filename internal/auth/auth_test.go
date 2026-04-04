package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/angoo/agentfoundry-ui/internal/config"
)

func newTestManager() *Manager {
	return &Manager{
		cfg: &config.Config{
			KeycloakURL:      "https://keycloak.example.com",
			KeycloakRealm:    "opendev",
			KeycloakClientID: "agentfoundry-ui",
			SessionSecret:    "test-secret",
		},
		sessions: make(map[string]*sessionData),
	}
}

func newDisabledManager() *Manager {
	return &Manager{
		cfg:      &config.Config{},
		sessions: make(map[string]*sessionData),
	}
}

func TestManager_Enabled(t *testing.T) {
	m := newTestManager()
	if !m.Enabled() {
		t.Error("expected enabled")
	}

	m2 := newDisabledManager()
	if m2.Enabled() {
		t.Error("expected disabled")
	}
}

func TestManager_CreateAndGetSession(t *testing.T) {
	m := newTestManager()
	sd := &sessionData{
		AccessToken: "token-123",
		UserInfo: UserInfo{
			Subject:  "user-1",
			Username: "alice",
			Email:    "alice@example.com",
		},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	id := m.CreateSession(sd)
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	got, ok := m.GetSession(id)
	if !ok {
		t.Fatal("expected to find session")
	}
	if got.AccessToken != "token-123" {
		t.Errorf("got access token %q", got.AccessToken)
	}
	if got.UserInfo.Username != "alice" {
		t.Errorf("got username %q", got.UserInfo.Username)
	}
}

func TestManager_GetSession_NotFound(t *testing.T) {
	m := newTestManager()
	_, ok := m.GetSession("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestManager_GetSession_Expired(t *testing.T) {
	m := newTestManager()
	sd := &sessionData{
		AccessToken: "token",
		UserInfo:    UserInfo{Subject: "u1"},
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	}

	id := m.CreateSession(sd)
	_, ok := m.GetSession(id)
	if ok {
		t.Error("expected expired session to not be found")
	}
}

func TestManager_DeleteSession(t *testing.T) {
	m := newTestManager()
	sd := &sessionData{
		AccessToken: "token",
		UserInfo:    UserInfo{Subject: "u1"},
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	id := m.CreateSession(sd)
	m.DeleteSession(id)

	_, ok := m.GetSession(id)
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestManager_Middleware_DisabledAuth(t *testing.T) {
	m := newDisabledManager()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := m.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Error("expected next handler to be called when auth is disabled")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}
}

func TestManager_Middleware_NoCookie(t *testing.T) {
	m := newTestManager()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	mw := m.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "agentfoundry") {
		t.Error("expected agentfoundry branding in login page")
	}
	if !strings.Contains(body, "Sign in with Keycloak") {
		t.Error("expected sign in button in login page")
	}
}

func TestManager_Middleware_InvalidSession(t *testing.T) {
	m := newTestManager()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	mw := m.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "invalid"})
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}

	cookie := rec.Result().Cookies()
	found := false
	for _, c := range cookie {
		if c.Name == "session" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie to be cleared")
	}
}

func TestManager_Middleware_ValidSession(t *testing.T) {
	m := newTestManager()
	sd := &sessionData{
		AccessToken: "my-token",
		UserInfo: UserInfo{
			Subject:  "user-42",
			Username: "bob",
			IsAdmin:  true,
		},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	sessionID := m.CreateSession(sd)

	var gotReq *http.Request
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		w.WriteHeader(http.StatusOK)
	})

	mw := m.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	ui := UserInfoFromRequest(gotReq)
	if ui == nil {
		t.Fatal("expected UserInfo in context")
	}
	if ui.Subject != "user-42" {
		t.Errorf("got subject %q, want %q", ui.Subject, "user-42")
	}
	if ui.Username != "bob" {
		t.Errorf("got username %q, want %q", ui.Username, "bob")
	}
	if !ui.IsAdmin {
		t.Error("expected IsAdmin=true")
	}

	token := AccessTokenFromContext(gotReq.Context())
	if token != "my-token" {
		t.Errorf("got access token %q, want %q", token, "my-token")
	}
}

func TestExtractClaim(t *testing.T) {
	claims := map[string]any{
		"realm_access": map[string]any{
			"roles": []any{"opendev-user", "team-admin"},
		},
		"groups": []any{"/engineering", "/product"},
		"simple": "single-value",
	}

	roles := extractClaim(claims, "realm_access.roles")
	if len(roles) != 2 || roles[0] != "opendev-user" {
		t.Errorf("got roles %v", roles)
	}

	groups := extractClaim(claims, "groups")
	if len(groups) != 2 || groups[0] != "/engineering" {
		t.Errorf("got groups %v", groups)
	}

	single := extractClaim(claims, "simple")
	if len(single) != 1 || single[0] != "single-value" {
		t.Errorf("got single %v", single)
	}

	missing := extractClaim(claims, "nonexistent.path")
	if missing != nil {
		t.Errorf("expected nil for missing, got %v", missing)
	}
}

func TestGenerateState(t *testing.T) {
	s1 := GenerateState()
	s2 := GenerateState()
	if s1 == "" || s2 == "" {
		t.Error("expected non-empty state")
	}
	if s1 == s2 {
		t.Error("expected different states on each call")
	}
}

func TestLoginURL(t *testing.T) {
	m := newTestManager()
	url := m.LoginURL("test-state", "http://localhost:8080/auth/callback")
	if !strings.Contains(url, "client_id=agentfoundry-ui") {
		t.Errorf("expected client_id in URL: %s", url)
	}
	if !strings.Contains(url, "state=test-state") {
		t.Errorf("expected state in URL: %s", url)
	}
	if !strings.Contains(url, "redirect_uri=http://localhost:8080/auth/callback") {
		t.Errorf("expected redirect_uri in URL: %s", url)
	}
	if !strings.Contains(url, "opendev/protocol/openid-connect/auth") {
		t.Errorf("expected auth endpoint in URL: %s", url)
	}
}

func TestHandler_Callback_MissingState(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc", nil)
	rec := httptest.NewRecorder()
	h.callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", rec.Code)
	}
}

func TestHandler_Callback_StateMismatch(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=abc&state=wrong-state", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct-state"})
	rec := httptest.NewRecorder()
	h.callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", rec.Code)
	}
}

func TestHandler_Callback_MissingCode(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state=test", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "test"})
	rec := httptest.NewRecorder()
	h.callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", rec.Code)
	}
}

func TestHandler_Me_NotAuthenticated(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()
	h.me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", rec.Code)
	}
}

func TestHandler_Me_Authenticated(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	ui := &UserInfo{
		Subject:  "user-1",
		Username: "alice",
		Email:    "alice@example.com",
		IsAdmin:  true,
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	ctx := ContextWithUserInfo(req.Context(), ui)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.me(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("got content type %q, want application/json", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"alice"`) {
		t.Errorf("expected username in response: %s", body)
	}
}

func TestHandler_Logout_ClearsCookie(t *testing.T) {
	m := newTestManager()
	cfg := &config.Config{
		KeycloakURL:      "https://keycloak.example.com",
		KeycloakRealm:    "opendev",
		KeycloakClientID: "agentfoundry-ui",
	}
	h := NewHandler(m, cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "some-session"})
	rec := httptest.NewRecorder()
	h.logout(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("got status %d, want 302", rec.Code)
	}

	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie to be cleared on logout")
	}

	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "openid-connect/logout") {
		t.Errorf("expected redirect to Keycloak logout, got %q", loc)
	}
}
