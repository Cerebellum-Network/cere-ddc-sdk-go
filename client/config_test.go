package client_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cerebellum-network/cere-ddc-sdk-go/client"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv(client.EnvConfig, "")
	t.Setenv(client.EnvToken, "")
	t.Setenv(client.EnvAPIURL, "")

	// No file: the defaults, no token.
	cfg, err := client.LoadConfig()
	if err != nil || cfg.BaseURL != client.DefaultBaseURL || cfg.Token != "" {
		t.Fatalf("%+v %v", cfg, err)
	}
	if _, err := client.NewFromEnv(); err == nil {
		t.Error("NewFromEnv without a token: no error")
	}

	// The file the CLI writes.
	path := filepath.Join(dir, "cere", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"api_url":"https://api.example/","token":" file-token ",`+
		`"login":{"key":"k","grant":"g"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = client.LoadConfig()
	if err != nil || cfg.BaseURL != "https://api.example" || cfg.Token != "file-token" || cfg.TokenFrom != path {
		t.Fatalf("file: %+v %v", cfg, err)
	}

	// The environment wins.
	t.Setenv(client.EnvToken, "env-token")
	t.Setenv(client.EnvAPIURL, "https://other.example")
	cfg, err = client.LoadConfig()
	if err != nil || cfg.BaseURL != "https://other.example" || cfg.Token != "env-token" || cfg.TokenFrom != client.EnvToken {
		t.Fatalf("env: %+v %v", cfg, err)
	}
	c, err := client.NewFromEnv()
	if err != nil || c.BaseURL() != "https://other.example" {
		t.Fatalf("NewFromEnv: %v", err)
	}

	// CERE_CONFIG names the file; a broken file is an error.
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(client.EnvConfig, broken)
	if _, err := client.LoadConfig(); err == nil {
		t.Error("a broken file: no error")
	}
}
