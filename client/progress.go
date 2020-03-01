package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/root-gg/plik/server/common"

	"github.com/camathieu/pb"

	"github.com/root-gg/plik/plik"
)

// Progress manage the progress bar pool
type Progress struct {
	bars []*pb.ProgressBar
	pool *pb.Pool

	prefixLength int

	mu   sync.Mutex
	once sync.Once
}

// NewProgress creates a progress bar pool
func NewProgress(files []*plik.File) (p *Progress) {
	p = new(Progress)

	for _, file := range files {
		if p.prefixLength < len(file.Name) {
			p.prefixLength = len(file.Name)
		}
	}

	return p
}

// Register a new progress bar
func (p *Progress) register(file *plik.File) {
	// Upload progress ( Size, Progressbar, Transfer Speed, Elapsed Time,... )
	linePrefix := fmt.Sprintf("%-"+strconv.Itoa(p.prefixLength)+"s : ", file.Name)

	bar := pb.New64(file.Size).SetUnits(pb.U_BYTES).Prefix(linePrefix)
	bar.Prefix(linePrefix)
	bar.ShowSpeed = true
	bar.ShowFinalTime = true
	bar.SetWidth(100)
	bar.SetMaxWidth(100)

	// Add the current progress bar to the progress bar pool
	p.mu.Lock()
	p.bars = append(p.bars, bar)
	p.mu.Unlock()

	file.WrapReader(func(fileReader io.ReadCloser) io.ReadCloser {
		return ioutil.NopCloser(io.TeeReader(fileReader, bar))
	})

	file.RegisterUploadCallback(func(metadata *common.File, err error) {
		if err != nil {
			// Keep only the first line of the error
			str := strings.TrimSuffix(strings.SplitAfterN(err.Error(), "\n", 2)[0], "\n")
			bar.FinishError(errors.New(str))
		} else {
			bar.Finish()
		}
	})
}

// Start the progress bar pool
func (p *Progress) start() {
	once := func() {
		p.pool, err = pb.StartPool(p.bars...)
		if err != nil {
			panic(err)
		}
	}
	p.once.Do(once)
}

// Stop the progress bar pool
func (p *Progress) stop() {
	if p.pool != nil {
		p.pool.Stop()
	}
}
