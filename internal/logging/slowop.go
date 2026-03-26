package logging

import (
	"log/slog"
	"sync"
	"time"
)

// SlowOpDetector tracks in-flight operations and logs warnings when any operation
// exceeds a configured threshold. The detector runs a background goroutine that
// periodically scans for stuck operations. Zero cost when debug mode is off.
type SlowOpDetector struct {
	mu     sync.Mutex
	ops    map[uint64]trackedOp
	nextID uint64

	threshold time.Duration
	logger    *slog.Logger
	done      chan struct{}
	wg        sync.WaitGroup
}

type trackedOp struct {
	name    string
	started time.Time
	attrs   []slog.Attr
	warned  bool // true once we've already emitted a warning for this op
}

// NewSlowOpDetector creates a detector that checks for stuck operations every checkInterval.
// Operations exceeding threshold trigger a Warn log. Returns nil if debug mode is off.
func NewSlowOpDetector(threshold, checkInterval time.Duration) *SlowOpDetector {
	if !debugEnabled.Load() {
		return nil
	}
	d := &SlowOpDetector{
		ops:       make(map[uint64]trackedOp),
		threshold: threshold,
		logger:    ForComponent(CompPerf),
		done:      make(chan struct{}),
	}
	d.wg.Add(1)
	go d.checkLoop(checkInterval)
	return d
}

// Start begins tracking a named operation. Returns an ID to pass to Finish.
func (d *SlowOpDetector) Start(name string, attrs ...slog.Attr) uint64 {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	id := d.nextID
	d.nextID++
	d.ops[id] = trackedOp{
		name:    name,
		started: time.Now(),
		attrs:   attrs,
	}
	d.mu.Unlock()
	return id
}

// Finish marks an operation as complete and removes it from tracking.
func (d *SlowOpDetector) Finish(id uint64) {
	if d == nil {
		return
	}
	d.mu.Lock()
	delete(d.ops, id)
	d.mu.Unlock()
}

func (d *SlowOpDetector) checkLoop(interval time.Duration) {
	defer d.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.scan()
		case <-d.done:
			return
		}
	}
}

func (d *SlowOpDetector) scan() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for id, op := range d.ops {
		elapsed := now.Sub(op.started)
		if elapsed > d.threshold && !op.warned {
			args := make([]any, 0, len(op.attrs)*2+4)
			args = append(args, slog.String("op", op.name), slog.Duration("elapsed", elapsed))
			for _, a := range op.attrs {
				args = append(args, a)
			}
			d.logger.Warn("stuck_operation", args...)
			op.warned = true
			d.ops[id] = op
		}
	}
}

// Stop shuts down the background goroutine.
func (d *SlowOpDetector) Stop() {
	if d == nil {
		return
	}
	close(d.done)
	d.wg.Wait()
}
