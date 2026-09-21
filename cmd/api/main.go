// Command api serves the LeX HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fastygo/lex/internal/adapters"
	"github.com/fastygo/lex/internal/adapters/openrouter"
	"github.com/fastygo/lex/internal/adapters/systemone"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/httpapi"
)

type typesafePort struct{ client typesafe.Client }

func (port typesafePort) AdapterID() string      { return typesafe.AdapterID }
func (port typesafePort) AdapterVersion() string { return typesafe.AdapterVersion }

func (port typesafePort) Evaluate(ctx context.Context, state any, questions map[string]any) (httpapi.Decision, error) {
	if n, ok := httpapi.AnswerBudget(ctx); ok {
		ctx = systemone.WithMaxResponseBytes(ctx, n)
	}
	decision, err := port.client.Evaluate(ctx, state, questions)
	if err != nil {
		return httpapi.Decision{}, err
	}
	return httpapi.Decision{
		ResolvedModel: decision.ResolvedModel, Answers: decision.Answers,
		RequestID: decision.RequestID, EvaluationTimeMS: decision.EvaluationTimeMS, Usage: decision.Usage,
	}, nil
}

type openrouterPort struct{ client openrouter.Client }

func (port openrouterPort) AdapterID() string      { return openrouter.AdapterID }
func (port openrouterPort) AdapterVersion() string { return openrouter.AdapterVersion }

func (port openrouterPort) Evaluate(ctx context.Context, state any, questions map[string]any) (httpapi.Decision, error) {
	if n, ok := httpapi.AnswerBudget(ctx); ok {
		ctx = systemone.WithMaxResponseBytes(ctx, n)
	}
	decision, err := port.client.Evaluate(ctx, state, questions)
	if err != nil {
		return httpapi.Decision{}, err
	}
	return httpapi.Decision{
		ResolvedModel: decision.ResolvedModel, Answers: decision.Answers,
		RequestID: decision.RequestID, EvaluationTimeMS: decision.EvaluationTimeMS, Usage: decision.Usage,
	}, nil
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
		pin := os.Getenv("LEX_HOSTED_RESOLVED_MODEL")
		if !openrouter.ValidResolvedPin(pin) {
			return nil, fmt.Errorf("hosted adapter requires LEX_HOSTED_RESOLVED_MODEL")
		}
		return openrouterPort{client: openrouter.New(hostedKey, pin)}, nil
	default:
		return nil, nil
	}
}

func httpServer(addr string, handler http.Handler, requestTimeout time.Duration) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       requestTimeout,
		WriteTimeout:      requestTimeout + 5*time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
	server := httpServer(":"+port, handler, config.RequestTimeout)

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
