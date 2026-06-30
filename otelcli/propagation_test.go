package otelcli

import (
	"testing"
)

// TestEnvCarrierTraceparentGraceful proves Postel's law on the read side: we
// accept input that the strict W3C propagator would reject (upper-case hex)
// and downcase it into our canonical representation.
func TestEnvCarrierTraceparentGraceful(t *testing.T) {
	t.Setenv("TRACEPARENT", "00-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA-BBBBBBBBBBBBBBBB-01")

	tp, err := envCarrierTraceparent()
	if err != nil {
		t.Fatalf("envCarrierTraceparent() returned an unexpected error: %s", err)
	}
	if !tp.Initialized {
		t.Fatal("expected an initialized traceparent from an upper-case TRACEPARENT")
	}
	if got := tp.TraceIdString(); got != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("trace id not downcased, got %q", got)
	}
	if got := tp.SpanIdString(); got != "bbbbbbbbbbbbbbbb" {
		t.Errorf("span id not downcased, got %q", got)
	}
	if !tp.Sampling {
		t.Error("expected the sampling flag to be set")
	}
}

// TestEnvCarrierTraceparentEmpty proves an absent carrier yields an
// uninitialized traceparent with no error, matching the historical contract.
func TestEnvCarrierTraceparentEmpty(t *testing.T) {
	t.Setenv("TRACEPARENT", "")

	tp, err := envCarrierTraceparent()
	if err != nil {
		t.Fatalf("envCarrierTraceparent() returned an unexpected error: %s", err)
	}
	if tp.Initialized {
		t.Error("expected an uninitialized traceparent when TRACEPARENT is unset")
	}
}

// TestInjectTraceparentStrict proves the emit side only ever produces
// spec-valid, lower-case W3C, even when the source was parsed from upper-case.
func TestInjectTraceparentStrict(t *testing.T) {
	t.Setenv("TRACEPARENT", "00-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA-BBBBBBBBBBBBBBBB-01")
	tp, err := envCarrierTraceparent()
	if err != nil {
		t.Fatalf("setup parse failed: %s", err)
	}

	got := map[string]string{}
	injectTraceparent(tp, func(k, v string) { got[k] = v })

	want := "00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01"
	if got["TRACEPARENT"] != want {
		t.Errorf("expected child TRACEPARENT=%q, got %q (full map: %v)", want, got["TRACEPARENT"], got)
	}
}

// TestInjectTraceparentSkipsInvalid proves we never emit a traceparent that
// would fail otel spec validation (e.g. an all-zero context).
func TestInjectTraceparentSkipsInvalid(t *testing.T) {
	// --tp-ignore-env yields a zeroed-but-Initialized traceparent, the realistic
	// "initialized yet invalid" case we must not propagate.
	zeroed := DefaultConfig().WithTraceparentIgnoreEnv(true).LoadTraceparent()
	called := false
	injectTraceparent(zeroed, func(k, v string) { called = true })
	if called {
		t.Error("an all-zero/invalid traceparent must not be emitted into the child env")
	}
}
