package tgbot

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	defaultReadTimeout     = time.Second * 10
	defaultWriteTimeout    = time.Second * 10
	defaultShutdownTimeout = time.Second * 30
)

type Probe func(ctx context.Context) error

type TGBot struct {
	bot              *bot.Bot
	opts             options
	middlewares      []Middleware
	commands         []models.BotCommand
	defaultHandlerFn Handler
}

// New creates a new TGBot instance with the given Telegram bot token and options.
// It initializes the bot API and applies all provided options.
// Returns an error if the bot API creation fails.
func New(
	token string,
	opts ...Option,
) (*TGBot, error) {
	tgBot := TGBot{
		opts: options{
			tracerFn: func() Tracer {
				return NewNoopTracer()
			},
			readTimeout:     defaultReadTimeout,
			writeTimeout:    defaultWriteTimeout,
			shutdownTimeout: defaultShutdownTimeout,
		},
	}

	tgBot.opts.apply(opts)

	botAPI, err := createBotAPI(
		token,
		tgBot.opts.webHookToken,
		tgBot.makeDefaultHandler(),
	)

	if err != nil {
		return nil, fmt.Errorf("create bot api: %w", err)
	}

	tgBot.bot = botAPI

	return &tgBot, nil
}

func createBotAPI(
	token string,
	webHookToken string,
	defaultHandler bot.HandlerFunc,
) (*bot.Bot, error) {
	opts := []bot.Option{
		bot.WithSkipGetMe(),
		bot.WithDefaultHandler(defaultHandler),
	}

	if webHookToken != "" {
		opts = append(opts, bot.WithWebhookSecretToken(webHookToken))
	}

	botAPI, err := bot.New(
		token,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}

	return botAPI, nil
}

func (t *TGBot) isWebHook() bool {
	return t.opts.webHookToken != ""
}
