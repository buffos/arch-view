package processprotocol

import (
	"bufio"
	"bytes"
	"os"
	"testing"
)

func TestReadBoundedLinePreservesBytesReturnedWithClosedPipe(t *testing.T) {
	reader := bufio.NewReader(&closedAfterDataReader{data: []byte(`{"type":"done"}`)})
	line, err := readBoundedLine(reader, DefaultMaxFrameBytes)
	if err != nil {
		t.Fatalf("read line: %v", err)
	}
	if !bytes.Equal(line, []byte(`{"type":"done"}`)) {
		t.Fatalf("line = %q", line)
	}
}

type closedAfterDataReader struct {
	data []byte
	done bool
}

func (r *closedAfterDataReader) Read(target []byte) (int, error) {
	if r.done {
		return 0, os.ErrClosed
	}
	r.done = true
	return copy(target, r.data), os.ErrClosed
}
