package usage

import "time"

// IProvider is the interface for fetching usage data.
type IProvider interface {
	LoadTokenSnapshot(provider UsageProvider, now time.Time, forceRefresh bool) (*TokenSnapshot, error)
}
