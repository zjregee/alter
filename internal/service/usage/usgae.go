package usage

import (
	"sync"
	"time"

	"github.com/zjregee/alter/internal/service/agents/provider/usage"
)

var (
	instance *usageService
	once     sync.Once
)

type usageService struct {
	fetcher usage.IProvider
}

func getInstance() *usageService {
	once.Do(func() {
		instance = &usageService{
			fetcher: usage.NewFetcher(),
		}
	})
	return instance
}

type ModelUsage struct {
	ModelName    string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

type UsageSummary struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalTokens       int
	TotalCostUSD      float64
	Models            []ModelUsage
	UpdatedAt         time.Time
}

func GetLastWeekUsage() (*UsageSummary, error) {
	return getUsageSummary(7, false)
}

func GetLastMonthUsage() (*UsageSummary, error) {
	return getUsageSummary(30, false)
}

func GetCurrentWeekUsage() (*UsageSummary, error) {
	return getUsageSummary(0, true)
}

func GetCurrentMonthUsage() (*UsageSummary, error) {
	return getUsageSummary(0, false)
}

func getUsageSummary(lastNDays int, isCurrentWeek bool) (*UsageSummary, error) {
	s := getInstance()
	now := time.Now()

	providers := []usage.UsageProvider{usage.CodexProvider, usage.ClaudeProvider}
	modelUsageMap := make(map[string]*ModelUsage)

	for _, provider := range providers {
		snapshot, err := s.fetcher.LoadTokenSnapshot(provider, now, false)
		if err != nil {
			continue
		}

		var filteredDaily []usage.DailyReportEntry
		if lastNDays == 30 {
			filteredDaily = snapshot.Daily
		} else if lastNDays > 0 {
			filteredDaily = filterLastNDays(snapshot.Daily, lastNDays, now)
		} else if isCurrentWeek {
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			monday := now.AddDate(0, 0, -(weekday - 1))
			mondayKey := dayKey(monday)
			filteredDaily = filterFromDate(snapshot.Daily, mondayKey)
		} else {
			firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			firstDayKey := dayKey(firstDayOfMonth)
			filteredDaily = filterFromDate(snapshot.Daily, firstDayKey)
		}

		for _, entry := range filteredDaily {
			modelsToProcess := entry.ModelsUsed
			if len(modelsToProcess) == 0 && len(entry.ModelBreakdowns) > 0 {
				modelsToProcess = make([]string, 0, len(entry.ModelBreakdowns))
				for _, breakdown := range entry.ModelBreakdowns {
					modelsToProcess = append(modelsToProcess, breakdown.ModelName)
				}
			}

			for _, breakdown := range entry.ModelBreakdowns {
				if _, exists := modelUsageMap[breakdown.ModelName]; !exists {
					modelUsageMap[breakdown.ModelName] = &ModelUsage{
						ModelName: breakdown.ModelName,
					}
				}
				if breakdown.CostUSD != nil {
					modelUsageMap[breakdown.ModelName].CostUSD += *breakdown.CostUSD
				}
			}

			if len(modelsToProcess) > 0 {
				if entry.InputTokens != nil {
					tokensPerModel := *entry.InputTokens / len(modelsToProcess)
					for _, modelName := range modelsToProcess {
						if _, exists := modelUsageMap[modelName]; !exists {
							modelUsageMap[modelName] = &ModelUsage{
								ModelName: modelName,
							}
						}
						modelUsageMap[modelName].InputTokens += tokensPerModel
					}
				}
				if entry.OutputTokens != nil {
					tokensPerModel := *entry.OutputTokens / len(modelsToProcess)
					for _, modelName := range modelsToProcess {
						if _, exists := modelUsageMap[modelName]; !exists {
							modelUsageMap[modelName] = &ModelUsage{
								ModelName: modelName,
							}
						}
						modelUsageMap[modelName].OutputTokens += tokensPerModel
					}
				}
				if entry.TotalTokens != nil {
					tokensPerModel := *entry.TotalTokens / len(modelsToProcess)
					for _, modelName := range modelsToProcess {
						if _, exists := modelUsageMap[modelName]; !exists {
							modelUsageMap[modelName] = &ModelUsage{
								ModelName: modelName,
							}
						}
						modelUsageMap[modelName].TotalTokens += tokensPerModel
					}
				}
			}
		}
	}

	summary := &UsageSummary{
		Models:    make([]ModelUsage, 0, len(modelUsageMap)),
		UpdatedAt: now,
	}

	for _, mu := range modelUsageMap {
		summary.TotalInputTokens += mu.InputTokens
		summary.TotalOutputTokens += mu.OutputTokens
		summary.TotalTokens += mu.TotalTokens
		summary.TotalCostUSD += mu.CostUSD
		summary.Models = append(summary.Models, *mu)
	}

	return summary, nil
}

func filterLastNDays(daily []usage.DailyReportEntry, n int, now time.Time) []usage.DailyReportEntry {
	if len(daily) == 0 {
		return daily
	}

	cutoffDate := now.AddDate(0, 0, -n+1)
	cutoffKey := dayKey(cutoffDate)

	filtered := make([]usage.DailyReportEntry, 0, len(daily))
	for _, entry := range daily {
		if entry.Date >= cutoffKey {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func filterFromDate(daily []usage.DailyReportEntry, fromDateKey string) []usage.DailyReportEntry {
	if len(daily) == 0 {
		return daily
	}

	filtered := make([]usage.DailyReportEntry, 0, len(daily))
	for _, entry := range daily {
		if entry.Date >= fromDateKey {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}
