package tgbot

import (
	"context"

	"github.com/go-telegram/bot"
)

func (t *TGBot) defaultHandler(ctx context.Context, msg BotMessage, sender MessageSender) error {
	if t.defaultHandlerFn != nil {
		return t.defaultHandlerFn(ctx, msg, sender)
	}

	return nil
}

func (t *TGBot) makeDefaultHandler() bot.HandlerFunc {
	return t.wrapHandler("default_handler", t.defaultHandler)
}
