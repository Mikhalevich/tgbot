package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Mikhalevich/tgbot"
)

const (
	downloadDir     = "downloads"
	downloadDirPerm = 0o750
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

	if err := os.MkdirAll(downloadDir, downloadDirPerm); err != nil {
		logger.Error("create download dir", slog.String("error", err.Error()))
		os.Exit(1)
	}

	bot.AddDefaultHandler(downloadHandler(logger))

	if err := bot.Start(context.Background()); err != nil {
		logger.Error("start bot", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("done")
}

func downloadHandler(logger *slog.Logger) tgbot.Handler {
	return func(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
		if msg.Type != tgbot.TypeDocument {
			return fmt.Errorf("message is not a document to download: %d", msg.Type)
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

		fileName := safeFileName(msg.Document)

		file, err := os.Create(filepath.Join(downloadDir, fileName))
		if err != nil {
			return fmt.Errorf("open file %s: %w", fileName, err)
		}

		defer file.Close()

		if _, err := io.Copy(file, rsp.Body); err != nil {
			return fmt.Errorf("copy file content: %w", err)
		}

		logger.Info("file downloaded", slog.String("file_name", fileName))

		return nil
	}
}

// safeFileName strips any directory components from the user-provided file name
// and falls back to the file id when nothing usable is left.
func safeFileName(doc tgbot.Document) string {
	fileName := filepath.Base(doc.FileName)
	if fileName == "." || fileName == ".." || fileName == string(filepath.Separator) {
		return doc.FileID
	}

	return fileName
}

type tracer struct {
	tgbot.NoopTracer

	logger *slog.Logger
}

func NewTracer(logger *slog.Logger) tracer {
	return tracer{
		logger: logger,
	}
}

func (t tracer) OnError(ctx context.Context, err error) {
	t.logger.Error("error", slog.String("error", err.Error()))
}
