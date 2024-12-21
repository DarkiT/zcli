# Service Manager

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

A flexible service management library for Go applications with comprehensive lifecycle management capabilities.

## Core Features

### Service Management
- Complete service lifecycle management (install, uninstall, start, stop, restart)
- Cross-platform support (Windows/Linux/macOS)
- Service status monitoring and reporting
- Graceful shutdown handling

### Parameter System
- Rich command-line parameter management
- Parameter validation and enumeration support
- Concurrent parameter processing
- Short and long parameter formats
- Required parameter enforcement
- Default value support
- Custom validation rules

### Internationalization
- Built-in multi-language support
- Easy language switching
- Customizable message templates

### Development Tools
- Debug mode with enhanced logging
- Custom command support
- Colorful console output with themes
- Detailed build information
- Comprehensive error handling

## Installation

```bash
go get github.com/darkit/zcli
```

## Quick Start

Here's a minimal example to get you started:

```go
package main

import (
    "log/slog"
    "github.com/darkit/zcli"
)

func main() {
    // Create service
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "My Application",
        Description: "Example service",
        Version:     "1.0.0",
        Run: func() error {
            slog.Info("Service is running...")
            return nil
        },
    })
    if err != nil {
        slog.Error("Failed to create service", "error", err)
        return
    }

    // Run service
    if err := svc.Run(); err != nil {
        slog.Error("Service failed", "error", err)
    }
}
```

## Advanced Usage

### Parameter Configuration

```go
pm := svc.ParamManager()

// Add required parameter
pm.AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "Config file path",
    Required:    true,
    Type:        "string",
})

// Add validated parameter
pm.AddParam(&zcli.Parameter{
    Name:        "port",
    Short:       "p",
    Description: "Service port",
    Default:     "8080",
    Type:        "string",
    Validate: func(val string) error {
        if val == "0" {
            return fmt.Errorf("port cannot be 0")
        }
        return nil
    },
})

// Add enum parameter
pm.AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Description: "Running mode",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
    Type:        "string",
})
```

### Custom Commands

```go
// Add version command
pm.AddCommand("version", "Show version info", func() {
    fmt.Printf("Version: %s\n", svc.GetVersion())
}, true)

// Add check command
pm.AddCommand("check", "Check service status", func() {
    // Add check logic
}, false)
```

### Debug Mode

```go
// Enable debug mode
svc.EnableDebug()

// Use debug logging
if svc.IsDebug() {
    slog.Debug("Debug information...")
}

// Disable debug mode
svc.DisableDebug()
```

## Command Line Interface

```bash
# Basic commands
./myapp install    # Install service
./myapp start     # Start service
./myapp stop      # Stop service
./myapp restart   # Restart service
./myapp status    # Show status
./myapp uninstall # Uninstall service

# Run with parameters or install service with parameters
./myapp --port 9090 --mode dev
./myapp install --port 9090 --mode dev

# Custom commands
./myapp version
./myapp check

# Help and version
./myapp -h
./myapp -v
```

## Complete Example

See [examples/main.go](examples/main.go) for a complete working example.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.