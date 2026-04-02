package audit

import (
	"sync"
	"time"
)

// IntentTracker groups tool calls into intent contexts by temporal proximity.
type IntentTracker struct {
	store        *Store
	sessionID    string
	gapThreshold time.Duration

	mu            sync.Mutex
	currentIntent *activeIntent
}

type activeIntent struct {
	id            int64
	lastCallTime  time.Time
	sequenceOrder int
}

// NewIntentTracker creates a tracker with the given gap threshold.
// Tool calls within the gap are grouped into the same intent.
func NewIntentTracker(store *Store, sessionID string, gap time.Duration) *IntentTracker {
	return &IntentTracker{
		store:        store,
		sessionID:    sessionID,
		gapThreshold: gap,
	}
}

// Track assigns a tool call to an intent context.
func (t *IntentTracker) Track(toolCallID int64, timestamp time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.currentIntent == nil || timestamp.Sub(t.currentIntent.lastCallTime) > t.gapThreshold {
		// Start new intent context.
		id, err := t.store.CreateIntentContext(t.sessionID)
		if err != nil {
			return err
		}
		t.currentIntent = &activeIntent{
			id:            id,
			lastCallTime:  timestamp,
			sequenceOrder: 0,
		}
	}

	t.currentIntent.sequenceOrder++
	t.currentIntent.lastCallTime = timestamp

	return t.store.LinkToolCallToIntent(
		t.currentIntent.id,
		toolCallID,
		t.currentIntent.sequenceOrder,
	)
}
