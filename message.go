package tgbot

import (
	"fmt"
)

type Type int

const (
	TypeInvalid Type = iota
	TypeText
	TypeCallbackQuery
	TypePayment
	TypeDocument
	TypePhoto
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

type Document struct {
	FileID   string
	FileName string
	FileSize int64
}

type PhotoSize struct {
	FileID   string
	Width    int
	Height   int
	FileSize int64
}

type Message struct {
	Type      Type
	MessageID int
	ChatID    int64
	IsGroup   bool
	User      User
	// for text message
	Text string
	Args string
	// for callback query
	Data     string
	Payment  Payment
	Document Document
	// photo sizes ordered from smallest to largest
	Photo []PhotoSize
}

// LargestPhoto returns the highest-resolution photo size.
// The second return value is false if the message has no photo.
func (m Message) LargestPhoto() (PhotoSize, bool) {
	if len(m.Photo) == 0 {
		return PhotoSize{}, false
	}

	return m.Photo[len(m.Photo)-1], true
}
