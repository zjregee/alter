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
	if provider != CodexProvider && provider != ClaudeProvider {
		return nil, fmt.Errorf("cost summary is not supported for %s", provider)
	}

	until := now
	// Rolling window: last 30 days (inclusive). Use -29 for inclusive boundaries.
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
	var currentDay *DailyReportEntry
	if len(daily.Data) > 0 {
		// Sort by date descending, then cost, then tokens
		sort.Slice(daily.Data, func(i, j int) bool {
			if daily.Data[i].Date != daily.Data[j].Date {
				return daily.Data[i].Date > daily.Data[j].Date
			}
			costI := 0.0
			if daily.Data[i].CostUSD != nil {
				costI = *daily.Data[i].CostUSD
			}
			costJ := 0.0
			if daily.Data[j].CostUSD != nil {
				costJ = *daily.Data[j].CostUSD
			}
			if costI != costJ {
				return costI > costJ
			}
			tokensI := 0
			if daily.Data[i].TotalTokens != nil {
				tokensI = *daily.Data[i].TotalTokens
			}
			tokensJ := 0
			if daily.Data[j].TotalTokens != nil {
				tokensJ = *daily.Data[j].TotalTokens
			}
			return tokensI > tokensJ
		})

		// Find the entry for today. The swift code seems to just pick the max, which if sorted by date,
		// would be the latest day. But if there is no data for 'now', it would pick a past day.
		// Let's find the entry for the current day based on 'now'.
		todayKey := dayKey(now)
		for i, entry := range daily.Data {
			if entry.Date == todayKey {
				currentDay = &daily.Data[i]
				break
			}
		}
		// If no entry for today, the latest entry is considered the "session"
		if currentDay == nil {
			currentDay = &daily.Data[0]
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
		Daily:             daily.Data,
		UpdatedAt:         now,
		Last30DaysCostUSD: last30DaysCostUSD,
	}

	if currentDay != nil {
		snapshot.SessionTokens = currentDay.TotalTokens
		snapshot.SessionCostUSD = currentDay.CostUSD
	}

	return snapshot
}
