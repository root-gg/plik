package utils

import (
	"errors"
	"fmt"
	"time"
)

var Running = errors.New("running")
var Stopped = errors.New("stopped")
var Uninitalized = errors.New("uninitalized")

type SplitTime struct {
	name  string
	start *time.Time
	split *time.Time
	stop  *time.Time
}

func NewSplitTime(name string) (split *SplitTime) {
	split = new(SplitTime)
	split.name = name
	return
}

func (split *SplitTime) Name() string {
	return split.name
}

func (split *SplitTime) Start() {
	if split.start == nil {
		now := time.Now()
		split.start = &now
	}
}

func (split *SplitTime) StartDate() *time.Time {
	return split.start
}

func (split *SplitTime) Split() (elapsed time.Duration) {
	if split.start != nil {
		if split.stop == nil {
			now := time.Now()
			if split.split == nil {
				elapsed = now.Sub(*split.start)
			} else {
				elapsed = now.Sub(*split.split)
			}
			split.split = &now
			return
		}
	}
	return
}

func (split *SplitTime) Stop() {
	if split.stop == nil {
		now := time.Now()
		split.stop = &now
		if split.start == nil {
			split.start = split.stop
		}
	}
}

func (split *SplitTime) StopDate() *time.Time {
	return split.stop
}

func (split *SplitTime) Elapsed() (elapsed time.Duration) {
	if split.start != nil {
		if split.stop == nil {
			elapsed = time.Since(*split.start)
		} else {
			elapsed = split.stop.Sub(*split.start)
		}
	}
	return
}

func (split *SplitTime) Status() error {
	if split.start == nil {
		return Uninitalized
	} else if split.stop == nil {
		return Running
	} else {
		return Stopped
	}
}

func (split *SplitTime) String() string {
	if split.start == nil {
		return fmt.Sprintf("%s : %s", split.name, split.Status())
	} else {
		return fmt.Sprintf("%s : %s : %s", split.name, split.Status(), split.Elapsed())
	}
}
