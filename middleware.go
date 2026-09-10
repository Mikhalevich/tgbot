package tgbot

type Middleware func(next Handler) Handler

// AddMiddleware adds a middleware function to the bot's middleware chain.
// Middlewares are applied in the order they were added.
func (t *TGBot) AddMiddleware(m Middleware) {
	t.middlewares = append(t.middlewares, m)
}

func (t *TGBot) applyMiddleware(hndlr Handler) Handler {
	//nolint:modernize
	for i := len(t.middlewares) - 1; i >= 0; i-- {
		hndlr = t.middlewares[i](hndlr)
	}

	return hndlr
}
