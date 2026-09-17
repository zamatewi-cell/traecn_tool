package proxy

import "strings"

// Think tags emitted by reasoning models inside the text stream.
const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
)

// Segment is a classified piece of model output.
type Segment struct {
	Text     string
	Thinking bool
}

// ThinkSplitter is a stateful stream classifier that separates
// <think>...</think> reasoning from the final answer. Tag boundaries may
// be split arbitrarily across Feed() calls; unclassifiable tail bytes are
// held back until more input arrives or Flush() is called.
type ThinkSplitter struct {
	buf     strings.Builder
	inThink bool
}

// NewThinkSplitter creates a splitter (initially outside a think block).
func NewThinkSplitter() *ThinkSplitter {
	return &ThinkSplitter{}
}

// Feed consumes one text increment and returns the classifiable segments.
func (s *ThinkSplitter) Feed(text string) []Segment {
	s.buf.WriteString(text)
	return s.drain(false)
}

// Flush returns any remaining buffered content, classifying an unterminated
// think block as reasoning.
func (s *ThinkSplitter) Flush() []Segment {
	return s.drain(true)
}

// drain extracts all complete segments; when flushing, the tail is emitted
// unconditionally, otherwise a possible partial-tag suffix is held back.
func (s *ThinkSplitter) drain(flush bool) []Segment {
	var out []Segment

	emit := func(text string, thinking bool) {
		if text != "" {
			out = append(out, Segment{Text: text, Thinking: thinking})
		}
	}

	for {
		data := s.buf.String()
		if s.inThink {
			idx := strings.Index(data, thinkCloseTag)
			if idx < 0 {
				if flush {
					emit(data, true)
					s.buf.Reset()
					return out
				}
				// Hold back a suffix that could be the start of "</think>".
				hold := partialTagSuffix(data, thinkCloseTag)
				emit(data[:len(data)-hold], true)
				s.buf.Reset()
				s.buf.WriteString(data[len(data)-hold:])
				return out
			}
			emit(data[:idx], true)
			s.buf.Reset()
			s.buf.WriteString(data[idx+len(thinkCloseTag):])
			s.inThink = false
			continue
		}

		idx := strings.Index(data, thinkOpenTag)
		if idx < 0 {
			if flush {
				emit(data, false)
				s.buf.Reset()
				return out
			}
			hold := partialTagSuffix(data, thinkOpenTag)
			emit(data[:len(data)-hold], false)
			s.buf.Reset()
			s.buf.WriteString(data[len(data)-hold:])
			return out
		}
		emit(data[:idx], false)
		s.buf.Reset()
		s.buf.WriteString(data[idx+len(thinkOpenTag):])
		s.inThink = true
	}
}

// partialTagSuffix returns the length of the longest suffix of data that is
// a proper prefix of tag (i.e. a tag possibly split across chunks).
func partialTagSuffix(data, tag string) int {
	max := len(tag) - 1
	if len(data) < max {
		max = len(data)
	}
	for n := max; n > 0; n-- {
		if strings.HasSuffix(data, tag[:n]) {
			return n
		}
	}
	return 0
}

// SplitThinking is the one-shot variant for complete (non-streaming) text.
func SplitThinking(text string) (content, reasoning string) {
	s := NewThinkSplitter()
	for _, seg := range s.Feed(text) {
		if seg.Thinking {
			reasoning += seg.Text
		} else {
			content += seg.Text
		}
	}
	for _, seg := range s.Flush() {
		if seg.Thinking {
			reasoning += seg.Text
		} else {
			content += seg.Text
		}
	}
	return content, reasoning
}

// WrapStreamHandler decorates a StreamHandler so EventText payloads
// carrying inline <think>...</think> blocks are re-classified into
// EventReasoning / EventText segments. The returned flush drains any
// held-back tail (call it when the stream ends without a finish event).
func WrapStreamHandler(handle StreamHandler) (wrapped StreamHandler, flush func() error) {
	sp := NewThinkSplitter()

	emit := func(seg Segment) error {
		if seg.Thinking {
			return handle(&StreamEvent{Type: EventReasoning, Reasoning: seg.Text})
		}
		return handle(&StreamEvent{Type: EventText, Text: seg.Text})
	}

	flush = func() error {
		for _, seg := range sp.Flush() {
			if err := emit(seg); err != nil {
				return err
			}
		}
		return nil
	}

	wrapped = func(evt *StreamEvent) error {
		if evt.Type == EventText {
			for _, seg := range sp.Feed(evt.Text) {
				if err := emit(seg); err != nil {
					return err
				}
			}
			return nil
		}
		if evt.Type == EventFinish || evt.Type == EventError {
			if err := flush(); err != nil {
				return err
			}
		}
		return handle(evt)
	}
	return wrapped, flush
}
