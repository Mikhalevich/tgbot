# tgbot – Telegram Bot API wrapper for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/Mikhalevich/tgbot)](https://goreportcard.com/report/github.com/Mikhalevich/tgbot)
[![GoDoc](https://godoc.org/github.com/Mikhalevich/tgbot?status.svg)](https://pkg.go.dev/github.com/Mikhalevich/tgbot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight wrapper around `github.com/go-telegram/bot`. It adds a simple handler API with a flattened `Message` type, middleware support, per-handler tracing hooks, file download links, and HTTP health-check endpoints for webhook mode.

## Table of Contents

| Section | Description |
|---|---|
| [Overview](#overview) | What the library does and why it exists |
| [Features](#features) | High-level capabilities |
| [Quick start: long polling](#quick-start-long-polling) | Minimal code to get a bot running in polling mode |
| [Quick start: webhook](#quick-start-webhook) | Minimal code to run the bot behind an HTTP webhook |
| [Configuration](#configuration) | YAML config file (used by the webhook CLI) |
| [Middleware](#middleware) | How to plug custom middleware |
| [Tracing & probes](#tracing--probes) | Tracer interface, no-op implementation, liveness/readiness endpoints |
| [Command registration](#command-registration) | `AddMenuCommand`, `AddTextCommand`, default / text / callback handlers |
| [Messages & file download](#messages--file-download) | `Message` types and downloading documents/photos |
| [Operations & HTTP endpoints](#operations--http-endpoints) | `SendMessage`, `DeleteMessage`, `GetDownloadLink`, `/live`, `/ready` |
| [Development](#development) | Building, testing, linting |
| [Examples](#examples) | Example bots and webhook CLI |
| [License](#license) | MIT License |

## Overview

`tgbot` simplifies the most common tasks when building a Telegram bot in Go:

* **Create a bot** with a token and optional options (`WithWebHookToken`, timeouts, tracer factory, probes).
* **Run it** with `Start(ctx)` – long polling by default, or webhook mode (with a built-in HTTP server) when a webhook secret token is set.
* **Register commands** that automatically appear in the bot's menu and match incoming updates.
* **Add middleware** that wraps every handler (e.g. logging, auth, rate limiting).
* **Attach a tracer** (or use the default no-op tracer) that is called before/after each handler.
* **Download files** (documents, photos) sent to the bot.
* **Expose `/live` and `/ready` health-check endpoints** in webhook mode, useful for Kubernetes / Docker.

The library is deliberately small – the core type is `TGBot` (see `tgbot.go`) and all functionality is exposed through a handful of exported functions and methods.

## Features

| Feature | Details |
|---|---|
| **Long-polling & webhook modes** | `Start(ctx)` uses webhook mode when `WithWebHookToken` is set, long polling otherwise. |
| **Middleware chain** | `AddMiddleware`; middleware run in the order they were added (first added is outermost). |
| **Tracer interface** | `Tracer.Before/OnSuccess/OnError/After`, created per handler call by `WithNewTracerFn`; `NewNoopTracer` included. |
| **Liveness / readiness probes** | `/live` and `/ready` HTTP handlers in webhook mode; probe functions are optional. |
| **Command registration** | `AddMenuCommand` (published to the bot menu on `Start`) & `AddTextCommand` (no menu entry). |
| **Default / text / callback handlers** | `AddDefaultHandler`, `AddDefaultTextHandler`, `AddDefaultCallbackQueryHandler`. |
| **Typed messages** | `Message.Type`: text, callback query, payment, document, photo. |
| **File download** | `GetDownloadLink` for document/photo file IDs; `Message.LargestPhoto()` helper. |
| **Operations** | `SendMessage`, `DeleteMessage`, `GetDownloadLink`. |
| **YAML config loading** | via `github.com/jinzhu/configor` (webhook CLI). |
| **Go 1.27+** | Module `go.mod` declares `go 1.27.0`. |
| **Standard tooling** | `make build`, `make test`, `make bench`, `make lint`, `make fmt` (golangci-lint). |

## Quick start: long polling

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

	bot, err := tgbot.New("YOUR_TELEGRAM_TOKEN")
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
		operations.SendMessage(ctx, msg.ChatID, "You said: "+msg.Text)

		return nil
	})

	if err := bot.Start(context.Background()); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

*Run:* `go run main.go`

If a handler returns an error, the bot replies to the message with `internal error`.

## Quick start: webhook

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mikhalevich/tgbot"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := tgbot.New(
		"YOUR_TELEGRAM_TOKEN",
		tgbot.WithWebHookToken("my-secret-token"), // enables webhook mode and validates incoming requests
		tgbot.WithLivenessProbe(func(ctx context.Context) error { return nil }),
		tgbot.WithReadinessProbe(func(ctx context.Context) error { return nil }),
	)
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
		operations.SendMessage(ctx, msg.ChatID, "You said: "+msg.Text)

		return nil
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// blocks until ctx is cancelled, then shuts the HTTP server down gracefully
	if err := bot.Start(ctx); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

*Run:* `go run main.go` – the bot listens on `:80` (the port is fixed) and serves:

* `POST /` – Telegram webhook updates,
* `GET /live` and `GET /ready` – health checks.

Server timeouts are configurable with `WithReadTimeout`, `WithWriteTimeout` and `WithShutdownTimeout`. The webhook itself must be registered with Telegram separately, e.g. with the `cmd/webhook` CLI (`--set`), using the same secret token.

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
func LoggingMiddleware(logger *slog.Logger) tgbot.Middleware {
	return func(next tgbot.Handler) tgbot.Handler {
		return func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
			logger.Info("incoming update", slog.String("text", msg.Text))

			return next(ctx, msg, operations)
		}
	}
}
```

Register it:

```go
bot.AddMiddleware(LoggingMiddleware(logger))
```

Middleware run in the order they were added: the first added is the outermost, the most recently added runs closest to the handler.

**Important:** the middleware chain is applied when a handler is registered, so call `AddMiddleware` **before** registering handlers – handlers registered earlier won't get middleware added later.

## Tracing & probes

* **Tracer** – implements `Before`, `OnSuccess`, `OnError`, `After`. A new tracer is created for every handler call by the factory passed to `WithNewTracerFn` (the default is `NewNoopTracer`). `examples/echotracer/main.go` shows a simple struct that logs each phase.

```go
type tracer struct{ logger *slog.Logger }

func (t tracer) Before(ctx context.Context, pattern string, msg tgbot.Message) context.Context {
	t.logger.Info("handler start", slog.String("pattern", pattern))
	return ctx
}
func (t tracer) OnSuccess(ctx context.Context) { /* … */ }
func (t tracer) OnError(ctx context.Context, err error) {
	t.logger.Error("handler error", slog.String("err", err.Error()))
}
func (t tracer) After(ctx context.Context) { /* … */ }

bot, err := tgbot.New(
	"YOUR_TELEGRAM_TOKEN",
	tgbot.WithNewTracerFn(func() tgbot.Tracer {
		return tracer{logger: logger}
	}),
)
```

To override only some methods, embed `tgbot.NoopTracer` in your struct (see `examples/downloadfile/main.go`).

* **Probes** – optional functions passed to `tgbot.New` with `WithLivenessProbe` / `WithReadinessProbe`. The endpoints respond `200 UP` when the probe succeeds and `500 DOWN` when it returns an error; when no probe is set they return an empty `200 OK`.

```go
bot, err := tgbot.New(
	"YOUR_TELEGRAM_TOKEN",
	tgbot.WithWebHookToken("my-secret-token"),
	tgbot.WithLivenessProbe(func(ctx context.Context) error { return nil }),
	tgbot.WithReadinessProbe(func(ctx context.Context) error { return nil }),
)
```

The HTTP handlers are at `/live` and `/ready` (see `probes.go`) and are only served in webhook mode.

## Command registration

```go
bot.AddMenuCommand("start", "Begin using the bot", func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
	operations.SendMessage(ctx, msg.ChatID, "Welcome!")
	return nil
})

bot.AddTextCommand("help", func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
	operations.SendMessage(ctx, msg.ChatID, "Help text, args: "+msg.Args)
	return nil
})
```

* `AddMenuCommand` – also registers the command as a `BotCommand`; menu commands are published to Telegram when `Start` is called.
* `AddTextCommand` – only matches the command (no menu entry).

A command matches when the message starts with a `/command` entity whose name has the registered command as a prefix. `msg.Args` contains the text after the command.

Catch-all handlers:

* `AddDefaultHandler` – any update not matched by another handler.
* `AddDefaultTextHandler` – any text message not matched by a command.
* `AddDefaultCallbackQueryHandler` – all callback queries (`msg.Data` holds the callback data).

## Messages & file download

Handlers receive a flattened `tgbot.Message`. `msg.Type` tells which fields are populated:

| `Type` | Fields |
|---|---|
| `TypeText` | `Text`, `Args` |
| `TypeCallbackQuery` | `Data` (plus `Text` of the original message, if accessible) |
| `TypePayment` | `Payment` (`IsSuccessful` for a successful payment, `IsCheckout` for a pre-checkout query) |
| `TypeDocument` | `Document` (`FileID`, `FileName`, `FileSize`) |
| `TypePhoto` | `Photo` (all available sizes; `LargestPhoto()` returns the biggest one) |

`MessageID`, `ChatID`, `IsGroup` and `User` are filled in for regular messages. For media messages `Text` holds the caption.

To download a file, resolve its file ID to a download URL with `GetDownloadLink` and fetch it over HTTP:

```go
bot.AddDefaultHandler(func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
	var fileID string

	switch msg.Type {
	case tgbot.TypeDocument:
		fileID = msg.Document.FileID
	case tgbot.TypePhoto:
		photo, ok := msg.LargestPhoto()
		if !ok {
			return errors.New("no photo sizes")
		}
		fileID = photo.FileID
	default:
		return nil
	}

	link, err := operations.GetDownloadLink(ctx, fileID)
	if err != nil {
		return fmt.Errorf("get download link: %w", err)
	}

	// download link with an HTTP GET request
	_ = link

	return nil
})
```

See `examples/downloadfile/main.go` for a complete example that saves files to disk.

## Operations & HTTP endpoints

Handlers receive an `tgbot.Operations` value (implemented by `*TGBot`):

| Method | Signature | Description |
|---|---|---|
| `SendMessage` | `SendMessage(ctx context.Context, chatID int64, msg string)` | Sends a text message; errors are logged. |
| `DeleteMessage` | `DeleteMessage(ctx context.Context, chatID int64, messageID int)` | Deletes a message; errors are logged. |
| `GetDownloadLink` | `GetDownloadLink(ctx context.Context, fileID string) (string, error)` | Returns a download URL for a document/photo file ID. |

HTTP endpoints (webhook mode only):

| Endpoint | Description |
|---|---|
| `GET /live` | `200 UP` if the liveness probe succeeds, otherwise `500 DOWN`. |
| `GET /ready` | `200 UP` if the readiness probe succeeds, otherwise `500 DOWN`. |

## Development

| Command | What it does |
|---|---|
| `make build` | `go build` – compiles the root package. Use `go build ./...` to also build the examples and CLI. |
| `make test` | `go test ./...` – runs all package tests (there are none yet). |
| `make bench` | `go test -bench=. -benchmem ./...` – runs benchmarks. |
| `make lint` | Runs `golangci-lint` (v2.15.0, installed on first use into `tools/bin/<version>/`). |
| `make fmt` | `golangci-lint fmt` – auto-format code. |

## Examples

| Example | Purpose |
|---|---|
| `examples/echotracer/main.go` | Minimal bot that echoes incoming text, with a custom `slog`-based tracer. |
| `examples/downloadfile/main.go` | Bot that saves incoming documents and photos to a `downloads/` directory. |
| `cmd/webhook/main.go` | CLI to manage webhooks (`--set`, `--get`, `--remove`). Uses a YAML config file and the `github.com/go-telegram/bot` low-level API. |

All of them compile with `go build ./...` and can be run after setting a real Telegram token.

## License

This project is released under the **MIT License** – see `LICENSE` for full text.
