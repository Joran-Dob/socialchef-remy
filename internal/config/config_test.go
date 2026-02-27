package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTranscriptionConfig(t *testing.T) {
	// Create a temporary config file for testing
	configContent := `transcription:
  provider: openai
  fallback_enabled: false
  fallback_provider: groq`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading config from YAML
	cfg := &Config{}
	err = cfg.LoadFromYAML(configPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Verify transcription config was loaded
	if cfg.Transcription.Provider != "openai" {
		t.Errorf("Expected provider to be 'openai', got '%s'", cfg.Transcription.Provider)
	}
	if cfg.Transcription.FallbackEnabled != false {
		t.Errorf("Expected fallback_enabled to be false, got %v", cfg.Transcription.FallbackEnabled)
	}
	if cfg.Transcription.FallbackProvider != "groq" {
		t.Errorf("Expected fallback_provider to be 'groq', got '%s'", cfg.Transcription.FallbackProvider)
	}
}

func TestLoadTranscriptionConfigPartial(t *testing.T) {
	// Test with partial config (only provider specified)
	configContent := `transcription:
  provider: custom-provider`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config_partial.yaml")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := &Config{}
	cfg.SetTranscriptionDefaults() // Set defaults first
	err = cfg.LoadFromYAML(configPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Verify provider was loaded but defaults applied for other fields
	if cfg.Transcription.Provider != "custom-provider" {
		t.Errorf("Expected provider to be 'custom-provider', got '%s'", cfg.Transcription.Provider)
	}
	if cfg.Transcription.FallbackEnabled != true {
		t.Errorf("Expected fallback_enabled to be true (default), got %v", cfg.Transcription.FallbackEnabled)
	}
	if cfg.Transcription.FallbackProvider != "openai" {
		t.Errorf("Expected fallback_provider to be 'openai' (default), got '%s'", cfg.Transcription.FallbackProvider)
	}
}

func TestLoadTranscriptionConfigDefaults(t *testing.T) {
	// Test without any YAML file
	cfg := &Config{}
	cfg.SetTranscriptionDefaults()

	// Verify defaults
	if cfg.Transcription.Provider != "groq" {
		t.Errorf("Expected provider to be 'groq' (default), got '%s'", cfg.Transcription.Provider)
	}
	if cfg.Transcription.FallbackEnabled != true {
		t.Errorf("Expected fallback_enabled to be true (default), got %v", cfg.Transcription.FallbackEnabled)
	}
	if cfg.Transcription.FallbackProvider != "openai" {
		t.Errorf("Expected fallback_provider to be 'openai' (default), got '%s'", cfg.Transcription.FallbackProvider)
	}
}

func TestLoadTranscriptionConfigFileNotFound(t *testing.T) {
	// Test with non-existent file
	cfg := &Config{}
	err := cfg.LoadFromYAML("non_existent_file.yaml")

	// Should not return an error for non-existent files
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
}

func TestLoadTranscriptionConfigInvalidYAML(t *testing.T) {
	// Test with invalid YAML content
	configContent := `transcription:
  provider: openai
  invalid_yaml: [unclosed`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config_invalid.yaml")

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := &Config{}
	err = cfg.LoadFromYAML(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
