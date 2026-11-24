// Command spine serves the API.
//
//	spine serve           run the API on SPINE_PORT
//	spine token <user>    print an access token for a user id (local use)
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dembygenesis/spine/internal/config"
	"github.com/dembygenesis/spine/internal/di"
	"github.com/dembygenesis/spine/pkg/observe"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "spine:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: spine serve | spine token <user-id>")
	}
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c, err := di.New(ctx, cfg, observe.NewSlog())
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	switch args[0] {
	case "serve":
		return serve(ctx, c, cfg.Port)
	case "token":
		if len(args) != 2 {
			return errors.New("usage: spine token <user-id>")
		}
		tok, err := c.JWT.Issue(args[1])
		if err != nil {
			return err
		}
		fmt.Println(tok)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context, c *di.Container, port int) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           c.Handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown) // best effort on the way out
	}()
	fmt.Printf("spine listening on :%d\n", port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
