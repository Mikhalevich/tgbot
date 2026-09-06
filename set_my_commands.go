package tgbot

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
)

func (t *TGBot) setMyCommands(ctx context.Context) error {
	if len(t.commands) == 0 {
		return nil
	}

	if _, err := t.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: t.commands,
	}); err != nil {
		return fmt.Errorf("set my commands: %w", err)
	}

	return nil
}
