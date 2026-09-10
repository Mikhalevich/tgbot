package tgbot

import (
	"time"
)

type options struct {
	webHookToken    string
	tracerFn        NewTracerFn
	livenessProbe   Probe
	readinessProbe  Probe
	readTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration
}

type Option func(o *options)

func (o *options) apply(opts []Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithWebHookToken sets the webhook secret token used to validate incoming webhook requests.
func WithWebHookToken(token string) Option {
	return func(o *options) {
		o.webHookToken = token
	}
}

// WithNewTracerFn sets the factory function used to create a new Tracer for each handler invocation.
func WithNewTracerFn(tracerFn NewTracerFn) Option {
	return func(o *options) {
		o.tracerFn = tracerFn
	}
}

// WithLivenessProbe sets the probe function used for the liveness HTTP health check endpoint.
func WithLivenessProbe(probe Probe) Option {
	return func(o *options) {
		o.livenessProbe = probe
	}
}

// WithReadinessProbe sets the probe function used for the readiness HTTP health check endpoint.
func WithReadinessProbe(probe Probe) Option {
	return func(o *options) {
		o.readinessProbe = probe
	}
}

// WithReadTimeout sets the read timeout for the HTTP server used in webhook mode.
func WithReadTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.readTimeout = timeout
	}
}

// WithWriteTimeout sets the write timeout for the HTTP server used in webhook mode.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = timeout
	}
}

// WithShutdownTimeout sets the maximum duration to wait for the HTTP server to shut down gracefully.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.shutdownTimeout = timeout
	}
}
