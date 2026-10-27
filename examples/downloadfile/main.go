package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/Mikhalevich/tgbot"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := tgbot.New(
		"TELEGRAM_BOT_TOKEN",
	)
	if err != nil {
		logger.Error("create bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(
		func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
			if msg.Type != tgbot.TypeDocument {
				logger.Error("message is not a document to download", slog.Int("msg_type", int(msg.Type)))

				return nil
			}

			link, err := operations.GetDownloadLink(ctx, msg.Document.FileID)
			if err != nil {
				return fmt.Errorf("get download link: %w", err)
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
			if err != nil {
				return fmt.Errorf("create http request: %w", err)
			}

			rsp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("download file: %w", err)
			}

			defer rsp.Body.Close()

			file, err := os.Open(msg.Document.FileName)
			if err != nil {
				return fmt.Errorf("open file %s: %w", msg.Document.FileName, err)
			}

			defer file.Close()

			if _, err := io.Copy(file, rsp.Body); err != nil {
				return fmt.Errorf("copy file content: %w", err)
			}

			return nil
		},
	)

	if err := bot.Start(context.Background()); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("done")
}
