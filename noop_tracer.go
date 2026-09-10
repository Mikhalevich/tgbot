package tgbot

import (
	"context"
)

type NoopTracer struct {
}

// NewNoopTracer returns a no-op Tracer implementation that performs no tracing actions.
func NewNoopTracer() NoopTracer {
	return NoopTracer{}
}

func (n NoopTracer) Before(ctx context.Context, pattern string, msg BotMessage) context.Context {
	return ctx
}

func (n NoopTracer) OnSuccess(ctx context.Context) {
}

func (n NoopTracer) OnError(ctx context.Context, err error) {
}

func (n NoopTracer) After(ctx context.Context) {
}
