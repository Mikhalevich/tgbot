package tgbot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Payment struct {
	IsSuccessful   bool
	IsCheckout     bool
	ID             string
	InvoicePayload string
	Currency       string
	TotalAmount    int
}

type User struct {
	FirstName string
	LastName  string
	Username  string
}

// FullName returns the full name of the user by combining the first and last name.
// If only one of the names is set, it returns that name.
// If both are empty, it returns an empty string.
func (u User) FullName() string {
	if u.FirstName == "" {
		return u.LastName
	}

	if u.LastName == "" {
		return u.FirstName
	}

	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

type BotMessage struct {
	MessageID int
	ChatID    int64
	IsGroup   bool
	User      User
	// for text message
	Text string
	// for callback query
	Data    string
	Payment Payment
	Args    string
}

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, msg string)
	DeleteMessage(ctx context.Context, chatID int64, messageID int)
}

type Handler func(ctx context.Context, msg BotMessage, sender MessageSender) error

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
			msg           = makeMsgFromUpdate(update)
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

func makeMsgFromUpdate(update *models.Update) BotMessage {
	if update.Message != nil {
		msg := fillBaseMessage(update.Message, update.Message.From)
		msg.Args = commandArgs(update.Message)

		if update.Message.SuccessfulPayment != nil {
			msg.Payment = Payment{
				IsSuccessful:   true,
				InvoicePayload: update.Message.SuccessfulPayment.InvoicePayload,
				Currency:       update.Message.SuccessfulPayment.Currency,
				TotalAmount:    update.Message.SuccessfulPayment.TotalAmount,
			}
		}

		return msg
	}

	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message.Message != nil {
			msg := fillBaseMessage(update.CallbackQuery.Message.Message, &update.CallbackQuery.From)
			msg.Data = update.CallbackQuery.Data

			return msg
		}

		if update.CallbackQuery.Message.InaccessibleMessage != nil {
			return BotMessage{
				MessageID: update.CallbackQuery.Message.InaccessibleMessage.MessageID,
				ChatID:    update.CallbackQuery.Message.InaccessibleMessage.Chat.ID,
				Data:      update.CallbackQuery.Data,
			}
		}
	}

	if update.PreCheckoutQuery != nil {
		return BotMessage{
			Payment: Payment{
				IsCheckout:     true,
				ID:             update.PreCheckoutQuery.ID,
				InvoicePayload: update.PreCheckoutQuery.InvoicePayload,
				Currency:       update.PreCheckoutQuery.Currency,
				TotalAmount:    update.PreCheckoutQuery.TotalAmount,
			},
		}
	}

	return BotMessage{}
}

func fillBaseMessage(msg *models.Message, user *models.User) BotMessage {
	botMsg := BotMessage{
		MessageID: msg.ID,
		ChatID:    msg.Chat.ID,
		IsGroup:   msg.Chat.Type != models.ChatTypePrivate,
		Text:      msg.Text,
	}

	if user != nil {
		botMsg.User = User{
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Username:  user.Username,
		}
	}

	return botMsg
}

func commandArgs(msg *models.Message) string {
	for _, e := range msg.Entities {
		if e.Type == models.MessageEntityTypeBotCommand {
			if e.Offset == 0 {
				return strings.TrimLeft(msg.Text[e.Offset+e.Length:], " ")
			}
		}
	}

	return ""
}
