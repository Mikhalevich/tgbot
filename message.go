package tgbot

import (
	"fmt"
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

type Message struct {
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
