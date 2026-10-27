package tgbot

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
)

func (t *TGBot) GetDownloadLink(
	ctx context.Context,
	fileID string,
) (string, error) {
	file, err := t.bot.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})

	if err != nil {
		return "", fmt.Errorf("get file: %w", err)
	}

	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.bot.Token(), file.FilePath), nil
}
