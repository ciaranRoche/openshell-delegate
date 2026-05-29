package creds

import (
	"fmt"
	"os"
	"strings"

	"github.com/ciaranRoche/openshell-delegate/internal/config"
)

// Validate checks that all required environment variables and files are present.
func Validate(cfg *config.Config) []string {
	var errors []string

	for _, envVar := range cfg.Env {
		if os.Getenv(envVar) == "" {
			errors = append(errors, fmt.Sprintf("environment variable %s is not set", envVar))
		}
	}

	for _, upload := range cfg.Uploads {
		src := config.ExpandPath(upload.Src)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("upload source not found: %s", upload.Src))
		}
	}

	return errors
}

// BuildLaunchScript generates the shell script that injects credentials into
// the sandbox's .bashrc and keeps the sandbox alive.
func BuildLaunchScript(cfg *config.Config) string {
	var envLines []string

	// Static env vars (fixed values from config)
	for k, v := range cfg.StaticEnv {
		envLines = append(envLines, fmt.Sprintf("export %s=%q", k, v))
	}

	// Forwarded env vars (read from host environment)
	for _, envVar := range cfg.Env {
		val := os.Getenv(envVar)
		if val != "" {
			envLines = append(envLines, fmt.Sprintf("export %s=%q", envVar, val))
		}
	}

	// Go config
	envLines = append(envLines,
		`export PATH="/usr/local/go/bin:/sandbox/go/bin:$PATH"`,
		`export GOPATH=/sandbox/go`,
		`export GOBIN=/sandbox/go/bin`,
	)

	// Build the script
	envBlock := strings.Join(envLines, "\n")

	script := fmt.Sprintf(`#!/bin/sh
cat >> /sandbox/.bashrc << 'ENVEOF'

# Injected by openshell-delegate
%s
ENVEOF
exec sleep infinity`, envBlock)

	return script
}

// UploadMap returns a map of source -> destination for files that should be
// uploaded during sandbox creation (via --upload flags).
func UploadMap(cfg *config.Config) map[string]string {
	uploads := make(map[string]string)
	for _, u := range cfg.Uploads {
		src := config.ExpandPath(u.Src)
		uploads[src] = u.Dst
	}
	return uploads
}
