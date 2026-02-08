package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
	"github.com/klabo/tinyclaw/plugins/discord"
)

func TestRunBotBadConfig(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{{invalid yaml")
	f.Close()

	cmd := Command{
		Action:     ActionRun,
		Token:      "fake-token",
		Channels:   []string{"123"},
		WorkDir:    ".",
		ConfigFile: f.Name(),
	}
	err = RunBot(cmd)
	if err == nil {
		t.Fatal("expected error for bad config")
	}
}

func TestRunBotConnectError(t *testing.T) {
	old := connectFunc
	defer func() { connectFunc = old }()
	connectFunc = func(token string, channelIDs []string) (discord.Client, plugin.Transport, error) {
		return nil, nil, fmt.Errorf("connect fail")
	}

	cmd := Command{
		Action:   ActionRun,
		Token:    "test-token",
		Channels: []string{"123"},
		WorkDir:  ".",
	}
	err := RunBot(cmd)
	if err == nil {
		t.Fatal("expected error for connect fail")
	}
}

// fakeRunBotTransport is a transport that closes immediately for RunBot tests.
type fakeRunBotTransport struct{}

func (f *fakeRunBotTransport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	ch := make(chan plugin.InboundEvent)
	close(ch) // immediate close → RunServe returns immediately
	return ch, nil
}
func (f *fakeRunBotTransport) Post(context.Context, plugin.OutboundOp) error { return nil }
func (f *fakeRunBotTransport) Close() error                                  { return nil }

type fakeRunBotClient struct{}

func (f *fakeRunBotClient) SendMessage(string, string) (string, error)        { return "", nil }
func (f *fakeRunBotClient) EditMessage(string, string, string) error          { return nil }
func (f *fakeRunBotClient) ChannelTyping(string) error                        { return nil }
func (f *fakeRunBotClient) SubscribeMessages(func(msg discord.Message)) error { return nil }
func (f *fakeRunBotClient) Close() error                                      { return nil }

func TestRunBotHappyPath(t *testing.T) {
	old := connectFunc
	defer func() { connectFunc = old }()
	connectFunc = func(token string, channelIDs []string) (discord.Client, plugin.Transport, error) {
		return &fakeRunBotClient{}, &fakeRunBotTransport{}, nil
	}

	cmd := Command{
		Action:   ActionRun,
		Token:    "test-token",
		Channels: []string{"123"},
		WorkDir:  t.TempDir(),
	}
	err := RunBot(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBotServeError(t *testing.T) {
	old := connectFunc
	defer func() { connectFunc = old }()

	subErrTransport := &stubServeTransport{subErr: fmt.Errorf("subscribe fail")}
	connectFunc = func(token string, channelIDs []string) (discord.Client, plugin.Transport, error) {
		return &fakeRunBotClient{}, subErrTransport, nil
	}

	cmd := Command{
		Action:   ActionRun,
		Token:    "test-token",
		Channels: []string{"123"},
		WorkDir:  t.TempDir(),
	}
	err := RunBot(cmd)
	if err == nil {
		t.Fatal("expected error from serve")
	}
}

func TestNewRunIDFunc(t *testing.T) {
	fn := newRunIDFunc()
	id1 := fn()
	id2 := fn()
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got %q and %q", id1, id2)
	}
	if id1 != "live-1" {
		t.Fatalf("expected %q, got %q", "live-1", id1)
	}
	if id2 != "live-2" {
		t.Fatalf("expected %q, got %q", "live-2", id2)
	}
}

func TestBuildServeParams(t *testing.T) {
	cmd := Command{
		Action:   ActionRun,
		Token:    "test-token",
		Channels: []string{"ch-1", "ch-2"},
		WorkDir:  "/tmp/work",
	}
	cfg := Config{BundleDir: "/tmp/bundles", LogLevel: "info"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tr := &stubServeTransport{}

	params := buildServeParams(cmd, cfg, tr, logger)

	if params.Transport != tr {
		t.Fatal("transport mismatch")
	}
	if params.WorkDir != "/tmp/work" {
		t.Fatalf("got workdir %q, want %q", params.WorkDir, "/tmp/work")
	}
	if params.BundleDir != "/tmp/bundles" {
		t.Fatalf("got bundledir %q, want %q", params.BundleDir, "/tmp/bundles")
	}
	if params.Routing.Default != "default" {
		t.Fatalf("got default routing %q, want %q", params.Routing.Default, "default")
	}
	if len(params.Routing.Rules) != 2 {
		t.Fatalf("expected 2 routing rules, got %d", len(params.Routing.Rules))
	}
	if params.Routing.Rules[0].Channel != "ch-1" {
		t.Fatalf("got rule 0 channel %q, want %q", params.Routing.Rules[0].Channel, "ch-1")
	}
	if params.Routing.Rules[1].Channel != "ch-2" {
		t.Fatalf("got rule 1 channel %q, want %q", params.Routing.Rules[1].Channel, "ch-2")
	}
	if params.Context == nil {
		t.Fatal("expected non-nil context provider")
	}
	if params.Logger != logger {
		t.Fatal("logger mismatch")
	}
	if params.IDFunc == nil {
		t.Fatal("expected non-nil IDFunc")
	}
	if params.NewHarness == nil {
		t.Fatal("expected non-nil NewHarness")
	}
	if params.Memory == nil {
		t.Fatal("expected non-nil Memory")
	}
	if params.RateLimiter == nil {
		t.Fatal("expected non-nil RateLimiter")
	}
}

func TestBuildServeParamsHarnessFactory(t *testing.T) {
	cmd := Command{
		Action:   ActionRun,
		Channels: []string{"ch-1"},
		WorkDir:  t.TempDir(),
	}
	cfg := Defaults()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tr := &stubServeTransport{}

	params := buildServeParams(cmd, cfg, tr, logger)
	h, err := params.NewHarness()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestBuildServeParamsSingleChannel(t *testing.T) {
	cmd := Command{
		Action:   ActionRun,
		Channels: []string{"only-one"},
		WorkDir:  ".",
	}
	cfg := Defaults()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	params := buildServeParams(cmd, cfg, &stubServeTransport{}, logger)
	if len(params.Routing.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(params.Routing.Rules))
	}
	if params.Routing.Rules[0] != (orchestrator.Rule{Channel: "only-one", Profile: "default"}) {
		t.Fatalf("unexpected rule: %+v", params.Routing.Rules[0])
	}
}

func TestDefaultConnect(t *testing.T) {
	// Test that defaultConnect returns an error for an invalid token.
	_, _, err := defaultConnect("invalid-token", []string{"ch-1"})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestBuildServeParamsSystemPrompt(t *testing.T) {
	cmd := Command{
		Action:   ActionRun,
		Channels: []string{"ch-1"},
		WorkDir:  t.TempDir(),
	}
	cfg := Defaults()
	cfg.SystemPrompt = "You are a helpful bot."
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tr := &stubServeTransport{}

	params := buildServeParams(cmd, cfg, tr, logger)
	// The harness factory should create a runner with the system prompt.
	// We verify it by calling the factory and checking the result is non-nil.
	h, err := params.NewHarness()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestRunBotLive(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live bot tests")
	}
	t.Log("live bot test placeholder — requires real Discord token and channel")
}
