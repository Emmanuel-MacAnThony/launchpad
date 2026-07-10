package main

import (
	"context"
	"encoding/hex"
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
	sharednginx "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/nginx"
	sharedssh "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/ssh"
	deployinfra "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/infra"
	deploycreate "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/create"
	deployget "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/getdeploy"
	deploylist "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/listdeploys"
	deployrollback "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/rollback"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/infra"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/list"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/update"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/crypto"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/logger"
)

// sshFactoryAdapter bridges sharedssh.Factory to create.SSHClientFactory.
// Use-cases define their own interfaces (dependency inversion); neither the
// use case nor the ssh package should depend on each other. The adapter lives
// here at the composition root — the one place that's allowed to know about
// both sides.
type sshFactoryAdapter struct{ f *sharedssh.Factory }

func (a *sshFactoryAdapter) New(host, user, keyPath string) create.SSHClient {
	return a.f.New(host, user, keyPath)
}

func main() {
	log := logger.New()
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	keyBytes, err := hex.DecodeString(cfg.Crypto.EncryptionKey)
	if err != nil {
		log.Error("invalid ENCRYPTION_KEY: must be hex-encoded 32 bytes", "err", err)
		os.Exit(1)
	}
	crypter, err := crypto.NewAESGCMCrypter(keyBytes)
	if err != nil {
		log.Error("failed to initialise crypter", "err", err)
		os.Exit(1)
	}

	pool := appdb.Connect(ctx, cfg.DB.URL)
	defer pool.Close()

	nginxClient := sharednginx.NewClient(cfg.Nginx.BaseDir)

	repo := infra.NewPostgresServiceRepository(ctx, pool, crypter)
	createSvc := create.New(repo, nginxClient, &sshFactoryAdapter{f: &sharedssh.Factory{}})

	getSvc := get.New(repo)
	updateSvc := update.New(repo)
	listSvc := list.New(repo)

	deployRepo := deployinfra.NewPostgresDeployRepository(ctx, pool)
	createDeploySvc := deploycreate.New(deployRepo)
	getDeploySvc := deployget.New(deployRepo)
	listDeploysSvc := deploylist.New(deployRepo)
	rollbackSvc := deployrollback.New(repo, deployRepo, nginxClient)

	router := api.NewRouter(api.RouterDeps{
		Service: api.NewServiceHandler(api.ServiceHandlerDeps{
			BaseURL:       cfg.Server.BaseURL,
			CreateService: createSvc,
			GetService:    getSvc,
			UpdateService: updateSvc,
			ListServices:  listSvc,
		}),
		Webhook: api.NewWebhookHandler(api.WebhookHandlerDeps{
			GetService:   getSvc,
			CreateDeploy: createDeploySvc,
		}),
		Deploy: api.NewDeployHandler(api.DeployHandlerDeps{
			GetDeploy:   getDeploySvc,
			ListDeploys: listDeploysSvc,
			Rollback:    rollbackSvc,
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
