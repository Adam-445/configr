# configr

A simple, format-agnostic, hot-reloading configuration loader for Go.

```go
// One-shot
cfg, err := configr.Load[MyConfig]("config.json")

// Live reload (Get() always returns the latest valid config)
loader, err := configr.New[MyConfig]("config.json",
    configr.WithOnChange(func(c MyConfig) {
        server.Reload(c)
    }),
)
cfg := loader.Get()
```

## Why

Most config libraries either lock you into a single format, require you to
inject a global, or make hot-reloading an afterthought. configr is built around
three constraints :

- **One interface to add a format**: Implement `Decoder` (one method) and any
  format works: JSON, YAML, TOML, or your own binary encoding!
- **Lock-free reads**: `Get()` reads from an `atomic.Pointer[T]` so it never
  blocks, never returns a partial write, and is safe in any goroutine.
- **Failed reloads keep previous config**: If the file changes but the new
  version fails validation, the last valid config stays in effect. Your server
  keeps running and you only have to fix the file.

## Installation

```bash
go get github.com/Adam-445/configr
```

For YAML support (optional, separate module):

```bash
go get github.com/Adam-445/configr/yaml
go get gopkg.in/yaml.v3
```
## Usage
 
### One-shot load
 
```go
type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}
 
cfg, err := configr.Load[Config]("config.json")
if err != nil {
    log.Fatal(err)
}
fmt.Println(cfg.Port) // 8080
```
 
### Hot-reload
 
```go
loader, err := configr.New[Config]("config.json",
    configr.WithPollInterval[Config](2*time.Second),
    configr.WithDefaults(func(c *Config) {
        if c.Port == 0 { c.Port = 8080 }
    }),
    configr.WithValidate(func(c Config) error {
        if c.Host == "" { return errors.New("host required") }
        return nil
    }),
    configr.WithOnChange(func(c Config) {
        log.Printf("reloaded: port=%d", c.Port)
    }),
)
if err != nil {
    log.Fatal(err)
}
defer loader.Stop()
 
// Anywhere in your code (safe from any goroutine and doesnt block):
cfg := loader.Get()
```
 
### YAML
 
```go
import (
    "github.com/Adam-445/configr"
    configryaml "github.com/Adam-445/configr/yaml"
)
 
loader, err := configr.New[Config]("config.yaml",
    configr.WithDecoder[Config](configryaml.YAML),
)
```

 
### Custom decoder
 
Implement the `Decoder` interface to support any format:
 
```go
type Decoder interface {
    Decode(r io.Reader, v any) error
}
 
type myTOMLDecoder struct{}
 
func (myTOMLDecoder) Decode(r io.Reader, v any) error {
    return toml.NewDecoder(r).Decode(v)
}
 
loader, err := configr.New[Config]("config.toml",
    configr.WithDecoder[Config](myTOMLDecoder{}),
)
```
 
## Options
 
| Option | Default | Description |
|---|---|---|
| `WithDecoder` | JSON | Format decoder |
| `WithPollInterval` | 2s | How often to check the file for changes |
| `WithDefaults` | — | Fill in zero-value fields before validation |
| `WithValidate` | — | Reject invalid configs, and keep the previous on failure |
| `WithOnChange` | — | Callback fired (in a goroutine) after each successful reload |

## Project structure
 
```
configr/
├── configr.go          package doc
├── decoder.go          Decoder interface
├── json.go             JSON decoder (stdlib, default)
├── loader.go           Loader[T], Load(), New(), Get(), Watch(), Stop()
├── options.go          Option funcs (WithDecoder, WithOnChange, ...)
├── watcher.go          Polling file watcher
├── loader_test.go      test suite
├── yaml/
    └── yaml.go         YAML decoder (optional, gopkg.in/yaml.v3)
```

## Migrating from a hand-rolled config loader
 
Before (typical pattern):
 
```go
f, _ := os.Open("config.json")
cfg := &Config{}
json.NewDecoder(f).Decode(cfg)
```
 
After:
 
```go
cfg, err := configr.Load[Config]("config.json",
    configr.WithDefaults(...),
    configr.WithValidate(...),
)
```
