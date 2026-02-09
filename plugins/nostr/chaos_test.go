package nostr

import (
	"math/rand"
	"sort"
	"testing"

	gonostr "github.com/nbd-wtf/go-nostr"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// TestChaosEncodeDecodeRoundTrip generates random sequences of ops,
// encodes them to Nostr events, shuffles, and decodes back.
func TestChaosEncodeDecodeRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for trial := 0; trial < 200; trial++ {
		ops := generateRandomOps(rng, 5+rng.Intn(20))

		// Encode all ops to Nostr events.
		var events []indexedEvent
		for i, op := range ops {
			ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
			if err != nil {
				t.Fatalf("trial %d, op %d: encode: %v", trial, i, err)
			}
			events = append(events, indexedEvent{index: i, event: ev})
		}

		// Shuffle the events.
		rng.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})

		// Decode shuffled events.
		var decoded []indexedOp
		for _, ie := range events {
			op, err := DecodeOutbound(&ie.event)
			if err != nil {
				t.Fatalf("trial %d: decode: %v", trial, err)
			}
			decoded = append(decoded, indexedOp{index: ie.index, op: op})
		}

		// Sort back by original index to verify content.
		sort.Slice(decoded, func(i, j int) bool {
			return decoded[i].index < decoded[j].index
		})

		// Verify each decoded op matches the original.
		for i, d := range decoded {
			orig := ops[i]
			if d.op.Kind != orig.Kind {
				t.Errorf("trial %d, op %d: kind = %q, want %q", trial, i, d.op.Kind, orig.Kind)
			}
			if d.op.Content != orig.Content {
				t.Errorf("trial %d, op %d: content mismatch", trial, i)
			}
		}
	}
}

// TestChaosDuplicateEvents verifies duplicates don't cause errors.
func TestChaosDuplicateEvents(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	for trial := 0; trial < 100; trial++ {
		ops := generateRandomOps(rng, 3+rng.Intn(10))

		var events []gonostrEvent
		for _, op := range ops {
			ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
			if err != nil {
				t.Fatalf("trial %d: encode: %v", trial, err)
			}
			events = append(events, ev)
			// Duplicate each event.
			events = append(events, ev)
		}

		// Shuffle.
		rng.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})

		// All should decode without error.
		for i, ev := range events {
			_, err := DecodeOutbound(&ev)
			if err != nil {
				t.Fatalf("trial %d, event %d: decode error: %v", trial, i, err)
			}
		}
	}
}

// TestChaosDeltaReassembly verifies deltas can be reassembled by seq number
// even when received out of order.
func TestChaosDeltaReassembly(t *testing.T) {
	rng := rand.New(rand.NewSource(77))

	for trial := 0; trial < 100; trial++ {
		n := 5 + rng.Intn(20)
		var ops []plugin.OutboundOp
		for i := 1; i <= n; i++ {
			ops = append(ops, plugin.OutboundOp{
				Kind:    plugin.OutboundDelta,
				Content: string(rune('A' + i%26)),
				Seq:     i,
			})
		}

		// Encode.
		var events []indexedEvent
		for i, op := range ops {
			ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
			if err != nil {
				t.Fatalf("trial %d: encode: %v", trial, err)
			}
			events = append(events, indexedEvent{index: i, event: ev})
		}

		// Shuffle.
		rng.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})

		// Decode and collect.
		var decoded []plugin.OutboundOp
		for _, ie := range events {
			op, err := DecodeOutbound(&ie.event)
			if err != nil {
				t.Fatalf("trial %d: decode: %v", trial, err)
			}
			decoded = append(decoded, op)
		}

		// Reassemble by sorting on Seq.
		sort.Slice(decoded, func(i, j int) bool {
			return decoded[i].Seq < decoded[j].Seq
		})

		// Verify correct order.
		for i, d := range decoded {
			expectedSeq := i + 1
			if d.Seq != expectedSeq {
				t.Errorf("trial %d: seq[%d] = %d, want %d", trial, i, d.Seq, expectedSeq)
			}
		}
	}
}

// TestChaosConversationSequence generates full conversation sequences
// (status → deltas → tool → deltas → response/error), shuffles events,
// and verifies all decode correctly.
func TestChaosConversationSequence(t *testing.T) {
	rng := rand.New(rand.NewSource(55))

	for trial := 0; trial < 100; trial++ {
		ops := generateConversationOps(rng)

		var events []gonostrEvent
		for _, op := range ops {
			ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
			if err != nil {
				t.Fatalf("trial %d: encode: %v", trial, err)
			}
			events = append(events, ev)
		}

		// Shuffle.
		rng.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})

		// Verify all decode without error.
		for i, ev := range events {
			_, err := DecodeOutbound(&ev)
			if err != nil {
				t.Fatalf("trial %d, event %d: decode: %v", trial, i, err)
			}
		}

		// Verify conversation always has a terminal event.
		hasTerminal := false
		for _, op := range ops {
			if op.Kind == plugin.OutboundResponse || op.Kind == plugin.OutboundError {
				hasTerminal = true
				break
			}
		}
		if !hasTerminal {
			t.Fatalf("trial %d: conversation missing terminal event", trial)
		}
	}
}

// --- helpers ---

type indexedEvent struct {
	index int
	event gonostrEvent
}

type indexedOp struct {
	index int
	op    plugin.OutboundOp
}

type gonostrEvent = gonostr.Event

func generateRandomOps(rng *rand.Rand, n int) []plugin.OutboundOp {
	kinds := []plugin.OutboundOpKind{
		plugin.OutboundStatus,
		plugin.OutboundDelta,
		plugin.OutboundTool,
		plugin.OutboundResponse,
		plugin.OutboundError,
	}
	phases := []string{"thinking", "tool_use", "writing", "done"}
	tools := []string{"bash", "read", "write", "grep"}
	faults := []string{"auth", "quota", "transient", "fatal"}

	var ops []plugin.OutboundOp
	for i := 0; i < n; i++ {
		kind := kinds[rng.Intn(len(kinds))]
		op := plugin.OutboundOp{Kind: kind}
		switch kind {
		case plugin.OutboundStatus:
			op.Phase = phases[rng.Intn(len(phases))]
		case plugin.OutboundDelta:
			op.Content = randomString(rng, 1+rng.Intn(100))
			op.Seq = i + 1
		case plugin.OutboundTool:
			op.Tool = tools[rng.Intn(len(tools))]
		case plugin.OutboundResponse:
			op.Content = randomString(rng, 1+rng.Intn(500))
		case plugin.OutboundError:
			op.Content = randomString(rng, 1+rng.Intn(100))
			op.Fault = faults[rng.Intn(len(faults))]
		}
		ops = append(ops, op)
	}
	return ops
}

func generateConversationOps(rng *rand.Rand) []plugin.OutboundOp {
	var ops []plugin.OutboundOp
	seq := 1

	// Status: thinking.
	ops = append(ops, plugin.OutboundOp{Kind: plugin.OutboundStatus, Phase: "thinking"})

	// Some deltas.
	nDeltas := 1 + rng.Intn(5)
	for i := 0; i < nDeltas; i++ {
		ops = append(ops, plugin.OutboundOp{
			Kind: plugin.OutboundDelta, Content: randomString(rng, 10), Seq: seq,
		})
		seq++
	}

	// Maybe a tool call.
	if rng.Float32() > 0.3 {
		ops = append(ops, plugin.OutboundOp{Kind: plugin.OutboundStatus, Phase: "tool_use"})
		ops = append(ops, plugin.OutboundOp{Kind: plugin.OutboundTool, Tool: "bash"})
		ops = append(ops, plugin.OutboundOp{Kind: plugin.OutboundStatus, Phase: "writing"})

		// More deltas after tool.
		nDeltas = 1 + rng.Intn(3)
		for i := 0; i < nDeltas; i++ {
			ops = append(ops, plugin.OutboundOp{
				Kind: plugin.OutboundDelta, Content: randomString(rng, 10), Seq: seq,
			})
			seq++
		}
	}

	// Terminal: response or error.
	if rng.Float32() > 0.1 {
		ops = append(ops, plugin.OutboundOp{
			Kind: plugin.OutboundResponse, Content: randomString(rng, 50),
		})
	} else {
		ops = append(ops, plugin.OutboundOp{
			Kind: plugin.OutboundError, Content: "error occurred", Fault: "fatal",
		})
	}

	return ops
}

func randomString(rng *rand.Rand, n int) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789 "
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}
