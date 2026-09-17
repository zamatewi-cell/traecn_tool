package sse

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestEvent_Struct(t *testing.T) {
	evt := Event{
		ID:    "123",
		Event: "message",
		Data:  "test data",
	}

	if evt.ID != "123" {
		t.Errorf("Event.ID = %v, want 123", evt.ID)
	}
	if evt.Event != "message" {
		t.Errorf("Event.Event = %v, want message", evt.Event)
	}
	if evt.Data != "test data" {
		t.Errorf("Event.Data = %v, want test data", evt.Data)
	}
}

func TestReader_NewReader(t *testing.T) {
	data := strings.NewReader("data: test\n\n")
	reader := NewReader(data)

	if reader == nil {
		t.Fatal("NewReader() returned nil")
	}
	if reader.br == nil {
		t.Error("NewReader() bufio reader is nil")
	}
}

func TestReader_ReadEvent_SingleEvent(t *testing.T) {
	input := "data: {\"message\": \"hello\"}\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt == nil {
		t.Fatal("ReadEvent() returned nil event")
	}
	if evt.Data != "{\"message\": \"hello\"}" {
		t.Errorf("ReadEvent() Data = %v, want {\"message\": \"hello\"}", evt.Data)
	}
}

func TestReader_ReadEvent_MultiLine(t *testing.T) {
	input := "data: line1\ndata: line2\ndata: line3\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "line1\nline2\nline3" {
		t.Errorf("ReadEvent() Data = %v, want line1\\nline2\\nline3", evt.Data)
	}
}

func TestReader_ReadEvent_WithEventAndID(t *testing.T) {
	input := "id: 456\nevent: message\ndata: test data\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.ID != "456" {
		t.Errorf("ReadEvent() ID = %v, want 456", evt.ID)
	}
	if evt.Event != "message" {
		t.Errorf("ReadEvent() Event = %v, want message", evt.Event)
	}
	if evt.Data != "test data" {
		t.Errorf("ReadEvent() Data = %v, want test data", evt.Data)
	}
}

func TestReader_ReadEvent_EOF(t *testing.T) {
	reader := NewReader(strings.NewReader(""))

	_, err := reader.ReadEvent()
	if err != io.EOF {
		t.Errorf("ReadEvent() error = %v, want io.EOF", err)
	}
}

func TestReader_ReadEvent_MultipleEvents(t *testing.T) {
	input := "data: event1\n\n\ndata: event2\n\n"
	reader := NewReader(strings.NewReader(input))

	// Read first event
	evt1, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() first error = %v", err)
	}
	if evt1.Data != "event1" {
		t.Errorf("ReadEvent() first Data = %v, want event1", evt1.Data)
	}

	// Read second event
	evt2, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() second error = %v", err)
	}
	if evt2.Data != "event2" {
		t.Errorf("ReadEvent() second Data = %v, want event2", evt2.Data)
	}
}

func TestWriter_NewWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	if writer == nil {
		t.Fatal("NewWriter() returned nil")
	}
	if writer.w == nil {
		t.Error("NewWriter() writer is nil")
	}
}

func TestWriter_WriteEvent(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteEvent("test data")
	if err != nil {
		t.Fatalf("WriteEvent() error = %v", err)
	}

	expected := "data: test data\n\n"
	if buf.String() != expected {
		t.Errorf("WriteEvent() output = %v, want %v", buf.String(), expected)
	}
}

func TestWriter_WriteDone(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteDone()
	if err != nil {
		t.Fatalf("WriteDone() error = %v", err)
	}

	expected := "data: [DONE]\n\n"
	if buf.String() != expected {
		t.Errorf("WriteDone() output = %v, want %v", buf.String(), expected)
	}
}

func TestWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	// Write multiple events
	events := []string{"event1", "event2", "event3"}
	for _, event := range events {
		err := writer.WriteEvent(event)
		if err != nil {
			t.Fatalf("WriteEvent(%v) error = %v", event, err)
		}
	}

	// Write done
	err := writer.WriteDone()
	if err != nil {
		t.Fatalf("WriteDone() error = %v", err)
	}

	// Verify output
	output := buf.String()
	if !strings.Contains(output, "data: event1\n\n") {
		t.Error("WriteEvent() missing event1")
	}
	if !strings.Contains(output, "data: event2\n\n") {
		t.Error("WriteEvent() missing event2")
	}
	if !strings.Contains(output, "data: event3\n\n") {
		t.Error("WriteEvent() missing event3")
	}
	if !strings.Contains(output, "data: [DONE]\n\n") {
		t.Error("WriteDone() missing [DONE]")
	}
}

func TestReaderWriter_RoundTrip(t *testing.T) {
	// Write events
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	events := []string{"message1", "message2", "message3"}
	for _, event := range events {
		err := writer.WriteEvent(event)
		if err != nil {
			t.Fatalf("WriteEvent() error = %v", err)
		}
	}

	// Read back events
	reader := NewReader(&buf)
	for i, expected := range events {
		evt, err := reader.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent() error = %v", err)
		}
		if evt.Data != expected {
			t.Errorf("ReadEvent() event %d Data = %v, want %v", i, evt.Data, expected)
		}
	}
}

func TestReader_ReadEvent_EmptyLines(t *testing.T) {
	// Multiple empty lines should be skipped
	input := "\n\n\ndata: actual data\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "actual data" {
		t.Errorf("ReadEvent() Data = %v, want actual data", evt.Data)
	}
}

func TestReader_ReadEvent_MalformedLines(t *testing.T) {
	// Lines without prefix should be ignored
	input := "data: valid\ninvalid line\nno prefix\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "valid" {
		t.Errorf("ReadEvent() Data = %v, want valid", evt.Data)
	}
}

func TestReader_ReadEvent_VeryLongLine(t *testing.T) {
	// A single data line far beyond bufio.Scanner's default 64KB limit
	// must survive intact (no JSON truncation).
	longPayload := strings.Repeat("x", 256*1024)
	input := "data: " + longPayload + "\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != longPayload {
		t.Errorf("ReadEvent() data length = %d, want %d", len(evt.Data), len(longPayload))
	}
}

func TestReader_ReadEvent_CRLF(t *testing.T) {
	input := "data: line1\r\ndata: line2\r\n\r\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "line1\nline2" {
		t.Errorf("ReadEvent() Data = %q, want line1\\nline2", evt.Data)
	}
}

// slowReader drips a few bytes per Read to simulate TCP fragmentation.
type slowReader struct {
	data []byte
	step int
}

func (s *slowReader) Read(p []byte) (int, error) {
	if len(s.data) == 0 {
		return 0, io.EOF
	}
	n := s.step
	if n > len(s.data) {
		n = len(s.data)
	}
	copy(p, s.data[:n])
	s.data = s.data[n:]
	return n, nil
}

func TestReader_ReadEvent_FragmentedDelivery(t *testing.T) {
	payload := `{"choices":[{"delta":{"content":"hello world"}}]}`
	input := "data: " + payload + "\n\ndata: [DONE]\n\n"
	reader := NewReader(&slowReader{data: []byte(input), step: 3})

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != payload {
		t.Errorf("ReadEvent() Data = %q, want %q", evt.Data, payload)
	}

	evt, err = reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() second error = %v", err)
	}
	if evt.Data != "[DONE]" {
		t.Errorf("ReadEvent() Data = %q, want [DONE]", evt.Data)
	}
}

func TestReader_ReadEvent_CommentKeepAlive(t *testing.T) {
	input := ": keep-alive\n\n: ping\ndata: real\n\n"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "real" {
		t.Errorf("ReadEvent() Data = %q, want real", evt.Data)
	}
}

func TestReader_ReadEvent_TrailingLineWithoutNewline(t *testing.T) {
	// Upstream closed connection right after a data line without \n\n.
	input := "data: tail"
	reader := NewReader(strings.NewReader(input))

	evt, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent() error = %v", err)
	}
	if evt.Data != "tail" {
		t.Errorf("ReadEvent() Data = %q, want tail", evt.Data)
	}
}

func TestWriter_WriteNamedEvent(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)
	if err := writer.WriteNamedEvent("message_start", "{}"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "event: message_start\ndata: {}\n\n" {
		t.Errorf("WriteNamedEvent() = %q", buf.String())
	}
}
