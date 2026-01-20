package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joshu-sajeev/goqueue/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	application, err := app.NewApiApp(ctx)
	if err != nil {
		log.Fatal("Couldn't start application:", err)
	}
	application.Run(ctx)
}
