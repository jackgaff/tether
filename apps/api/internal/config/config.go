package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	AppName             string
	AppEnv              string
	Port                string
	FrontendOrigin      string
	DatabaseURL         string
	AuthMode            string
	InternalAPIKey      string
	AWSRegion           string
	BedrockRegion       string
	NovaVoiceModelID    string
	NovaAnalysisModelID string
}

func Load() (Config, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	return LoadFrom(baseDir)
}

func LoadFrom(baseDir string) (Config, error) {
	if err := loadEnvFiles(envLoadPaths(baseDir)...); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:             getEnv("APP_NAME", "Nova Echoes"),
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("API_PORT", "8080"),
		FrontendOrigin:      getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		AuthMode:            getEnv("AUTH_MODE", "off"),
		InternalAPIKey:      os.Getenv("INTERNAL_API_KEY"),
		AWSRegion:           getEnv("AWS_REGION", "us-east-1"),
		BedrockRegion:       getEnv("BEDROCK_REGION", "us-east-1"),
		NovaVoiceModelID:    getEnv("NOVA_VOICE_MODEL_ID", "amazon.nova-2-sonic-v1:0"),
		NovaAnalysisModelID: getEnv("NOVA_ANALYSIS_MODEL_ID", "amazon.nova-2-lite-v1:0"),
	}

	switch cfg.AuthMode {
	case "off", "api-key":
	default:
		return Config{}, fmt.Errorf("AUTH_MODE must be one of off or api-key, got %q", cfg.AuthMode)
	}

	if cfg.AuthMode == "api-key" && strings.TrimSpace(cfg.InternalAPIKey) == "" {
		return Config{}, errors.New("INTERNAL_API_KEY is required when AUTH_MODE=api-key")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func envLoadPaths(baseDir string) []string {
	absoluteBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		absoluteBaseDir = filepath.Clean(baseDir)
	}

	repoRoot := findRepoRoot(absoluteBaseDir)
	paths := []string{
		filepath.Join(repoRoot, ".env"),
		filepath.Join(repoRoot, ".env.local"),
	}

	if absoluteBaseDir != repoRoot {
		paths = append(paths,
			filepath.Join(absoluteBaseDir, ".env"),
			filepath.Join(absoluteBaseDir, ".env.local"),
		)
	}

	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, exists := seen[path]; exists {
			continue
		}

		seen[path] = struct{}{}
		unique = append(unique, path)
	}

	return unique
}

func findRepoRoot(start string) string {
	current := filepath.Clean(start)

	for {
		if hasRepoMarker(current) {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return current
		}

		current = parent
	}
}

func hasRepoMarker(dir string) bool {
	for _, name := range []string{".git", "package.json", "compose.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}

	return false
}

func loadEnvFiles(paths ...string) error {
	lockedKeys := make(map[string]struct{})
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			lockedKeys[key] = struct{}{}
		}
	}

	for _, path := range paths {
		if err := loadEnvFile(path, lockedKeys); err != nil {
			return err
		}
	}

	return nil
}

func loadEnvFile(path string, lockedKeys map[string]struct{}) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("open %s: %w", path, err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" {
			return fmt.Errorf("%s:%d: environment variable name is required", path, lineNumber)
		}

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, locked := lockedKeys[key]; locked {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s:%d: %w", key, path, lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}

	return nil
}
