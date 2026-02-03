package scheduler

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pranavdwivedi/aegis/pkg/config"
	"github.com/pranavdwivedi/aegis/pkg/crypto"
	"github.com/pranavdwivedi/aegis/pkg/engine"
	"github.com/pranavdwivedi/aegis/pkg/storage"
)

type Scheduler struct {
	cfg     *config.Config
	key     crypto.MasterKey
	repoDir string
}

func New(cfg *config.Config, repoDir string, key crypto.MasterKey) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		repoDir: repoDir,
		key:     key,
	}
}

func (s *Scheduler) Start() {
	var wg sync.WaitGroup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	fmt.Printf("Starting Aegis Daemon with %d jobs...\n", len(s.cfg.Jobs))

	// Create Backend
	var backend storage.Backend
	var err error

	if s.cfg.Storage != nil && s.cfg.Storage.Type == "s3" {
		// Use S3
		fmt.Printf("Using S3 Storage Backend (%s)\n", s.cfg.Storage.Bucket)
		backend, err = storage.NewS3Backend(
			s.cfg.Storage.Endpoint,
			s.cfg.Storage.AccessKey, // Should use Env but Config allows override
			s.cfg.Storage.SecretKey,
			s.cfg.Storage.Bucket,
			s.cfg.Storage.UseSSL,
		)
	} else {
		// Default to Local
		fmt.Println("Using Local Storage Backend")
		backend, err = storage.NewLocalBackend(s.repoDir)
	}
	if err != nil {
		fmt.Printf("[%s] ERROR initializing backend: %v\n", time.Now().Format(time.TimeOnly), err)
		return
	}
	defer backend.Close()

	for _, job := range s.cfg.Jobs {
		wg.Add(1)
		go func(j config.Job) {
			defer wg.Done()
			s.runJobLoop(j, backend, quit)
		}(job)
	}

	<-quit
	fmt.Println("\nShutting down scheduler...")
}

func (s *Scheduler) runJobLoop(job config.Job, backend storage.Backend, quit <-chan os.Signal) {
	interval, err := job.GetDuration()
	if err != nil {
		fmt.Printf("Error parsing interval for job %s: %v\n", job.Name, err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Scheduled: %s every %s\n", job.Name, interval)

	for {
		select {
		case <-ticker.C:
			fmt.Printf("[%s] Starting backup: %s\n", time.Now().Format(time.TimeOnly), job.Name)
			snapshotID, err := engine.Backup(s.repoDir, backend, s.key, job.Path)
			if err != nil {
				fmt.Printf("[%s] ERROR backup %s: %v\n", time.Now().Format(time.TimeOnly), job.Name, err)
			} else {
				fmt.Printf("[%s] SUCCESS %s (Snapshot %d)\n", time.Now().Format(time.TimeOnly), job.Name, snapshotID)
			}
		case <-quit:
			return
		}
	}
}
