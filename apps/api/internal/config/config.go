package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"nova-echoes/api/internal/modules/voicecatalog"
)

type Config struct {
	AppName                    string
	AppEnv                     string
	Port                       string
	FrontendOrigin             string
	AllowedFrontendOrigins     []string
	DatabaseURL                string
	VoiceLabExportDir          string
	AuthMode                   string
	InternalAPIKey             string
	AWSRegion                  string
	BedrockRegion              string
	NovaVoiceModelID           string
	NovaAnalysisModelID        string
	NovaDefaultVoiceID         string
	NovaAllowedVoiceIDs        []string
	NovaInputSampleRate        int
	NovaOutputSampleRate       int
	NovaEndpointingSensitivity string
}

func Load() (Config, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}

	return LoadFrom(baseDir)
}

func LoadFrom(baseDir string) (Config, error) {
	absoluteBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		absoluteBaseDir = filepath.Clean(baseDir)
	}

	repoRoot := findRepoRoot(absoluteBaseDir)
	if err := loadEnvFiles(envLoadPaths(baseDir)...); err != nil {
		return Config{}, err
	}

	var parseErr error

	cfg := Config{
		AppName:             getEnv("APP_NAME", "Nova Echoes"),
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("API_PORT", "8080"),
		FrontendOrigin:      getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		VoiceLabExportDir:   getEnv("VOICE_LAB_EXPORT_DIR", filepath.Join(repoRoot, "apps", "api", "testdata", "voice-lab")),
		AuthMode:            getEnv("AUTH_MODE", "off"),
		InternalAPIKey:      os.Getenv("INTERNAL_API_KEY"),
		AWSRegion:           getEnv("AWS_REGION", "us-east-1"),
		BedrockRegion:       getEnv("BEDROCK_REGION", "us-east-1"),
		NovaVoiceModelID:    getEnv("NOVA_VOICE_MODEL_ID", "amazon.nova-2-sonic-v1:0"),
		NovaAnalysisModelID: getEnv("NOVA_ANALYSIS_MODEL_ID", "amazon.nova-2-lite-v1:0"),
		NovaDefaultVoiceID:  getEnv("NOVA_DEFAULT_VOICE_ID", "matthew"),
		NovaAllowedVoiceIDs: getEnvList("NOVA_ALLOWED_VOICE_IDS", voicecatalog.KnownIDs()),
	}

	cfg.AllowedFrontendOrigins = getEnvList("ALLOWED_FRONTEND_ORIGINS", []string{cfg.FrontendOrigin})
	if !slices.Contains(cfg.AllowedFrontendOrigins, cfg.FrontendOrigin) {
		cfg.AllowedFrontendOrigins = append(cfg.AllowedFrontendOrigins, cfg.FrontendOrigin)
	}
	if !filepath.IsAbs(cfg.VoiceLabExportDir) {
		cfg.VoiceLabExportDir = filepath.Join(repoRoot, cfg.VoiceLabExportDir)
	}

	cfg.NovaInputSampleRate, parseErr = getEnvInt("NOVA_INPUT_SAMPLE_RATE", 16000)
	if parseErr != nil {
		return Config{}, parseErr
	}

	cfg.NovaOutputSampleRate, parseErr = getEnvInt("NOVA_OUTPUT_SAMPLE_RATE", 24000)
	if parseErr != nil {
		return Config{}, parseErr
	}

	cfg.NovaEndpointingSensitivity = strings.ToUpper(getEnv("NOVA_ENDPOINTING_SENSITIVITY", "LOW"))

	switch cfg.AuthMode {
	case "off", "api-key":
	default:
		return Config{}, fmt.Errorf("AUTH_MODE must be one of off or api-key, got %q", cfg.AuthMode)
	}

	if cfg.AuthMode == "api-key" && strings.TrimSpace(cfg.InternalAPIKey) == "" {
		return Config{}, errors.New("INTERNAL_API_KEY is required when AUTH_MODE=api-key")
	}

	for _, sampleRate := range []int{cfg.NovaInputSampleRate, cfg.NovaOutputSampleRate} {
		if !slices.Contains([]int{8000, 16000, 24000}, sampleRate) {
			return Config{}, fmt.Errorf("audio sample rate must be one of 8000, 16000, or 24000, got %d", sampleRate)
		}
	}

	if !slices.Contains([]string{"LOW", "MEDIUM", "HIGH"}, cfg.NovaEndpointingSensitivity) {
		return Config{}, fmt.Errorf("NOVA_ENDPOINTING_SENSITIVITY must be one of LOW, MEDIUM, or HIGH, got %q", cfg.NovaEndpointingSensitivity)
	}

	if len(cfg.NovaAllowedVoiceIDs) == 0 {
		return Config{}, errors.New("NOVA_ALLOWED_VOICE_IDS must include at least one voice")
	}

	if !slices.Contains(cfg.NovaAllowedVoiceIDs, cfg.NovaDefaultVoiceID) {
		return Config{}, fmt.Errorf("NOVA_DEFAULT_VOICE_ID %q must be present in NOVA_ALLOWED_VOICE_IDS", cfg.NovaDefaultVoiceID)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer, got %q", key, value)
	}

	return parsed, nil
}

func getEnvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return append([]string(nil), fallback...)
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}

	return items
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
