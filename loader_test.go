package configr_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// --- Load ---

func TestLoad(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) string // returns path to config file
		wantErr  bool
		wantHost string // only if no error expected
		wantPort int
	}

	cases := []testCase{
		{
			name: "valid config",
			setup: func(t *testing.T) string {
				return writeJSON(t, testConfig{Host: "localhost", Port: 8080})
			},
			wantErr:  false,
			wantHost: "localhost",
			wantPort: 8080,
		},
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/config.json"
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				f, err := os.CreateTemp(t.TempDir(), "*.json")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.WriteString(`{bad json`); err != nil {
					t.Fatal(err)
				}
				f.Close()
				return f.Name()
			},
			wantErr: true,
		},
		{
			name: "unknown field (typo)",
			setup: func(t *testing.T) string {
				f, err := os.CreateTemp(t.TempDir(), "*.json")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.WriteString(`{"host":"localhost","typo_field":1}`); err != nil {
					t.Fatal(err)
				}
				f.Close()
				return f.Name()
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			cfg, err := configr.Load[testConfig](path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v, wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if cfg.Host != tc.wantHost || cfg.Port != tc.wantPort {
					t.Errorf("got %+v, want host=%s port=%d", cfg, tc.wantHost, tc.wantPort)
				}
			}
		})
	}
}

// --- WithDefaults ---

func TestLoad_Defaults(t *testing.T) {
	path := writeJSON(t, testConfig{Host: "localhost", Port: 8080})
	cfg, err := configr.Load[testConfig](path,
		configr.WithDefaults(func(c *testConfig) {
			if c.Timeout == 0 {
				c.Timeout = 30
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 30 {
		t.Errorf("expected default timeout=30, got %d", cfg.Timeout)
	}
}

// --- WithValidate ---

func TestLoad_Validation(t *testing.T) {
	type testCase struct {
		name       string
		config     testConfig
		wantErr    bool
		wantErrMsg string // optional substring
	}

	cases := []testCase{
		{
			name:    "validation passes",
			config:  testConfig{Host: "localhost", Port: 8080},
			wantErr: false,
		},
		{
			name:       "validation fails (negative port)",
			config:     testConfig{Host: "localhost", Port: -1},
			wantErr:    true,
			wantErrMsg: "port must be positive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeJSON(t, tc.config)
			_, err := configr.Load[testConfig](path,
				configr.WithValidate(func(c testConfig) error {
					if c.Port <= 0 {
						return errors.New("port must be positive")
					}
					return nil
				}),
			)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("expected error containing %q, got %v", tc.wantErrMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// --- Custom Decoder ---

// brokenDecoder always returns an error. It's used to test decoder error paths.
type brokenDecoder struct{}

func (brokenDecoder) Decode(_ io.Reader, _ any) error {
	return errors.New("broken decoder")
}

func TestLoad_CustomDecoder(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(`{}`)
	f.Close()

	_, err := configr.Load[testConfig](f.Name(),
		configr.WithDecoder[testConfig](brokenDecoder{}),
	)
	if err == nil {
		t.Fatal("expected error from broken decoder, got nil")
	}
}

// --- Get (atomic read) ---

func TestGet_ReturnsCopy(t *testing.T) {
	path := writeJSON(t, testConfig{Host: "original"})
	loader, err := configr.New[testConfig](path)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Stop()

	cfg := loader.Get()
	cfg.Host = "mutated" // mutate the local copy

	// The loader must still hold the original value.
	if loader.Get().Host != "original" {
		t.Error("Get() should return a copy. Mutating it must not affect the loader")
	}
}

// --- Hot reload and Watch ---

func TestNew_HotReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write initial config.
	initial := testConfig{Host: "v1", Port: 1}
	data, _ := json.Marshal(initial)
	os.WriteFile(path, data, 0o644)

	var callbackCount atomic.Int32
	var lastHost atomic.Value
	lastHost.Store("")

	loader, err := configr.New[testConfig](path,
		configr.WithPollInterval[testConfig](50*time.Millisecond),
		configr.WithOnChange(func(c testConfig) {
			callbackCount.Add(1)
			lastHost.Store(c.Host)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Stop()

	// Sanity: initial read
	if loader.Get().Host != "v1" {
		t.Fatalf("expected initial host=v1, got %q", loader.Get().Host)
	}

	// Wait a tick so the watcher captures the current mtime before we write.
	time.Sleep(100 * time.Millisecond)

	// Update the file.
	updated := testConfig{Host: "v2", Port: 2}
	data, _ = json.Marshal(updated)
	os.WriteFile(path, data, 0o644)

	// Allow up to 500ms for the reload to propagate.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if loader.Get().Host == "v2" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if loader.Get().Host != "v2" {
		t.Errorf("expected hot-reloaded host=v2, got %q", loader.Get().Host)
	}
	if callbackCount.Load() == 0 {
		t.Error("onChange callback was never called")
	}
}
