package discord

import (
	"strings"
	"testing"
)

func TestChunkEmpty(t *testing.T) {
	result := chunk("", 2000)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestChunkFitsWithinLimit(t *testing.T) {
	text := "Hello, world!"
	result := chunk(text, 2000)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != text {
		t.Fatalf("got %q, want %q", result[0], text)
	}
}

func TestChunkParagraphSplit(t *testing.T) {
	// Two paragraphs, each under maxSize, but combined they exceed it.
	p1 := strings.Repeat("a", 60)
	p2 := strings.Repeat("b", 60)
	text := p1 + "\n\n" + p2
	result := chunk(text, 100)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(result), result)
	}
	if result[0] != p1 {
		t.Fatalf("chunk[0] = %q, want %q", result[0], p1)
	}
	if result[1] != p2 {
		t.Fatalf("chunk[1] = %q, want %q", result[1], p2)
	}
}

func TestChunkFencePreservation(t *testing.T) {
	// A large fenced code block should remain intact even if it exceeds maxSize.
	code := strings.Repeat("x", 150)
	text := "```\n" + code + "\n```"
	result := chunk(text, 100)
	// The fence block stays whole via coalesce (even if > maxSize).
	found := false
	for _, c := range result {
		if strings.Contains(c, "```") && strings.Contains(c, code) {
			found = true
		}
	}
	if !found {
		t.Fatalf("fenced block not preserved: %v", result)
	}
}

func TestChunkHardCut(t *testing.T) {
	// A single long line with no newlines — must be hard-cut.
	text := strings.Repeat("z", 250)
	result := chunk(text, 100)
	if len(result) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(result))
	}
	total := 0
	for _, c := range result {
		total += len(c)
		if len(c) > 100 {
			t.Fatalf("chunk exceeds maxSize: %d", len(c))
		}
	}
	if total != 250 {
		t.Fatalf("total length = %d, want 250", total)
	}
}

func TestCoalesceMergesSmallChunks(t *testing.T) {
	// Two small segments that should be merged.
	result := coalesce([]string{"aaa"}, "bbb", 20, "\n\n")
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != "aaa\n\nbbb" {
		t.Fatalf("got %q, want %q", result[0], "aaa\n\nbbb")
	}
}

func TestCoalesceDoesNotMergeOverLimit(t *testing.T) {
	result := coalesce([]string{"aaa"}, "bbb", 5, "\n\n")
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
}

func TestCoalesceEmpty(t *testing.T) {
	result := coalesce(nil, "hello", 100, "\n")
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
	if result[0] != "hello" {
		t.Fatalf("got %q, want %q", result[0], "hello")
	}
}

func TestMultipleParagraphsSplit(t *testing.T) {
	// Three paragraphs separated by double newlines.
	p1 := strings.Repeat("a", 40)
	p2 := strings.Repeat("b", 40)
	p3 := strings.Repeat("c", 40)
	text := p1 + "\n\n" + p2 + "\n\n" + p3
	result := chunk(text, 50)
	if len(result) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %v", len(result), result)
	}
}

func TestNewlineSplitting(t *testing.T) {
	// Lines separated by single newlines that together exceed maxSize.
	lines := make([]string, 5)
	for i := range lines {
		lines[i] = strings.Repeat("x", 30)
	}
	text := strings.Join(lines, "\n")
	result := chunk(text, 65)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d: %v", len(result), result)
	}
	for _, c := range result {
		if len(c) > 65 {
			t.Fatalf("chunk exceeds maxSize: len=%d", len(c))
		}
	}
}

func TestSplitSegmentsWithFence(t *testing.T) {
	text := "before\n```\ncode\n```\nafter"
	segs := splitSegments(text)
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d: %v", len(segs), segs)
	}
	if !strings.HasPrefix(segs[1], "```") {
		t.Fatalf("segment[1] should be fence block: %q", segs[1])
	}
}

func TestSplitSegmentsNoFence(t *testing.T) {
	text := "para1\n\npara2"
	segs := splitSegments(text)
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d: %v", len(segs), segs)
	}
}

func TestAppendSegmentsTrimsEmpty(t *testing.T) {
	// "a\n\n\n\nb" has empty parts between double newlines
	result := appendSegments(nil, "a\n\n\n\nb")
	for _, s := range result {
		if s == "" {
			t.Fatal("appendSegments should not produce empty segments")
		}
	}
}

func TestSplitAtNewlinesSingleLine(t *testing.T) {
	// Single line longer than max → hardCut
	text := strings.Repeat("a", 50)
	result := splitAtNewlines(text, 20)
	for _, c := range result {
		if len(c) > 20 {
			t.Fatalf("chunk exceeds maxSize: %d", len(c))
		}
	}
}

func TestSplitAtNewlinesWithLongLine(t *testing.T) {
	// Mix of short and long lines
	text := "short\n" + strings.Repeat("x", 50) + "\nshort2"
	result := splitAtNewlines(text, 20)
	for _, c := range result {
		if len(c) > 20 {
			t.Fatalf("chunk exceeds maxSize: %d", len(c))
		}
	}
}

func TestHardCutExact(t *testing.T) {
	text := strings.Repeat("a", 40)
	result := hardCut(text, 20)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result))
	}
	if result[0] != strings.Repeat("a", 20) {
		t.Fatalf("chunk[0] wrong length")
	}
	if result[1] != strings.Repeat("a", 20) {
		t.Fatalf("chunk[1] wrong length")
	}
}

func TestHardCutShort(t *testing.T) {
	result := hardCut("abc", 100)
	if len(result) != 1 || result[0] != "abc" {
		t.Fatalf("expected [abc], got %v", result)
	}
}

func TestHardCutEmpty(t *testing.T) {
	result := hardCut("", 100)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestIsFenceBlock(t *testing.T) {
	if !isFenceBlock("```go\ncode\n```") {
		t.Fatal("expected fence block")
	}
	if !isFenceBlock("  ```\ncode\n```") {
		t.Fatal("expected fence block with leading space")
	}
	if isFenceBlock("no fence here") {
		t.Fatal("expected non-fence")
	}
}

func TestSplitSegmentsUnclosedFence(t *testing.T) {
	// Fence that is never closed — should still produce segments.
	text := "before\n```\ncode\nmore code"
	segs := splitSegments(text)
	if len(segs) == 0 {
		t.Fatal("expected segments")
	}
}

func TestSplitAtNewlinesFlush(t *testing.T) {
	// Ensure the buffer is flushed at the end.
	text := "line1\nline2"
	result := splitAtNewlines(text, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d: %v", len(result), result)
	}
	if result[0] != "line1\nline2" {
		t.Fatalf("got %q, want %q", result[0], "line1\nline2")
	}
}
