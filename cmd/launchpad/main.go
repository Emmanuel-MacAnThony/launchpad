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

	"github.com/Emmanuel-MacAnThony/launchpad/internal/api"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/config"
	appdb "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/db"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/infra"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/list"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/update"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/logger"
)

func main() {
	log := logger.New()
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool := appdb.Connect(ctx, cfg.DB.URL)
	defer pool.Close()

	repo := infra.NewPostgresServiceRepository(ctx, pool)
	createSvc := create.New(repo, nil) // nginx client wired up once implemented
	getSvc := get.New(repo)
	updateSvc := update.New(repo)
	listSvc := list.New(repo)

	router := api.NewRouter(api.RouterDeps{
		Service: api.NewServiceHandler(api.ServiceHandlerDeps{
			BaseURL:       cfg.Server.BaseURL,
			CreateService: createSvc,
			GetService:    getSvc,
			UpdateService: updateSvc,
			ListServices:  listSvc,
		}),
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", "err", err)
		}
	}()

	log.Info("launchpad server starting", "port", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}
