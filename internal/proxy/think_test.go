package proxy

import (
	"strings"
	"testing"
)

func feedAll(s *ThinkSplitter, chunks ...string) []Segment {
	var out []Segment
	for _, c := range chunks {
		out = append(out, s.Feed(c)...)
	}
	out = append(out, s.Flush()...)
	return out
}

func joinSegments(segs []Segment) (content, reasoning string) {
	for _, seg := range segs {
		if seg.Thinking {
			reasoning += seg.Text
		} else {
			content += seg.Text
		}
	}
	return
}

func TestThinkSplitter_SimpleBlock(t *testing.T) {
	s := NewThinkSplitter()
	content, reasoning := joinSegments(feedAll(s, "<think>let me think</think>final answer"))
	if content != "final answer" {
		t.Errorf("content = %q", content)
	}
	if reasoning != "let me think" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

func TestThinkSplitter_TagSplitAcrossChunks(t *testing.T) {
	s := NewThinkSplitter()
	// The open tag, close tag and content are fragmented byte by byte.
	var segs []Segment
	full := "<think>deep thought</think>answer"
	for i := 0; i < len(full); i += 3 {
		end := i + 3
		if end > len(full) {
			end = len(full)
		}
		segs = append(segs, s.Feed(full[i:end])...)
	}
	segs = append(segs, s.Flush()...)

	content, reasoning := joinSegments(segs)
	if content != "answer" {
		t.Errorf("content = %q, want answer", content)
	}
	if reasoning != "deep thought" {
		t.Errorf("reasoning = %q, want deep thought", reasoning)
	}
}

func TestThinkSplitter_NoTags(t *testing.T) {
	s := NewThinkSplitter()
	content, reasoning := joinSegments(feedAll(s, "plain ", "text ", "only"))
	if content != "plain text only" {
		t.Errorf("content = %q", content)
	}
	if reasoning != "" {
		t.Errorf("reasoning = %q, want empty", reasoning)
	}
}

func TestThinkSplitter_UnterminatedBlock(t *testing.T) {
	s := NewThinkSplitter()
	content, reasoning := joinSegments(feedAll(s, "start <think>never closed"))
	if content != "start " {
		t.Errorf("content = %q", content)
	}
	if reasoning != "never closed" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

func TestThinkSplitter_PartialTagIsNotTag(t *testing.T) {
	s := NewThinkSplitter()
	// "<thinkx" looks like a tag prefix but is not one.
	content, _ := joinSegments(feedAll(s, "a <thinkx b"))
	if content != "a <thinkx b" {
		t.Errorf("content = %q, want literal text preserved", content)
	}
}

func TestThinkSplitter_MultipleBlocks(t *testing.T) {
	s := NewThinkSplitter()
	content, reasoning := joinSegments(feedAll(s, "<think>t1</think>c1<think>t2</think>c2"))
	if content != "c1c2" {
		t.Errorf("content = %q", content)
	}
	if reasoning != "t1t2" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

func TestSplitThinking_OneShot(t *testing.T) {
	content, reasoning := SplitThinking("<think>why</think>because")
	if content != "because" || reasoning != "why" {
		t.Errorf("SplitThinking = (%q,%q)", content, reasoning)
	}

	content, reasoning = SplitThinking("no tags here")
	if content != "no tags here" || reasoning != "" {
		t.Errorf("SplitThinking = (%q,%q)", content, reasoning)
	}
}

func TestThinkSplitter_NoContentLossAcrossRandomSplits(t *testing.T) {
	full := "pre <think>some reasoning with <angle> brackets</think> post <think>more</think>done"
	for _, step := range []int{1, 2, 5, 7, 13, 100} {
		s := NewThinkSplitter()
		var segs []Segment
		for i := 0; i < len(full); i += step {
			end := i + step
			if end > len(full) {
				end = len(full)
			}
			segs = append(segs, s.Feed(full[i:end])...)
		}
		segs = append(segs, s.Flush()...)

		var rebuilt strings.Builder
		for _, seg := range segs {
			rebuilt.WriteString(seg.Text)
		}
		content, reasoning := joinSegments(segs)
		if content != "pre  post done" {
			t.Errorf("step=%d content = %q", step, content)
		}
		if reasoning != "some reasoning with <angle> bracketsmore" {
			t.Errorf("step=%d reasoning = %q", step, reasoning)
		}
		// No byte may be lost or duplicated outside tag markers.
		want := strings.ReplaceAll(strings.ReplaceAll(full, "<think>", ""), "</think>", "")
		if rebuilt.String() != want {
			t.Errorf("step=%d rebuilt = %q, want %q", step, rebuilt.String(), want)
		}
	}
}
