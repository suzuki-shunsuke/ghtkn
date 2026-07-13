package agent

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/suzuki-shunsuke/ghtkn/pkg/controller/agent"
)

// Units for the d/w/m TTL suffixes. A month is counted as 30 days.
const (
	day   = 24 * time.Hour
	week  = 7 * day
	month = 30 * day
)

// ttlUnit maps a d/w/m suffix (units time.ParseDuration does not understand but that are
// practical for a TTL measured in days, weeks, or months) to its duration.
func ttlUnit(b byte) (time.Duration, bool) {
	switch b {
	case 'd':
		return day, true
	case 'w':
		return week, true
	case 'm':
		return month, true
	default:
		return 0, false
	}
}

// parseRefreshTokenTTL parses a --refresh-token-ttl value. It accepts a number with a
// d (day), w (week), or m (30-day month) suffix, e.g. "3d", "4w", "2m", "1.5w", and
// falls back to time.ParseDuration for other inputs (e.g. "168h"). It rejects empty,
// negative, and out-of-range values: the TTL must not be negative and must not exceed
// six months. Zero is allowed and means "use the agent default".
func parseRefreshTokenTTL(s string) (time.Duration, error) {
	d, err := parseDurationValue(s)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("refresh-token-ttl must not be negative: %q", s)
	}
	if d > agent.MaxRefreshTokenTTL {
		return 0, fmt.Errorf("refresh-token-ttl must not exceed 6 months: %q", s)
	}
	return d, nil
}

func parseDurationValue(s string) (time.Duration, error) {
	if s == "" {
		return 0, errors.New("refresh-token-ttl must not be empty")
	}
	if unit, ok := ttlUnit(s[len(s)-1]); ok {
		n, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, fmt.Errorf("parse refresh-token-ttl %q: %w", s, err)
		}
		return time.Duration(n * float64(unit)), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("parse refresh-token-ttl %q: %w", s, err)
	}
	return d, nil
}
