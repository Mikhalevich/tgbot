package tgbot

import (
	"time"
)

type options struct {
	webHookToken     string
	tracerFn         NewTracerFn
	livenessProbe    Probe
	readinessProbe   Probe
	readTimeout      time.Duration
	writeTimeout     time.Duration
	shoutdownTimeout time.Duration
}

type Option func(o *options)

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func WithWebHookToken(token string) Option {
	return func(o *options) {
		o.webHookToken = token
	}
}

func WithNewTracerFn(tracerFn NewTracerFn) Option {
	return func(o *options) {
		o.tracerFn = tracerFn
	}
}

func WithLivenessProbe(probe Probe) Option {
	return func(o *options) {
		o.livenessProbe = probe
	}
}

func WithRedinessProbe(probe Probe) Option {
	return func(o *options) {
		o.readinessProbe = probe
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.readTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = timeout
	}
}

func WithShoutdownTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.shoutdownTimeout = timeout
	}
}
