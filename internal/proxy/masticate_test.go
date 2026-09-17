package proxy

import (
	"bytes"
	"testing"
)

func TestMasticate_RoundTrip(t *testing.T) {
	plaintext := []byte(`[{"role":"user","content":[{"type":"text","text":"hi"}]}]`)

	message, pin, at, err := masticate(plaintext)
	if err != nil {
		t.Fatalf("masticate() error = %v", err)
	}
	if message == "" || len(pin) != 16 || at <= 0 {
		t.Fatalf("masticate() returned message=%d pin=%q at=%d", len(message), pin, at)
	}

	got, err := Demasticate(message, pin, at)
	if err != nil {
		t.Fatalf("Demasticate() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("round trip mismatch: got %q", got)
	}
}

func TestDemasticate_WrongPin(t *testing.T) {
	plaintext := []byte("secret")
	message, _, at, err := masticate(plaintext)
	if err != nil {
		t.Fatalf("masticate() error = %v", err)
	}
	// Zero pin => wrong key => GCM auth failure.
	if _, err := Demasticate(message, "0000000000000000", at); err == nil {
		t.Fatal("Demasticate() with wrong pin should fail")
	}
}

func TestDemasticate_WrongTimestamp(t *testing.T) {
	plaintext := []byte("secret")
	message, pin, at, err := masticate(plaintext)
	if err != nil {
		t.Fatalf("masticate() error = %v", err)
	}
	// AAD mismatch => GCM auth failure.
	if _, err := Demasticate(message, pin, at+1); err == nil {
		t.Fatal("Demasticate() with wrong requestAt should fail")
	}
}

func TestMasticate_Randomized(t *testing.T) {
	plaintext := []byte("same input")
	m1, p1, _, err1 := masticate(plaintext)
	m2, p2, _, err2 := masticate(plaintext)
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v %v", err1, err2)
	}
	if m1 == m2 || p1 == p2 {
		t.Error("masticate must randomize iv and pin per call")
	}
}
