package tgbot

import (
	"context"
)

type Tracer interface {
	Before(ctx context.Context, pattern string, msg BotMessage) context.Context
	OnSuccess(ctx context.Context)
	OnError(ctx context.Context, err error)
	After(ctx context.Context)
}

type NewTracerFn func() Tracer
