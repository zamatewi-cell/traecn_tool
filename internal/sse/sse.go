package sse

import (
	"bufio"
	"io"
	"strings"
)

// Event represents an SSE event
type Event struct {
	ID    string
	Event string
	Data  string
}

// Reader reads SSE events from a stream. It is built on bufio.Reader with
// unbounded line buffering so arbitrarily long data lines survive TCP
// fragmentation and reassembly ("粘包/断包") without JSON truncation.
type Reader struct {
	br *bufio.Reader
}

// NewReader creates a new SSE reader
func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, 64*1024)}
}

// readLine reads one line, tolerating very long lines and CRLF endings.
func (r *Reader) readLine() (string, error) {
	var sb strings.Builder
	for {
		frag, err := r.br.ReadString('\n')
		sb.WriteString(frag)
		if err != nil {
			// Return what we have on EOF so a trailing unterminated
			// line is still processed.
			return strings.TrimRight(sb.String(), "\r\n"), err
		}
		if strings.HasSuffix(sb.String(), "\n") {
			return strings.TrimRight(sb.String(), "\r\n"), nil
		}
	}
}

// ReadEvent reads the next SSE event. Comment lines (":...") are skipped,
// multiple data: lines are joined with "\n" per the SSE spec.
func (r *Reader) ReadEvent() (*Event, error) {
	var evt Event
	hasData := false

	for {
		line, err := r.readLine()

		if line == "" {
			// Blank line: dispatch buffered event if any.
			if hasData {
				return &evt, nil
			}
			if err != nil {
				return nil, io.EOF
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			// Comment / keep-alive line.
			if err != nil {
				return nil, io.EOF
			}
			continue
		}

		field, value, found := strings.Cut(line, ":")
		if !found {
			// Per spec: line without colon is a field name with empty value.
			field, value = line, ""
		} else {
			value = strings.TrimPrefix(value, " ")
		}

		switch field {
		case "data":
			if hasData {
				evt.Data += "\n" + value
			} else {
				evt.Data = value
				hasData = true
			}
		case "event":
			evt.Event = value
		case "id":
			evt.ID = value
		}

		if err != nil {
			if hasData {
				return &evt, nil
			}
			return nil, io.EOF
		}
	}
}

// Writer writes SSE events to a stream
type Writer struct {
	w io.Writer
}

// NewWriter creates a new SSE writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteEvent writes an SSE data event
func (w *Writer) WriteEvent(data string) error {
	_, err := io.WriteString(w.w, "data: "+data+"\n\n")
	return err
}

// WriteNamedEvent writes an SSE event with an explicit event: field.
func (w *Writer) WriteNamedEvent(event, data string) error {
	_, err := io.WriteString(w.w, "event: "+event+"\ndata: "+data+"\n\n")
	return err
}

// WriteDone writes the [DONE] event
func (w *Writer) WriteDone() error {
	_, err := io.WriteString(w.w, "data: [DONE]\n\n")
	return err
}
