# tgbot – Telegram Bot API wrapper for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/Mikhalevich/tgbot)](https://goreportcard.com/report/github.com/Mikhalevich/tgbot)
[![GoDoc](https://godoc.org/github.com/Mikhalevich/tgbot?status.svg)](https://pkg.go.dev/github.com/Mikhalevich/tgbot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight, idiomatic wrapper around the official Telegram Bot API (`github.com/go-telegram/bot`). It adds convenience methods, middleware support, HTTP health‑check endpoints, and configurable tracing/profiling hooks.

## Table of Contents

| Section | Description |
|---|---|
| [Overview](#overview) | What the library does and why it exists |
| [Features](#features) | High‑level capabilities |
| [Quick start – Long‑polling](#quick-start---long-polling) | Minimal code to get a bot running in polling mode |
| [Quick start – Webhook](#quick-start---webhook) | Minimal code to run the bot behind an HTTP webhook |
| [Configuration](#configuration) | YAML config file (used by the webhook CLI) |
| [Middleware](#middleware) | How to plug custom middleware |
| [Tracing & probes](#tracing--probes) | Tracer interface, no‑op implementation, liveness/readiness endpoints |
| [Command registration](#command-registration) | `AddMenuCommand`, `AddTextCommand`, default / text / callback handlers |
| [HTTP helpers](#http-helpers) | `SendMessage`, `DeleteMessage`, `/live`, `/ready` |
| [Development](#development) | Building, testing, linting, and releasing |
| [Examples](#examples) | Echo‑tracer bot and webhook CLI |
| [License](#license) | MIT License |

## Overview

`tgbot` simplifies the most common tasks when building a Telegram bot in Go:

* **Create a bot** with a token and optional options (`WithWebHookToken`, timeouts, tracers, probes).
* **Run in long‑polling mode** (`bot.Start(ctx)`) or **webhook mode** (`bot.StartWebhook(ctx)` + HTTP server).
* **Register commands** that automatically appear in the bot’s menu and match incoming updates.
* **Add middleware** that runs for every update (e.g. logging, auth, rate‑limiting).
* **Attach a tracer** (or a no‑op tracer) to log before/after each handler.
* **Expose `/live` and `/ready` health‑check endpoints** useful for Kubernetes / Docker.
* **Load configuration** from a YAML file (the webhook example uses `github.com/jinzhu/configor`).

The library is deliberately small – the core type is `TGBot` (see `tgbot.go`) and all functionality is exposed through a handful of exported functions and methods.

## Features

| Feature | Details |
|---|---|
| **Long‑polling & Webhook modes** | Choose at runtime via `WithWebHookToken` or the `isWebHook()` guard. |
| **Middleware chain** | `AddMiddleware` / `applyMiddleware`; middleware run in reverse‑add order. |
| **Tracer interface** | `Tracer.Before/OnSuccess/OnError/After`; `NewNoopTracer` included. |
| **Liveness / readiness probes** | `/live` and `/ready` HTTP handlers; probe functions are optional. |
| **Command registration** | `AddMenuCommand` (adds to BotCommand list) & `AddTextCommand` (matches text). |
| **Default / text / callback handlers** | `AddDefaultHandler`, `AddDefaultTextHandler`, `AddDefaultCallbackQueryHandler`. |
| **Helper methods** | `SendMessage`, `DeleteMessage`, `setMyCommands`. |
| **YAML config loading** | via `github.com/jinzhu/configor` (webhook CLI). |
| **Go 1.26+ compatible** | Module `go.mod` declares `go 1.26.6`. |
| **Standard tooling** | `make build`, `make test`, `make vet`, `make lint` (golangci‑lint). |

## Quick start – Long‑polling

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Mikhalevich/tgbot"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := tgbot.New(
		"YOUR_TELEGRAM_TOKEN",
		tgbot.WithNewTracerFn(func() tgbot.Tracer {
			return tgbot.NewTracer(logger) // your own tracer
		}),
	)
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(func(ctx context.Context, msg tgbot.BotMessage, sender tgbot.MessageSender) error {
		_, err := sender.SendMessage(ctx, msg.ChatID, "You said: "+msg.Text)
		return err
	})

	if err := bot.Start(context.Background()); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

*Run:* `go run main.go`
## Quick start – Webhook

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Mikhalevich/tgbot"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := tgbot.New(
		"YOUR_TELEGRAM_TOKEN",
		tgbot.WithWebHookToken("my-secret-token"),   // required for webhook validation
		tgbot.WithLivenessProbe(func(ctx context.Context) error { return nil }),
		tgbot.WithReadinessProbe(func(ctx context.Context) error { return nil }),
	)
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// HTTP mux that also serves /live and /ready
	httpMux := ... // your mux, see start.go for the built‑in pattern

	if err := bot.Start(ctx); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

*Run:* `go run main.go` – the bot will listen on `:80` (configurable via `WithReadTimeout`/`WithWriteTimeout`).

## Configuration

The webhook CLI (`cmd/webhook/main.go`) loads a YAML config:

```yaml
token: <BOT_TOKEN>
webhook_secret_token: <SECRET_TOKEN>
url: https://example.com/hook
certificate_path: ./certs/public.pem
```

The struct tags map directly:

```go
type Config struct {
    Token              string `yaml:"token" required:"true"`
    WebHookSecretToken string `yaml:"webhook_secret_token" required:"true"`
    URL                string `yaml:"url" required:"true"`
    CertificatePath    string `yaml:"certificate_path" required:"true"`
}
```

You can adjust the config path with the `-config` flag:

```
./webhook -config ./config/config.yaml
```

Available flags:

| Flag | Default | Description |
|---|---|---|
| `--config` | `config/config.yaml` | Path to YAML file |
| `--get` | `false` | Dump webhook info |
| `--set` | `false` | Set webhook |
| `--remove` | `false` | Remove webhook |

## Middleware

```go
func LoggingMiddleware(next tgbot.Handler) tgbot.Handler {
	return func(ctx context.Context, msg tgbot.BotMessage, sender tgbot.MessageSender) error {
		logger := slog.FromContext(ctx)
		logger.Info("incoming update", slog.String("text", msg.Text))
		return next(ctx, msg, sender)
	}
}
```

Register it:

```go
bot.AddMiddleware(LoggingMiddleware)
```

Middleware are applied **last‑in‑first‑out** (the most recently added runs closest to the handler).

## Tracing & probes

* **Tracer** – implements `Before`, `OnSuccess`, `OnError`, `After`. The example `examples/echotracer/main.go` shows a simple struct that logs each phase.

```go
type tracer struct{ logger *slog.Logger }

func (t tracer) Before(ctx context.Context, pattern string, msg tgbot.BotMessage) context.Context {
	t.logger.Info("handler start", slog.String("pattern", pattern))
	return ctx
}
func (t tracer) OnSuccess(ctx context.Context)  { /* … */ }
func (t tracer) OnError(ctx context.Context, err error) {
	t.logger.Error("handler error", slog.String("err", err.Error()))
}
func (t tracer) After(ctx context.Context) { /* … */ }
```

* **Probes** – optional functions passed with `WithLivenessProbe` / `WithReadinessProbe`. When nil the endpoints simply return `200 OK` without checking anything.

```go
bot.WithLivenessProbe(func(ctx context.Context) error { return nil })
bot.WithReadinessProbe(func(ctx context.Context) error { return nil })
```

The HTTP handlers are at `/live` and `/ready` (see `probes.go`).

## Command registration

```go
bot.AddMenuCommand("start", "Begin using the bot")
bot.AddTextCommand("help", func(ctx context.Context, msg tgbot.BotMessage, sender tgbot.MessageSender) error {
	_, err := sender.SendMessage(ctx, msg.ChatID, "Help text")
	return err
})
```

* `AddMenuCommand` – also registers the command as a `BotCommand` visible in Telegram's UI.
* `AddTextCommand` – only matches the command text (no menu entry).

You can also set a **default handler** that catches any update not matched by a registered command (`AddDefaultHandler`).

## HTTP helpers

| Method | Signature | Description |
|---|---|---|
| `SendMessage` | `func (t *TGBot) SendMessage(ctx context.Context, chatID int64, msg string)` | Sends a text message; errors are logged. |
| `DeleteMessage` | `func (t *TGBot) DeleteMessage(ctx, chatID int64, messageID int)` | Deletes a message; errors are logged. |
| `/live` | `http.HandlerFunc` | Returns `200 OK` if the liveness probe succeeds, otherwise `500`. |
| `/ready` | `http.HandlerFunc` | Returns `200 OK` if the readiness probe succeeds. |

## Development

| Command | What it does |
|---|---|
| `make build` | `go build ./…` – compiles the library and examples. |
| `make test` | `go test ./…` – runs all package tests. |
| `make vet` | `go vet ./…` – static analysis. |
| `make lint` | Runs `golangci-lint` (v2.12.2) from `tools/bin/`. |
| `make fmt` | `golangci-lint fmt` – auto‑format code. |

The `Makefile` also defines the tag `APP_TAG := 0.5.2` (used for release artefacts).

## Examples

| Example | Purpose |
|---|---|
| `examples/echotracer/main.go` | Minimal bot that echoes incoming text, with a custom `slog`‑based tracer. |
| `cmd/webhook/main.go` | CLI to manage webhooks (`--set`, `--get`, `--remove`). Uses a YAML config file and the `github.com/go-telegram/bot` low‑level API. |

Both examples compile with `go build ./…` and can be run directly after setting a real Telegram token.

## License

This project is released under the **MIT License** – see `LICENSE` for full text.
