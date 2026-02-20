package stream

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeFactory struct {
	mu      sync.Mutex
	queue   []*fakeCommand
	names   []string
	argSets [][]string
}

func (f *fakeFactory) Command(name string, args ...string) command {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 {
		panic("no fake commands remaining")
	}
	c := f.queue[0]
	f.queue = f.queue[1:]
	f.names = append(f.names, name)
	f.argSets = append(f.argSets, append([]string(nil), args...))
	return c
}

type fakeCommand struct {
	pid           int
	startErr      error
	waitErr       error
	signalErr     error
	killErr       error
	waitCh        chan struct{}
	onSignalClose bool
	onKillClose   bool

	mu          sync.Mutex
	signalCount int
	killCount   int
	closeOnce   sync.Once
}

func (c *fakeCommand) Start() error {
	return c.startErr
}

func (c *fakeCommand) Wait() error {
	if c.waitCh != nil {
		<-c.waitCh
	}
	return c.waitErr
}

func (c *fakeCommand) Signal(_ os.Signal) error {
	c.mu.Lock()
	c.signalCount++
	c.mu.Unlock()
	if c.onSignalClose {
		c.closeWait()
	}
	return c.signalErr
}

func (c *fakeCommand) Kill() error {
	c.mu.Lock()
	c.killCount++
	c.mu.Unlock()
	if c.onKillClose {
		c.closeWait()
	}
	return c.killErr
}

func (c *fakeCommand) PID() int {
	return c.pid
}

func (c *fakeCommand) closeWait() {
	if c.waitCh == nil {
		return
	}
	c.closeOnce.Do(func() {
		close(c.waitCh)
	})
}

func (c *fakeCommand) counts() (signals int, kills int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.signalCount, c.killCount
}

func TestStartAndStatusLifecycle(t *testing.T) {
	t.Parallel()

	cmd := &fakeCommand{pid: 101, waitCh: make(chan struct{})}
	factory := &fakeFactory{queue: []*fakeCommand{cmd}}

	m := NewManager(ManagerOptions{FFmpegBinary: "ffmpeg-test", StopTimeout: 200 * time.Millisecond})
	m.factory = factory

	cfg := HLSConfig{InputPath: "input.mp4", OutputDir: t.TempDir()}

	st, err := m.Start(context.Background(), "movie-night", cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if st.State != StateRunning {
		t.Fatalf("expected running state, got %s", st.State)
	}
	if st.PID != 101 {
		t.Fatalf("expected pid 101, got %d", st.PID)
	}

	running := m.ListRunning()
	if len(running) != 1 {
		t.Fatalf("expected 1 running stream, got %d", len(running))
	}
	if running[0].StreamID != "movie-night" {
		t.Fatalf("expected stream id movie-night, got %s", running[0].StreamID)
	}

	cmd.closeWait()
	eventually(t, time.Second, func() bool {
		status, ok := m.Status("movie-night")
		return ok && status.State == StateStopped
	})

	if got := m.ListRunning(); len(got) != 0 {
		t.Fatalf("expected no running streams, got %d", len(got))
	}
}

func TestStartFailureMarksStreamFailed(t *testing.T) {
	t.Parallel()

	cmd := &fakeCommand{startErr: errors.New("exec failed")}
	factory := &fakeFactory{queue: []*fakeCommand{cmd}}

	m := NewManager(ManagerOptions{})
	m.factory = factory

	st, err := m.Start(context.Background(), "broken", HLSConfig{InputPath: "in.mp4", OutputDir: t.TempDir()})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "start ffmpeg") {
		t.Fatalf("expected contextual start error, got %v", err)
	}
	if st.State != StateFailed {
		t.Fatalf("expected failed state, got %s", st.State)
	}

	status, ok := m.Status("broken")
	if !ok {
		t.Fatalf("expected stream to exist")
	}
	if status.State != StateFailed {
		t.Fatalf("expected failed state in status, got %s", status.State)
	}
	if status.LastError == "" {
		t.Fatalf("expected last error to be set")
	}
}

func TestStopSignalsProcessAndStops(t *testing.T) {
	t.Parallel()

	cmd := &fakeCommand{
		pid:           202,
		waitCh:        make(chan struct{}),
		onSignalClose: true,
	}
	factory := &fakeFactory{queue: []*fakeCommand{cmd}}

	m := NewManager(ManagerOptions{StopTimeout: 500 * time.Millisecond})
	m.factory = factory

	_, err := m.Start(context.Background(), "live", HLSConfig{InputPath: "in.ts", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if err := m.Stop(context.Background(), "live"); err != nil {
		t.Fatalf("expected nil error stopping stream, got %v", err)
	}

	eventually(t, time.Second, func() bool {
		status, ok := m.Status("live")
		return ok && status.State == StateStopped
	})

	signals, kills := cmd.counts()
	if signals == 0 {
		t.Fatalf("expected at least one signal")
	}
	if kills != 0 {
		t.Fatalf("expected no kill, got %d", kills)
	}
}

func TestUpdateRestartsWithNewConfig(t *testing.T) {
	t.Parallel()

	cmd1 := &fakeCommand{pid: 301, waitCh: make(chan struct{}), onSignalClose: true}
	cmd2 := &fakeCommand{pid: 302, waitCh: make(chan struct{})}
	factory := &fakeFactory{queue: []*fakeCommand{cmd1, cmd2}}

	m := NewManager(ManagerOptions{StopTimeout: 500 * time.Millisecond})
	m.factory = factory

	dir1 := t.TempDir()
	dir2 := t.TempDir()
	_, err := m.Start(context.Background(), "chan1", HLSConfig{InputPath: "one.ts", OutputDir: dir1, SegmentDurationSeconds: 2})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	updated, err := m.Update(context.Background(), "chan1", HLSConfig{InputPath: "two.ts", OutputDir: dir2, SegmentDurationSeconds: 4})
	if err != nil {
		t.Fatalf("expected nil error from update, got %v", err)
	}
	if updated.State != StateRunning {
		t.Fatalf("expected running state after update, got %s", updated.State)
	}
	if updated.PID != 302 {
		t.Fatalf("expected new pid 302, got %d", updated.PID)
	}
	if updated.Config.InputPath != "two.ts" {
		t.Fatalf("expected new input path, got %s", updated.Config.InputPath)
	}

	factory.mu.Lock()
	if len(factory.names) != 2 {
		factory.mu.Unlock()
		t.Fatalf("expected 2 ffmpeg command invocations, got %d", len(factory.names))
	}
	if factory.names[0] != "ffmpeg" || factory.names[1] != "ffmpeg" {
		factory.mu.Unlock()
		t.Fatalf("expected ffmpeg binary for both invocations, got %#v", factory.names)
	}
	if !containsSequence(factory.argSets[1], []string{"-i", "two.ts"}) {
		factory.mu.Unlock()
		t.Fatalf("expected updated args to include new input path, got %#v", factory.argSets[1])
	}
	factory.mu.Unlock()

	cmd2.closeWait()
	eventually(t, time.Second, func() bool {
		status, ok := m.Status("chan1")
		return ok && status.State == StateStopped
	})
}

func TestListRunningIDs(t *testing.T) {
	t.Parallel()

	cmdA := &fakeCommand{pid: 401, waitCh: make(chan struct{})}
	cmdB := &fakeCommand{pid: 402, waitCh: make(chan struct{})}
	factory := &fakeFactory{queue: []*fakeCommand{cmdA, cmdB}}

	m := NewManager(ManagerOptions{})
	m.factory = factory

	_, err := m.Start(context.Background(), "a", HLSConfig{InputPath: "a.ts", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	_, err = m.Start(context.Background(), "b", HLSConfig{InputPath: "b.ts", OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	ids := m.ListRunningIDs()
	expected := []string{"a", "b"}
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("expected %v, got %v", expected, ids)
	}

	cmdA.closeWait()
	cmdB.closeWait()
}

func containsSequence(values []string, seq []string) bool {
	if len(seq) == 0 {
		return true
	}
	for i := 0; i <= len(values)-len(seq); i++ {
		if reflect.DeepEqual(values[i:i+len(seq)], seq) {
			return true
		}
	}
	return false
}

func eventually(t *testing.T, timeout time.Duration, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}
