package tgbot

import (
	"context"

	"github.com/go-telegram/bot"
)

func (t *TGBot) defaultHandler(ctx context.Context, msg Message, operations Operations) error {
	if t.defaultHandlerFn != nil {
		return t.defaultHandlerFn(ctx, msg, operations)
	}

	return nil
}

func (t *TGBot) makeDefaultHandler() bot.HandlerFunc {
	return t.wrapHandler("default_handler", t.defaultHandler)
}
