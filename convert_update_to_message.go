package tgbot

import (
	"strings"

	"github.com/go-telegram/bot/models"
)

func convertToMessage(update *models.Update) Message {
	if update.Message != nil {
		return fillMessage(update.Message)
	}

	if update.CallbackQuery != nil {
		return fillCallbackQuery(update.CallbackQuery)
	}

	if update.PreCheckoutQuery != nil {
		return Message{
			Type: TypePayment,
			Payment: Payment{
				IsCheckout:     true,
				ID:             update.PreCheckoutQuery.ID,
				InvoicePayload: update.PreCheckoutQuery.InvoicePayload,
				Currency:       update.PreCheckoutQuery.Currency,
				TotalAmount:    update.PreCheckoutQuery.TotalAmount,
			},
		}
	}

	return Message{}
}

func fillBaseMessage(msg *models.Message, user *models.User) Message {
	botMsg := Message{
		MessageID: msg.ID,
		ChatID:    msg.Chat.ID,
		IsGroup:   msg.Chat.Type != models.ChatTypePrivate,
		Text:      msg.Text,
	}

	// media messages (document, photo, etc.) carry their text in caption
	if botMsg.Text == "" {
		botMsg.Text = msg.Caption
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

func fillMessage(update *models.Message) Message {
	msg := fillBaseMessage(update, update.From)

	if update.SuccessfulPayment != nil {
		msg.Type = TypePayment
		msg.Payment = Payment{
			IsSuccessful:   true,
			InvoicePayload: update.SuccessfulPayment.InvoicePayload,
			Currency:       update.SuccessfulPayment.Currency,
			TotalAmount:    update.SuccessfulPayment.TotalAmount,
		}

		return msg
	}

	if update.Document != nil {
		msg.Type = TypeDocument
		msg.Document = Document{
			FileID:   update.Document.FileID,
			FileName: update.Document.FileName,
			FileSize: update.Document.FileSize,
		}

		return msg
	}

	if len(update.Photo) > 0 {
		msg.Type = TypePhoto
		msg.Photo = make([]PhotoSize, 0, len(update.Photo))

		for _, photo := range update.Photo {
			msg.Photo = append(msg.Photo, PhotoSize{
				FileID:   photo.FileID,
				Width:    photo.Width,
				Height:   photo.Height,
				FileSize: int64(photo.FileSize),
			})
		}

		return msg
	}

	msg.Type = TypeText
	msg.Args = commandArgs(update)

	return msg
}

func fillCallbackQuery(update *models.CallbackQuery) Message {
	if update.Message.Message != nil {
		msg := fillBaseMessage(update.Message.Message, &update.From)
		msg.Type = TypeCallbackQuery
		msg.Data = update.Data

		return msg
	}

	if update.Message.InaccessibleMessage != nil {
		return Message{
			Type:      TypeCallbackQuery,
			MessageID: update.Message.InaccessibleMessage.MessageID,
			ChatID:    update.Message.InaccessibleMessage.Chat.ID,
			Data:      update.Data,
		}
	}

	return Message{}
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
