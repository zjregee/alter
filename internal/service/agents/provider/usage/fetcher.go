package usage

import (
	"fmt"
	"sort"
	"time"
)

var _ IProvider = (*Fetcher)(nil)

type Fetcher struct{}

func NewFetcher() *Fetcher {
	return &Fetcher{}
}

type UsageProvider string

const (
	CodexProvider  UsageProvider = "codex"
	ClaudeProvider UsageProvider = "claude"
)

func (f *Fetcher) LoadTokenSnapshot(provider UsageProvider, now time.Time, forceRefresh bool) (*TokenSnapshot, error) {
	if now.IsZero() {
		return nil, fmt.Errorf("now time cannot be zero")
	}

	if provider != CodexProvider && provider != ClaudeProvider {
		return nil, fmt.Errorf("cost summary is not supported for %s", provider)
	}

	until := now
	since := now.AddDate(0, 0, -29)

	options := ScannerOptions{}
	if forceRefresh {
		options.RefreshMinInterval = 0
	} else {
		options.RefreshMinInterval = 60 * time.Second
	}

	daily, err := LoadDailyReport(string(provider), since, until, now, options)
	if err != nil {
		return nil, err
	}

	return tokenSnapshotFromDaily(daily, now), nil
}

func tokenSnapshotFromDaily(daily *DailyReport, now time.Time) *TokenSnapshot {
	var todayEntry *DailyReportEntry
	sortedData := make([]DailyReportEntry, 0)

	if len(daily.Data) > 0 {
		sortedData = make([]DailyReportEntry, len(daily.Data))
		copy(sortedData, daily.Data)

		sort.Slice(sortedData, func(i, j int) bool {
			if sortedData[i].Date != sortedData[j].Date {
				return sortedData[i].Date > sortedData[j].Date
			}
			costI := 0.0
			if sortedData[i].CostUSD != nil {
				costI = *sortedData[i].CostUSD
			}
			costJ := 0.0
			if sortedData[j].CostUSD != nil {
				costJ = *sortedData[j].CostUSD
			}
			if costI != costJ {
				return costI > costJ
			}
			tokensI := 0
			if sortedData[i].TotalTokens != nil {
				tokensI = *sortedData[i].TotalTokens
			}
			tokensJ := 0
			if sortedData[j].TotalTokens != nil {
				tokensJ = *sortedData[j].TotalTokens
			}
			return tokensI > tokensJ
		})

		todayKey := dayKey(now)
		for i, entry := range sortedData {
			if entry.Date == todayKey {
				todayEntry = &sortedData[i]
				break
			}
		}
	}

	var last30DaysCostUSD *float64
	if daily.Summary != nil && daily.Summary.TotalCostUSD != nil {
		last30DaysCostUSD = daily.Summary.TotalCostUSD
	} else {
		var total float64
		costSeen := false
		for _, entry := range daily.Data {
			if entry.CostUSD != nil {
				total += *entry.CostUSD
				costSeen = true
			}
		}
		if costSeen {
			last30DaysCostUSD = &total
		}
	}

	snapshot := &TokenSnapshot{
		Daily:             sortedData,
		UpdatedAt:         now,
		Last30DaysCostUSD: last30DaysCostUSD,
	}

	if todayEntry != nil {
		snapshot.SessionTokens = todayEntry.TotalTokens
		snapshot.SessionCostUSD = todayEntry.CostUSD
	}

	return snapshot
}
