package tgbot

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func (t *TGBot) Start(ctx context.Context) error {
	if err := t.setMyCommands(ctx); err != nil {
		return fmt.Errorf("set my commands: %w", err)
	}

	if !t.isWebHook() {
		t.bot.Start(ctx)

		return nil
	}

	go t.bot.StartWebhook(ctx)

	mux := http.NewServeMux()
	mux.Handle("POST /", t.bot.WebhookHandler())
	mux.HandleFunc("GET /live", t.httpLivenessProbe())
	mux.HandleFunc("GET /ready", t.httpReadinessProbe())

	if err := t.listenHTTP(ctx, mux); err != nil {
		return fmt.Errorf("listen http webhook: %w", err)
	}

	return nil
}

func (t *TGBot) listenHTTP(ctx context.Context, hndlr http.Handler) error {
	var (
		srv = &http.Server{
			Addr:         ":80",
			Handler:      hndlr,
			ReadTimeout:  t.opts.readTimeout,
			WriteTimeout: t.opts.writeTimeout,
		}

		srvErrCh = make(chan error)
	)

	defer close(srvErrCh)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrCh <- err
		}
	}()

	select {
	case err := <-srvErrCh:
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), t.opts.shutdownTimeout)
	defer cancel()

	//nolint:contextcheck
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
