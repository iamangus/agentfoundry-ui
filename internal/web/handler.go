package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/angoo/agentfoundry-ui/internal/api"
	"github.com/angoo/agentfoundry-ui/internal/auth"
)

//go:embed templates/*.html
var templateFS embed.FS

type UserInfoData struct {
	Username    string
	Email       string
	IsAdmin     bool
	IsTeamAdmin bool
	Teams       []string
}

type chatPageData struct {
	ActivePage  string
	Agents      []*api.Definition
	Sessions    []*api.Session
	Current     *api.Session
	ActiveRunID string
	User        *UserInfoData
}

type postMessageData struct {
	Content   string
	SessionID string
	RunID     string
}

type agentsPageData struct {
	ActivePage string
	Agents     []*api.Definition
	User       *UserInfoData
}

type agentEditorData struct {
	Def                  *api.Definition
	StructuredOutputJSON string
	User                 *UserInfoData
}

type saveYamlData struct {
	Editor agentEditorData
	Agents []*api.Definition
}

type toolsPageData struct {
	ActivePage string
	Servers    []serverTools
	User       *UserInfoData
}

type apiKeysPageData struct {
	ActivePage string
	Keys       []apiKeyData
	NewKey     *apiKeyData
	User       *UserInfoData
}

type apiKeyData struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	FullKey    string     `json:"full_key,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type toolInfo struct {
	QualifiedName string
	Server        string
	Name          string
	Description   string
}

type serverTools struct {
	Name  string
	Tools []toolInfo
}

type toolGenerateData struct {
	YAML     string
	Selected int
	Lines    int
}

type Handler struct {
	client  *api.Client
	tmpl    *template.Template
	authMgr *auth.Manager
}

func NewHandler(backendURL string, authMgr *auth.Manager) (*Handler, error) {
	client, err := api.NewClient(backendURL)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	h := &Handler{
		client:  client,
		authMgr: authMgr,
	}

	funcMap := template.FuncMap{
		"renderMarkdown": renderMarkdown,
		"dict": func(kvs ...any) map[string]any {
			m := make(map[string]any, len(kvs)/2)
			for i := 0; i+1 < len(kvs); i += 2 {
				if key, ok := kvs[i].(string); ok {
					m[key] = kvs[i+1]
				} else {
					slog.Warn("dict: non-string key ignored in template helper", "index", i, "key", kvs[i])
				}
			}
			if len(kvs)%2 != 0 {
				slog.Warn("dict: odd number of arguments in template helper", "count", len(kvs))
			}
			return m
		},
		"json": func(v any) template.JS {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"jsonAttr": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				return "[]"
			}
			return string(b)
		},
		"truncate": func(s string, max int) string {
			if len(s) <= max {
				return s
			}
			return s[:max] + "..."
		},
		"joinLines": func(ss []string) string {
			return strings.Join(ss, "\n")
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	h.tmpl = tmpl

	return h, nil
}

func (h *Handler) Client() *api.Client {
	return h.client
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.redirectToChat)
	mux.HandleFunc("GET /chat", h.chatPage)
	mux.HandleFunc("POST /chat/sessions", h.newSession)
	mux.HandleFunc("GET /chat/sessions/list", h.sessionListPartial)
	mux.HandleFunc("POST /chat/sessions/{id}/messages", h.postMessage)
	mux.HandleFunc("GET /chat/runs/{id}/events", h.runEvents)
	mux.HandleFunc("GET /agents", h.agentsPage)
	mux.HandleFunc("GET /agents/list", h.agentListPartial)
	mux.HandleFunc("GET /agents/new", h.newAgentEditor)
	mux.HandleFunc("GET /agents/{name}/edit", h.agentEditPartial)
	mux.HandleFunc("PUT /agents/{name}", h.saveAgentForm)
	mux.HandleFunc("POST /agents/form", h.createAgentFormNew)
	mux.HandleFunc("POST /agents/{name}/clone", h.cloneAgent)
	mux.HandleFunc("DELETE /agents/{name}", h.deleteAgentWeb)
	mux.HandleFunc("GET /tools", h.toolsPage)
	mux.HandleFunc("GET /tools/list", h.toolListPartial)
	mux.HandleFunc("POST /tools/generate", h.toolGeneratePartial)
	mux.HandleFunc("GET /api-keys", h.apiKeysPage)
	mux.HandleFunc("POST /api-keys", h.createAPIKey)
	mux.HandleFunc("DELETE /api-keys/{id}", h.revokeAPIKey)
	slog.Info("web UI routes registered")
}

func (h *Handler) userFromRequest(r *http.Request) *UserInfoData {
	ui := auth.UserInfoFromRequest(r)
	if ui == nil {
		return nil
	}
	return &UserInfoData{
		Username:    ui.Username,
		Email:       ui.Email,
		IsAdmin:     ui.IsAdmin,
		IsTeamAdmin: ui.IsTeamAdmin,
		Teams:       ui.Teams,
	}
}

func (h *Handler) redirectToChat(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/chat", http.StatusFound)
}

func (h *Handler) chatPage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")

	agents, err := h.client.ListAgents(r.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}

	sessions, err := h.client.ListSessions(r.Context())
	if err != nil {
		slog.Error("failed to list sessions", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}

	var current *api.Session
	var activeRunID string
	if sessionID != "" {
		for _, s := range sessions {
			if s.ID == sessionID {
				current = s
				activeRunID = s.ActiveRunID
				break
			}
		}
	}

	data := chatPageData{
		ActivePage:  "chat",
		Agents:      agents,
		Sessions:    sessions,
		Current:     current,
		ActiveRunID: activeRunID,
		User:        h.userFromRequest(r),
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, "chat-content", data)
		return
	}

	h.render(w, "chat.html", data)
}

func (h *Handler) newSession(w http.ResponseWriter, r *http.Request) {
	agentName := r.FormValue("agent")
	if agentName == "" {
		http.Error(w, "agent is required", http.StatusBadRequest)
		return
	}

	sess, err := h.client.CreateSession(r.Context(), agentName)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}

	agents, _ := h.client.ListAgents(r.Context())
	sessions, _ := h.client.ListSessions(r.Context())

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", "/chat?session="+sess.ID)
		data := chatPageData{
			ActivePage: "chat",
			Agents:     agents,
			Sessions:   sessions,
			Current:    sess,
			User:       h.userFromRequest(r),
		}
		h.renderPartial(w, "new-session-response", data)
		return
	}

	http.Redirect(w, r, "/chat?session="+sess.ID, http.StatusSeeOther)
}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	content := r.FormValue("message")
	if content == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	result, err := h.client.PostMessage(r.Context(), sessionID, content)
	if err != nil {
		slog.Error("failed to post message", "session", sessionID, "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderPartial(w, "post-message-response", postMessageData{
		Content:   content,
		SessionID: sessionID,
		RunID:     result.RunID,
	})
}

func (h *Handler) runEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	reader, err := h.client.StreamRunEventsReader(r.Context(), runID)
	if err != nil {
		slog.Error("failed to connect to SSE stream", "run", runID, "error", err)
		http.Error(w, "stream error", http.StatusBadGateway)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				slog.Error("SSE read error", "run", runID, "error", err)
			}
			return
		}
	}
}

func (h *Handler) sessionListPartial(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.client.ListSessions(r.Context())
	if err != nil {
		slog.Error("failed to list sessions", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	data := chatPageData{
		Sessions: sessions,
		User:     h.userFromRequest(r),
	}
	h.renderPartial(w, "session-list", data)
}

func (h *Handler) agentsPage(w http.ResponseWriter, r *http.Request) {
	agents, err := h.client.ListAgents(r.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	data := agentsPageData{
		ActivePage: "agents",
		Agents:     agents,
		User:       h.userFromRequest(r),
	}
	h.render(w, "agents.html", data)
}

func (h *Handler) agentListPartial(w http.ResponseWriter, r *http.Request) {
	agents, err := h.client.ListAgents(r.Context())
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	data := agentsPageData{
		Agents: agents,
		User:   h.userFromRequest(r),
	}
	h.renderPartial(w, "agent-list-items", data)
}

func (h *Handler) agentEditPartial(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	def, err := h.client.GetAgent(r.Context(), name)
	if err != nil {
		slog.Error("failed to get agent", "name", name, "error", err)
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	h.renderPartial(w, "agent-editor", agentEditorData{
		Def:                  def,
		StructuredOutputJSON: structuredOutputJSON(def),
		User:                 h.userFromRequest(r),
	})
}

func (h *Handler) newAgentEditor(w http.ResponseWriter, r *http.Request) {
	h.renderPartial(w, "agent-editor-new", agentEditorData{Def: &api.Definition{}, User: h.userFromRequest(r)})
}

func (h *Handler) saveAgentForm(w http.ResponseWriter, r *http.Request) {
	originalName := r.PathValue("name")
	def, err := definitionFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newName := r.FormValue("name")
	def.Name = newName

	if err := def.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := h.client.UpdateAgent(r.Context(), originalName, def)
	if err != nil {
		slog.Error("failed to save agent", "name", newName, "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}

	agents, _ := h.client.ListAgents(r.Context())
	h.renderPartial(w, "save-agent-response", saveYamlData{
		Editor: agentEditorData{Def: saved, StructuredOutputJSON: structuredOutputJSON(saved), User: h.userFromRequest(r)},
		Agents: agents,
	})
}

func (h *Handler) createAgentFormNew(w http.ResponseWriter, r *http.Request) {
	def, err := definitionFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	def.Name = r.FormValue("name")

	if err := def.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	saved, err := h.client.CreateAgent(r.Context(), def)
	if err != nil {
		slog.Error("failed to create agent", "name", def.Name, "error", err)
		http.Error(w, "failed to create", http.StatusInternalServerError)
		return
	}

	agents, _ := h.client.ListAgents(r.Context())
	h.renderPartial(w, "save-agent-response", saveYamlData{
		Editor: agentEditorData{Def: saved, StructuredOutputJSON: structuredOutputJSON(saved), User: h.userFromRequest(r)},
		Agents: agents,
	})
}

func (h *Handler) cloneAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	src, err := h.client.GetAgent(r.Context(), name)
	if err != nil {
		http.Error(w, "agent not found: "+name, http.StatusNotFound)
		return
	}

	agents, _ := h.client.ListAgents(r.Context())
	existing := make(map[string]bool, len(agents))
	for _, a := range agents {
		existing[a.Name] = true
	}

	cloneName, err := cloneAgentName(name, func(s string) bool { return existing[s] })
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	clone := *src
	clone.Name = cloneName
	clone.Tools = append([]string(nil), src.Tools...)
	if src.StructuredOutput != nil {
		so := *src.StructuredOutput
		clone.StructuredOutput = &so
	}

	saved, err := h.client.CreateAgent(r.Context(), &clone)
	if err != nil {
		slog.Error("failed to clone agent", "source", name, "clone", cloneName, "error", err)
		http.Error(w, "failed to clone", http.StatusInternalServerError)
		return
	}

	agents, _ = h.client.ListAgents(r.Context())
	h.renderPartial(w, "save-agent-response", saveYamlData{
		Editor: agentEditorData{Def: saved, StructuredOutputJSON: structuredOutputJSON(saved), User: h.userFromRequest(r)},
		Agents: agents,
	})
}

func (h *Handler) deleteAgentWeb(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.client.DeleteAgent(r.Context(), name); err != nil {
		slog.Error("failed to delete agent", "name", name, "error", err)
		http.Error(w, "failed to delete", http.StatusInternalServerError)
		return
	}
	agents, _ := h.client.ListAgents(r.Context())
	h.renderPartial(w, "agent-list-items", agentsPageData{Agents: agents})
}

func (h *Handler) toolsPage(w http.ResponseWriter, r *http.Request) {
	servers, err := h.buildServerTools(r.Context())
	if err != nil {
		slog.Error("failed to list tools", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	data := toolsPageData{
		ActivePage: "tools",
		Servers:    servers,
		User:       h.userFromRequest(r),
	}
	h.render(w, "tools.html", data)
}

func (h *Handler) toolListPartial(w http.ResponseWriter, r *http.Request) {
	servers, err := h.buildServerTools(r.Context())
	if err != nil {
		slog.Error("failed to list tools", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}
	data := toolsPageData{
		Servers: servers,
		User:    h.userFromRequest(r),
	}
	h.renderPartial(w, "tool-list", data)
}

func (h *Handler) toolGeneratePartial(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	tools := r.Form["tool"]
	sort.Strings(tools)

	var buf strings.Builder
	if len(tools) > 0 {
		buf.WriteString("tools:\n")
		for _, t := range tools {
			buf.WriteString("  - ")
			buf.WriteString(t)
			buf.WriteString("\n")
		}
	}

	h.renderPartial(w, "tool-generate-result", toolGenerateData{
		YAML:     buf.String(),
		Selected: len(tools),
		Lines:    len(tools) + 1,
	})
}

func (h *Handler) apiKeysPage(w http.ResponseWriter, r *http.Request) {
	keys, err := h.client.ListAPIKeys(r.Context())
	if err != nil {
		slog.Error("failed to list api keys", "error", err)
		http.Error(w, "backend error", http.StatusBadGateway)
		return
	}

	data := apiKeysPageData{
		ActivePage: "api-keys",
		User:       h.userFromRequest(r),
	}

	keyData := make([]apiKeyData, len(keys))
	for i, k := range keys {
		keyData[i] = apiKeyData{
			ID:         k.ID,
			Name:       k.Name,
			KeyPrefix:  k.KeyPrefix,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
			ExpiresAt:  k.ExpiresAt,
		}
	}
	data.Keys = keyData

	h.render(w, "api-keys.html", data)
}

func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	key, err := h.client.CreateAPIKey(r.Context(), req.Name)
	if err != nil {
		slog.Error("failed to create api key", "error", err)
		http.Error(w, "failed to create key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(key)
}

func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.client.RevokeAPIKey(r.Context(), id); err != nil {
		slog.Error("failed to revoke api key", "id", id, "error", err)
		http.Error(w, "failed to revoke", http.StatusInternalServerError)
		return
	}

	keys, _ := h.client.ListAPIKeys(r.Context())
	keyData := make([]apiKeyData, len(keys))
	for i, k := range keys {
		keyData[i] = apiKeyData{
			ID:         k.ID,
			Name:       k.Name,
			KeyPrefix:  k.KeyPrefix,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
			ExpiresAt:  k.ExpiresAt,
		}
	}

	data := apiKeysPageData{
		ActivePage: "api-keys",
		Keys:       keyData,
		User:       h.userFromRequest(r),
	}
	h.renderPartial(w, "api-key-list", data)
}

func (h *Handler) buildServerTools(ctx context.Context) ([]serverTools, error) {
	allTools, err := h.client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	byServer := make(map[string][]toolInfo)
	for _, t := range allTools {
		byServer[t.Server] = append(byServer[t.Server], toolInfo{
			QualifiedName: t.QualifiedName,
			Server:        t.Server,
			Name:          t.Name,
			Description:   t.Description,
		})
	}

	servers := make([]serverTools, 0, len(byServer))
	for srv, tools := range byServer {
		servers = append(servers, serverTools{Name: srv, Tools: tools})
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })
	return servers, nil
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render template", "name", name, "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render partial", "name", name, "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func structuredOutputJSON(def *api.Definition) string {
	if def == nil || def.StructuredOutput == nil {
		return ""
	}
	b, err := json.MarshalIndent(def.StructuredOutput, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

func definitionFromForm(r *http.Request) (*api.Definition, error) {
	def := &api.Definition{
		Kind:         api.KindAgent,
		Description:  r.FormValue("description"),
		Model:        r.FormValue("model"),
		SystemPrompt: r.FormValue("system_prompt"),
		ForceJSON:    r.FormValue("force_json") != "",
		Scope:        r.FormValue("scope"),
		Team:         r.FormValue("team"),
	}
	if v := r.FormValue("max_turns"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			def.MaxTurns = n
		}
	}
	if v := r.FormValue("max_concurrent_tools"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			def.MaxConcurrentTools = n
		}
	}
	toolsStr := r.FormValue("tools")
	for _, line := range strings.Split(toolsStr, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			def.Tools = append(def.Tools, t)
		}
	}
	soJSON := r.FormValue("structured_output_json")
	soEnabled := r.FormValue("structured_output_enabled") == "true"
	if soEnabled && soJSON != "" {
		var so api.StructuredOutput
		if err := json.Unmarshal([]byte(soJSON), &so); err != nil {
			return nil, fmt.Errorf("invalid structured output JSON: %w", err)
		}
		def.StructuredOutput = &so
	}
	return def, nil
}

func cloneAgentName(src string, exists func(string) bool) (string, error) {
	candidate := src + "-copy"
	if !exists(candidate) {
		return candidate, nil
	}
	for i := 2; i <= 10; i++ {
		candidate = fmt.Sprintf("%s-copy-%d", src, i)
		if !exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("too many copies of %q", src)
}
