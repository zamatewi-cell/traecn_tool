// Package protect implements the anti-abuse layer: context projection
// (payload compression), sensitive-word smoothing and rate limiting.
package protect

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Config tunes the protection layer.
type Config struct {
	// MaxPayloadBytes is the total budget for the serialized message list
	// (0 = unlimited). Prevents 413 Payload Too Large / 504 timeouts.
	MaxPayloadBytes int `json:"max_payload_bytes,omitempty"`
	// MaxMessageBytes is the per-message cap before head/tail truncation
	// (0 = unlimited).
	MaxMessageBytes int `json:"max_message_bytes,omitempty"`
	// FilterEnabled turns on sensitive-word smoothing.
	FilterEnabled bool `json:"filter_enabled,omitempty"`
	// FilterReplacements maps sensitive words to smooth replacements.
	FilterReplacements map[string]string `json:"filter_replacements,omitempty"`
	// MaxConcurrent caps concurrent upstream requests (0 = unlimited).
	MaxConcurrent int `json:"max_concurrent,omitempty"`
	// MinIntervalMs enforces a minimum interval between upstream requests.
	MinIntervalMs int `json:"min_interval_ms,omitempty"`
}

// DefaultConfig returns sane defaults matching the observed upstream
// tolerance (captured chat payloads were ~300KB).
func DefaultConfig() Config {
	return Config{
		MaxPayloadBytes: 512 * 1024,
		MaxMessageBytes: 64 * 1024,
		FilterEnabled:   false,
		MaxConcurrent:   4,
		MinIntervalMs:   0,
	}
}

// ---------------------------------------------------------------------------
// Context Projector
// ---------------------------------------------------------------------------

// stackTraceLine matches common stack-trace lines (Go, Python, Java, JS).
var stackTraceLine = regexp.MustCompile(`^\s*(at\s+.*\(.*:\d+(:\d+)?\)\s*$|File\s+".*",\s+line\s+\d+|\S+\.(go|java|py|js|ts):\d+(:\d+)?|goroutine\s+\d+\s+\[)`)

// MessageView is the minimal message shape the projector needs (kept
// import-free to avoid cycles; callers convert to their message type).
type MessageView struct {
	Role    string
	Content string
}

// Projector compresses oversized message histories so the upstream payload
// stays within budget: stack traces are stripped, oversized messages are
// head/tail truncated, and oldest non-system messages shrink first when the
// total budget is exceeded.
type Projector struct {
	MaxPayloadBytes int
	MaxMessageBytes int
}

// NewProjector creates a projector (0 = unlimited for either budget).
func NewProjector(maxPayloadBytes, maxMessageBytes int) *Projector {
	return &Projector{MaxPayloadBytes: maxPayloadBytes, MaxMessageBytes: maxMessageBytes}
}

const truncationMarker = "\n...[middle truncated]...\n"

// StripStackTraces removes stack-trace lines from text while keeping the
// error headline and surrounding context.
func StripStackTraces(text string) string {
	if text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	removed := 0
	for _, ln := range lines {
		if stackTraceLine.MatchString(ln) {
			removed++
			continue
		}
		kept = append(kept, ln)
	}
	if removed > 0 {
		kept = append(kept, "["+strconv.Itoa(removed)+" stack trace lines removed]")
	}
	return strings.Join(kept, "\n")
}

// trimToRuneEdge shortens s so it does not end mid-rune.
func trimToRuneEdge(s string) string {
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r == utf8.RuneError && size <= 1 {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}

// skipToRuneEdge drops leading bytes that are mid-rune continuations.
func skipToRuneEdge(s string) string {
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size <= 1 {
			s = s[1:]
			continue
		}
		break
	}
	return s
}

// TruncateHeadTail keeps the first head and last tail bytes of text with a
// marker in between, never splitting a UTF-8 rune at the cut points.
func TruncateHeadTail(text string, budget int) string {
	if budget <= 0 || len(text) <= budget {
		return text
	}
	tail := budget / 4
	head := budget - tail - len(truncationMarker)
	if head <= 0 {
		// Too small for head+marker+tail: hard cut at the budget.
		return trimToRuneEdge(text[:budget])
	}
	h := trimToRuneEdge(text[:head])
	t := skipToRuneEdge(text[len(text)-tail:])
	return h + truncationMarker + t
}

// Project applies stack-trace stripping, per-message caps and the total
// budget to a message list. System messages and the last two messages are
// shrunk last (they carry the most important context).
func (p *Projector) Project(messages []MessageView) []MessageView {
	if p == nil {
		return messages
	}

	out := make([]MessageView, len(messages))
	copy(out, messages)

	// 1. Stack-trace stripping on long tool outputs.
	for i := range out {
		if len(out[i].Content) > 4096 {
			out[i].Content = StripStackTraces(out[i].Content)
		}
	}

	// 2. Per-message cap.
	if p.MaxMessageBytes > 0 {
		for i := range out {
			out[i].Content = TruncateHeadTail(out[i].Content, p.MaxMessageBytes)
		}
	}

	// 3. Total budget: shrink oldest non-system messages (except the last
	// two) in passes of decreasing size until the estimate fits.
	if p.MaxPayloadBytes > 0 {
		totalLen := func() int {
			t := 0
			for _, m := range out {
				t += len(m.Content) + 32 // role + framing estimate
			}
			return t
		}
		for _, shrink := range []int{2048, 1024, 512, 256} {
			if totalLen() <= p.MaxPayloadBytes {
				break
			}
			for i := 0; i < len(out)-2 && totalLen() > p.MaxPayloadBytes; i++ {
				if out[i].Role == "system" || len(out[i].Content) <= shrink {
					continue
				}
				out[i].Content = TruncateHeadTail(out[i].Content, shrink)
			}
		}
	}

	return out
}

// ---------------------------------------------------------------------------
// Sensitive-word filter
// ---------------------------------------------------------------------------

// Filter performs smooth sensitive-word replacement on outbound prompts.
type Filter struct {
	replacements map[string]string
	enabled      bool
}

// NewFilter builds a filter; disabled filters pass text through unchanged.
func NewFilter(enabled bool, replacements map[string]string) *Filter {
	return &Filter{enabled: enabled, replacements: replacements}
}

// Apply replaces all configured sensitive words in text.
func (f *Filter) Apply(text string) string {
	if f == nil || !f.enabled || text == "" {
		return text
	}
	for word, repl := range f.replacements {
		if word == "" {
			continue
		}
		text = strings.ReplaceAll(text, word, repl)
	}
	return text
}

// ---------------------------------------------------------------------------
// Rate limiter (semaphore + minimum interval)
// ---------------------------------------------------------------------------

// Limiter throttles upstream requests to avoid tripping account risk
// control: a semaphore caps concurrency and a minimum interval spaces out
// request starts.
type Limiter struct {
	sem     chan struct{}
	mu      sync.Mutex
	lastAt  time.Time
	minWait time.Duration
}

// NewLimiter creates a limiter (maxConcurrent <= 0 = unlimited,
// minInterval <= 0 = no spacing).
func NewLimiter(maxConcurrent int, minInterval time.Duration) *Limiter {
	l := &Limiter{minWait: minInterval}
	if maxConcurrent > 0 {
		l.sem = make(chan struct{}, maxConcurrent)
	}
	return l
}

// Acquire blocks until the request may proceed; the returned release
// function must be called when the upstream request completes.
func (l *Limiter) Acquire() (release func()) {
	rel, _ := l.AcquireContext(context.Background())
	return rel
}

// AcquireContext blocks until the request may proceed or ctx is cancelled;
// the returned release function must be called when the upstream request completes.
func (l *Limiter) AcquireContext(ctx context.Context) (release func(), err error) {
	if l == nil {
		return func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if l.sem != nil {
		select {
		case l.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if l.minWait > 0 {
		l.mu.Lock()
		d := time.Since(l.lastAt)
		if d < l.minWait {
			wait := l.minWait - d
			l.mu.Unlock()
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				if l.sem != nil {
					<-l.sem
				}
				return nil, ctx.Err()
			}
			l.mu.Lock()
		}
		l.lastAt = time.Now()
		l.mu.Unlock()
	}
	return func() {
		if l.sem != nil {
			<-l.sem
		}
	}, nil
}
