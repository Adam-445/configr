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
