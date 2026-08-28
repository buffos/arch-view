package processprotocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Decoder reads one JSON object per line and enforces a bounded line size.
// Use one Decoder for a stream; the package-level ReadFrame helper is intended
// for one-shot reads.
type Decoder struct {
	reader *bufio.Reader
	max    int
}

func NewDecoder(reader io.Reader) *Decoder {
	return NewDecoderWithLimit(reader, DefaultMaxFrameBytes)
}

func NewDecoderWithLimit(reader io.Reader, maxBytes int) *Decoder {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	if reader == nil {
		return &Decoder{max: maxBytes}
	}
	if buffered, ok := reader.(*bufio.Reader); ok {
		return &Decoder{reader: buffered, max: maxBytes}
	}
	return &Decoder{reader: bufio.NewReader(reader), max: maxBytes}
}

func (d *Decoder) ReadFrame() (Frame, error) {
	if d == nil || d.reader == nil {
		return Frame{}, io.EOF
	}
	line, err := readBoundedLine(d.reader, d.max)
	if err != nil {
		return Frame{}, err
	}
	if len(bytes.TrimSpace(line)) == 0 {
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "blank lines are not protocol frames", ErrMalformedJSON)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(line, &fields); err != nil || fields == nil {
		if err == nil {
			err = errors.New("frame must be a JSON object")
		}
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "frame must be one JSON object", err)
	}
	var frame Frame
	if err := json.Unmarshal(line, &frame); err != nil {
		return Frame{}, newProtocolError(ErrorMalformedJSON, "", "", "frame fields have invalid JSON types", err)
	}
	if err := validateFrameShape(frame, fields); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

func readBoundedLine(reader *bufio.Reader, maxBytes int) ([]byte, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(part) > 0 {
			if len(line)+len(part) > maxBytes {
				return nil, newProtocolError(ErrorFrameTooLarge, "", "", fmt.Sprintf("line exceeds %d bytes", maxBytes), ErrFrameTooLarge)
			}
			line = append(line, part...)
		}
		switch {
		case err == nil:
			return line, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(line) == 0 {
				return nil, io.EOF
			}
			return line, nil
		default:
			return nil, err
		}
	}
}

func ReadFrame(reader io.Reader) (Frame, error) {
	return NewDecoder(reader).ReadFrame()
}

func ReadFrameWithLimit(reader io.Reader, maxBytes int) (Frame, error) {
	return NewDecoderWithLimit(reader, maxBytes).ReadFrame()
}

// Encoder writes protocol frames with exactly one trailing newline per frame.
type Encoder struct {
	writer io.Writer
	max    int
}

func NewEncoder(writer io.Writer) *Encoder {
	return NewEncoderWithLimit(writer, DefaultMaxFrameBytes)
}

func NewEncoderWithLimit(writer io.Writer, maxBytes int) *Encoder {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	return &Encoder{writer: writer, max: maxBytes}
}

func (e *Encoder) WriteFrame(frame Frame) error {
	if e == nil {
		return errors.New("protocol encoder is nil")
	}
	return writeFrame(e.writer, frame, e.max)
}

func WriteFrame(writer io.Writer, frame Frame) error {
	return writeFrame(writer, frame, DefaultMaxFrameBytes)
}

func WriteFrameWithLimit(writer io.Writer, frame Frame, maxBytes int) error {
	return writeFrame(writer, frame, maxBytes)
}

func writeFrame(writer io.Writer, frame Frame, maxBytes int) error {
	if writer == nil {
		return errors.New("protocol writer is nil")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxFrameBytes
	}
	if err := ValidateFrame(frame); err != nil {
		return err
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return newProtocolError(ErrorInvalidPayload, frame.Type, frame.RequestID, "frame could not be encoded as JSON", err)
	}
	if len(data)+1 > maxBytes {
		return newProtocolError(ErrorFrameTooLarge, frame.Type, frame.RequestID, fmt.Sprintf("line exceeds %d bytes", maxBytes), ErrFrameTooLarge)
	}
	data = append(data, '\n')
	for len(data) > 0 {
		written, writeErr := writer.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if writeErr != nil {
			return writeErr
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
