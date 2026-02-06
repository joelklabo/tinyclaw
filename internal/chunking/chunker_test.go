package chunking

import (
	"strings"
	"testing"
)

func TestEmptyInput(t *testing.T) {
	chunks := Chunk("", 100)
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}

func TestFitsInOneChunk(t *testing.T) {
	text := "hello world"
	chunks := Chunk(text, 100)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("expected %q, got %q", text, chunks[0])
	}
}

func TestExactBoundary(t *testing.T) {
	text := "12345"
	chunks := Chunk(text, 5)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for exact boundary, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("expected %q, got %q", text, chunks[0])
	}
}

func TestSplitAtParagraph(t *testing.T) {
	text := "first paragraph\n\nsecond paragraph"
	chunks := Chunk(text, 20)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != "first paragraph" {
		t.Fatalf("chunk 0: expected %q, got %q", "first paragraph", chunks[0])
	}
	if chunks[1] != "second paragraph" {
		t.Fatalf("chunk 1: expected %q, got %q", "second paragraph", chunks[1])
	}
}

func TestSplitAtNewline(t *testing.T) {
	text := "line one\nline two\nline three"
	chunks := Chunk(text, 15)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d: %v", len(chunks), chunks)
	}
	// Rejoin should reproduce original
	rejoined := strings.Join(chunks, "\n")
	if rejoined != text {
		t.Fatalf("rejoin mismatch:\n  expected: %q\n  got:      %q", text, rejoined)
	}
}

func TestHardCut(t *testing.T) {
	text := "abcdefghij"
	chunks := Chunk(text, 3)
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d: %v", len(chunks), chunks)
	}
	if strings.Join(chunks, "") != text {
		t.Fatalf("hard-cut chunks don't rejoin: %v", chunks)
	}
}

func TestSingleCharOverLimit(t *testing.T) {
	text := "abcdef"
	chunks := Chunk(text, 5)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != "abcde" || chunks[1] != "f" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestCodeFencePreservation(t *testing.T) {
	text := "before\n\n```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n\nafter"
	// Max size big enough for the fence block but not the whole thing
	chunks := Chunk(text, 60)
	// Verify no chunk splits inside the code fence
	for i, c := range chunks {
		opens := strings.Count(c, "```")
		if opens%2 != 0 {
			t.Fatalf("chunk %d has unmatched code fence: %q", i, c)
		}
	}
	// Rejoin should be original
	rejoined := strings.Join(chunks, "\n\n")
	if rejoined != text {
		t.Fatalf("rejoin mismatch:\n  expected: %q\n  got:      %q", text, rejoined)
	}
}

func TestCodeFenceLargerThanMax(t *testing.T) {
	// When a code fence block is larger than maxSize, it must still be kept whole
	fence := "```\n" + strings.Repeat("x", 50) + "\n```"
	text := "before\n\n" + fence + "\n\nafter"
	chunks := Chunk(text, 30)
	// The fence block should appear intact in one chunk
	found := false
	for _, c := range chunks {
		if strings.Contains(c, "```") {
			if strings.Count(c, "```") != 2 {
				t.Fatalf("code fence split across chunks: %q", c)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("code fence not found in any chunk")
	}
}

func TestMixedContent(t *testing.T) {
	text := "intro\n\n```\ncode\n```\n\nmiddle paragraph\n\n```\nmore code\n```\n\nend"
	chunks := Chunk(text, 30)
	for i, c := range chunks {
		opens := strings.Count(c, "```")
		if opens%2 != 0 {
			t.Fatalf("chunk %d has unmatched code fence: %q", i, c)
		}
	}
	rejoined := strings.Join(chunks, "\n\n")
	if rejoined != text {
		t.Fatalf("rejoin mismatch:\n  expected: %q\n  got:      %q", text, rejoined)
	}
}

func TestCoalesce(t *testing.T) {
	// A trailing tiny piece should be merged into previous chunk
	text := "aaaa\n\nbb"
	chunks := Chunk(text, 10)
	// "aaaa" (4) + "\n\n" + "bb" (2) = 8, fits in 10
	if len(chunks) != 1 {
		t.Fatalf("expected coalesced into 1 chunk, got %d: %v", len(chunks), chunks)
	}
}

func TestSplitAtNewlinesWithLongLine(t *testing.T) {
	// A paragraph that has short lines followed by a long line
	// This exercises the buf-flush-then-hardcut path in splitAtNewlines
	short := "ab"
	long := strings.Repeat("x", 20)
	text := short + "\n" + long
	// maxSize 10: "ab" fits, but the 20-char line doesn't fit and buf has content
	chunks := Chunk(text, 10)
	// Should get: "ab", then hard-cut of 20 chars into 2 chunks
	all := strings.Join(chunks, "")
	// All content should be preserved (minus the newline separator that was between them)
	if !strings.Contains(all, short) || !strings.Contains(all, long) {
		t.Fatalf("content lost: %v", chunks)
	}
}

func TestSplitAtNewlinesSingleLongLine(t *testing.T) {
	// A single long line with no newlines inside a paragraph
	// This should fallback directly to hardCut via splitAtNewlines
	text := strings.Repeat("y", 15)
	chunks := Chunk(text, 5)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d: %v", len(chunks), chunks)
	}
	if strings.Join(chunks, "") != text {
		t.Fatalf("content mismatch: %v", chunks)
	}
}

func TestNewlineSplitFlushBeforeLongLine(t *testing.T) {
	// Exercises: buf has content, next line is too long, buf flushed then hardcut
	text := "aa\nbb\n" + strings.Repeat("z", 12)
	chunks := Chunk(text, 5)
	// "aa\nbb" is 5 which fits, then the 12-char line gets hard-cut
	found := false
	for _, c := range chunks {
		if c == "aa\nbb" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'aa\\nbb' chunk, got: %v", chunks)
	}
}
