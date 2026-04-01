package analytics

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrCollectorClosed = errors.New("collector is closed")
	ErrBufferFull      = errors.New("event buffer is full")
	ErrStoreNotSet     = errors.New("store not configured")
)

type Collector interface {
	Record(ctx context.Context, event *Event) error
	Flush(ctx context.Context) error
	Close() error
	IsEnabled() bool
	SetEnabled(enabled bool)
}

type EventCollector struct {
	config   CollectorConfig
	store    AnalyticsStore
	buffer   []*Event
	bufferMu sync.Mutex
	flushCh  chan struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	closed   bool
	closeMu  sync.RWMutex
	rand     *rand.Rand
}

func NewCollector(store AnalyticsStore, config CollectorConfig) *EventCollector {
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultCollectorConfig().BatchSize
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = DefaultCollectorConfig().FlushInterval
	}
	if config.BufferSize <= 0 {
		config.BufferSize = DefaultCollectorConfig().BufferSize
	}
	if config.SampleRate <= 0 {
		config.SampleRate = 1.0
	}
	if config.SampleRate > 1.0 {
		config.SampleRate = 1.0
	}

	c := &EventCollector{
		config:  config,
		store:   store,
		buffer:  make([]*Event, 0, config.BatchSize),
		flushCh: make(chan struct{}, 1),
		done:    make(chan struct{}),
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if config.Enabled {
		c.startFlushLoop()
	}

	return c
}

func (c *EventCollector) startFlushLoop() {
	c.wg.Add(1)
	go c.flushLoop()
}

func (c *EventCollector) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			_ = c.Flush(context.Background())
		case <-c.flushCh:
			_ = c.Flush(context.Background())
		}
	}
}

func (c *EventCollector) Record(ctx context.Context, event *Event) error {
	c.closeMu.RLock()
	defer c.closeMu.RUnlock()

	if c.closed {
		return ErrCollectorClosed
	}

	if !c.config.Enabled {
		return nil
	}

	if c.config.SampleRate < 1.0 {
		if c.rand.Float64() > c.config.SampleRate {
			return nil
		}
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()

	if len(c.buffer) >= c.config.BufferSize {
		return ErrBufferFull
	}

	c.buffer = append(c.buffer, event)

	if len(c.buffer) >= c.config.BatchSize {
		select {
		case c.flushCh <- struct{}{}:
		default:
		}
	}

	return nil
}

func (c *EventCollector) Flush(ctx context.Context) error {
	c.closeMu.RLock()
	defer c.closeMu.RUnlock()

	if c.closed {
		return ErrCollectorClosed
	}

	c.bufferMu.Lock()
	if len(c.buffer) == 0 {
		c.bufferMu.Unlock()
		return nil
	}

	events := c.buffer
	c.buffer = make([]*Event, 0, c.config.BatchSize)
	c.bufferMu.Unlock()

	if c.store == nil {
		return ErrStoreNotSet
	}

	return c.store.StoreEvents(ctx, events)
}

func (c *EventCollector) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.done)

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}

	return c.Flush(context.Background())
}

func (c *EventCollector) IsEnabled() bool {
	return c.config.Enabled
}

func (c *EventCollector) SetEnabled(enabled bool) {
	c.config.Enabled = enabled
}

func (c *EventCollector) GetBufferedCount() int {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()
	return len(c.buffer)
}

type BatchCollector struct {
	*EventCollector
	aggregator *Aggregator
}

func NewBatchCollector(store AnalyticsStore, aggregator *Aggregator, config CollectorConfig) *BatchCollector {
	return &BatchCollector{
		EventCollector: NewCollector(store, config),
		aggregator:     aggregator,
	}
}

func (bc *BatchCollector) RecordWithAggregation(ctx context.Context, event *Event) error {
	if err := bc.Record(ctx, event); err != nil {
		return err
	}

	if bc.aggregator != nil {
		bc.aggregator.RecordEvent(event)
	}

	return nil
}

type AsyncCollector struct {
	*EventCollector
	eventCh chan *Event
}

func NewAsyncCollector(store AnalyticsStore, config CollectorConfig) *AsyncCollector {
	ac := &AsyncCollector{
		EventCollector: NewCollector(store, config),
		eventCh:        make(chan *Event, config.BufferSize),
	}

	ac.wg.Add(1)
	go ac.processLoop()

	return ac
}

func (ac *AsyncCollector) processLoop() {
	defer ac.wg.Done()

	for {
		select {
		case <-ac.done:
			return
		case event := <-ac.eventCh:
			_ = ac.EventCollector.Record(context.Background(), event)
		}
	}
}

func (ac *AsyncCollector) RecordAsync(event *Event) error {
	ac.closeMu.RLock()
	defer ac.closeMu.RUnlock()

	if ac.closed {
		return ErrCollectorClosed
	}

	select {
	case ac.eventCh <- event:
		return nil
	default:
		return ErrBufferFull
	}
}

func (ac *AsyncCollector) Close() error {
	close(ac.eventCh)
	return ac.EventCollector.Close()
}
