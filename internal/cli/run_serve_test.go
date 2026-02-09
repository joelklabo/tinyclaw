package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
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
		PrivateKey: "fake-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
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
	connectFunc = func(privateKey, sessionKey string, relayURLs []string) (plugin.Transport, error) {
		return nil, fmt.Errorf("connect fail")
	}

	cmd := Command{
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    ".",
	}
	err := RunBot(cmd)
	if err == nil {
		t.Fatal("expected error for connect fail")
	}
}

// fakeRunBotTransport is a transport that closes immediately for RunBot tests.
type fakeRunBotTransport struct{}

func (f *fakeRunBotTransport) Subscribe(_ context.Context) (<-chan plugin.InboundEvent, error) {
	ch := make(chan plugin.InboundEvent)
	close(ch) // immediate close → RunServe returns immediately
	return ch, nil
}
func (f *fakeRunBotTransport) Post(_ context.Context, _ plugin.OutboundOp) error { return nil }
func (f *fakeRunBotTransport) Close() error                                      { return nil }

func TestRunBotHappyPath(t *testing.T) {
	old := connectFunc
	defer func() { connectFunc = old }()
	connectFunc = func(privateKey, sessionKey string, relayURLs []string) (plugin.Transport, error) {
		return &fakeRunBotTransport{}, nil
	}

	cmd := Command{
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    t.TempDir(),
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
	connectFunc = func(privateKey, sessionKey string, relayURLs []string) (plugin.Transport, error) {
		return subErrTransport, nil
	}

	cmd := Command{
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    t.TempDir(),
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
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    "/tmp/work",
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
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    t.TempDir(),
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

func TestDefaultConnect(t *testing.T) {
	// Test that defaultConnect returns an error for an invalid key.
	_, err := defaultConnect("invalid-key", "session", []string{})
	if err == nil {
		t.Fatal("expected error for invalid key or empty relays")
	}
}

func TestBuildServeParamsSystemPrompt(t *testing.T) {
	cmd := Command{
		Action:     ActionRun,
		PrivateKey: "test-key",
		Relays:     []string{"wss://relay.example.com"},
		SessionKey: "test",
		WorkDir:    t.TempDir(),
	}
	cfg := Defaults()
	cfg.SystemPrompt = "You are a helpful bot."
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

func TestRunBotLive(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live bot tests")
	}
	t.Log("live bot test placeholder — requires real Nostr key and relays")
}

func TestParseRelayList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"wss://relay.example.com", []string{"wss://relay.example.com"}},
		{"wss://r1.com, wss://r2.com", []string{"wss://r1.com", "wss://r2.com"}},
		{"wss://r1.com,wss://r2.com,wss://r3.com", []string{"wss://r1.com", "wss://r2.com", "wss://r3.com"}},
		{"  wss://r1.com , wss://r2.com  ", []string{"wss://r1.com", "wss://r2.com"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := parseRelayList(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseRelayList(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parseRelayList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
