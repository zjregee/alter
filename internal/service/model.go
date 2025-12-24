package service

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"

	"github.com/zjregee/alter/internal/models"
)

const (
	DeepSeekChatModelID        = "deepseek-chat"
	DeepSeekReasonerModelID    = "deepseek-reasoner"
	DoubaoSeed18251215ModelID  = "doubao-seed-1-8-251215"
	KimiK2TurboModelID         = "kimi-k2-turbo-preview"
	KimiK2ThinkingTurboModelID = "kimi-k2-thinking-turbo"
	XGrok41FastModelID         = "x-ai/grok-4.1-fast"
	Qwen3CoderModelID          = "qwen/qwen3-coder:free"
	XiaoMiMimoV2FlashModelID   = "xiaomi/mimo-v2-flash:free"
)

const defaultModelID = DeepSeekChatModelID

const (
	DeepSeekModelProvider   = "DeepSeek"
	ByteDanceModelProvider  = "ByteDance"
	MoonshotModelProvider   = "Moonshot"
	OpenRouterModelProvider = "OpenRouter"
)

const (
	DeepSeekModelBaseURL   = "https://api.deepseek.com"
	ByteDanceModelBaseURL  = "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	MoonshotModelBaseURL   = "https://api.moonshot.cn"
	OpenRouterModelBaseURL = "https://openrouter.ai/api/v1"
)

var (
	DeepSeekModelAPIKey   string
	ByteDanceModelAPIKey  string
	MoonshotModelAPIKey   string
	OpenRouterModelAPIKey string
)

func init() {
	DeepSeekModelAPIKey = os.Getenv("DEEPSEEK_API_KEY")
	ByteDanceModelAPIKey = os.Getenv("BYTE_DANCE_API_KEY")
	MoonshotModelAPIKey = os.Getenv("MOONSHOT_API_KEY")
	OpenRouterModelAPIKey = os.Getenv("OPENROUTER_API_KEY")

	if DeepSeekModelAPIKey == "" {
		panic("DEEPSEEK_API_KEY is not set")
	}
	if ByteDanceModelAPIKey == "" {
		panic("BYTE_DANCE_API_KEY is not set")
	}
	if MoonshotModelAPIKey == "" {
		panic("MOONSHOT_API_KEY is not set")
	}
	if OpenRouterModelAPIKey == "" {
		panic("OPENROUTER_API_KEY is not set")
	}
}

type ModelConfig struct {
	Info    *models.ModelInfo
	APIKey  string
	BaseURL string
}

var availableModels = map[string]*ModelConfig{
	DeepSeekChatModelID: {
		Info: &models.ModelInfo{
			ID:            DeepSeekChatModelID,
			Name:          "deepseek-chat",
			Provider:      DeepSeekModelProvider,
			ContextWindow: "128k",
		},
		APIKey:  DeepSeekModelAPIKey,
		BaseURL: DeepSeekModelBaseURL,
	},
	DeepSeekReasonerModelID: {
		Info: &models.ModelInfo{
			ID:            DeepSeekReasonerModelID,
			Name:          "deepseek-reasoner",
			Provider:      DeepSeekModelProvider,
			ContextWindow: "128k",
		},
		APIKey:  DeepSeekModelAPIKey,
		BaseURL: DeepSeekModelBaseURL,
	},
	DoubaoSeed18251215ModelID: {
		Info: &models.ModelInfo{
			ID:            DoubaoSeed18251215ModelID,
			Name:          "doubao-seed-1.8",
			Provider:      ByteDanceModelProvider,
			ContextWindow: "256k",
		},
		APIKey:  ByteDanceModelAPIKey,
		BaseURL: ByteDanceModelBaseURL,
	},
	KimiK2TurboModelID: {
		Info: &models.ModelInfo{
			ID:            KimiK2TurboModelID,
			Name:          "kimi-k2",
			Provider:      MoonshotModelProvider,
			ContextWindow: "256k",
		},
		APIKey:  MoonshotModelAPIKey,
		BaseURL: MoonshotModelBaseURL,
	},
	KimiK2ThinkingTurboModelID: {
		Info: &models.ModelInfo{
			ID:            KimiK2ThinkingTurboModelID,
			Name:          "kimi-k2-thinking",
			Provider:      MoonshotModelProvider,
			ContextWindow: "256k",
		},
		APIKey:  MoonshotModelAPIKey,
		BaseURL: MoonshotModelBaseURL,
	},
	XGrok41FastModelID: {
		Info: &models.ModelInfo{
			ID:            XGrok41FastModelID,
			Name:          "grok-4.1-fast",
			Provider:      OpenRouterModelProvider,
			ContextWindow: "2M",
		},
		APIKey:  OpenRouterModelAPIKey,
		BaseURL: OpenRouterModelBaseURL,
	},
	Qwen3CoderModelID: {
		Info: &models.ModelInfo{
			ID:            Qwen3CoderModelID,
			Name:          "qwen3-coder",
			Provider:      OpenRouterModelProvider,
			ContextWindow: "262k",
		},
		APIKey:  OpenRouterModelAPIKey,
		BaseURL: OpenRouterModelBaseURL,
	},
	XiaoMiMimoV2FlashModelID: {
		Info: &models.ModelInfo{
			ID:            XiaoMiMimoV2FlashModelID,
			Name:          "mimo-v2-flash",
			Provider:      OpenRouterModelProvider,
			ContextWindow: "262k",
		},
		APIKey:  OpenRouterModelAPIKey,
		BaseURL: OpenRouterModelBaseURL,
	},
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
	case ByteDanceModelProvider:
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
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
