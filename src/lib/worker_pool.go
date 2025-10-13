package lib

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool represents a dynamic worker pool with load balancing
type WorkerPool struct {
	workers      []*Worker
	workQueue    chan WorkItem
	resultQueue  chan WorkResult
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	maxWorkers   int
	minWorkers   int
	activeCount  int64
	totalWork    int64
	completedWork int64
	errorCount   int64
	mu           sync.RWMutex
	stats        WorkerPoolStats
	options      PoolOptions
}

// Worker represents a single worker goroutine
type Worker struct {
	id          int
	pool        *WorkerPool
	workQueue   <-chan WorkItem
	resultQueue chan<- WorkResult
	ctx         context.Context
	healthCheck time.Duration
	quit        chan bool
}

// WorkItem represents a unit of work to be processed
type WorkItem struct {
	ID       int
	Task     func() (interface{}, error)
	Priority int
	Timeout  time.Duration
	Metadata map[string]interface{}
}

// WorkResult represents the result of processing a work item
type WorkResult struct {
	WorkID     int
	Result     interface{}
	Error      error
	Duration   time.Duration
	WorkerID   int
	StartTime  time.Time
	EndTime    time.Time
	Metadata   map[string]interface{}
}

// PoolOptions contains configuration options for the worker pool
type PoolOptions struct {
	MinWorkers      int           `json:"min_workers"`
	MaxWorkers      int           `json:"max_workers"`
	QueueSize       int           `json:"queue_size"`
	HealthCheck     time.Duration `json:"health_check"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	MaxRetries      int           `json:"max_retries"`
	EnableMetrics   bool          `json:"enable_metrics"`
	LoadBalanceMode string        `json:"load_balance_mode"` // "round_robin", "least_loaded", "random"
}

// WorkerPoolStats contains statistics about the worker pool
type WorkerPoolStats struct {
	TotalWorkers     int64         `json:"total_workers"`
	ActiveWorkers    int64         `json:"active_workers"`
	QueuedWork       int64         `json:"queued_work"`
	CompletedWork    int64         `json:"completed_work"`
	ErrorCount       int64         `json:"error_count"`
	AverageWaitTime  time.Duration `json:"average_wait_time"`
	AverageWorkTime  time.Duration `json:"average_work_time"`
	ThroughputPerSec float64       `json:"throughput_per_sec"`
	LastUpdate       time.Time     `json:"last_update"`
}

// DefaultPoolOptions returns default options for the worker pool
func DefaultPoolOptions() PoolOptions {
	cpuCount := runtime.NumCPU()

	return PoolOptions{
		MinWorkers:      max(1, cpuCount/2),
		MaxWorkers:      max(2, cpuCount*2),
		QueueSize:       1000,
		HealthCheck:     30 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxRetries:      3,
		EnableMetrics:   true,
		LoadBalanceMode: "least_loaded",
	}
}

// NewWorkerPool creates a new dynamic worker pool
func NewWorkerPool(options PoolOptions) *WorkerPool {
	if options.MinWorkers == 0 {
		options.MinWorkers = 1
	}
	if options.MaxWorkers == 0 {
		options.MaxWorkers = runtime.NumCPU() * 2
	}
	if options.QueueSize == 0 {
		options.QueueSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workQueue:   make(chan WorkItem, options.QueueSize),
		resultQueue: make(chan WorkResult, options.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		minWorkers:  options.MinWorkers,
		maxWorkers:  options.MaxWorkers,
		options:     options,
		stats: WorkerPoolStats{
			LastUpdate: time.Now(),
		},
	}

	// Start initial workers
	for i := 0; i < options.MinWorkers; i++ {
		pool.addWorker()
	}

	// Start monitoring goroutine
	go pool.monitor()

	return pool
}

// Submit submits work to the pool
func (p *WorkerPool) Submit(task func() (interface{}, error)) *WorkFuture {
	return p.SubmitWithOptions(task, 0, 0, nil)
}

// SubmitWithOptions submits work with custom options
func (p *WorkerPool) SubmitWithOptions(task func() (interface{}, error), priority int, timeout time.Duration, metadata map[string]interface{}) *WorkFuture {
	workID := int(atomic.AddInt64(&p.totalWork, 1))

	workItem := WorkItem{
		ID:       workID,
		Task:     task,
		Priority: priority,
		Timeout:  timeout,
		Metadata: metadata,
	}

	future := &WorkFuture{
		workID: workID,
		done:   make(chan struct{}),
	}

	select {
	case p.workQueue <- workItem:
		atomic.AddInt64(&p.stats.QueuedWork, 1)
	case <-p.ctx.Done():
		future.error = p.ctx.Err()
		close(future.done)
	default:
		// Queue is full
		future.error = ErrQueueFull
		close(future.done)
	}

	return future
}

// SubmitBatch submits multiple work items
func (p *WorkerPool) SubmitBatch(tasks []func() (interface{}, error)) []*WorkFuture {
	futures := make([]*WorkFuture, len(tasks))

	for i, task := range tasks {
		futures[i] = p.Submit(task)
	}

	return futures
}

// ProcessAndWait processes work and waits for completion
func (p *WorkerPool) ProcessAndWait(tasks []func() (interface{}, error)) ([]interface{}, []error) {
	futures := p.SubmitBatch(tasks)

	results := make([]interface{}, len(futures))
	errors := make([]error, len(futures))

	for i, future := range futures {
		results[i], errors[i] = future.Get()
	}

	return results, errors
}

// WorkFuture represents a future result for a work item
type WorkFuture struct {
	workID int
	result interface{}
	error  error
	done   chan struct{}
	mu     sync.RWMutex
}

// Get waits for and returns the result
func (f *WorkFuture) Get() (interface{}, error) {
	<-f.done
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.result, f.error
}

// Done returns a channel that's closed when the work is complete
func (f *WorkFuture) Done() <-chan struct{} {
	return f.done
}

// setResult sets the result of the work item
func (f *WorkFuture) setResult(result interface{}, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.result = result
	f.error = err
	close(f.done)
}

// addWorker adds a new worker to the pool
func (p *WorkerPool) addWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) >= p.maxWorkers {
		return
	}

	workerID := len(p.workers)
	worker := &Worker{
		id:          workerID,
		pool:        p,
		workQueue:   p.workQueue,
		resultQueue: p.resultQueue,
		ctx:         p.ctx,
		healthCheck: p.options.HealthCheck,
		quit:        make(chan bool),
	}

	p.workers = append(p.workers, worker)
	atomic.AddInt64(&p.stats.TotalWorkers, 1)

	p.wg.Add(1)
	go worker.start()
}

// removeWorker removes a worker from the pool
func (p *WorkerPool) removeWorker(workerID int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, worker := range p.workers {
		if worker.id == workerID {
			close(worker.quit)
			p.workers = append(p.workers[:i], p.workers[i+1:]...)
			atomic.AddInt64(&p.stats.TotalWorkers, -1)
			break
		}
	}
}

// start starts the worker's main loop
func (w *Worker) start() {
	defer w.pool.wg.Done()

	atomic.AddInt64(&w.pool.activeCount, 1)
	defer atomic.AddInt64(&w.pool.activeCount, -1)

	for {
		select {
		case workItem := <-w.workQueue:
			atomic.AddInt64(&w.pool.stats.ActiveWorkers, 1)
			result := w.processWorkItem(workItem)
			atomic.AddInt64(&w.pool.stats.ActiveWorkers, -1)

			select {
			case w.resultQueue <- result:
			case <-w.ctx.Done():
				return
			}

		case <-w.quit:
			return

		case <-w.ctx.Done():
			return
		}
	}
}

// processWorkItem processes a single work item
func (w *Worker) processWorkItem(workItem WorkItem) WorkResult {
	startTime := time.Now()

	result := WorkResult{
		WorkID:    workItem.ID,
		WorkerID:  w.id,
		StartTime: startTime,
		Metadata:  workItem.Metadata,
	}

	// Handle timeout if specified
	if workItem.Timeout > 0 {
		ctx, cancel := context.WithTimeout(w.ctx, workItem.Timeout)
		defer cancel()

		done := make(chan struct{})
		var taskResult interface{}
		var taskError error

		go func() {
			defer close(done)
			taskResult, taskError = workItem.Task()
		}()

		select {
		case <-done:
			result.Result, result.Error = taskResult, taskError
		case <-ctx.Done():
			result.Error = ErrTaskTimeout
		}
	} else {
		result.Result, result.Error = workItem.Task()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Update statistics
	atomic.AddInt64(&w.pool.completedWork, 1)
	atomic.AddInt64(&w.pool.stats.CompletedWork, 1)

	if result.Error != nil {
		atomic.AddInt64(&w.pool.errorCount, 1)
		atomic.AddInt64(&w.pool.stats.ErrorCount, 1)
	}

	return result
}

// monitor monitors the pool and adjusts worker count based on load
func (p *WorkerPool) monitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	resultProcessor := time.NewTicker(100 * time.Millisecond)
	defer resultProcessor.Stop()

	for {
		select {
		case <-ticker.C:
			p.adjustWorkerCount()
			p.updateStats()

		case result := <-p.resultQueue:
			// Process results (this could be extended with result callbacks)
			_ = result

		case <-p.ctx.Done():
			return
		}
	}
}

// adjustWorkerCount dynamically adjusts the number of workers based on load
func (p *WorkerPool) adjustWorkerCount() {
	p.mu.RLock()
	currentWorkers := len(p.workers)
	queueLength := len(p.workQueue)
	p.mu.RUnlock()

	// Calculate optimal worker count based on queue length and CPU usage
	targetWorkers := currentWorkers

	if queueLength > 10 && currentWorkers < p.maxWorkers {
		// Add workers if queue is backing up
		targetWorkers = min(currentWorkers+2, p.maxWorkers)
	} else if queueLength == 0 && currentWorkers > p.minWorkers {
		// Remove idle workers
		targetWorkers = max(currentWorkers-1, p.minWorkers)
	}

	// Adjust worker count
	if targetWorkers > currentWorkers {
		for i := currentWorkers; i < targetWorkers; i++ {
			p.addWorker()
		}
	} else if targetWorkers < currentWorkers {
		// Remove the most recently added workers
		for i := currentWorkers; i > targetWorkers; i-- {
			if len(p.workers) > 0 {
				workerToRemove := p.workers[len(p.workers)-1]
				p.removeWorker(workerToRemove.id)
			}
		}
	}
}

// updateStats updates the pool statistics
func (p *WorkerPool) updateStats() {
	if !p.options.EnableMetrics {
		return
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.ActiveWorkers = atomic.LoadInt64(&p.activeCount)
	p.stats.QueuedWork = int64(len(p.workQueue))
	p.stats.CompletedWork = atomic.LoadInt64(&p.completedWork)
	p.stats.LastUpdate = time.Now()

	// Calculate throughput
	if p.stats.CompletedWork > 0 {
		elapsed := time.Since(p.stats.LastUpdate)
		if elapsed > 0 {
			p.stats.ThroughputPerSec = float64(p.stats.CompletedWork) / elapsed.Seconds()
		}
	}
}

// GetStats returns the current pool statistics
func (p *WorkerPool) GetStats() WorkerPoolStats {
	p.updateStats()
	return p.stats
}

// Stop gracefully shuts down the worker pool
func (p *WorkerPool) Stop(timeout time.Duration) error {
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}

// Close immediately closes the worker pool
func (p *WorkerPool) Close() {
	p.cancel()
	p.wg.Wait()
}

// Resize changes the pool size limits
func (p *WorkerPool) Resize(minWorkers, maxWorkers int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.minWorkers = max(1, minWorkers)
	p.maxWorkers = max(p.minWorkers, maxWorkers)
}

// Errors
var (
	ErrQueueFull       = &PoolError{"work queue is full"}
	ErrTaskTimeout     = &PoolError{"task timed out"}
	ErrShutdownTimeout = &PoolError{"shutdown timeout exceeded"}
	ErrPoolClosed      = &PoolError{"worker pool is closed"}
)

// PoolError represents a worker pool error
type PoolError struct {
	message string
}

func (e *PoolError) Error() string {
	return e.message
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}