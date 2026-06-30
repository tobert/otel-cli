package otelcli

import (
	"context"

	"github.com/tobert/otel-cli/w3c/traceparent"
	"go.opentelemetry.io/contrib/propagators/envcar"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// propagator is the OpenTelemetry propagator otel-cli uses to emit context into
// child process environments. Routing emission through the standard W3C
// propagator guarantees the output is always spec-valid: this is the strict
// half of otel-cli's Postel's-law stance (accept leniently when parsing, only
// ever emit valid W3C). TRACESTATE and BAGGAGE join via a composite propagator
// in a follow-up.
var propagator propagation.TextMapPropagator = propagation.TraceContext{}

// envCarrierTraceparent reads a traceparent from the process environment using
// the OpenTelemetry env-carrier key normalization, then parses the value with
// otel-cli's lenient parser. Parsing leniency is deliberate: we accept input
// the strict propagator would reject (e.g. upper-case hex) and canonicalize it.
// An absent carrier yields an uninitialized Traceparent with no error.
func envCarrierTraceparent() (traceparent.Traceparent, error) {
	carrier := envcar.Carrier{}
	raw := carrier.Get("traceparent")
	if raw == "" {
		return traceparent.Traceparent{}, nil
	}
	return traceparent.Parse(raw)
}

// injectTraceparent writes tp into a child process environment via setEnv,
// routing through the standard W3C propagator so the emitted value is always
// spec-valid. An invalid (e.g. all-zero) span context is silently not emitted
// rather than propagating a context that downstream tooling would reject.
func injectTraceparent(tp traceparent.Traceparent, setEnv func(key, value string)) {
	sc := spanContextFromTraceparent(tp)
	if !sc.IsValid() {
		return
	}
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	propagator.Inject(ctx, &envcar.Carrier{SetEnvFunc: setEnv})
}

// spanContextFromTraceparent converts otel-cli's internal traceparent into the
// SDK trace.SpanContext the propagator operates on.
func spanContextFromTraceparent(tp traceparent.Traceparent) trace.SpanContext {
	var traceID trace.TraceID
	var spanID trace.SpanID
	copy(traceID[:], tp.TraceId)
	copy(spanID[:], tp.SpanId)

	flags := trace.TraceFlags(0)
	if tp.Sampling {
		flags = trace.FlagsSampled
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	})
}
