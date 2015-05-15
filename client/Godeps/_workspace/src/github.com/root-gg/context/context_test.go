package context

import (
	"errors"
	"fmt"
	"github.com/root-gg/plik/client/Godeps/_workspace/src/github.com/root-gg/utils"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1")
	child.Fork("fork2")
	if child.Status() != Running {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), Running)
	}
	child.Finalize(nil)
	if child.Status() != Success {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), Success)
	}
	children := root.AllChildren()
	if len(children) != 2 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 2)
	}
}

func TestDefaultName(t *testing.T) {
	root := NewContext("")
	defaultName := "TestDefaultName"
	if root.Name() != defaultName {
		t.Errorf("Invalid child default name %s instead of %s", root.Name, defaultName)
	}
	child := root.Fork("")
	if child.Name() != defaultName {
		t.Errorf("Invalid child default name %s instead of %s", child.Name, defaultName)
	}
}

func TestDates(t *testing.T) {
	root := NewContext("ROOT")
	fmt.Printf("StartDate : %s\n", root.StartDate().String())
	fmt.Printf("Running since : %s\n", root.Elapsed().String())
	if root.EndDate() != nil {
		t.Error("EndDate on running context")
	}
	root.Finalize(Success)
	fmt.Printf("EndDate : %s\n", root.StartDate().String())
	fmt.Printf("Has run : %s\n", root.Elapsed().String())
}

func TestTimers(t *testing.T) {
	root := NewContext("ROOT")
	root.Time("t1").Stop()
	root.Time("t2")
	timers := root.Timers()
	if len(timers) != 2 {
		t.Errorf("Invalid timer count %d instead of %d", len(root.Timers()), 2)
	}
	if timers[0].Status() != utils.Stopped {
		t.Errorf("Invalid timer %s status %s instead of %s", timers[0].Name(), timers[0].Status(), utils.Stopped)
	}
	if timers[1].Status() != utils.Running {
		t.Errorf("Invalid timer %s status %s instead of %s", timers[1].Name(), timers[1].Status(), utils.Running)
	}
}

func TestFinalize(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1")
	go func() { child.Finalize(Success) }()
	child.Wait()
	if child.Status() != Success {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), Success)
	}
}

func TestWaitAllChildren(t *testing.T) {
	root := NewContext("ROOT")
	child1 := root.Fork("fork1")
	child2 := child1.Fork("fork2")
	child3 := child2.Fork("fork3")
	go func() {
		time.Sleep(100 * time.Millisecond)
		child1.Finalize(Success)
		time.Sleep(100 * time.Millisecond)
		child3.Finalize(Success)
		time.Sleep(100 * time.Millisecond)
		child2.Finalize(Success)
	}()
	root.WaitAllChildren()
	children := root.AllChildren()
	if len(children) != 3 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 3)
	}
	for _, child := range children {
		if child.Status() != Success {
			t.Errorf("Invalid child status %s instead of %s", child.Status(), Timedout)
		}
	}
}

func TestStatusOverride(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1")
	var err = errors.New("error")
	go func() { child.Finalize(err) }()
	child.Wait()
	child.Finalize(Success)
	if child.Status() != err {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), err)
	}
}

func TestTimeoutOk(t *testing.T) {
	root := NewContext("ROOT")
	child := root.ForkWithTimeout("", 200*time.Millisecond)
	go func() {
		time.Sleep(100 * time.Millisecond)
		child.Finalize(Success)
	}()
	<-child.Done()
	if child.Status() != Success {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), Success)
	}
}

func TestTimeoutKo(t *testing.T) {
	root := NewContext("ROOT")
	child := root.ForkWithTimeout("", 100*time.Millisecond)
	go func() {
		time.Sleep(200 * time.Millisecond)
		child.Finalize(Success)
	}()
	<-child.Done()
	if child.Status() != Timedout {
		t.Errorf("Invalid child status %s instead of %s", child.Status(), Timedout)
	}
}

func TestTimeoutDates(t *testing.T) {
	root := NewContextWithTimeout("", 100*time.Millisecond)
	fmt.Printf("Deadline is : %s\n", root.Deadline().String())
	fmt.Printf("Remaining time : %s\n", root.Remaining().String())
	child := root.Fork("")
	if child.Deadline() != *child.StartDate() {
		t.Errorf("Invalid deadline for non timed context : %s\n", child.Deadline().String())
	}
	if child.Remaining().Seconds() > 0 {
		t.Errorf("Invalid remaining for non timed context : %s\n", child.Remaining().String())
	}
}

func TestCancel(t *testing.T) {
	root := NewContext("ROOT")
	root.Fork("").Fork("").Fork("")
	root.Cancel()
	children := root.AllChildren()
	if len(children) != 3 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 3)
	}
	for _, child := range children {
		if child.Status() != Canceled {
			t.Errorf("Invalid child status %s instead of %s", child.Status(), Timedout)
		}
	}
}

func TestAutoCancel(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1").AutoCancel()
	child.Fork("").Fork("").Fork("")
	child.Finalize(Success)
	time.Sleep(100 * time.Millisecond)
	children := child.AllChildren()
	if len(children) != 3 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 3)
	}
	for _, child := range children {
		if child.Status() != Canceled {
			t.Errorf("Invalid child status %s instead of %s", child.Status(), Canceled)
		}
	}
}

func TestDetach(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("")
	child.Fork("").Fork("")
	children := root.AllChildren()
	if len(children) != 3 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 3)
	}
	root.DetachChild(child)
	children = root.AllChildren()
	if len(children) != 0 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 1)
	}
}

func TestAutoDetach(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1").AutoDetach()
	children := root.AllChildren()
	if len(children) != 1 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 1)
	}
	child.Finalize(Success)
	time.Sleep(100 * time.Millisecond)
	children = root.AllChildren()
	if len(children) != 0 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 0)
	}
}

func TestDetachChild(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1")
	children := root.AllChildren()
	if len(children) != 1 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 1)
	}
	root.DetachChild(child)
	children = root.AllChildren()
	if len(children) != 0 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 0)
	}
}

func TestAutoDetachChild(t *testing.T) {
	root := NewContext("ROOT")
	child := root.Fork("fork1")
	root.AutoDetachChild(child)
	children := root.AllChildren()
	if len(children) != 1 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 1)
	}
	child.Finalize(Success)
	time.Sleep(100 * time.Millisecond)
	children = root.AllChildren()
	if len(children) != 0 {
		t.Errorf("Invalid childen count %d instead of %d", len(children), 0)
	}
}

func TestValue(t *testing.T) {
	root := NewContext("ROOT")
	root.Set("foo", "bar")
	value, ok := root.Get("foo")
	if !ok {
		t.Error("Missing value for key \"foo\"")
	}
	if value.(string) != "bar" {
		t.Error("Invalid value \"%s\" for key \"foo\" sould be \"bar\"", value)
	}
	child := root.Fork("fork1")
	value, ok = child.Get("foo")
	if !ok {
		t.Error("Missing value for key \"foo\" in child context")
	}
	if value.(string) != "bar" {
		t.Error("Invalid value \"%s\" for key \"foo\" child context sould be \"bar\"", value)
	}
}

func TestMissingValue(t *testing.T) {
	root := NewContext("ROOT")
	root.Set("go", "lang")
	child := root.Fork("scala")
	child.Set("sca", "la")
	child2 := child.Fork("java")
	child2.Get("ja")
	value, ok := child.Get("foo")
	if ok {
		t.Error("Missing key \"ja\" should be missing")
	}
	if value != nil {
		t.Error("Missing value \"%s\" for key \"foo\" should be missing", value)
	}
}

func TestValueOverride(t *testing.T) {
	root := NewContext("ROOT")
	root.Set("foo", "bar")
	child := root.Fork("")
	child.Set("foo", "baz")
	value, ok := root.Get("foo")
	if !ok {
		t.Error("Missing value for key foo")
	}
	if value.(string) != "bar" {
		t.Error("Invalid value \"%s\" for key foo sould be \"bar\"", value)
	}
	value, ok = child.Get("foo")
	if !ok {
		t.Error("Missing value for key foo in child context")
	}
	if value.(string) != "baz" {
		t.Error("Invalid value \"%s\" for key foo child context sould be \"baz\"", value)
	}
}

func TestDisplay(t *testing.T) {
	root := NewContext("ROOT")
	fork1 := root.Fork("fork1")
	fork1.Fork("fork11")
	fork1.Fork("fork12").Fork("fork121")
	fork1.Finalize(Success)
	fork1.Cancel()
	fork2 := root.Fork("fork2")
	fork2.Fork("fork21")
	fork2.Fork("fork22").Fork("fork221")
	fork2.Time("t1").Stop()
	fork2.Time("t2").Stop()
	fork2.Time("t3")
	fmt.Println(root.String())
}
