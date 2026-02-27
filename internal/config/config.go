package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Env            string
	ServiceName    string
	ServiceVersion string

	DatabaseURL string

	SupabaseURL            string
	SupabaseJWTSecret      string
	SupabaseServiceRoleKey string

	RedisURL string

	OpenAIKey string
	GroqKey   string

	ApifyAPIKey    string
	ProxyServerURL string
	ProxyAPIKey    string

	OtelExporterOTLPEndpoint string
	OtelExporterOTLPHeaders  string
	SentryDSN                string

	Port string

	Transcription TranscriptionConfig
}

type TranscriptionConfig struct {
	Provider         string `yaml:"provider"`
	FallbackEnabled  bool   `yaml:"fallback_enabled"`
	FallbackProvider string `yaml:"fallback_provider"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:                      os.Getenv("ENV"),
		ServiceName:              os.Getenv("SERVICE_NAME"),
		ServiceVersion:           os.Getenv("SERVICE_VERSION"),
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		SupabaseURL:              os.Getenv("SUPABASE_URL"),
		SupabaseJWTSecret:        os.Getenv("SUPABASE_JWT_SECRET"),
		SupabaseServiceRoleKey:   os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		RedisURL:                 os.Getenv("REDIS_URL"),
		OpenAIKey:                os.Getenv("OPENAI_API_KEY"),
		GroqKey:                  os.Getenv("GROQ_API_KEY"),
		ApifyAPIKey:              os.Getenv("APIFY_API_KEY"),
		ProxyServerURL:           os.Getenv("PROXY_SERVER_URL"),
		ProxyAPIKey:              os.Getenv("PROXY_API_KEY"),
		OtelExporterOTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OtelExporterOTLPHeaders:  os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"),
		SentryDSN:                os.Getenv("SENTRY_DSN"),
		Port:                     os.Getenv("PORT"),
	}

	// Load from YAML file if available
	if err := cfg.LoadFromYAML("config.yaml"); err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	// Set defaults
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "socialchef-remy"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Set transcription defaults
	cfg.SetTranscriptionDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func (c *Config) LoadFromYAML(path string) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File not found is not an error
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlConfig struct {
		Transcription TranscriptionConfig `yaml:"transcription"`
	}

	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply transcription config with defaults
	if yamlConfig.Transcription.Provider != "" {
		c.Transcription.Provider = yamlConfig.Transcription.Provider
	}
	if yamlConfig.Transcription.FallbackEnabled {
		c.Transcription.FallbackEnabled = yamlConfig.Transcription.FallbackEnabled
	}
	if yamlConfig.Transcription.FallbackProvider != "" {
		c.Transcription.FallbackProvider = yamlConfig.Transcription.FallbackProvider
	}

	return nil
}

func (c *Config) SetTranscriptionDefaults() {
	if c.Transcription.Provider == "" {
		c.Transcription.Provider = "groq"
	}
	if !c.Transcription.FallbackEnabled {
		c.Transcription.FallbackEnabled = true
	}
	if c.Transcription.FallbackProvider == "" {
		c.Transcription.FallbackProvider = "openai"
	}
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.SupabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL is required")
	}
	if c.SupabaseJWTSecret == "" {
		return fmt.Errorf("SUPABASE_JWT_SECRET is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	return nil
}
