package config

import "os"

type Config struct {
	Listen     string
	BackendURL string

	KeycloakURL          string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
	SessionSecret        string
}

func Load() *Config {
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":8080"
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:3000"
	}

	return &Config{
		Listen:     listen,
		BackendURL: backendURL,

		KeycloakURL:          os.Getenv("KEYCLOAK_URL"),
		KeycloakRealm:        os.Getenv("KEYCLOAK_REALM"),
		KeycloakClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		KeycloakClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		SessionSecret:        os.Getenv("SESSION_SECRET"),
	}
}

func (c *Config) AuthEnabled() bool {
	return c.KeycloakURL != "" && c.KeycloakRealm != "" && c.KeycloakClientID != "" && c.SessionSecret != ""
}
