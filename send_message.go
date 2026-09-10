package tgbot

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
)

// SendMessage sends a text message to the specified chat.
// Errors are logged but not returned to the caller.
func (t *TGBot) SendMessage(ctx context.Context, chatID int64, msg string) {
	if _, err := t.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   msg,
	}); err != nil {
		log.Printf("send message: %v\n", err)
	}
}

// DeleteMessage deletes a message from a chat by its ID.
// Errors are logged but not returned to the caller.
func (t *TGBot) DeleteMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
) {
	if _, err := t.bot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	}); err != nil {
		log.Printf("delete message: %v\n", err)
	}
}
