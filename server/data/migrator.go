package data

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
)

// MigrateOptions controls the behavior of MigrateFiles
type MigrateOptions struct {
	IgnoreErrors bool
	Workers      int
	DryRun       bool
}

// MigrateStats holds counters returned by MigrateFiles
type MigrateStats struct {
	Copied  int64
	Skipped int64
	Errors  int64
	Bytes   int64
}

// MigrateFiles copies file blobs for all uploaded/removed files from src to dst.
// It uses a configurable worker pool (default: 4) for parallel transfers.
// Files with status missing, uploading, or deleted are skipped (no data in backend).
func MigrateFiles(src Backend, dst Backend, metaBackend *metadata.Backend, options *MigrateOptions) (stats MigrateStats, err error) {
	if options == nil {
		options = &MigrateOptions{}
	}
	workers := options.Workers
	if workers <= 0 {
		workers = 4
	}

	type job struct {
		file *common.File
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan job, workers*2)
	var wg sync.WaitGroup

	var copied, skipped, errCount, byteCount int64

	// fatalErr holds the first non-ignorable worker error.
	var fatalErr error
	var fatalMu sync.Mutex

	workerFn := func() {
		defer wg.Done()
		for j := range jobs {
			file := j.file

			// Only copy files that actually have data in the backend
			if file.Status != common.FileUploaded && file.Status != common.FileRemoved {
				atomic.AddInt64(&skipped, 1)
				continue
			}

			if options.DryRun {
				fmt.Printf("[dry-run] would copy file %s (%s, %d bytes, status: %s)\n",
					file.ID, file.Name, file.Size, file.Status)
				atomic.AddInt64(&copied, 1)
				atomic.AddInt64(&byteCount, file.Size)
				continue
			}

			reader, e := src.GetFile(file)
			if e != nil {
				atomic.AddInt64(&errCount, 1)
				msg := fmt.Sprintf("error reading file %s (%s): %s", file.ID, file.Name, e)
				if options.IgnoreErrors {
					fmt.Println(msg)
					continue
				}
				fatalMu.Lock()
				if fatalErr == nil {
					fatalErr = fmt.Errorf("%s", msg)
				}
				fatalMu.Unlock()
				cancel()
				return
			}

			// Caller (us) is responsible for closing the reader — AddFile does not close it.
			e = dst.AddFile(file, reader)
			_ = reader.Close()
			if e != nil {
				atomic.AddInt64(&errCount, 1)
				msg := fmt.Sprintf("error writing file %s (%s): %s", file.ID, file.Name, e)
				if options.IgnoreErrors {
					fmt.Println(msg)
					continue
				}
				fatalMu.Lock()
				if fatalErr == nil {
					fatalErr = fmt.Errorf("%s", msg)
				}
				fatalMu.Unlock()
				cancel()
				return
			}

			atomic.AddInt64(&copied, 1)
			atomic.AddInt64(&byteCount, file.Size)
		}
	}

	for range workers {
		wg.Add(1)
		go workerFn()
	}

	// Enumerate all files and feed to workers; stop early if a fatal error is signalled.
	iterErr := metaBackend.ForEachFile(func(file *common.File) error {
		select {
		case <-ctx.Done():
			fatalMu.Lock()
			e := fatalErr
			fatalMu.Unlock()
			return e
		case jobs <- job{file: file}:
			return nil
		}
	})

	close(jobs)
	wg.Wait()

	stats = MigrateStats{
		Copied:  atomic.LoadInt64(&copied),
		Skipped: atomic.LoadInt64(&skipped),
		Errors:  atomic.LoadInt64(&errCount),
		Bytes:   atomic.LoadInt64(&byteCount),
	}

	if iterErr != nil {
		return stats, iterErr
	}
	fatalMu.Lock()
	e := fatalErr
	fatalMu.Unlock()
	if e != nil {
		return stats, e
	}
	return stats, nil
}
