package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// readEvents reads a text/event-stream and calls fn with each event: lines
// "event: <name>" and "data: <json>" (several data lines are joined with a
// newline), ended by a blank line. Comments (": ping") and the id and retry
// fields are skipped. It returns fn's first error, a read error (as a
// transport error), or nil at the end of the stream.
func readEvents(r io.Reader, fn func(Event) error) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var name string
	var data bytes.Buffer
	hasData := false
	dispatch := func() error {
		if !hasData {
			name = ""
			return nil
		}
		ev := Event{Name: name, Data: bytes.Clone(data.Bytes())}
		if ev.Name == "" {
			ev.Name = "message"
		}
		name, hasData = "", false
		data.Reset()
		return fn(ev)
	}
	for {
		line, err := readLine(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// A stream may end without a final blank line.
				if len(line) == 0 {
					return dispatch()
				}
			} else {
				return &transportError{err: fmt.Errorf("read the stream: %w", err)}
			}
		}
		switch {
		case len(line) == 0:
			if derr := dispatch(); derr != nil {
				return derr
			}
		case line[0] == ':':
			// a comment: the keep-alive ping
		default:
			field, value, _ := bytes.Cut(line, []byte(":"))
			value = bytes.TrimPrefix(value, []byte(" "))
			switch string(field) {
			case "event":
				name = string(value)
			case "data":
				if hasData {
					data.WriteByte('\n')
				}
				if data.Len()+len(value) > maxEventBytes {
					return fmt.Errorf("cere: a stream event is larger than %d bytes", maxEventBytes)
				}
				data.Write(value)
				hasData = true
			}
		}
		if errors.Is(err, io.EOF) {
			return dispatch()
		}
	}
}

// readLine reads one line without its ending (\n, \r\n or \r is not split:
// the API ends lines with \n). A line longer than maxEventBytes is an error.
func readLine(br *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, err := br.ReadSlice('\n')
		line = append(line, chunk...)
		if len(line) > maxEventBytes {
			return nil, fmt.Errorf("a stream line is longer than %d bytes", maxEventBytes)
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))
		return line, err
	}
}
