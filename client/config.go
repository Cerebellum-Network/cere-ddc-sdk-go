package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The environment variables the `cere` CLI, the Python SDK and LoadConfig
// read.
const (
	// EnvToken is an access token; it overrides the file's.
	EnvToken = "CERE_TOKEN"
	// EnvAPIURL is the API's base URL; it overrides the file's.
	EnvAPIURL = "CERE_API_URL"
	// EnvConfig is the path of the configuration file.
	EnvConfig = "CERE_CONFIG"
)

// Config is where to call and with what token, as the `cere` CLI saves it.
type Config struct {
	// BaseURL is the API's base URL.
	BaseURL string
	// Token is the access token: after `cere login` in the browser, the
	// bucket owner's grant to the CLI's key with the level that key signed.
	// Empty when none is configured. It never prints (see StaticToken);
	// string(Token) is the token itself.
	Token StaticToken
	// TokenFrom says where Token came from: the file's path or EnvToken.
	TokenFrom string
}

// ConfigPath is the `cere` CLI's configuration file: $CERE_CONFIG, else
// $XDG_CONFIG_HOME/cere/config.json, else ~/.config/cere/config.json.
func ConfigPath() (string, error) {
	if p := os.Getenv(EnvConfig); p != "" {
		return p, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "cere", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cere: config: %w", err)
	}
	return filepath.Join(home, ".config", "cere", "config.json"), nil
}

// LoadConfig reads what `cere login` saved, then the environment on top
// (CERE_TOKEN, CERE_API_URL), then the defaults: the same file and rules as the
// CLI and the Python SDK. A missing file is not an error.
func LoadConfig() (Config, error) {
	cfg := Config{}
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return cfg, fmt.Errorf("cere: config: %w", err)
	default:
		var file struct {
			APIURL string `json:"api_url"`
			Token  string `json:"token"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			return cfg, fmt.Errorf("cere: config: %s is not valid JSON: %w", path, err)
		}
		cfg.BaseURL, cfg.Token = file.APIURL, StaticToken(strings.TrimSpace(file.Token))
		if cfg.Token != "" {
			cfg.TokenFrom = path
		}
	}
	if t := strings.TrimSpace(os.Getenv(EnvToken)); t != "" {
		cfg.Token, cfg.TokenFrom = StaticToken(t), EnvToken
	}
	if u := os.Getenv(EnvAPIURL); u != "" {
		cfg.BaseURL = u
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}

// NewFromEnv is New with the base URL and token from LoadConfig, then opts.
// It returns an error when no token is configured.
func NewFromEnv(opts ...Option) (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Token == "" {
		return nil, errors.New("cere: not logged in: run `cere login`, or set CERE_TOKEN to an access token " +
			"from the developer console")
	}
	return New(append([]Option{WithBaseURL(cfg.BaseURL), WithToken(string(cfg.Token))}, opts...)...)
}
