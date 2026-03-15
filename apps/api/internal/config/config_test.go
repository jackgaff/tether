package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nova-echoes/api/internal/config"
)

var configKeys = []string{
	"APP_NAME",
	"APP_ENV",
	"API_PORT",
	"FRONTEND_ORIGIN",
	"ALLOWED_FRONTEND_ORIGINS",
	"DATABASE_URL",
	"VOICE_LAB_EXPORT_DIR",
	"AUTH_MODE",
	"INTERNAL_API_KEY",
	"AWS_REGION",
	"BEDROCK_REGION",
	"NOVA_VOICE_MODEL_ID",
	"NOVA_ANALYSIS_MODEL_ID",
	"NOVA_DEFAULT_VOICE_ID",
	"NOVA_ALLOWED_VOICE_IDS",
	"NOVA_INPUT_SAMPLE_RATE",
	"NOVA_OUTPUT_SAMPLE_RATE",
	"NOVA_ENDPOINTING_SENSITIVITY",
	"ANALYSIS_WORKER_ENABLED",
	"ANALYSIS_WORKER_POLL_INTERVAL",
	"SCREENING_SCHEDULER_ENABLED",
	"SCREENING_SCHEDULER_POLL_INTERVAL",
	"ADMIN_USERNAME",
	"ADMIN_PASSWORD",
	"ADMIN_SESSION_SECRET",
}

func TestLoadFromEnvLocalOverridesEnv(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), strings.Join([]string{
		"APP_NAME=FromEnv",
		"FRONTEND_ORIGIN=http://env.example",
		"AUTH_MODE=off",
	}, "\n"))
	writeFile(t, filepath.Join(baseDir, ".env.local"), strings.Join([]string{
		"FRONTEND_ORIGIN=http://env-local.example",
		"API_PORT=9090",
	}, "\n"))

	cfg, err := config.LoadFrom(baseDir)
	if err != nil {
		t.Fatalf("LoadFrom returned error: %v", err)
	}

	if cfg.AppName != "FromEnv" {
		t.Fatalf("expected APP_NAME from .env, got %q", cfg.AppName)
	}

	if cfg.FrontendOrigin != "http://env-local.example" {
		t.Fatalf("expected .env.local to override FRONTEND_ORIGIN, got %q", cfg.FrontendOrigin)
	}

	if cfg.Port != "9090" {
		t.Fatalf("expected API_PORT from .env.local, got %q", cfg.Port)
	}
}

func TestLoadFromKeepsShellEnvHighestPriority(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), "APP_NAME=FromFile\nAUTH_MODE=off\n")

	if err := os.Setenv("APP_NAME", "FromShell"); err != nil {
		t.Fatalf("Setenv(APP_NAME): %v", err)
	}

	cfg, err := config.LoadFrom(baseDir)
	if err != nil {
		t.Fatalf("LoadFrom returned error: %v", err)
	}

	if cfg.AppName != "FromShell" {
		t.Fatalf("expected shell APP_NAME to win, got %q", cfg.AppName)
	}
}

func TestLoadFromRejectsMalformedEnvFiles(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), "AUTH_MODE=off\nthis is not valid\n")

	_, err := config.LoadFrom(baseDir)
	if err == nil {
		t.Fatal("expected malformed env file to return an error")
	}

	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestLoadFromUsesRepoRootAndServiceOverrides(t *testing.T) {
	resetManagedEnv(t)

	repoRoot := t.TempDir()
	serviceDir := filepath.Join(repoRoot, "apps", "api")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	writeFile(t, filepath.Join(repoRoot, "package.json"), "{}\n")
	writeFile(t, filepath.Join(repoRoot, ".env"), "APP_NAME=RootEnv\nAUTH_MODE=off\nAPI_PORT=8080\n")
	writeFile(t, filepath.Join(serviceDir, ".env.local"), "API_PORT=8181\n")

	cfg, err := config.LoadFrom(serviceDir)
	if err != nil {
		t.Fatalf("LoadFrom returned error: %v", err)
	}

	if cfg.AppName != "RootEnv" {
		t.Fatalf("expected root .env to be loaded, got %q", cfg.AppName)
	}

	if cfg.Port != "8181" {
		t.Fatalf("expected service override from apps/api/.env.local, got %q", cfg.Port)
	}

	expectedExportDir := filepath.Join(repoRoot, "apps", "api", "testdata", "voice-lab")
	if cfg.VoiceLabExportDir != expectedExportDir {
		t.Fatalf("expected default export dir %q, got %q", expectedExportDir, cfg.VoiceLabExportDir)
	}
}

func TestLoadFromIncludesFrontendOriginInAllowedOrigins(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), strings.Join([]string{
		"AUTH_MODE=off",
		"FRONTEND_ORIGIN=http://localhost:5173",
		"ALLOWED_FRONTEND_ORIGINS=http://localhost:5174",
	}, "\n"))

	cfg, err := config.LoadFrom(baseDir)
	if err != nil {
		t.Fatalf("LoadFrom returned error: %v", err)
	}

	if !contains(cfg.AllowedFrontendOrigins, "http://localhost:5173") {
		t.Fatalf("expected FrontendOrigin to be included in allowed origins, got %v", cfg.AllowedFrontendOrigins)
	}

	if !contains(cfg.AllowedFrontendOrigins, "http://localhost:5174") {
		t.Fatalf("expected configured allowed origins to be included, got %v", cfg.AllowedFrontendOrigins)
	}
}

func TestLoadFromRequiresAPIKeyWhenEnabled(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), "AUTH_MODE=api-key\n")

	_, err := config.LoadFrom(baseDir)
	if err == nil {
		t.Fatal("expected missing INTERNAL_API_KEY to return an error")
	}

	if !strings.Contains(err.Error(), "INTERNAL_API_KEY is required") {
		t.Fatalf("expected missing api key error, got %v", err)
	}
}

func TestLoadFromRejectsProductionDemoAdminDefaults(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), strings.Join([]string{
		"APP_ENV=production",
		"AUTH_MODE=off",
		"ADMIN_USERNAME=demo-admin",
		"ADMIN_PASSWORD=demo-admin-password",
		"ADMIN_SESSION_SECRET=demo-admin-session-secret-change-me",
	}, "\n"))

	_, err := config.LoadFrom(baseDir)
	if err == nil {
		t.Fatal("expected production demo admin defaults to be rejected")
	}

	if !strings.Contains(err.Error(), "must not use demo defaults") {
		t.Fatalf("expected demo defaults error, got %v", err)
	}
}

func TestLoadFromRejectsWildcardOriginsInProduction(t *testing.T) {
	resetManagedEnv(t)

	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, ".env"), strings.Join([]string{
		"APP_ENV=production",
		"AUTH_MODE=off",
		"ADMIN_USERNAME=prod-admin",
		"ADMIN_PASSWORD=not-the-demo-password",
		"ADMIN_SESSION_SECRET=production-secret-value",
		"ALLOWED_FRONTEND_ORIGINS=*",
	}, "\n"))

	_, err := config.LoadFrom(baseDir)
	if err == nil {
		t.Fatal("expected wildcard origin to be rejected in production")
	}

	if !strings.Contains(err.Error(), "cannot contain * in production") {
		t.Fatalf("expected wildcard origin error, got %v", err)
	}
}

func resetManagedEnv(t *testing.T) {
	t.Helper()

	original := make(map[string]*string, len(configKeys))
	for _, key := range configKeys {
		if value, ok := os.LookupEnv(key); ok {
			value := value
			original[key] = &value
		} else {
			original[key] = nil
		}

		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%s): %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, key := range configKeys {
			value := original[key]
			var err error
			if value == nil {
				err = os.Unsetenv(key)
			} else {
				err = os.Setenv(key, *value)
			}

			if err != nil {
				t.Fatalf("restore %s: %v", key, err)
			}
		}
	})
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
