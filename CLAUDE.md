# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make build` — `go build` of the root package only; use `go build ./...` to also compile `cmd/webhook` and `examples/echotracer`.
- `make test` — `go test ./...`. Run one test with `go test -run TestName ./...`. There are currently no test files.
- `make bench` — benchmarks with `-benchmem`.
- `make lint` / `make fmt` — golangci-lint (version pinned by `LINTER_VERSION` in the Makefile), installed on first use into `tools/bin/<version>/`. The config (`.golangci.yml`) enables **all** linters except a short disable list, so expect strict checks (varnamelen, funlen, contextcheck, modernize, etc.). The code silences specific false positives with targeted `//nolint:<linter>` comments; follow that pattern rather than loosening the config.

## Architecture

Single-package library (`package tgbot`, module `github.com/Mikhalevich/tgbot`) wrapping `github.com/go-telegram/bot`. `TGBot` (`tgbot.go`) holds the underlying `*bot.Bot`, functional `options` (`options.go`), the middleware list, and the menu commands to publish.

**Handler pipeline** (`register_handler.go`): every registration (`AddMenuCommand`, `AddTextCommand`, `AddDefaultTextHandler`, `AddDefaultCallbackQueryHandler`) goes through `wrapHandler`, which:
1. applies the middleware chain **at registration time** — so `AddMiddleware` must be called before registering handlers, or those handlers won't get it;
2. converts the raw `models.Update` into the library's flattened `BotMessage` (`makeMsgFromUpdate` handles messages, callback queries incl. inaccessible messages, successful payments, and pre-checkout queries);
3. creates a fresh `Tracer` per invocation via the `tracerFn` factory and calls `Before`/`OnSuccess`/`OnError`/`After`;
4. on handler error, replies to the user with `"internal error"`.

Handlers receive `TGBot` itself as the `MessageSender` (its `SendMessage`/`DeleteMessage` log errors and return nothing).

**Default handler** (`default_handler.go`): the go-telegram default handler is wired once in `New()` to `t.defaultHandler`, which dispatches to `defaultHandlerFn` set later by `AddDefaultHandler` (which applies the middleware chain itself).

**Command matching**: `commandMatchFn` matches only a bot-command entity at offset 0 and uses `strings.HasPrefix` on the command name. `BotMessage.Args` is the text after that command. Only commands registered with a description (`AddMenuCommand`) are published via `SetMyCommands`, which runs at the start of `Start()`.

**Run modes** (`start.go`): `Start(ctx)` picks the mode from whether `WithWebHookToken` was set (`isWebHook()`):
- long polling — blocks in `bot.Start(ctx)`; no HTTP server, no probes;
- webhook — runs `bot.StartWebhook` and an `http.Server` on hardcoded `:80` serving `POST /` (webhook), `GET /live`, `GET /ready` (`probes.go`), with graceful shutdown on ctx cancellation using the configured read/write/shutdown timeouts.

`cmd/webhook` is a standalone CLI (YAML config via `configor`, flags `--get`/`--set`/`--remove`) that registers/inspects the webhook with Telegram directly through go-telegram/bot; it does not use the `tgbot` package.

The README's code samples are not fully in sync with the API (e.g. they treat `SendMessage` as returning values); treat the source as authoritative.
