package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/angoo/agentfoundry-ui/internal/config"
)

type Handler struct {
	mgr *Manager
	cfg *config.Config
}

func NewHandler(mgr *Manager, cfg *config.Config) *Handler {
	return &Handler{mgr: mgr, cfg: cfg}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	if !h.mgr.Enabled() {
		return
	}
	mux.HandleFunc("GET /auth/login", h.login)
	mux.HandleFunc("GET /auth/callback", h.callback)
	mux.HandleFunc("GET /auth/logout", h.logout)
	mux.HandleFunc("GET /auth/me", h.me)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	state := GenerateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   600,
	})

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	callbackURL := scheme + "://" + r.Host + "/auth/callback"

	loginURL := h.mgr.LoginURL(state, callbackURL)
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}

	returnedState := r.URL.Query().Get("state")
	if returnedState == "" || returnedState != stateCookie.Value {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	redirectURL := scheme + "://" + r.Host + "/auth/callback"

	sd, err := h.mgr.ExchangeCode(r.Context(), code, redirectURL)
	if err != nil {
		slog.Error("failed to exchange code", "error", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	sessionID := h.mgr.CreateSession(sd)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("user logged in", "user", sd.UserInfo.Username, "subject", sd.UserInfo.Subject)
	http.Redirect(w, r, "/chat", http.StatusFound)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		h.mgr.DeleteSession(cookie.Value)
	}

	base := strings.TrimRight(h.cfg.KeycloakURL, "/")
	logoutURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout?client_id=%s&post_logout_redirect_uri=%s",
		base, h.cfg.KeycloakRealm, h.cfg.KeycloakClientID,
		"http://"+r.Host+"/auth/login",
	)

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, logoutURL, http.StatusFound)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	ui := UserInfoFromRequest(r)
	if ui == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ui)
}
