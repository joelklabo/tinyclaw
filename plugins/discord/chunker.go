package discord

import "strings"

// chunk splits text into chunks of at most maxSize bytes.
// It never splits inside a code fence (``` block), prefers paragraph
// boundaries (\n\n), falls back to newlines, and hard-cuts as a last resort.
func chunk(text string, maxSize int) []string {
	if text == "" {
		return nil
	}

	segments := splitSegments(text)

	var chunks []string
	for _, seg := range segments {
		if len(seg) <= maxSize {
			chunks = coalesce(chunks, seg, maxSize, "\n\n")
			continue
		}
		if isFenceBlock(seg) {
			chunks = coalesce(chunks, seg, maxSize, "\n\n")
			continue
		}
		for _, part := range splitAtNewlines(seg, maxSize) {
			chunks = coalesce(chunks, part, maxSize, "\n")
		}
	}
	return chunks
}

func splitSegments(text string) []string {
	var segments []string
	var buf strings.Builder
	inFence := false

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		isFenceLine := strings.HasPrefix(strings.TrimSpace(line), "```")

		if isFenceLine {
			if !inFence {
				if buf.Len() > 0 {
					segments = appendSegments(segments, buf.String())
					buf.Reset()
				}
				inFence = true
				buf.WriteString(line)
				continue
			}
			buf.WriteByte('\n')
			buf.WriteString(line)
			segments = append(segments, buf.String())
			buf.Reset()
			inFence = false
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	if buf.Len() > 0 {
		segments = appendSegments(segments, buf.String())
	}
	return segments
}

func appendSegments(segments []string, text string) []string {
	parts := strings.Split(text, "\n\n")
	for _, p := range parts {
		p = strings.TrimRight(p, "\n")
		if p != "" {
			segments = append(segments, p)
		}
	}
	return segments
}

func splitAtNewlines(text string, maxSize int) []string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return hardCut(text, maxSize)
	}
	var result []string
	var buf strings.Builder
	for _, line := range lines {
		needed := len(line)
		if buf.Len() > 0 {
			needed++
		}
		if buf.Len()+needed > maxSize && buf.Len() > 0 {
			result = append(result, buf.String())
			buf.Reset()
		}
		if len(line) > maxSize {
			result = append(result, hardCut(line, maxSize)...)
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	if buf.Len() > 0 {
		result = append(result, buf.String())
	}
	return result
}

func hardCut(text string, maxSize int) []string {
	var result []string
	for len(text) > maxSize {
		result = append(result, text[:maxSize])
		text = text[maxSize:]
	}
	if len(text) > 0 {
		result = append(result, text)
	}
	return result
}

func coalesce(chunks []string, seg string, maxSize int, sep string) []string {
	if len(chunks) == 0 {
		return append(chunks, seg)
	}
	prev := chunks[len(chunks)-1]
	if len(prev)+len(sep)+len(seg) <= maxSize {
		chunks[len(chunks)-1] = prev + sep + seg
		return chunks
	}
	return append(chunks, seg)
}

func isFenceBlock(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "```")
}
