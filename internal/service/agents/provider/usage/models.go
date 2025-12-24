package usage

import "time"

type TokenSnapshot struct {
	SessionTokens     *int               `json:"sessionTokens"`
	SessionCostUSD    *float64           `json:"sessionCostUSD"`
	Last30DaysCostUSD *float64           `json:"last30DaysCostUSD"`
	Daily             []DailyReportEntry `json:"daily"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

type DailyReport struct {
	Data    []DailyReportEntry  `json:"data"`
	Summary *DailyReportSummary `json:"summary"`
}

type DailyReportEntry struct {
	Date            string           `json:"date"`
	InputTokens     *int             `json:"inputTokens"`
	OutputTokens    *int             `json:"outputTokens"`
	TotalTokens     *int             `json:"totalTokens"`
	CostUSD         *float64         `json:"costUSD"`
	ModelsUsed      []string         `json:"modelsUsed"`
	ModelBreakdowns []ModelBreakdown `json:"modelBreakdowns"`
}

type ModelBreakdown struct {
	ModelName string   `json:"modelName"`
	CostUSD   *float64 `json:"costUSD"`
}

type DailyReportSummary struct {
	TotalInputTokens  *int     `json:"totalInputTokens"`
	TotalOutputTokens *int     `json:"totalOutputTokens"`
	TotalTokens       *int     `json:"totalTokens"`
	TotalCostUSD      *float64 `json:"totalCostUSD"`
}
