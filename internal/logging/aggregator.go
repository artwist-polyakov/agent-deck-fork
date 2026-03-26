package logging

import (
	"log/slog"
	"sync"
	"time"
)

// aggregateKey uniquely identifies an event type for batching.
type aggregateKey struct {
	Component string
	Event     string
}

// aggregateEntry tracks a batched event's count and last-seen fields.
type aggregateEntry struct {
	Count  int64
	Fields []slog.Attr
}

// Aggregator batches high-frequency events and emits summaries periodically.
type Aggregator struct {
	logger   *slog.Logger
	interval time.Duration

	mu      sync.Mutex
	entries map[aggregateKey]*aggregateEntry

	done chan struct{}
	wg   sync.WaitGroup
}

// NewAggregator creates an aggregator that flushes every intervalSecs seconds.
// If logger is nil, recorded events are silently dropped.
func NewAggregator(logger *slog.Logger, intervalSecs int) *Aggregator {
	if intervalSecs <= 0 {
		intervalSecs = 30
	}
	return &Aggregator{
		logger:   logger,
		interval: time.Duration(intervalSecs) * time.Second,
		entries:  make(map[aggregateKey]*aggregateEntry),
		done:     make(chan struct{}),
	}
}

// Start begins the background flush goroutine.
func (a *Aggregator) Start() {
	a.wg.Add(1)
	go a.flushLoop()
}

// Stop flushes remaining entries and stops the background goroutine.
func (a *Aggregator) Stop() {
	close(a.done)
	a.wg.Wait()
	a.flush() // Final flush
}

// Record increments the counter for an event type.
// fields are kept from the most recent call (last-writer-wins for context).
func (a *Aggregator) Record(component, event string, fields ...slog.Attr) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := aggregateKey{Component: component, Event: event}
	entry, ok := a.entries[key]
	if !ok {
		entry = &aggregateEntry{}
		a.entries[key] = entry
	}
	entry.Count++
	if len(fields) > 0 {
		entry.Fields = fields
	}
}

func (a *Aggregator) flushLoop() {
	defer a.wg.Done()
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.flush()
		case <-a.done:
			return
		}
	}
}

func (a *Aggregator) flush() {
	a.mu.Lock()
	if len(a.entries) == 0 {
		a.mu.Unlock()
		return
	}
	// Swap out entries under lock
	entries := a.entries
	a.entries = make(map[aggregateKey]*aggregateEntry)
	a.mu.Unlock()

	if a.logger == nil {
		return
	}

	for key, entry := range entries {
		attrs := []any{
			slog.String("component", key.Component),
			slog.String("event", key.Event),
			slog.Int64("count", entry.Count),
			slog.Int("window_seconds", int(a.interval.Seconds())),
		}
		for _, f := range entry.Fields {
			attrs = append(attrs, f)
		}
		a.logger.Info("event_summary", attrs...)
	}
}
