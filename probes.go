package tgbot

import (
	"fmt"
	"net/http"
)

func (t *TGBot) httpLivenessProbe() http.HandlerFunc {
	return doProbe(t.opts.livenessProbe)
}

func (t *TGBot) httpReadinessProbe() http.HandlerFunc {
	return doProbe(t.opts.readinessProbe)
}

func doProbe(probe Probe) http.HandlerFunc {
	return http.HandlerFunc(
		//nolint:varnamelen
		func(w http.ResponseWriter, r *http.Request) {
			if probe == nil {
				return
			}

			if err := probe(r.Context()); err != nil {
				http.Error(w, "DOWN", http.StatusInternalServerError)
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "UP")
		},
	)
}
