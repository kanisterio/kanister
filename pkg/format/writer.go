package format

import (
	"io"
)

// Writer formats strings before writing them to its underlying writer.
type Writer struct {
	W         io.Writer
	Pod       string
	Container string
}

// Write formats p and write it to w's writer.
func (w *Writer) Write(p []byte) (int, error) {
	LogTo(w.W, w.Pod, w.Container, string(p))
	return len(p), nil
}
