package main

import (
	"context"
	"errors"
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
		var fileID, fileName string

		//nolint:exhaustive
		switch msg.Type {
		case tgbot.TypeDocument:
			fileID = msg.Document.FileID
			fileName = safeFileName(msg.Document)
		case tgbot.TypePhoto:
			photo, ok := msg.LargestPhoto()
			if !ok {
				return errors.New("photo message without photo sizes")
			}

			fileID = photo.FileID
			fileName = photo.FileID + ".jpg"
		default:
			return fmt.Errorf("message is not a document or photo to download: %d", msg.Type)
		}

		if err := downloadFile(ctx, operations, fileID, fileName); err != nil {
			return fmt.Errorf("download file: %w", err)
		}

		logger.Info("file downloaded", slog.String("file_name", fileName))

		return nil
	}
}

func downloadFile(ctx context.Context, operations tgbot.Operations, fileID, fileName string) error {
	link, err := operations.GetDownloadLink(ctx, fileID)
	if err != nil {
		return fmt.Errorf("get download link: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do http request: %w", err)
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http status: %s", rsp.Status)
	}

	if err := saveFile(rsp.Body, fileName); err != nil {
		return fmt.Errorf("save file %s: %w", fileName, err)
	}

	return nil
}

// saveFile writes content to a temporary file and renames it to fileName on success,
// so a failed download never leaves a partial file under the final name.
func saveFile(content io.Reader, fileName string) error {
	tmpFile, err := os.CreateTemp(downloadDir, fileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmpFile.Name()

	if _, err := io.Copy(tmpFile, content); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)

		return fmt.Errorf("copy file content: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)

		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filepath.Join(downloadDir, fileName)); err != nil {
		os.Remove(tmpName)

		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
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
