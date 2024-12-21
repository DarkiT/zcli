# Service Manager

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/zcli.svg)](https://pkg.go.dev/github.com/darkit/zcli)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/zcli)](https://goreportcard.com/report/github.com/darkit/zcli)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/zcli/blob/master/LICENSE)

A flexible service management library for Go applications.

## Features

- Full service lifecycle management (install, uninstall, start, stop, restart, status)
- Multi-language support (English/Chinese built-in)
- Rich command-line parameter system with validation
- Runtime configuration management
- Detailed build information support
- Colorful console output with customizable themes
- Cross-platform service support (Windows/Linux/macOS)
- Custom command support
- Parameter validation and enumeration support
- Concurrent parameter processing
- Debug mode support

## Installation

```bash
go get github.com/darkit/zcli
```

## Requirements

- Go 1.21.5 or higher
- Windows, Linux, or macOS

## Dependencies

- github.com/kardianos/service v1.2.2
- github.com/fatih/color v1.18.0

## Quick Start

```go
package main

import (
    "fmt"
    "log/slog"
    "time"
    "github.com/darkit/zcli"
)

func main() {
    // Create build info
    buildInfo := zcli.NewBuildInfo().
        SetDebug(true).
        SetVersion("1.0.0").
        SetBuildTime(time.Now())

    // Create service
    svc, err := zcli.New(&zcli.Options{
        Name:        "myapp",
        DisplayName: "My Application",
        Description: "This is my application service",
        Version:     "1.0.0",
        Language:    "en",
        BuildInfo:   buildInfo,
        Run: func() error {
            slog.Info("Service is running...")
            return nil
        },
        Stop: func() error {
            slog.Info("Service is stopping...")
            return nil
        },
    })
    if err != nil {
        slog.Error("Failed to create service", "error", err)
        return
    }

    // Add parameters
    pm := svc.ParamManager()
    
    // Add config parameter
    pm.AddParam(&zcli.Parameter{
        Name:        "config",
        Short:       "c",
        Long:        "config", 
        Description: "Config file path",
        Required:    true,
        Type:        "string",
    })

    // Add port parameter with validation
    pm.AddParam(&zcli.Parameter{
        Name:        "port",
        Short:       "p",
        Long:        "port",
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

    // Add mode parameter with enum values
    pm.AddParam(&zcli.Parameter{
        Name:        "mode",
        Short:       "m",
        Long:        "mode",
        Description: "Running mode",
        Default:     "prod",
        EnumValues:  []string{"dev", "test", "prod"},
        Type:        "string",
    })

    // Run service
    if err := svc.Run(); err != nil {
        slog.Error("Service failed", "error", err)
    }
}
```

## Command Line Usage

```bash
# Show help information
./myapp -h

# Show version information
./myapp -v

# Install and start service
./myapp install

# Start service
./myapp start

# Stop service
./myapp stop

# Restart service
./myapp restart

# Show service status
./myapp status

# Uninstall service
./myapp uninstall

# Run with custom parameters
./myapp --port 9090 --mode dev --config config.toml

# Use Chinese language
./myapp --lang zh
```

## Parameter System

The library provides a comprehensive parameter management system:

```go
// Add required parameter
pm.AddParam(&zcli.Parameter{
    Name:        "config",
    Short:       "c",
    Long:        "config",
    Description: "Config file path",
    Required:    true,
    Type:        "string",
})

// Add parameter with validation
pm.AddParam(&zcli.Parameter{
    Name:        "workers",
    Short:       "w",
    Long:        "workers",
    Description: "Number of workers",
    Default:     "5",
    Type:        "string",
    Validate: func(val string) error {
        if val == "0" {
            return fmt.Errorf("workers cannot be 0")
        }
        return nil
    },
})

// Add enum parameter
pm.AddParam(&zcli.Parameter{
    Name:        "mode",
    Short:       "m",
    Long:        "mode",
    Description: "Running mode",
    Default:     "prod",
    EnumValues:  []string{"dev", "test", "prod"},
    Type:        "string",
})

// Get parameter values
port := pm.GetString("port")
workers := pm.GetInt("workers")
isDebug := pm.GetBool("debug")
```

## Build Information

Support for detailed build information with build script:

```bash
#!/bin/bash
VERSION="1.0.0"
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
GIT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

go build \
    -ldflags "-X main.version=${VERSION} \
              -X buildInfo.buildTime=${BUILD_TIME} \
              -X buildInfo.gitCommit=${GIT_COMMIT} \
              -X buildInfo.gitBranch=${GIT_BRANCH} \
              -X buildInfo.gitTag=${GIT_TAG}" \
    -o myapp
```

## Custom Commands

Support for adding custom commands:

```go
pm.AddCommand("custom", "Custom command description", func() {
    // Custom command logic
    return nil
}, false)
```

## Configuration Management

Runtime configuration management:

```go
// Set configuration value
svc.SetConfigValue("lastStartTime", time.Now().Unix())

// Get configuration value
value, exists := svc.GetConfigValue("lastStartTime")

// Delete configuration value
svc.DeleteConfigValue("lastStartTime")

// Check if configuration exists
exists := svc.HasConfigValue("lastStartTime")

// Get all configuration keys
keys := svc.GetConfigKeys()
```

## Debug Mode

Support for debug mode:

```go
// Enable debug mode
svc.EnableDebug()

// Disable debug mode
svc.DisableDebug()

// Check debug status
isDebug := svc.IsDebug()
```

## Examples

See the `examples` directory for complete working examples.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.