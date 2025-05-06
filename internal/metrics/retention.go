package metrics

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// RetentionManager handles scheduled pruning of the metrics database
type RetentionManager struct {
	dao       *DAO
	config    RetentionConfig
	ticker    *time.Ticker
	stopCh    chan struct{}
	wg        sync.WaitGroup
	running   bool
	runningMu sync.Mutex
}

// NewRetentionManager creates a new retention manager with the given configuration
func NewRetentionManager(dao *DAO, config RetentionConfig) *RetentionManager {
	return &RetentionManager{
		dao:    dao,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the scheduled pruning process
func (r *RetentionManager) Start() error {
	r.runningMu.Lock()
	defer r.runningMu.Unlock()

	if r.running {
		return fmt.Errorf("retention manager is already running")
	}

	if !r.config.Enabled {
		log.Println("Metrics retention manager is disabled")
		return nil
	}

	// Run an initial pruning immediately
	go r.runPruning()

	// Set up regular pruning
	r.ticker = time.NewTicker(r.config.PruneInterval)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.ticker.C:
				go r.runPruning()
			case <-r.stopCh:
				return
			}
		}
	}()

	r.running = true
	log.Printf("Metrics retention manager started with interval %s", r.config.PruneInterval)
	return nil
}

// Stop terminates the scheduled pruning process
func (r *RetentionManager) Stop() {
	r.runningMu.Lock()
	defer r.runningMu.Unlock()

	if !r.running {
		return
	}

	r.ticker.Stop()
	close(r.stopCh)
	r.wg.Wait()
	r.running = false
	log.Println("Metrics retention manager stopped")
}

// UpdateConfig updates the retention configuration
func (r *RetentionManager) UpdateConfig(config RetentionConfig) error {
	r.runningMu.Lock()

	// If currently running, stop first
	if r.running {
		r.ticker.Stop()
		close(r.stopCh)
		r.wg.Wait()
		r.running = false
	}

	// Apply new config
	r.config = config
	r.stopCh = make(chan struct{})

	shouldStart := config.Enabled // decide after lock
	r.runningMu.Unlock()

	// Start outside the lock to avoid deadlock
	if shouldStart {
		return r.Start()
	}
	return nil
}

// RunPruningNow triggers an immediate pruning operation
// This is useful for manual pruning or testing
func (r *RetentionManager) RunPruningNow() (map[string]int64, error) {
	return r.dao.PruneDatabase(r.config)
}

// runPruning performs the actual database pruning operation
func (r *RetentionManager) runPruning() {
	start := time.Now()
	results, err := r.dao.PruneDatabase(r.config)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Error during scheduled metrics pruning: %v", err)
		return
	}

	var totalPruned int64
	for _, count := range results {
		totalPruned += count
	}

	if totalPruned > 0 {
		log.Printf("Metrics pruning complete in %s: removed %d records (time: %d, count: %d, size: %d)",
			duration,
			totalPruned,
			results["time_based"],
			results["count_based"],
			results["size_based"])
	}
}
