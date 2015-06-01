package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	timer := NewSplitTime("main")
	if timer.Name() != "main" {
		t.Errorf("Invalid timer name %s instead of %s", timer.Name(), "main")
	}
}

func TestTimerStatus(t *testing.T) {
	timer := NewSplitTime("timer")
	if timer.Status() != Uninitalized {
		t.Errorf("Invalid timer status %s instead of %s", timer.Status(), Uninitalized)
	}
	timer.Start()
	if timer.Status() != Running {
		t.Errorf("Invalid timer status %s instead of %s", timer.Status(), Running)
	}
	timer.Stop()
	if timer.Status() != Stopped {
		t.Errorf("Invalid timer status %s instead of %s", timer.Status(), Stopped)
	}
}

func TestTimerDates(t *testing.T) {
	timer := NewSplitTime("timer")
	if timer.StartDate() != nil {
		t.Error("Start date on uninitalized timer : %s", timer.StartDate().String())
	}
	if timer.StopDate() != nil {
		t.Error("Stop date on uninitalized timer : %s", timer.StopDate().String())
	}
	timer.Start()
	if timer.StartDate() == nil {
		t.Error("Missing start date on running timer")
	}
	if timer.StopDate() != nil {
		t.Error("Stop date on running timer : %s", timer.StopDate().String())
	}
	timer.Stop()
	if timer.StartDate() == nil {
		t.Error("Missing start date on stopped timer")
	}
	if timer.StopDate() == nil {
		t.Error("Missing stop date on stopped timer")
	}
}

func TestTimerImmutability(t *testing.T) {
	timer := NewSplitTime("timer")
	timer.Start()
	startDate1 := timer.StartDate()
	timer.Start()
	startDate2 := timer.StartDate()
	if startDate1 != startDate2 {
		t.Errorf("Non immutable start date : %s != %s", startDate1.String(), startDate2.String())
	}
	timer.Stop()
	stopDate1 := timer.StopDate()
	timer.Stop()
	stopDate2 := timer.StopDate()
	if stopDate1 != stopDate2 {
		t.Errorf("Non immutable stop date : %s != %s", stopDate1.String(), stopDate2.String())
	}
	timer.Start()
	if timer.Status() != Stopped {
		t.Errorf("Non immutable timer status %s instead of %s", timer.Status(), Stopped)
	}
	startDate3 := timer.StartDate()
	if startDate1 != startDate3 {
		t.Errorf("Non immutable start date : %s != %s", startDate1.String(), startDate3.String())
	}
}

func TestTimerElapsed(t *testing.T) {
	timer := NewSplitTime("timer")
	if timer.Elapsed() != time.Duration(0) {
		t.Errorf("Invalid uninitialized timer elapsed time %s", timer.Elapsed().String())
	}
	timer.Start()
	if timer.Elapsed() <= time.Duration(0) {
		t.Errorf("Invalid running timer elapsed time %s", timer.Elapsed().String())
	}
	timer.Stop()
	if timer.Elapsed() <= time.Duration(0) {
		t.Errorf("Invalid stopped timer elapsed time %s", timer.Elapsed().String())
	}
}

func TestTimerStopUninitalizedTimer(t *testing.T) {
	timer := NewSplitTime("timer")
	timer.Stop()
	if timer.Status() != Stopped {
		t.Errorf("Invalid timer status %s instead of %s", timer.Status(), Stopped)
	}
	if timer.Elapsed() != time.Duration(0) {
		t.Errorf("Invalid uninitialized stopped timer elapsed time %s", timer.Elapsed().String())
	}
}

func TestTimerString(t *testing.T) {
	timer := NewSplitTime("timer")
	fmt.Println(timer.String())
	timer.Start()
	fmt.Println(timer.String())
	timer.Stop()
	fmt.Println(timer.String())
}
