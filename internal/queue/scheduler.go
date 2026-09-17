package queue

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// deque represents a double-ended queue
type deque struct {
	list *list.List
}

func newDeque() *deque {
	return &deque{list: list.New()}
}

func (d *deque) pushBack(req *QueuedRequest) {
	d.list.PushBack(req)
}

func (d *deque) popFront() *QueuedRequest {
	if d.list.Len() == 0 {
		return nil
	}
	return d.list.Remove(d.list.Front()).(*QueuedRequest)
}

func (d *deque) len() int {
	return d.list.Len()
}

// Scheduler implements Multi-Level Feedback Queue (MLFQ) scheduler
type Scheduler struct {
	mu        sync.Mutex
	queues    map[Priority]*deque
	maxLen    int
	workers   int
	semaphore chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	MaxQueueLength int
	WorkerCount    int
	MaxConcurrency int
}

// DefaultSchedulerConfig returns default scheduler config
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		MaxQueueLength: 1000,
		WorkerCount:    4,
		MaxConcurrency: 10,
	}
}

// NewScheduler creates a new MLFQ scheduler
func NewScheduler(config SchedulerConfig) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Scheduler{
		queues: map[Priority]*deque{
			PriorityVIP:    newDeque(),
			PriorityHigh:   newDeque(),
			PriorityNormal: newDeque(),
			PriorityLow:    newDeque(),
		},
		maxLen:    config.MaxQueueLength,
		workers:   config.WorkerCount,
		semaphore: make(chan struct{}, config.MaxConcurrency),
		ctx:       ctx,
		cancel:    cancel,
	}

	return s
}

// Enqueue adds a request to the queue
func (s *Scheduler) Enqueue(req *QueuedRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check queue length
	totalLen := 0
	for _, q := range s.queues {
		totalLen += q.len()
	}
	if totalLen >= s.maxLen {
		return ErrQueueFull
	}

	req.AddedAt = time.Now()
	req.UpdatedAt = time.Now()

	queue := s.queues[req.Priority]
	if queue == nil {
		queue = s.queues[PriorityNormal]
	}
	queue.pushBack(req)

	return nil
}

// Dequeue gets the next request based on priority
func (s *Scheduler) Dequeue() *QueuedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check queues in priority order (VIP -> High -> Normal -> Low)
	priorities := []Priority{PriorityVIP, PriorityHigh, PriorityNormal, PriorityLow}

	for _, p := range priorities {
		queue := s.queues[p]
		if req := queue.popFront(); req != nil {
			return req
		}
	}

	return nil
}

// StartWorkers starts worker goroutines
func (s *Scheduler) StartWorkers(processor RequestProcessor) {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(processor)
	}
}

// worker processes requests from the queue
func (s *Scheduler) worker(processor RequestProcessor) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			req := s.Dequeue()
			if req == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Acquire semaphore
			s.semaphore <- struct{}{}

			// Process request
			go func(r *QueuedRequest) {
				defer func() { <-s.semaphore }()
				processor.Process(r)
			}(req)
		}
	}
}

// Stop stops all workers
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// GetQueueLength returns total queue length
func (s *Scheduler) GetQueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := 0
	for _, q := range s.queues {
		total += q.len()
	}
	return total
}

// GetQueueStats returns queue statistics
func (s *Scheduler) GetQueueStats() *QueueStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := &QueueStats{
		ByPriority: make(map[Priority]int),
	}

	for p, q := range s.queues {
		stats.ByPriority[p] = q.len()
		stats.TotalLength += q.len()
	}

	stats.SemaphoreAvailable = len(s.semaphore)
	stats.SemaphoreCapacity = cap(s.semaphore)

	return stats
}

// QueueStats holds queue statistics
type QueueStats struct {
	TotalLength        int
	ByPriority         map[Priority]int
	SemaphoreAvailable int
	SemaphoreCapacity  int
}

// RequestProcessor processes queued requests
type RequestProcessor interface {
	Process(req *QueuedRequest)
}

// QueueFullError represents queue full error
type QueueFullError struct{}

func (e *QueueFullError) Error() string {
	return "queue is full"
}
