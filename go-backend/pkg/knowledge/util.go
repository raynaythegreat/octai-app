// OctAi - Knowledge Base Utilities
package knowledge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// newID generates a short random hex ID.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "kb-" + hex.EncodeToString(b)
}

// splitIntoChunks splits text into overlapping word-based chunks.
// chunkSize is the target size in characters, overlap is the overlap in characters.
func splitIntoChunks(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var buf strings.Builder
	i := 0
	for i < len(words) {
		buf.Reset()
		j := i
		for j < len(words) && buf.Len() < chunkSize {
			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(words[j])
			j++
		}
		chunks = append(chunks, buf.String())

		// Advance by (chunkSize - overlap) worth of words.
		advanceChars := chunkSize - overlap
		advancedChars := 0
		for i < j && advancedChars < advanceChars {
			advancedChars += len(words[i]) + 1
			i++
		}
		if i == j {
			break // safety: don't loop forever
		}
	}
	return chunks
}

// marshalMetadata serializes metadata to JSON, returning "" on nil/error.
func marshalMetadata(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// unmarshalMetadata deserializes a JSON metadata string.
func unmarshalMetadata(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	_ = json.Unmarshal([]byte(s), &m)
	return m
}
