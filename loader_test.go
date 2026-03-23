package configr_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Adam-445/configr"
)

// testConfig is a simple struct used across all tests.
type testConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Debug   bool   `json:"debug"`
	Timeout int    `json:"timeout"`
}

// writeJSON writes v as JSON to a temp file and returns its path.
func writeJSON(t *testing.T, v any) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// -- Load (one-shot) --
func TestLoad_Valid(t *testing.T) {
	path := writeJSON(t, testConfig{Host: "localhost", Port: 8080})
	cfg, err := configr.Load[testConfig](path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" || cfg.Port != 8080 {
		t.Errorf("got %+v, want host=localhost port=8080", cfg)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := configr.Load[testConfig]("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*json")
	f.WriteString(`{bad json`)
	f.Close()

	_, err := configr.Load[testConfig](f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	// the JSON decoder uses DisallowUnknownFields to catch typos.
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(`{"host":"localhost:,"typo_field":1}`)
	f.Close()

	_, err := configr.Load[testConfig](f.Name())
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}
