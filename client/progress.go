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
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

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

	file.RegisterDoneCallback(func() {
		if file.Error() != nil {
			// Keep only the first line of the error
			str := strings.TrimSuffix(strings.SplitAfterN(file.Error().Error(), "\n", 2)[0], "\n")
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
