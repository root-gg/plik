package context

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"sync"
	"time"
)

var Running = errors.New("running")
var Success = errors.New("success")
var Canceled = errors.New("canceled")
var Timedout = errors.New("timedout")

type Context struct {
	parent   *Context
	name     string
	elapsed  utils.SplitTime
	splits   []*utils.SplitTime
	done     chan struct{}
	children []*Context
	timeout  time.Duration
	timer    *time.Timer
	status   error
	lock     sync.RWMutex
	values   map[interface{}]interface{}
}

func NewContext(name string) (ctx *Context) {
	if name == "" {
		_, _, name = utils.GetCaller(2)
		_, name = utils.ParseFunction(name)
	}
	ctx = new(Context)
	ctx.status = Running
	ctx.elapsed = *utils.NewSplitTime("")
	ctx.elapsed.Start()
	ctx.name = name
	ctx.done = make(chan struct{})
	ctx.children = make([]*Context, 0)
	ctx.values = make(map[interface{}]interface{})
	return
}

func NewContextWithTimeout(name string, timeout time.Duration) (ctx *Context) {
	if name == "" {
		_, _, name = utils.GetCaller(2)
		_, name = utils.ParseFunction(name)
	}
	ctx = NewContext(name)
	ctx.timeout = timeout
	ctx.timer = time.NewTimer(timeout)
	go func() {
		select {
		case <-ctx.timer.C:
			ctx.Finalize(Timedout)
		case <-ctx.Done():
			ctx.timer.Stop()
		}
	}()
	return
}

func (ctx *Context) Fork(name string) (fork *Context) {
	if name == "" {
		_, _, name = utils.GetCaller(2)
		_, name = utils.ParseFunction(name)
	}
	fork = NewContext(name)
	fork.parent = ctx
	ctx.children = append(ctx.children, fork)
	return
}

func (ctx *Context) ForkWithTimeout(name string, timeout time.Duration) (fork *Context) {
	if name == "" {
		_, _, name = utils.GetCaller(2)
		_, name = utils.ParseFunction(name)
	}
	fork = NewContextWithTimeout(name, timeout)
	fork.parent = ctx
	ctx.children = append(ctx.children, fork)
	return
}

func (ctx *Context) Name() string {
	return ctx.name
}

func (ctx *Context) Done() (done <-chan struct{}) {
	done = ctx.done
	return
}

func (ctx *Context) Wait() {
	if ctx.status == Running {
		<-ctx.done
	}
}

func (ctx *Context) waitAllChildren(root bool) {
	for _, child := range ctx.children {
		child.waitAllChildren(false)
	}
	if !root {
		ctx.Wait()
	}
}

func (ctx *Context) WaitAllChildren() {
	ctx.waitAllChildren(true)
	return
}

func (ctx *Context) Status() (status error) {
	if ctx.status == nil {
		status = Success
	} else {
		status = ctx.status
	}
	return ctx.status
}

func (ctx *Context) Finalize(err error) {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	if ctx.status != Running {
		return
	}
	ctx.status = err
	ctx.elapsed.Stop()
	close(ctx.done)
}

func (ctx *Context) Cancel() {
	ctx.Finalize(Canceled)
	for _, child := range ctx.Children() {
		child.Cancel()
	}
}

func (ctx *Context) AutoCancel() *Context {
	go func() {
		<-ctx.Done()
		ctx.Cancel()
	}()
	return ctx
}

func (ctx *Context) DetachChild(child *Context) {
	for i := 0; i < len(ctx.children); i++ {
		if ctx.children[i] == child {
			ctx.children = append(ctx.children[:i], ctx.children[i+1:]...)
		}
	}
}

func (ctx *Context) AutoDetach() *Context {
	go func() {
		<-ctx.Done()
		if ctx.parent != nil {
			ctx.parent.DetachChild(ctx)
		}
	}()
	return ctx
}

func (ctx *Context) AutoDetachChild(child *Context) {
	go func() {
		<-child.Done()
		ctx.DetachChild(child)
	}()
}

func (ctx *Context) allChildren(children []*Context) []*Context {
	children = append(children, ctx.children...)
	for _, child := range ctx.children {
		children = child.allChildren(children)
	}
	return children
}

func (ctx *Context) AllChildren() []*Context {
	return ctx.allChildren([]*Context{})
}

func (ctx *Context) Children() []*Context {
	return ctx.children
}

func (ctx *Context) Set(key interface{}, value interface{}) {
	ctx.values[key] = value
}

func (ctx *Context) Get(key interface{}) (interface{}, bool) {
	if value, ok := ctx.values[key]; ok {
		return value, true
	} else {
		if ctx.parent != nil {
			return ctx.parent.Get(key)
		}
	}
	return nil, false
}

func (ctx *Context) StartDate() *time.Time {
	return ctx.elapsed.StartDate()
}

func (ctx *Context) EndDate() *time.Time {
	return ctx.elapsed.StopDate()
}

func (ctx *Context) Elapsed() time.Duration {
	return ctx.elapsed.Elapsed()
}

func (ctx *Context) Deadline() time.Time {
	return ctx.StartDate().Add(ctx.timeout)
}

func (ctx *Context) Remaining() time.Duration {
	return ctx.Deadline().Sub(time.Now())
}

func (ctx *Context) Time(name string) (split *utils.SplitTime) {
	if ctx.splits == nil {
		ctx.splits = make([]*utils.SplitTime, 0)
	}
	split = utils.NewSplitTime(name)
	ctx.splits = append(ctx.splits, split)
	split.Start()
	return
}

func (ctx *Context) Timers() []*utils.SplitTime {
	return ctx.splits
}

func (ctx *Context) string(depth int) string {
	str := bytes.NewBufferString("")
	var pad string
	for i := 0; i < depth; i++ {
		pad += " "
	}
	str.WriteString(pad)
	if depth > 0 {
		str.WriteString("`->")
	}
	str.WriteString(fmt.Sprintf("%s : status %s, elapsed %s\n", ctx.name, ctx.Status(), ctx.Elapsed().String()))
	if ctx.splits != nil {
		for _, split := range ctx.splits {
			str.WriteString(pad)
			str.WriteString("  - ")
			str.WriteString(split.String())
			str.WriteString("\n")
		}
	}
	for _, child := range ctx.Children() {
		str.WriteString(child.string(depth + 1))
	}
	return str.String()
}

func (ctx *Context) String() string {
	return ctx.string(0)
}
