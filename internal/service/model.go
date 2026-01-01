package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/pelletier/go-toml/v2"

	"github.com/zjregee/alter/internal/models"
)

const (
	defaultDir        = ".alter"
	defaultConfigFile = "config.toml"
)

var defaultModelID string

const (
	DeepSeekModelProvider   = "DeepSeek"
	MoonshotModelProvider   = "Moonshot"
	OpenRouterModelProvider = "OpenRouter"
)

const (
	defaultDeepSeekModelBaseURL   = "https://api.deepseek.com"
	defaultMoonshotModelBaseURL   = "https://api.moonshot.cn/v1"
	defaultOpenRouterModelBaseURL = "https://openrouter.ai/api/v1"
)

var (
	availableModels map[string]*ModelConfig
)

type ModelConfig struct {
	Info    *models.ModelInfo
	APIKey  string
	BaseURL string
}

type fileConfig struct {
	Model modelFileConfig `toml:"model"`
}

type modelFileConfig struct {
	Default   string                         `toml:"default"`
	Providers map[string]modelProviderConfig `toml:"providers"`
}

type modelProviderConfig struct {
	APIKey  string             `toml:"api_key"`
	BaseURL string             `toml:"base_url"`
	Models  []modelEntryConfig `toml:"models"`
}

type modelEntryConfig struct {
	ID            string `toml:"id"`
	Name          string `toml:"name"`
	ContextWindow string `toml:"context_window"`
}

func init() {
	cfg, err := loadModelConfig()
	if err != nil {
		panic(err)
	}

	availableModels, err = buildAvailableModels(cfg)
	if err != nil {
		panic(err)
	}

	if cfg.Default == "" {
		panic(fmt.Sprintf("model default is empty in %s", defaultConfigFile))
	}
	if _, ok := availableModels[cfg.Default]; !ok {
		panic(fmt.Sprintf("default model not found: %s", cfg.Default))
	}
	defaultModelID = cfg.Default
}

func loadModelConfig() (*modelFileConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	alterDir := filepath.Join(homeDir, defaultDir)
	if err := os.MkdirAll(alterDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .alter directory: %w", err)
	}

	configPath := filepath.Join(alterDir, defaultConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read model config %s: %w", configPath, err)
	}

	var cfg fileConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse model config %s: %w", configPath, err)
	}

	return &cfg.Model, nil
}

func buildAvailableModels(cfg *modelFileConfig) (map[string]*ModelConfig, error) {
	if cfg == nil || len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("model providers are missing in %s", defaultConfigFile)
	}

	providerNames := map[string]string{
		"deepseek":   DeepSeekModelProvider,
		"moonshot":   MoonshotModelProvider,
		"openrouter": OpenRouterModelProvider,
	}
	providerDefaults := map[string]string{
		"deepseek":   defaultDeepSeekModelBaseURL,
		"moonshot":   defaultMoonshotModelBaseURL,
		"openrouter": defaultOpenRouterModelBaseURL,
	}

	modelsByID := make(map[string]*ModelConfig)
	for key, provider := range cfg.Providers {
		providerName, ok := providerNames[key]
		if !ok {
			return nil, fmt.Errorf("model provider %s is not supported", key)
		}
		if provider.APIKey == "" {
			return nil, fmt.Errorf("model provider %s api_key is empty in %s", key, defaultConfigFile)
		}
		if provider.BaseURL == "" {
			provider.BaseURL = providerDefaults[key]
		}
		if provider.BaseURL == "" {
			return nil, fmt.Errorf("model provider %s base_url is empty in %s", key, defaultConfigFile)
		}
		if len(provider.Models) == 0 {
			return nil, fmt.Errorf("model provider %s models are empty in %s", key, defaultConfigFile)
		}

		for _, entry := range provider.Models {
			if entry.ID == "" {
				return nil, fmt.Errorf("model provider %s has a model with empty id in %s", key, defaultConfigFile)
			}
			if entry.Name == "" {
				return nil, fmt.Errorf("model provider %s has a model with empty name: %s", key, entry.ID)
			}
			if _, exists := modelsByID[entry.ID]; exists {
				return nil, fmt.Errorf("model id %s is duplicated in %s", entry.ID, defaultConfigFile)
			}

			modelsByID[entry.ID] = &ModelConfig{
				Info: &models.ModelInfo{
					ID:            entry.ID,
					Name:          entry.Name,
					Provider:      providerName,
					ContextWindow: entry.ContextWindow,
				},
				APIKey:  provider.APIKey,
				BaseURL: provider.BaseURL,
			}
		}
	}

	return modelsByID, nil
}

func getDefaultModelInfo() *models.ModelInfo {
	if config, ok := availableModels[defaultModelID]; ok {
		return config.Info
	}

	return nil
}

func getAvailableModelInfos() []*models.ModelInfo {
	models := make([]*models.ModelInfo, 0, len(availableModels))
	for _, model := range availableModels {
		models = append(models, model.Info)
	}

	return models
}

func isModelAvailable(modelID string) bool {
	_, ok := availableModels[modelID]
	return ok
}

func getModel(ctx context.Context, modelID string) (model.ToolCallingChatModel, error) {
	config, ok := availableModels[modelID]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	switch config.Info.Provider {
	case DeepSeekModelProvider:
		return deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  config.APIKey,
			BaseURL: config.BaseURL,
			Model:   modelID,
		})
	case MoonshotModelProvider, OpenRouterModelProvider:
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  config.APIKey,
			BaseURL: config.BaseURL,
			Model:   modelID,
		})
	default:
	}

	return nil, fmt.Errorf("unsupported agent model: %s", modelID)
}
