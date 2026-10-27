package tgbot

import (
	"context"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Operations interface {
	SendMessage(ctx context.Context, chatID int64, msg string)
	DeleteMessage(ctx context.Context, chatID int64, messageID int)
	GetDownloadLink(ctx context.Context, fileID string) (string, error)
}

type Handler func(ctx context.Context, msg Message, operations Operations) error

// AddMenuCommand registers a bot command with a description in the bot's menu.
// The command is also registered as a handler for incoming messages matching the command.
func (t *TGBot) AddMenuCommand(command string, description string, handler Handler) {
	t.addCommand(command, description, handler)
}

// AddTextCommand registers a command handler without a menu description.
// The handler is triggered when a message starts with the given command.
func (t *TGBot) AddTextCommand(command string, handler Handler) {
	t.addCommand(command, "", handler)
}

func (t *TGBot) addCommand(command string, description string, handler Handler) {
	if description != "" {
		t.commands = append(t.commands, models.BotCommand{
			Command:     command,
			Description: description,
		})
	}

	t.bot.RegisterHandlerMatchFunc(
		commandMatchFn(command),
		t.wrapHandler(command, handler),
	)
}

func commandMatchFn(command string) bot.MatchFunc {
	return func(update *models.Update) bool {
		if update.Message == nil {
			return false
		}

		var (
			data     = update.Message.Text
			entities = update.Message.Entities
		)

		for _, entity := range entities {
			if entity.Type == models.MessageEntityTypeBotCommand {
				if entity.Offset != 0 {
					continue
				}

				entityData := data[entity.Offset+1 : entity.Offset+entity.Length]

				if strings.HasPrefix(entityData, command) {
					return true
				}
			}
		}

		return false
	}
}

// AddDefaultHandler sets the handler that is called when no other command handler matches an incoming message.
func (t *TGBot) AddDefaultHandler(h Handler) {
	h = t.applyMiddleware(h)
	t.defaultHandlerFn = h
}

// AddDefaultTextHandler registers a handler for all text messages that do not match any registered command.
// The handler is called with the full message text as the pattern.
func (t *TGBot) AddDefaultTextHandler(h Handler) {
	t.bot.RegisterHandler(
		bot.HandlerTypeMessageText,
		"",
		bot.MatchTypePrefix,
		t.wrapHandler("default_text_handler", h),
	)
}

// AddDefaultCallbackQueryHandler registers a handler for all callback query updates.
func (t *TGBot) AddDefaultCallbackQueryHandler(h Handler) {
	t.bot.RegisterHandler(
		bot.HandlerTypeCallbackQueryData,
		"",
		bot.MatchTypePrefix,
		t.wrapHandler("default_callback_query", h),
	)
}

func (t *TGBot) wrapHandler(pattern string, handler Handler) bot.HandlerFunc {
	handler = t.applyMiddleware(handler)

	return func(ctx context.Context, botAPI *bot.Bot, update *models.Update) {
		var (
			msg           = convertToMessage(update)
			handlerTracer = t.opts.tracerFn()
		)

		ctx = handlerTracer.Before(ctx, pattern, msg)
		defer handlerTracer.After(ctx)

		if err := handler(ctx, msg, t); err != nil {
			handlerTracer.OnError(ctx, err)

			if _, err := botAPI.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: msg.ChatID,
				ReplyParameters: &models.ReplyParameters{
					MessageID: msg.MessageID,
				},
				Text: "internal error",
			}); err != nil {
				log.Printf("send message error for pattern %q: %v", pattern, err)
			}

			return
		}

		handlerTracer.OnSuccess(ctx)
	}
}
