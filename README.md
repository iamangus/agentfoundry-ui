# agentfoundry-ui

Web frontend for [agentfoundry](https://github.com/angoo/agentfoundry). Provides a browser-based UI for chatting with agents, managing agent definitions, and browsing available tools. Connects to the agentfoundry backend via REST API — never to Temporal or the worker directly.

## Quick Start

### Build and Run

```bash
go build -o agentfoundry-ui ./cmd/server/
BACKEND_URL=http://localhost:3000 ./agentfoundry-ui
```

With auth enabled:

```bash
KEYCLOAK_URL=https://keycloak.example.com \
KEYCLOAK_REALM=opendev \
KEYCLOAK_CLIENT_ID=agentfoundry-ui \
SESSION_SECRET=$(openssl rand -base64 32) \
BACKEND_URL=http://localhost:3000 \
./agentfoundry-ui
```

### Docker

```bash
docker build -t agentfoundry-ui .
docker run -p 8080:8080 \
  -e BACKEND_URL=http://host.docker.internal:3000 \
  -e KEYCLOAK_URL=https://keycloak.example.com \
  -e KEYCLOAK_REALM=opendev \
  -e KEYCLOAK_CLIENT_ID=agentfoundry-ui \
  -e SESSION_SECRET=$(openssl rand -base64 32) \
  agentfoundry-ui
```

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN` | `:8080` | HTTP listen address |
| `BACKEND_URL` | `http://localhost:3000` | agentfoundry backend API URL |
| `KEYCLOAK_URL` | *(empty, auth disabled)* | Base Keycloak URL |
| `KEYCLOAK_REALM` | *(empty, auth disabled)* | Realm name |
| `KEYCLOAK_CLIENT_ID` | *(empty, auth disabled)* | Public client ID for the UI |
| `KEYCLOAK_CLIENT_SECRET` | *(empty)* | Client secret (if confidential client) |
| `SESSION_SECRET` | *(empty, auth disabled)* | Required to enable auth |
| `AUTH_ADMIN_ROLES` | `opendev-admin` | Realm role for admin badge in navbar |
| `AUTH_TEAM_ADMIN_ROLE` | `team-admin` | Realm role for team-admin badge in navbar |

Auth is disabled by default. Set all of `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`, and `SESSION_SECRET` to enable OIDC login.

## Keycloak Setup (for OIDC login)

The UI acts as an OIDC relying party. When auth is enabled, unauthenticated users see a styled login page with a "Sign in with Keycloak" button. After login, the session is stored in a cookie and the access token is forwarded to the backend on every API call.

### Keycloak Client Configuration

Create a client in your Keycloak realm:

| Setting | Value |
|---------|-------|
| **Client ID** | `agentfoundry-ui` |
| **Client authentication** | Off (public client) |
| **Valid redirect URIs** | `http://localhost:8080/auth/callback` |
| **Web origins** | `http://localhost:8080` |
| **Root URL** | `http://localhost:8080` |

For production, replace `localhost:8080` with your actual domain.

### Auth Flow

```
Browser ──> UI (/chat) ──> no session cookie ──> styled login page with Keycloak link
                                                         │
                                               user clicks "Sign in with Keycloak"
                                                         │
Browser <── UI (/auth/callback) <── Keycloak callback with code
    │
    │  UI exchanges code for tokens, stores session cookie
    │
Browser ──> UI (/chat) ──> session valid ──> proxy to backend with Bearer token
```

The UI stores the user's access token in an in-memory session store. On every request to the backend, it reads the session cookie and forwards the token as an `Authorization: Bearer` header. Sessions expire when the access token expires.

## Features

- **Auth** — OIDC login via Keycloak with styled login page. Navbar shows username, admin/team-admin badges, and sign-out link.
- **Chat** — Real-time streaming conversations with agents via SSE. Markdown rendering for both streaming responses (client-side `marked.js`) and historical messages (server-side goldmark). Sessions are owner-scoped.
- **Agents** — Create, edit, clone, and delete agent definitions via a form UI. Scope selector for `global`, `team`, or `user` visibility. Team dropdown for team-scoped agents.
- **Tools** — Browse all discovered MCP tools grouped by server.
- **API Keys** — Create, list, and revoke personal API keys via a dedicated page.

## Architecture

```
                    ┌─────────────────────┐
                    │   agentfoundry-ui   │
                    │     (:8080)         │
                    │                     │
   Browser ────────>│  auth middleware     │
   (HTMX/JS)        │  (session cookie)    │
                    │         │           │
                    │         v           │
                    │  API Client ────────┼──> agentfoundry backend (:3000)
                    │  (Bearer token)     │        │
                    │                     │        v
                    │  SSE byte-copy      │    Keycloak (JWT verify)
                    │  proxy              │
                    └─────────────────────┘
```

The UI never connects to Temporal or the agent worker. All communication goes through the agentfoundry REST API. SSE streams from `GET /api/v1/chat/runs/{id}/events` are proxied as raw byte copies.

## Project Structure

```
agentfoundry-ui/
├── cmd/server/main.go              # Server entrypoint
├── internal/
│   ├── api/
│   │   ├── client.go               # HTTP client for backend API + SSE proxy
│   │   └── types.go                # Definition, Session, Message, API key types
│   ├── auth/
│   │   ├── manager.go              # OIDC provider, session store, auth middleware
│   │   ├── handler.go              # /auth/login, /auth/callback, /auth/logout, /auth/me
│   │   └── auth_test.go            # Tests
│   ├── config/
│   │   └── config.go               # All env vars (listen, backend, Keycloak)
│   └── web/
│       ├── handler.go              # All UI routes (agents, chat, tools, API keys)
│       ├── handler_test.go         # Tests
│       ├── markdown.go             # goldmark renderer for historical messages
│       └── templates/
│           ├── layout.html          # Base layout with navbar + user menu
│           ├── chat.html            # Chat page with streaming
│           ├── agents.html          # Agent management page with scope selector
│           ├── tools.html           # Tool browser page
│           └── api-keys.html        # API key management page
├── Dockerfile
└── go.mod
```
