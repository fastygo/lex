// Command api serves the LeX HTTP API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fastygo/lex/internal/adapters"
	"github.com/fastygo/lex/internal/adapters/openrouter"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/httpapi"
)

type typesafePort struct{ client typesafe.Client }

func (port typesafePort) AdapterID() string      { return typesafe.AdapterID }
func (port typesafePort) AdapterVersion() string { return typesafe.AdapterVersion }

func (port typesafePort) Evaluate(ctx context.Context, state any, questions map[string]any) (httpapi.Decision, error) {
	decision, err := port.client.Evaluate(ctx, state, questions)
	if err != nil {
		return httpapi.Decision{}, err
	}
	return httpapi.Decision{ResolvedModel: decision.ResolvedModel, Answers: decision.Answers}, nil
}

type openrouterPort struct{ client openrouter.Client }

func (port openrouterPort) AdapterID() string      { return openrouter.AdapterID }
func (port openrouterPort) AdapterVersion() string { return openrouter.AdapterVersion }

func (port openrouterPort) Evaluate(ctx context.Context, state any, questions map[string]any) (httpapi.Decision, error) {
	decision, err := port.client.Evaluate(ctx, state, questions)
	if err != nil {
		return httpapi.Decision{}, err
	}
	return httpapi.Decision{ResolvedModel: decision.ResolvedModel, Answers: decision.Answers}, nil
}

func selectDecider() (httpapi.Decider, error) {
	directKey := os.Getenv("LEX_TYPESAFE_API_KEY")
	hostedKey := os.Getenv("LEX_OPENROUTER_API_KEY")
	kind, err := adapters.Select(os.Getenv("LEX_DECISION_ADAPTER"), directKey != "", hostedKey != "")
	if err != nil {
		return nil, err
	}
	switch kind {
	case adapters.KindDirect:
		return typesafePort{client: typesafe.New(directKey)}, nil
	case adapters.KindHosted:
		return openrouterPort{client: openrouter.New(hostedKey)}, nil
	default:
		return nil, nil
	}
}

func main() {
	config, err := httpapi.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	decider, err := selectDecider()
	if err != nil {
		log.Fatal(err)
	}
	config.Decider = decider
	handler, err := httpapi.NewHandler(config)
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      config.RequestTimeout,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
