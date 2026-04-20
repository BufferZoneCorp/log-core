# log-core

![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)
![License](https://img.shields.io/badge/license-MIT-green)

`log-core` is a structured logging library for Go with an API fully compatible with [sirupsen/logrus](https://github.com/sirupsen/logrus). It can be used as a drop-in replacement in projects that already import `logrus`.

Features:

- Logrus-compatible package-level API (`Info`, `Warn`, `Error`, `Debug`, `Fatal`, `WithField`, `WithFields`)
- `Logger` struct with `WithField` / `WithFields` / level-filtered methods
- `Fields` type alias — same as `logrus.Fields`
- Pluggable `Formatter` interface with built-in `TextFormatter` and `JSONFormatter`
- Zero external dependencies

## Installation

```sh
go get github.com/BufferZoneCorp/log-core
```

## Import path

```go
import logcore "github.com/BufferZoneCorp/log-core"
```

## Usage

### Package-level API (logrus drop-in)

```go
package main

import (
    logcore "github.com/BufferZoneCorp/log-core"
)

func main() {
    logcore.SetLevel(logcore.DebugLevel)

    logcore.Info("application started")
    logcore.WithField("version", "1.4.2").Info("build info")
    logcore.WithFields(logcore.Fields{
        "user_id": 42,
        "action":  "login",
    }).Warn("suspicious login attempt")

    logcore.Error("something went wrong")
}
```

### Named logger

```go
package main

import (
    "os"

    logcore "github.com/BufferZoneCorp/log-core"
)

func main() {
    log := logcore.New()
    log.Level = logcore.DebugLevel
    log.Formatter = &logcore.JSONFormatter{}
    log.Out = os.Stdout

    log.WithFields(logcore.Fields{
        "component": "db",
        "host":      "postgres:5432",
    }).Info("connected to database")

    log.WithField("query_ms", 12).Debug("query executed")
    log.Warn("connection pool nearing limit")
    log.Error("query timeout")
}
```

### JSON output

```go
log := logcore.New()
log.Formatter = &logcore.JSONFormatter{}
log.WithField("request_id", "abc-123").Info("request received")
// {"level":"info","msg":"request received","request_id":"abc-123","time":"2024-01-15T09:30:00Z"}
```

## Migrating from logrus

Replace your import:

```go
// Before
import "github.com/sirupsen/logrus"

// After
import logcore "github.com/BufferZoneCorp/log-core"
```

Then rename `logrus.` to `logcore.` (or alias the import as `logrus` for a zero-diff migration):

```go
import logrus "github.com/BufferZoneCorp/log-core"
```

All `logrus.Fields`, `logrus.WithFields`, `logrus.Info`, etc. calls continue to work unchanged.

## API reference

### Types

| Type | Description |
|---|---|
| `Logger` | Named logger with configurable level, output, and formatter |
| `Entry` | Log entry returned by `WithField` / `WithFields` |
| `Fields` | `map[string]interface{}` — structured field set |
| `Level` | Log level (`TraceLevel` … `PanicLevel`) |
| `TextFormatter` | Human-readable text output (default) |
| `JSONFormatter` | Machine-readable JSON output |

### Package-level functions

`Info`, `Warn`, `Error`, `Debug`, `Fatal`, `WithField`, `WithFields`, `SetLevel`, `SetOutput`

## Requirements

- Go 1.21 or later
- No external dependencies

## License

MIT — see [LICENSE](LICENSE).
