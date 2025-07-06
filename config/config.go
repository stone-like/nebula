package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/sashabaranov/go-openai"
)

// Config represents the nebula configuration
type Config struct {
	Model        string `json:"model"`
	DatabasePath string `json:"database_path"`
	MaxSessions  int    `json:"max_sessions"`
	APIKey       string `json:"-"` // APIキーは設定ファイルに保存しない
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(homeDir, ".nebula", "memory.db")
	
	return &Config{
		Model:        "gpt-4.1-nano", // デフォルトはgpt-4.1-nano
		DatabasePath: defaultDBPath,
		MaxSessions:  100,
	}
}

// LoadConfig loads configuration from file or creates default
func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	// 設定ファイルが存在しない場合はデフォルト設定を作成
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := SaveConfig(config); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	// 設定ファイルを読み込み
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// APIキーは環境変数から取得
	config.APIKey = os.Getenv("OPENAI_API_KEY")

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := getConfigPath()

	// 設定ディレクトリを作成
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// JSONとして保存
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigPath returns the path to the configuration file
func getConfigPath() string {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// フォールバック: カレントディレクトリに.nebulaフォルダを作成
		return ".nebula/config.json"
	}
	return filepath.Join(homeDir, ".nebula", "config.json")
}

// GetOpenAIModel returns the appropriate OpenAI model identifier
func (c *Config) GetOpenAIModel() string {
	switch c.Model {
	case "gpt-4.1-nano":
		return openai.GPT4Dot1Nano // OpenAIライブラリでの実際の識別子
	case "gpt-4.1-mini":
		return openai.GPT4Dot1Mini // 現在は同じモデルを使用（将来的に変更可能）
	default:
		return openai.GPT4Dot1Nano // デフォルト
	}
}

// SetModel updates the model in configuration
func (c *Config) SetModel(model string) error {
	validModels := []string{"gpt-4.1-nano", "gpt-4.1-mini"}

	if slices.Contains(validModels, model) {
		c.Model = model
		return SaveConfig(c)
	}

	return fmt.Errorf("invalid model: %s. Valid models: %v", model, validModels)
}
