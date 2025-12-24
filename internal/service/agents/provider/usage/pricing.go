package usage

import (
	"regexp"
	"strings"
)

type CodexPricing struct {
	InputCostPerToken          float64
	OutputCostPerToken         float64
	CacheReadInputCostPerToken float64
}

type ClaudePricing struct {
	InputCostPerToken                            float64
	OutputCostPerToken                           float64
	CacheCreationInputCostPerToken               float64
	CacheReadInputCostPerToken                   float64
	ThresholdTokens                              *int
	InputCostPerTokenAboveThreshold              *float64
	OutputCostPerTokenAboveThreshold             *float64
	CacheCreationInputCostPerTokenAboveThreshold *float64
	CacheReadInputCostPerTokenAboveThreshold     *float64
}

var codexPricing = map[string]CodexPricing{
	"gpt-5": {
		InputCostPerToken:          1.25e-6,
		OutputCostPerToken:         1e-5,
		CacheReadInputCostPerToken: 1.25e-7,
	},
	"gpt-5-codex": {
		InputCostPerToken:          1.25e-6,
		OutputCostPerToken:         1e-5,
		CacheReadInputCostPerToken: 1.25e-7,
	},
	"gpt-5.1": {
		InputCostPerToken:          1.25e-6,
		OutputCostPerToken:         1e-5,
		CacheReadInputCostPerToken: 1.25e-7,
	},
	"gpt-5.2": {
		InputCostPerToken:          1.75e-6,
		OutputCostPerToken:         1.4e-5,
		CacheReadInputCostPerToken: 1.75e-7,
	},
	"gpt-5.2-codex": {
		InputCostPerToken:          1.75e-6,
		OutputCostPerToken:         1.4e-5,
		CacheReadInputCostPerToken: 1.75e-7,
	},
}

var claudePricing = map[string]ClaudePricing{
	"claude-haiku-4-5-20251001": {
		InputCostPerToken:              1e-6,
		OutputCostPerToken:             5e-6,
		CacheCreationInputCostPerToken: 1.25e-6,
		CacheReadInputCostPerToken:     1e-7,
	},
	"claude-opus-4-5-20251101": {
		InputCostPerToken:              5e-6,
		OutputCostPerToken:             2.5e-5,
		CacheCreationInputCostPerToken: 6.25e-6,
		CacheReadInputCostPerToken:     5e-7,
	},
	"claude-sonnet-4-5": {
		InputCostPerToken:                            3e-6,
		OutputCostPerToken:                           1.5e-5,
		CacheCreationInputCostPerToken:               3.75e-6,
		CacheReadInputCostPerToken:                   3e-7,
		ThresholdTokens:                              intPtr(200_000),
		InputCostPerTokenAboveThreshold:              float64Ptr(6e-6),
		OutputCostPerTokenAboveThreshold:             float64Ptr(2.25e-5),
		CacheCreationInputCostPerTokenAboveThreshold: float64Ptr(7.5e-6),
		CacheReadInputCostPerTokenAboveThreshold:     float64Ptr(6e-7),
	},
	"claude-sonnet-4-5-20250929": {
		InputCostPerToken:                            3e-6,
		OutputCostPerToken:                           1.5e-5,
		CacheCreationInputCostPerToken:               3.75e-6,
		CacheReadInputCostPerToken:                   3e-7,
		ThresholdTokens:                              intPtr(200_000),
		InputCostPerTokenAboveThreshold:              float64Ptr(6e-6),
		OutputCostPerTokenAboveThreshold:             float64Ptr(2.25e-5),
		CacheCreationInputCostPerTokenAboveThreshold: float64Ptr(7.5e-6),
		CacheReadInputCostPerTokenAboveThreshold:     float64Ptr(6e-7),
	},
	"claude-opus-4-20250514": {
		InputCostPerToken:              1.5e-5,
		OutputCostPerToken:             7.5e-5,
		CacheCreationInputCostPerToken: 1.875e-5,
		CacheReadInputCostPerToken:     1.5e-6,
	},
	"claude-opus-4-1": {
		InputCostPerToken:              1.5e-5,
		OutputCostPerToken:             7.5e-5,
		CacheCreationInputCostPerToken: 1.875e-5,
		CacheReadInputCostPerToken:     1.5e-6,
	},
	"claude-sonnet-4-20250514": {
		InputCostPerToken:                            3e-6,
		OutputCostPerToken:                           1.5e-5,
		CacheCreationInputCostPerToken:               3.75e-6,
		CacheReadInputCostPerToken:                   3e-7,
		ThresholdTokens:                              intPtr(200_000),
		InputCostPerTokenAboveThreshold:              float64Ptr(6e-6),
		OutputCostPerTokenAboveThreshold:             float64Ptr(2.25e-5),
		CacheCreationInputCostPerTokenAboveThreshold: float64Ptr(7.5e-6),
		CacheReadInputCostPerTokenAboveThreshold:     float64Ptr(6e-7),
	},
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func normalizeCodexModel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "openai/")
	if strings.Contains(trimmed, "-codex") {
		base := strings.Split(trimmed, "-codex")[0]
		if _, ok := codexPricing[base]; ok {
			return base
		}
	}
	return trimmed
}

var claudeVersionRegex = regexp.MustCompile(`-v\d+:\d+$`)
var claudeDateRegex = regexp.MustCompile(`-\d{8}$`)

func normalizeClaudeModel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "anthropic.")

	if lastDot := strings.LastIndex(trimmed, "."); lastDot != -1 && strings.Contains(trimmed, "claude-") {
		tail := trimmed[lastDot+1:]
		if strings.HasPrefix(tail, "claude-") {
			trimmed = tail
		}
	}

	trimmed = claudeVersionRegex.ReplaceAllString(trimmed, "")

	if baseRange := claudeDateRegex.FindStringIndex(trimmed); baseRange != nil {
		base := trimmed[:baseRange[0]]
		if _, ok := claudePricing[base]; ok {
			return base
		}
	}

	return trimmed
}

func GetCodexCostUSD(model string, inputTokens, cachedInputTokens, outputTokens int) *float64 {
	key := normalizeCodexModel(model)
	pricing, ok := codexPricing[key]
	if !ok {
		return nil
	}
	cached := min(max(0, cachedInputTokens), max(0, inputTokens))
	nonCached := max(0, inputTokens-cached)
	cost := float64(nonCached)*pricing.InputCostPerToken +
		float64(cached)*pricing.CacheReadInputCostPerToken +
		float64(max(0, outputTokens))*pricing.OutputCostPerToken
	return &cost
}

func GetClaudeCostUSD(model string, inputTokens, cacheReadInputTokens, cacheCreationInputTokens, outputTokens int) *float64 {
	key := normalizeClaudeModel(model)
	pricing, ok := claudePricing[key]
	if !ok {
		return nil
	}

	tiered := func(tokens int, base float64, above *float64, threshold *int) float64 {
		if threshold == nil || above == nil {
			return float64(tokens) * base
		}
		below := min(tokens, *threshold)
		over := max(tokens-*threshold, 0)
		return float64(below)*base + float64(over)**above
	}

	cost := tiered(max(0, inputTokens), pricing.InputCostPerToken, pricing.InputCostPerTokenAboveThreshold, pricing.ThresholdTokens) +
		tiered(max(0, cacheReadInputTokens), pricing.CacheReadInputCostPerToken, pricing.CacheReadInputCostPerTokenAboveThreshold, pricing.ThresholdTokens) +
		tiered(max(0, cacheCreationInputTokens), pricing.CacheCreationInputCostPerToken, pricing.CacheCreationInputCostPerTokenAboveThreshold, pricing.ThresholdTokens) +
		tiered(max(0, outputTokens), pricing.OutputCostPerToken, pricing.OutputCostPerTokenAboveThreshold, pricing.ThresholdTokens)

	return &cost
}
