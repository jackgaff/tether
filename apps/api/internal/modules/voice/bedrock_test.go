package voice

import "testing"

func TestSplitTextInputChunksRespectsByteLimit(t *testing.T) {
	t.Parallel()

	text := "abcdEFGHijkl"
	chunks := splitTextInputChunks(text, 4)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}

	expected := []string{"abcd", "EFGH", "ijkl"}
	for index, chunk := range chunks {
		if chunk != expected[index] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", index, expected[index], chunk)
		}
	}
}

func TestSplitTextInputChunksPreservesUTF8Boundaries(t *testing.T) {
	t.Parallel()

	text := "aé世b"
	chunks := splitTextInputChunks(text, 4)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	expected := []string{"aé", "世b"}
	for index, chunk := range chunks {
		if chunk != expected[index] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", index, expected[index], chunk)
		}
	}
}
