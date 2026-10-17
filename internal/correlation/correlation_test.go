package correlation

import (
	"context"
	"testing"
	"time"
)

func TestNewIsValidAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id := New()
		if !Valid(id) {
			t.Fatalf("invalid id %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestValid(t *testing.T) {
	cases := map[string]bool{
		"crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T": true,
		"crosswatc-req-01JCA9S8G4B2Z7":             false, // too short
		"crosswatc-req-01jca9s8g4b2z7qx3m5n6p8r9t": false, // lowercase
		"crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9U": false, // U not in alphabet
		"other-req-01JCA9S8G4B2Z7QX3M5N6P8R9T":     false,
		"":                                         false,
	}
	for id, want := range cases {
		if got := Valid(id); got != want {
			t.Errorf("Valid(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestTimeRoundTrip(t *testing.T) {
	at := time.Date(2026, 10, 17, 10, 37, 42, 0, time.UTC)
	got, ok := Time(NewAt(at))
	if !ok || !got.Equal(at) {
		t.Fatalf("got %v %v, want %v", got, ok, at)
	}
	if _, ok := Time("nope"); ok {
		t.Fatal("expected false for invalid id")
	}
}

func TestContext(t *testing.T) {
	ctx := WithID(context.Background(), "crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T")
	if id, ok := FromContext(ctx); !ok || id == "" {
		t.Fatal("missing id")
	}
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("unexpected id")
	}
}
