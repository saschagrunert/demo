package demo

import (
	"context"
	"errors"
	"io"
	"testing"
)

// chunkReader returns at most one chunk per Read call, like a terminal in raw
// mode.
type chunkReader struct {
	chunks []string
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}

	n := copy(p, c.chunks[0])

	c.chunks[0] = c.chunks[0][n:]
	if c.chunks[0] == "" {
		c.chunks = c.chunks[1:]
	}

	return n, nil
}

func TestKeypressConsumesEscapeSequences(t *testing.T) {
	t.Parallel()

	input := newInputReader(&chunkReader{chunks: []string{"\x1b[A", "x"}})

	for range 2 {
		if err := input.keypress(t.Context()); err != nil {
			t.Fatalf("keypress: %v", err)
		}
	}

	if err := input.keypress(t.Context()); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after two keypresses, got %v", err)
	}
}

func TestKeypressDetectsCtrlC(t *testing.T) {
	t.Parallel()

	input := newInputReader(&chunkReader{chunks: []string{"a\x03"}})

	if err := input.keypress(t.Context()); !errors.Is(err, errInterrupt) {
		t.Fatalf("expected interrupt, got %v", err)
	}
}

func TestLineSplitsChunks(t *testing.T) {
	t.Parallel()

	input := newInputReader(&chunkReader{chunks: []string{"first\nsec", "ond\nlast"}})

	for range 3 {
		if err := input.line(t.Context()); err != nil {
			t.Fatalf("line: %v", err)
		}
	}

	if err := input.line(t.Context()); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after three lines, got %v", err)
	}
}

func TestReadResumesAfterCancellation(t *testing.T) {
	t.Parallel()

	reader, writer := io.Pipe()
	input := newInputReader(reader)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := input.line(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}

	go func() {
		_, _ = writer.Write([]byte("later\n"))
	}()

	if err := input.line(t.Context()); err != nil {
		t.Fatalf("line after cancellation: %v", err)
	}
}
