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
// newline), ended by a blank line. An "id: <id>" line sets the id of that
// event and of the ones after it, until the next id line (as the stream's
// last event id). Comments (": ping") and the retry field are skipped. It
// returns fn's first error, a read error (as a transport error), or nil at
// the end of the stream. A stream that ends in the middle of a line was cut:
// the event that line belongs to is incomplete, so it is dropped and the cut
// is a transport error (Follow opens the stream again).
func readEvents(r io.Reader, fn func(Event) error) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var name, lastID string
	var data bytes.Buffer
	hasData := false
	dispatch := func() error {
		if !hasData {
			name = ""
			return nil
		}
		ev := Event{Name: name, ID: lastID, Data: bytes.Clone(data.Bytes())}
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
			if !errors.Is(err, io.EOF) {
				return &transportError{err: fmt.Errorf("read the stream: %w", err)}
			}
			if len(line) > 0 {
				return &transportError{err: errors.New("read the stream: it ended in the middle of a line")}
			}
			// A stream may end without a final blank line after its last
			// complete line.
			return dispatch()
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
			case "id":
				if !bytes.ContainsRune(value, 0) {
					lastID = string(value)
				}
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
