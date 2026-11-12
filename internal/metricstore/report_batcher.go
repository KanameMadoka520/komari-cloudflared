package metricstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/komari-monitor/komari/internal/trafficcounter"
	"sync"
	"time"

	logger "github.com/komari-monitor/komari/utils/log"

	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/pkg/metric"
	v2 "github.com/komari-monitor/komari/protocol/v2"
)

type reportTrafficState struct {
	mu sync.Mutex

	reportTrafficValues
}

type reportTrafficValues struct {
	hasLifetime              bool
	lifetimeUp, lifetimeDown int64
	counter                  trafficcounter.State
	initialized              bool
	timestamp                time.Time
	hasUp                    bool
	totalUp                  int64
	totalUpAt                time.Time
	hasDown                  bool
	totalDown                int64
	totalDownAt              time.Time
}

var (
	reportTrafficStates sync.Map
	reportWriteMu       sync.Mutex
)

const (
	reportBatchInterval     = 3 * time.Second
	reportBatchQueueSize    = 512
	pingBatchQueueSize      = 4096
	pingBatchMaxRecords     = 512
	reportBatchWriteTimeout = 10 * time.Second
	// Longer gaps have no reliable per-bucket traffic attribution.
	maxTrafficReportGap = 15 * time.Minute
)

var (
	reportBatcherMu sync.Mutex
	reportBatcher   *reportBatchWorker
)

var (
	ErrReportBatchQueueFull = errors.New("metric report batch queue is full")
	ErrReportBatchStopped   = errors.New("metric report batcher is stopped")
	ErrPingBatchQueueFull   = errors.New("ping batch queue is full")
)

type reportBatchRequest struct {
	ctx  context.Context
	done chan error
	stop bool
}

type reportBatchWorker struct {
	mu        sync.Mutex
	queue     chan v2.Report
	pingQueue chan models.PingRecord
	requests  chan reportBatchRequest
	done      chan struct{}
	stopping  bool
}

// StartReportBatcher starts the shared report writer. Exact samples are kept in
// the short raw window while the same writes update in-memory minute buckets.
func StartReportBatcher() {
	reportBatcherMu.Lock()
	defer reportBatcherMu.Unlock()
	if reportBatcher != nil {
		return
	}
	worker := &reportBatchWorker{
		queue:     make(chan v2.Report, reportBatchQueueSize),
		pingQueue: make(chan models.PingRecord, pingBatchQueueSize),
		requests:  make(chan reportBatchRequest, 1),
		done:      make(chan struct{}),
	}
	reportBatcher = worker
	go worker.run()
}

// StopReportBatcher stops the report writer after flushing all queued reports.
func StopReportBatcher(ctx context.Context) error {
	reportBatcherMu.Lock()
	worker := reportBatcher
	if worker == nil {
		reportBatcherMu.Unlock()
		return nil
	}
	worker.mu.Lock()
	worker.stopping = true
	worker.mu.Unlock()
	reportBatcherMu.Unlock()

	request := reportBatchRequest{ctx: ctx, done: make(chan error, 1), stop: true}
	select {
	case worker.requests <- request:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-request.done:
		<-worker.done
		reportBatcherMu.Lock()
		if reportBatcher == worker {
			reportBatcher = nil
		}
		reportBatcherMu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// FlushReportBatch synchronously flushes the current queue. It is useful for
// controlled handoff points and deterministic tests; normal operation uses the
// worker ticker and the active mode's flush interval.
func FlushReportBatch(ctx context.Context) error {
	reportBatcherMu.Lock()
	worker := reportBatcher
	reportBatcherMu.Unlock()
	if worker == nil {
		return nil
	}
	request := reportBatchRequest{ctx: ctx, done: make(chan error, 1)}
	worker.mu.Lock()
	stopping := worker.stopping
	worker.mu.Unlock()
	if stopping {
		return ErrReportBatchStopped
	}
	select {
	case worker.requests <- request:
	case <-worker.done:
		return ErrReportBatchStopped
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-request.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WriteReport persists one agent report and adds it to in-memory minute
// summaries using the same server receive time. Traffic deltas remain summable
// after rollup.
func WriteReport(ctx context.Context, report v2.Report) (v2.Report, error) {
	if report.UUID == "" {
		return v2.Report{}, fmt.Errorf("report UUID is required")
	}
	if report.UpdatedAt.IsZero() {
		return v2.Report{}, fmt.Errorf("report receive time is required")
	}
	report.UpdatedAt = report.UpdatedAt.UTC()
	if GetStore() == nil {
		return v2.Report{}, fmt.Errorf("metric store not enabled")
	}

	reportBatcherMu.Lock()
	worker := reportBatcher
	reportBatcherMu.Unlock()
	if worker != nil {
		if err := worker.enqueue(ctx, report); err != nil {
			return v2.Report{}, err
		}
		return report, nil
	}

	saved, err := writeReportBatch(ctx, []v2.Report{report})
	if err != nil {
		return v2.Report{}, err
	}
	return saved[0], nil
}

func (w *reportBatchWorker) enqueue(ctx context.Context, report v2.Report) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopping {
		return ErrReportBatchStopped
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case w.queue <- report:
		return nil
	default:
		return ErrReportBatchQueueFull
	}
}

func (w *reportBatchWorker) run() {
	ticker := time.NewTicker(reportBatchInterval)
	defer ticker.Stop()

	var pending []v2.Report
	var pendingPings []models.PingRecord
	for {
		select {
		case request := <-w.requests:
			pending = append(pending, drainQueue(w.queue, reportBatchQueueSize)...)
			pendingPings = append(pendingPings, drainQueue(w.pingQueue, pingBatchQueueSize)...)
			err := errors.Join(
				writePendingReports(request.ctx, &pending),
				writePendingPingRecords(request.ctx, &pendingPings),
			)
			if request.stop {
				if err != nil {
					logger.Errorf("metricstore", "failed to flush metric report batch during shutdown: %v", err)
				}
				close(w.done)
				request.done <- err
				return
			}
			request.done <- err
		case <-ticker.C:
			pendingPings = append(pendingPings, drainQueue(w.pingQueue, pingBatchQueueSize)...)
			if err := writePendingPingRecords(context.Background(), &pendingPings); err != nil {
				logger.Errorf("metricstore", "failed to flush ping batch: %v", err)
			}
			pending = append(pending, drainQueue(w.queue, reportBatchQueueSize)...)
			if err := writePendingReports(context.Background(), &pending); err != nil {
				logger.Errorf("metricstore", "failed to flush metric report batch: %v", err)
			}
		}
	}
}

func (w *reportBatchWorker) enqueuePing(ctx context.Context, record models.PingRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopping {
		return ErrReportBatchStopped
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case w.pingQueue <- record:
		return nil
	default:
		return ErrPingBatchQueueFull
	}
}

// drainQueue takes up to limit queued items without blocking.
func drainQueue[T any](queue <-chan T, limit int) []T {
	n := min(len(queue), limit)
	if n <= 0 {
		return nil
	}
	items := make([]T, 0, n)
	for len(items) < limit {
		select {
		case item := <-queue:
			items = append(items, item)
		default:
			return items
		}
	}
	return items
}

// writePendingReports commits all pending reports as one batch and keeps them
// for the next attempt when the write fails.
func writePendingReports(ctx context.Context, pending *[]v2.Report) error {
	if len(*pending) == 0 {
		return nil
	}
	writeCtx, cancel := context.WithTimeout(ctx, reportBatchWriteTimeout)
	defer cancel()
	if _, err := writeReportBatch(writeCtx, *pending); err != nil {
		return err
	}
	*pending = (*pending)[:0]
	return nil
}

func writePendingPingRecords(ctx context.Context, pending *[]models.PingRecord) error {
	for len(*pending) > 0 {
		batchSize := len(*pending)
		if batchSize > pingBatchMaxRecords {
			batchSize = pingBatchMaxRecords
		}
		writeCtx, cancel := context.WithTimeout(ctx, reportBatchWriteTimeout)
		err := writePingRecords(writeCtx, (*pending)[:batchSize])
		cancel()
		if err != nil {
			return err
		}
		*pending = (*pending)[batchSize:]
	}
	return nil
}

func writeReportBatch(ctx context.Context, reports []v2.Report) ([]v2.Report, error) {
	if len(reports) == 0 {
		return nil, nil
	}
	if err := storeOperations.AcquireShared(ctx); err != nil {
		return nil, fmt.Errorf("wait for metric store operation before writing reports: %w", err)
	}
	defer storeOperations.ReleaseShared()

	// Serialize baseline reads and commits, including direct writes without
	// the background batcher.
	reportWriteMu.Lock()
	defer reportWriteMu.Unlock()

	s := GetStore()
	if s == nil {
		return nil, fmt.Errorf("metric store not enabled")
	}

	prepared := make([]v2.Report, len(reports))
	copy(prepared, reports)
	points := make([]metric.Point, 0, len(reports)*20)
	pendingStates := make(map[*reportTrafficState]reportTrafficValues)
	for i, report := range prepared {
		stateValue, _ := reportTrafficStates.LoadOrStore(report.UUID, &reportTrafficState{})
		state := stateValue.(*reportTrafficState)
		values, ok := pendingStates[state]
		if !ok {
			state.mu.Lock()
			values = state.reportTrafficValues
			state.mu.Unlock()
		}
		if !values.initialized {
			totalUp, hasUp, err := latestReportPoint(ctx, s, MetricNetTotalUp, report.UUID, report.UpdatedAt)
			if err != nil {
				logger.Errorf("metricstore", "failed to restore previous upload counter for %s: %v", report.UUID, err)
			} else {
				values.totalUp = int64(totalUp.Value)
				values.totalUpAt = totalUp.Timestamp
				values.hasUp = hasUp
			}
			totalDown, hasDown, err := latestReportPoint(ctx, s, MetricNetTotalDown, report.UUID, report.UpdatedAt)
			if err != nil {
				logger.Errorf("metricstore", "failed to restore previous download counter for %s: %v", report.UUID, err)
			} else {
				values.totalDown = int64(totalDown.Value)
				values.totalDownAt = totalDown.Timestamp
				values.hasDown = hasDown
			}
			highUp, hasHighUp, highErr := latestReportCounter(ctx, s, MetricTrafficHighUp, report.UUID, report.UpdatedAt)
			if highErr != nil {
				return nil, highErr
			}
			highDown, hasHighDown, highErr := latestReportCounter(ctx, s, MetricTrafficHighDown, report.UUID, report.UpdatedAt)
			if highErr != nil {
				return nil, highErr
			}
			if !hasHighUp {
				highUp = values.totalUp
			}
			if !hasHighDown {
				highDown = values.totalDown
			}
			values.counter = trafficcounter.State{Initialized: values.hasUp || values.hasDown, Up: trafficcounter.Direction{HighWater: highUp}, Down: trafficcounter.Direction{HighWater: highDown}, Epoch: report.Network.CounterEpoch, Bits: report.Network.CounterBits}
			acceptedUp, hasAcceptedUp, err := latestReportCounter(ctx, s, MetricTrafficAcceptedUp, report.UUID, report.UpdatedAt)
			if err != nil {
				return nil, err
			}
			acceptedDown, hasAcceptedDown, err := latestReportCounter(ctx, s, MetricTrafficAcceptedDown, report.UUID, report.UpdatedAt)
			if err != nil {
				return nil, err
			}
			values.hasLifetime = hasAcceptedUp && hasAcceptedDown
			values.lifetimeUp, values.lifetimeDown = acceptedUp, acceptedDown
			values.initialized = true
			// Keep only the restored baseline on failure. WriteBatch may have
			// accepted raw samples before a rollup flush fails, so restoring
			// again on retry could otherwise read this uncommitted batch.
			state.mu.Lock()
			state.reportTrafficValues = values
			state.mu.Unlock()
		}

		if !values.timestamp.IsZero() && !report.UpdatedAt.After(values.timestamp) {
			report.UpdatedAt = values.timestamp.Add(time.Millisecond)
		}
		trafficUp, trafficDown := int64(0), int64(0)
		if report.Traffic != nil {
			if report.Traffic.Skip {
				continue
			}
			trafficUp, trafficDown = report.Traffic.Up, report.Traffic.Down
			// A main-DB observation can survive a crash before its metric queue is
			// flushed. Bridge it from durable accepted totals, never raw counters.
			if report.Traffic.HasLifetime {
				observation := *report.Traffic
				report.Traffic = &observation
				if values.hasLifetime {
					if observation.LifetimeUp < values.lifetimeUp || observation.LifetimeDown < values.lifetimeDown {
						return nil, fmt.Errorf("accepted traffic total moved backwards for %s", report.UUID)
					}
					trafficUp = observation.LifetimeUp - values.lifetimeUp
					trafficDown = observation.LifetimeDown - values.lifetimeDown
				}
				// Metrics persist millisecond timestamps while main-DB observations
				// retain nanoseconds. Compare at storage precision so a clean restart
				// does not turn the same flushed sample into an apparent lost report.
				if !observation.PreviousAt.IsZero() && (observation.PreviousAt.UnixMilli() > values.totalUpAt.UnixMilli() || observation.PreviousAt.UnixMilli() > values.totalDownAt.UnixMilli()) {
					report.Traffic.Uncertain = true
				}
				if (!values.totalUpAt.IsZero() && report.UpdatedAt.Sub(values.totalUpAt) > maxTrafficReportGap) || (!values.totalDownAt.IsZero() && report.UpdatedAt.Sub(values.totalDownAt) > maxTrafficReportGap) {
					report.Traffic.Gap = true
				}
				values.hasLifetime = true
				values.lifetimeUp, values.lifetimeDown = observation.LifetimeUp, observation.LifetimeDown
			}

			if report.Traffic.Gap {
				trafficUp, trafficDown = 0, 0
			}
			values.counter.Up.HighWater, values.counter.Down.HighWater = report.Traffic.HighWaterUp, report.Traffic.HighWaterDown
			values.counter.Initialized = true
		} else {
			next, delta, err := values.counter.AdvanceObserved(report.Network.TotalUp, report.Network.TotalDown, report.Network.CounterEpoch, report.Network.CounterBits, report.Uptime, report.UpdatedAt)
			if err != nil {
				return nil, err
			}
			values.counter = next
			trafficUp, trafficDown = delta.Up, delta.Down
			gap := (!values.totalUpAt.IsZero() && report.UpdatedAt.Sub(values.totalUpAt) > maxTrafficReportGap) || (!values.totalDownAt.IsZero() && report.UpdatedAt.Sub(values.totalDownAt) > maxTrafficReportGap)
			if !values.hasUp || gap {
				trafficUp = 0
			}
			if !values.hasDown || gap {
				trafficDown = 0
			}
			report.Traffic = &v2.TrafficObservation{Up: trafficUp, Down: trafficDown, HighWaterUp: next.Up.HighWater, HighWaterDown: next.Down.HighWater, Uncertain: delta.Uncertain, Gap: gap}
		}
		points = append(points, reportMetricPoints(report, trafficUp, trafficDown)...)
		values.timestamp = report.UpdatedAt
		values.hasUp = true
		values.totalUp = report.Network.TotalUp
		values.totalUpAt = report.UpdatedAt
		values.hasDown = true
		values.totalDown = report.Network.TotalDown
		values.totalDownAt = report.UpdatedAt
		pendingStates[state] = values
		prepared[i] = report
	}

	if err := s.WriteBatch(ctx, points); err != nil {
		return nil, err
	}

	// Advance counters and timestamps only after success. Retried batches
	// must replace the same samples, not manufacture resets or new timestamps.
	for state, values := range pendingStates {
		state.mu.Lock()
		state.reportTrafficValues = values
		state.mu.Unlock()
	}

	return prepared, nil
}
