package usage

import (
	"time"
)

type IProvider interface {
	LoadTokenSnapshot(provider UsageProvider, now time.Time, forceRefresh bool) (*TokenSnapshot, error)
}
