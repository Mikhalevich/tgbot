package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Mikhalevich/tgbot"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := tgbot.New(
		"TELEGRAM_BOT_TOKEN",
		tgbot.WithNewTracerFn(
			func() tgbot.Tracer {
				return NewTracer(logger)
			},
		),
	)
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(
		func(ctx context.Context, msg tgbot.BotMessage, sender tgbot.MessageSender) error {
			sender.SendMessage(
				ctx,
				msg.ChatID,
				fmt.Sprintf("reply for %q", msg.Text),
			)

			return nil
		},
	)

	if err := bot.Start(context.Background()); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("done")
}

type tracer struct {
	logger *slog.Logger
}

func NewTracer(logger *slog.Logger) tracer {
	return tracer{
		logger: logger,
	}
}

func (t tracer) Before(ctx context.Context, pattern string, msg tgbot.BotMessage) context.Context {
	t.logger.Info("before handler log", slog.String("msg_text", msg.Text))

	return ctx
}

func (t tracer) OnSuccess(ctx context.Context) {
	t.logger.Info("on success log")
}

func (t tracer) OnError(ctx context.Context, err error) {
	t.logger.Error("on error log", slog.String("error", err.Error()))
}

func (t tracer) After(ctx context.Context) {
	t.logger.Info("after handler log")
}
