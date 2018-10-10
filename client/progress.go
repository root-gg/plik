/**

    Plik upload client

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/camathieu/pb"

	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/plik/server/common"
)

// Progress manage the progress bar pool
type Progress struct {
	bars []*pb.ProgressBar
	pool *pb.Pool

	mu   sync.Mutex
	wg   sync.WaitGroup
	once sync.Once
}

// Create the progress bar pool
func newProgress(uploadInfo *common.Upload) (p *Progress) {
	p = new(Progress)

	if !config.Config.Quiet {
		p.wg.Add(len(uploadInfo.Files))
	}

	return p
}

// Register a new progress bar
func (p *Progress) register(fileToUpload *config.FileToUpload, writer io.Writer) (multiWriter io.Writer, done func(error)) {
	if config.Config.Quiet {
		// Noop
		return writer, func(error) {}
	}

	// Upload progress ( Size, Progressbar, Transfer Speed, Elapsed Time,... )
	linePrefix := fmt.Sprintf("%-"+strconv.Itoa(config.GetLongestFilename())+"s : ", fileToUpload.Name)
	bar := pb.New64(fileToUpload.CurrentSize).SetUnits(pb.U_BYTES).Prefix(linePrefix)
	bar.Prefix(linePrefix)
	bar.ShowSpeed = true
	bar.ShowFinalTime = true
	bar.SetWidth(100)
	bar.SetMaxWidth(100)

	// Add the current progress bar to the progress bar pool
	p.mu.Lock()
	p.bars = append(p.bars, bar)
	p.mu.Unlock()

	// Write to the progress bar and to the multipart MIME writer
	multiWriter = io.MultiWriter(writer, bar)

	// Callback to call when the upload is finished or encounters an error
	done = func(err error) {
		if err != nil {
			// Keep only the first line of the error
			str := strings.TrimSuffix(strings.SplitAfterN(err.Error(), "\n", 2)[0], "\n")
			bar.FinishError(errors.New(str))
		} else {
			bar.Finish()
		}
	}

	// Wait for all progress bars to be initialized
	p.wg.Done()
	p.wg.Wait()

	// Start the progress bar pool
	p.start()

	return multiWriter, done
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
