package stream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrStreamAlreadyRunning = errors.New("stream already running")
	ErrStreamNotFound       = errors.New("stream not found")
)

type State string

const (
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

type HLSConfig struct {
	InputPath              string
	OutputDir              string
	PlaylistName           string
	SegmentPattern         string
	SegmentDurationSeconds int
	ListSize               int
	LoopInput              bool
	DeleteSegments         bool
	VideoCodec             string
	AudioCodec             string
	ExtraArgs              []string
}

type Status struct {
	StreamID      string
	State         State
	PID           int
	Config        HLSConfig
	StartedAt     time.Time
	UpdatedAt     time.Time
	StoppedAt     time.Time
	LastError     string
	PlaylistPath  string
	CommandBinary string
	CommandArgs   []string
}

type ManagerOptions struct {
	FFmpegBinary string
	StopTimeout  time.Duration
}

type command interface {
	Start() error
	Wait() error
	Signal(sig os.Signal) error
	Kill() error
	PID() int
}

type commandFactory interface {
	Command(name string, args ...string) command
}

type execCommandFactory struct{}

func (f execCommandFactory) Command(name string, args ...string) command {
	return &execCommand{cmd: exec.Command(name, args...)}
}

type execCommand struct {
	cmd *exec.Cmd
}

func (c *execCommand) Start() error {
	return c.cmd.Start()
}

func (c *execCommand) Wait() error {
	return c.cmd.Wait()
}

func (c *execCommand) Signal(sig os.Signal) error {
	if c.cmd.Process == nil {
		return errors.New("process not started")
	}
	return c.cmd.Process.Signal(sig)
}

func (c *execCommand) Kill() error {
	if c.cmd.Process == nil {
		return errors.New("process not started")
	}
	return c.cmd.Process.Kill()
}

func (c *execCommand) PID() int {
	if c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
}

type managedStream struct {
	id           string
	state        State
	cfg          HLSConfig
	pid          int
	lastErr      string
	startedAt    time.Time
	updatedAt    time.Time
	stoppedAt    time.Time
	playlistPath string
	binary       string
	args         []string
	cmd          command
	done         chan struct{}
	stopReq      bool
}

func (s *managedStream) isActive() bool {
	switch s.state {
	case StateStarting, StateRunning, StateStopping:
		return true
	default:
		return false
	}
}

func (s *managedStream) toStatus() Status {
	cfg := s.cfg
	cfg.ExtraArgs = append([]string(nil), s.cfg.ExtraArgs...)
	return Status{
		StreamID:      s.id,
		State:         s.state,
		PID:           s.pid,
		Config:        cfg,
		StartedAt:     s.startedAt,
		UpdatedAt:     s.updatedAt,
		StoppedAt:     s.stoppedAt,
		LastError:     s.lastErr,
		PlaylistPath:  s.playlistPath,
		CommandBinary: s.binary,
		CommandArgs:   append([]string(nil), s.args...),
	}
}

type Manager struct {
	mu          sync.RWMutex
	streams     map[string]*managedStream
	factory     commandFactory
	now         func() time.Time
	ffmpegBin   string
	stopTimeout time.Duration
}

func NewManager(opts ManagerOptions) *Manager {
	ffmpegBin := strings.TrimSpace(opts.FFmpegBinary)
	if ffmpegBin == "" {
		ffmpegBin = "ffmpeg"
	}
	stopTimeout := opts.StopTimeout
	if stopTimeout <= 0 {
		stopTimeout = 8 * time.Second
	}

	return &Manager{
		streams:     make(map[string]*managedStream),
		factory:     execCommandFactory{},
		now:         time.Now,
		ffmpegBin:   ffmpegBin,
		stopTimeout: stopTimeout,
	}
}

func (m *Manager) Start(_ context.Context, streamID string, cfg HLSConfig) (Status, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return Status{}, errors.New("stream id is required")
	}

	norm, err := normalizeConfig(cfg)
	if err != nil {
		return Status{}, err
	}
	if err := os.MkdirAll(norm.OutputDir, 0o755); err != nil {
		return Status{}, fmt.Errorf("create output dir: %w", err)
	}

	args := buildHLSArgs(norm)
	playlistPath := filepath.Join(norm.OutputDir, norm.PlaylistName)
	now := m.now()

	m.mu.Lock()
	if existing, ok := m.streams[streamID]; ok && existing.isActive() {
		m.mu.Unlock()
		return Status{}, ErrStreamAlreadyRunning
	}

	s := &managedStream{
		id:           streamID,
		state:        StateStarting,
		cfg:          norm,
		startedAt:    now,
		updatedAt:    now,
		playlistPath: playlistPath,
		binary:       m.ffmpegBin,
		args:         append([]string(nil), args...),
		done:         make(chan struct{}),
	}
	s.cmd = m.factory.Command(m.ffmpegBin, args...)
	m.streams[streamID] = s
	m.mu.Unlock()

	if err := s.cmd.Start(); err != nil {
		m.mu.Lock()
		s.state = StateFailed
		s.lastErr = err.Error()
		s.updatedAt = m.now()
		s.stoppedAt = s.updatedAt
		close(s.done)
		st := s.toStatus()
		m.mu.Unlock()
		return st, fmt.Errorf("start ffmpeg: %w", err)
	}

	m.mu.Lock()
	s.state = StateRunning
	s.pid = s.cmd.PID()
	s.updatedAt = m.now()
	st := s.toStatus()
	m.mu.Unlock()

	go m.waitForExit(streamID, s)

	return st, nil
}

func (m *Manager) waitForExit(streamID string, s *managedStream) {
	err := s.cmd.Wait()
	now := m.now()

	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.streams[streamID]
	if !ok || current != s {
		close(s.done)
		return
	}

	s.updatedAt = now
	s.stoppedAt = now
	if s.stopReq {
		s.state = StateStopped
	} else if err != nil {
		s.state = StateFailed
		s.lastErr = err.Error()
	} else {
		s.state = StateStopped
	}

	close(s.done)
}

func (m *Manager) Stop(ctx context.Context, streamID string) error {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return errors.New("stream id is required")
	}

	m.mu.Lock()
	s, ok := m.streams[streamID]
	if !ok {
		m.mu.Unlock()
		return ErrStreamNotFound
	}
	if !s.isActive() {
		m.mu.Unlock()
		return nil
	}
	s.stopReq = true
	s.state = StateStopping
	s.updatedAt = m.now()
	cmd := s.cmd
	done := s.done
	timeout := m.stopTimeout
	m.mu.Unlock()

	if cmd != nil {
		_ = cmd.Signal(os.Interrupt)
	}
	if waitDone(ctx, done, timeout) {
		return nil
	}

	if cmd != nil {
		_ = cmd.Kill()
	}
	if waitDone(ctx, done, timeout) {
		return nil
	}

	return errors.New("timed out stopping stream")
}

func (m *Manager) Update(ctx context.Context, streamID string, cfg HLSConfig) (Status, error) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return Status{}, errors.New("stream id is required")
	}

	m.mu.RLock()
	_, ok := m.streams[streamID]
	m.mu.RUnlock()
	if !ok {
		return Status{}, ErrStreamNotFound
	}

	if err := m.Stop(ctx, streamID); err != nil {
		return Status{}, err
	}
	return m.Start(ctx, streamID, cfg)
}

func (m *Manager) StopAll(ctx context.Context) error {
	streamIDs := m.ListRunningIDs()
	for _, streamID := range streamIDs {
		if err := m.Stop(ctx, streamID); err != nil {
			return fmt.Errorf("stop %s: %w", streamID, err)
		}
	}
	return nil
}

func (m *Manager) Status(streamID string) (Status, bool) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return Status{}, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streams[streamID]
	if !ok {
		return Status{}, false
	}
	return s.toStatus(), true
}

func (m *Manager) List() []Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Status, 0, len(m.streams))
	for _, s := range m.streams {
		out = append(out, s.toStatus())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StreamID < out[j].StreamID
	})
	return out
}

func (m *Manager) ListRunning() []Status {
	all := m.List()
	out := make([]Status, 0, len(all))
	for _, s := range all {
		switch s.State {
		case StateStarting, StateRunning, StateStopping:
			out = append(out, s)
		}
	}
	return out
}

func (m *Manager) ListRunningIDs() []string {
	running := m.ListRunning()
	ids := make([]string, 0, len(running))
	for _, s := range running {
		ids = append(ids, s.StreamID)
	}
	return ids
}

func (m *Manager) Remove(streamID string) error {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return errors.New("stream id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.streams[streamID]
	if !ok {
		return ErrStreamNotFound
	}
	if s.isActive() {
		return errors.New("cannot remove active stream")
	}
	delete(m.streams, streamID)
	return nil
}

func normalizeConfig(cfg HLSConfig) (HLSConfig, error) {
	cfg.InputPath = strings.TrimSpace(cfg.InputPath)
	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)
	cfg.PlaylistName = strings.TrimSpace(cfg.PlaylistName)
	cfg.SegmentPattern = strings.TrimSpace(cfg.SegmentPattern)
	cfg.VideoCodec = strings.TrimSpace(cfg.VideoCodec)
	cfg.AudioCodec = strings.TrimSpace(cfg.AudioCodec)

	if cfg.InputPath == "" {
		return HLSConfig{}, errors.New("input path is required")
	}
	if cfg.OutputDir == "" {
		return HLSConfig{}, errors.New("output dir is required")
	}
	if cfg.PlaylistName == "" {
		cfg.PlaylistName = "index.m3u8"
	}
	if cfg.SegmentPattern == "" {
		cfg.SegmentPattern = "segment_%06d.ts"
	}
	if cfg.SegmentDurationSeconds <= 0 {
		cfg.SegmentDurationSeconds = 6
	}
	if cfg.ListSize < 0 {
		return HLSConfig{}, errors.New("list size must be non-negative")
	}
	if cfg.ListSize == 0 {
		cfg.ListSize = 6
	}
	if cfg.VideoCodec == "" {
		cfg.VideoCodec = "copy"
	}
	if cfg.AudioCodec == "" {
		cfg.AudioCodec = "copy"
	}
	cfg.ExtraArgs = append([]string(nil), cfg.ExtraArgs...)

	return cfg, nil
}

func buildHLSArgs(cfg HLSConfig) []string {
	args := []string{
		"-hide_banner",
		"-nostdin",
		"-loglevel", "error",
	}
	if cfg.LoopInput {
		args = append(args, "-stream_loop", "-1")
	}
	args = append(args,
		"-i", cfg.InputPath,
		"-c:v", cfg.VideoCodec,
		"-c:a", cfg.AudioCodec,
		"-f", "hls",
		"-hls_time", strconv.Itoa(cfg.SegmentDurationSeconds),
		"-hls_list_size", strconv.Itoa(cfg.ListSize),
	)

	flags := []string{"independent_segments"}
	if cfg.DeleteSegments {
		flags = append(flags, "delete_segments")
	}
	args = append(args, "-hls_flags", strings.Join(flags, "+"))
	args = append(args, "-hls_segment_filename", filepath.Join(cfg.OutputDir, cfg.SegmentPattern))
	args = append(args, cfg.ExtraArgs...)
	args = append(args, filepath.Join(cfg.OutputDir, cfg.PlaylistName))
	return args
}

func waitDone(ctx context.Context, done <-chan struct{}, timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-done:
			return true
		case <-ctx.Done():
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	case <-timer.C:
		return false
	}
}
