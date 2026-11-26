package demo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

const inputChunkSize = 256

// inputReader reads user input in a way that can be cancelled through a
// context. It only reads from the source while a caller waits for input, so
// that step commands sharing the same stdin still receive their input. A read
// that is still pending after a cancellation keeps waiting for the next input
// in the background, which then goes to the next caller.
type inputReader struct {
	src     io.Reader
	pending []byte
	results chan readResult
	reading bool
}

type readResult struct {
	data []byte
	err  error
}

func newInputReader(src io.Reader) *inputReader {
	return &inputReader{
		src:     src,
		pending: nil,
		results: make(chan readResult, 1),
		reading: false,
	}
}

// read returns the next chunk of input. A read that is still in progress when
// the context gets cancelled is picked up by the next call.
func (i *inputReader) read(ctx context.Context, size int) ([]byte, error) {
	if !i.reading {
		i.reading = true

		go func() {
			buf := make([]byte, size)

			n, err := i.src.Read(buf)
			i.results <- readResult{data: buf[:n], err: err}
		}()
	}

	select {
	case res := <-i.results:
		i.reading = false

		return res.data, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for input: %w", ctx.Err())
	}
}

// keypress waits for a single keypress in raw mode. All bytes that arrive
// together, like an escape sequence or pasted text, count as one keypress.
func (i *inputReader) keypress(ctx context.Context) error {
	data := i.pending
	i.pending = nil

	for len(data) == 0 {
		chunk, err := i.read(ctx, inputChunkSize)
		data = chunk

		if err != nil && len(data) == 0 {
			return fmt.Errorf("unable to read keypress: %w", err)
		}
	}

	if bytes.IndexByte(data, ctrlC) >= 0 {
		return errInterrupt
	}

	return nil
}

// line waits for a newline. A final line without a newline counts as well. It
// reads byte by byte, so that piped input after the newline is left for the
// step commands.
func (i *inputReader) line(ctx context.Context) error {
	for {
		if idx := bytes.IndexByte(i.pending, '\n'); idx >= 0 {
			i.pending = i.pending[idx+1:]

			return nil
		}

		chunk, err := i.read(ctx, 1)
		i.pending = append(i.pending, chunk...)

		if err == nil {
			continue
		}

		if bytes.IndexByte(i.pending, '\n') >= 0 {
			continue
		}

		if errors.Is(err, io.EOF) && len(i.pending) > 0 {
			i.pending = nil

			return nil
		}

		return fmt.Errorf("unable to read newline: %w", err)
	}
}
