# Service Manager

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

A flexible service management library for Go applications.

## Features

- Full service lifecycle management (install, uninstall, start, stop, restart)
- Multi-language support (English/Chinese built-in)
- Rich command-line parameter system
- Runtime configuration management
- Detailed build information support
- Colorful console output
- Windows service support

## Installation

```bash
go get github.com/darkit/zcli
```

## Requirements

- Go 1.16 or higher
- Windows, Linux, or macOS

## Dependencies

- github.com/kardianos/service
- github.com/fatih/color

## Quick Start

```go
package main

import (
    "log"
    "github.com/darkit/zcli"
)

func main() {
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "My Application",
        Description: "This is my application service",
        Version:     "1.0.0",
    })
    if err != nil {
        log.Fatal(err)
    }

    if err := svc.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Command Line Usage

```bash
# Show help
./myapp -h

# Install service
./myapp install

# Start service
./myapp start

# Stop service
./myapp stop

# Show status
./myapp status

# Uninstall service
./myapp uninstall

# Use Chinese language
./myapp install --lang zh

# Run with custom parameters
./myapp --port 9090 --mode dev
```

## Parameter System

The library provides a flexible parameter management system:

```go
// Add required parameter
svc.ParamManager().AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "Config file path",
    Required:    true,
})

// Add enum parameter
svc.ParamManager().AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Long:        "mode",
    Description: "Running mode",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
})
```

## Multi-language Support

Built-in support for English and Chinese, easily add more languages:

```go
// Add new language
svc.AddLanguage("fr", zcli.Messages{
    Install:   "Installer le service",
    Uninstall: "Désinstaller le service",
    Start:     "Démarrer le service",
    Stop:      "Arrêter le service",
    // ...
})

// Switch language
svc.SetLanguage("fr")
```

## Build Information

Support for detailed build information:

```go
svc, err := zcli.New(&zcli.Options{
    Name:    "myapp",
    Version: "1.0.0",
    BuildInfo: zcli.NewBuildInfo().
        SetVersion("1.0.0").
        SetBuildTime(time.Now()).
        SetDebug(true),
})
```

## Runtime Configuration

All parameters and settings are managed in memory:

```go
// Set custom configuration value
svc.SetConfigValue("custom_key", "custom_value")

// Get configuration value
value, exists := svc.GetConfigValue("custom_key")

// Delete configuration value
svc.DeleteConfigValue("custom_key")
```

## Service Events

Support for service lifecycle events:

```go
svc, err := zcli.New(&zcli.Options{
    Name: "myapp",
    Run: func() error {
        // Service run logic
        return nil
    },
    Stop: func() error {
        // Service stop logic
        return nil
    },
})
```

## Advanced Usage

See `examples` directory for more advanced usage examples.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.