package create

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
	"github.com/google/uuid"
)

const maxIDRetries = 3

// SSHConfig holds connection parameters for the customer server.
type SSHConfig struct {
	Host     string
	User     string
	KeyBytes []byte
}

// SSHResult holds the output of a remote command.
type SSHResult struct {
	Stdout string
	Stderr string
}

// SSHExecutor is a persistent connection to the customer server for the
// lifetime of a single Execute call. Defined here so this use case owns
// its dependency contract — the concrete sharedssh.Executor satisfies it
// via an adapter at the composition root.
type SSHExecutor interface {
	Run(cmd string) (SSHResult, error)
	Close() error
}

// SSHExecutorFactory mirrors the naming from sharedssh.Factory — it dials
// once and hands back a stateful executor the caller must Close.
type SSHExecutorFactory interface {
	NewExecutor(cfg SSHConfig) (SSHExecutor, error)
}

type Repo interface {
	Save(service domain.Service) error
	ExistsByDomain(domain string) (bool, error)
	Delete(id string) error
}

type UseCase struct {
	repo       Repo
	sshFactory SSHExecutorFactory
}

func New(repo Repo, sshFactory SSHExecutorFactory) *UseCase {
	return &UseCase{repo: repo, sshFactory: sshFactory}
}

func (uc *UseCase) Execute(input CreateInput) result.Result[CreateOutput] {
	if input.Name == "" || input.RepoURL == "" || input.Domain == "" ||
		input.HealthCheckURL == "" || input.WebhookSecret == "" ||
		input.Host == "" || input.SSHUser == "" || input.SSHKey == "" ||
		input.BluePort == 0 || input.GreenPort == 0 || input.ContainerPort == 0 {
		return result.Fail[CreateOutput](ErrInvalidInput)
	}
	if input.ComposeSvc == "" {
		input.ComposeSvc = "app"
	}

	if input.BluePort == input.GreenPort {
		return result.Fail[CreateOutput](fmt.Errorf("%w: blue and green ports must differ", ErrInvalidInput))
	}

	exists, err := uc.repo.ExistsByDomain(input.Domain)
	if err != nil {
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrPersistFailed, err))
	}
	if exists {
		return result.Fail[CreateOutput](ErrDomainTaken)
	}

	ex, err := uc.sshFactory.NewExecutor(SSHConfig{
		Host:     input.Host,
		User:     input.SSHUser,
		KeyBytes: []byte(input.SSHKey),
	})
	if err != nil {
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrSSHFailed, err))
	}
	defer ex.Close()

	if err := bootstrap(ex); err != nil {
		return result.Fail[CreateOutput](err)
	}

	free, err := portsAreFree(ex, input.BluePort, input.GreenPort)
	if err != nil {
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrPortScanFailed, err))
	}
	if !free {
		return result.Fail[CreateOutput](ErrPortConflict)
	}

	svc, err := uc.saveWithRetry(input)
	if err != nil {
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrPersistFailed, err))
	}

	return result.Ok(CreateOutput{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoURL:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckURL: svc.HealthCheckURL,
		Host:           svc.Host,
		SSHUser:        svc.SSHUser,
		SSHKey:     svc.SSHKey,
		BluePort:       svc.BluePort,
		GreenPort:      svc.GreenPort,
		ContainerPort:  svc.ContainerPort,
		ComposeSvc:     svc.ComposeSvc,
		ActiveSlot:     svc.ActiveSlot,
		CreatedAt:      svc.CreatedAt,
	})
}

// bootstrap prepares the customer server to accept Launchpad-managed nginx configs.
// Creates /etc/nginx/launchpad/ and idempotently adds the include directive to
// nginx.conf, then validates with nginx -t. No reload here — the first activation
// triggers the reload when there is actually a service config to serve.
func bootstrap(ex SSHExecutor) error {
	// confirm docker is present before committing to the service record
	if _, err := ex.Run("docker info >/dev/null 2>&1"); err != nil {
		return ErrDockerNotInstalled
	}

	// confirm nginx is present; we will write configs into its directory
	if _, err := ex.Run("nginx -v 2>/dev/null"); err != nil {
		return ErrNginxNotInstalled
	}

	if _, err := ex.Run("mkdir -p /etc/nginx/launchpad"); err != nil {
		return fmt.Errorf("%w: creating launchpad dir: %s", ErrBootstrapFailed, err)
	}

	// grep exits non-zero when the line is absent — the || branch appends it.
	// Run returning an error means the SSH command itself failed, not "line not found".
	includeCmd := `grep -qF 'include /etc/nginx/launchpad/*.conf;' /etc/nginx/nginx.conf || ` +
		`printf '\ninclude /etc/nginx/launchpad/*.conf;\n' >> /etc/nginx/nginx.conf`
	if _, err := ex.Run(includeCmd); err != nil {
		return fmt.Errorf("%w: adding include directive: %s", ErrBootstrapFailed, err)
	}

	// validate the config parses cleanly with the new include line in place
	res, err := ex.Run("nginx -t")
	if err != nil {
		return fmt.Errorf("%w: nginx -t: %s", ErrBootstrapFailed, res.Stderr)
	}

	return nil
}

// portsAreFree checks all given ports in a single ss invocation.
func portsAreFree(ex SSHExecutor, ports ...int) (bool, error) {
	res, err := ex.Run("ss -tln")
	if err != nil {
		return false, err
	}
	for _, port := range ports {
		if strings.Contains(res.Stdout, fmt.Sprintf(":%d ", port)) {
			return false, nil
		}
	}
	return true, nil
}

func buildService(input CreateInput) domain.Service {
	return domain.Service{
		Name:           input.Name,
		RepoURL:        input.RepoURL,
		Domain:         input.Domain,
		HealthCheckURL: input.HealthCheckURL,
		WebhookSecret:  input.WebhookSecret,
		Host:           input.Host,
		SSHUser:        input.SSHUser,
		SSHKey:     input.SSHKey,
		BluePort:       input.BluePort,
		GreenPort:      input.GreenPort,
		ContainerPort:  input.ContainerPort,
		ComposeSvc:     input.ComposeSvc,
		ActiveSlot:     nil,
		CreatedAt:      time.Now().UTC(),
	}
}

func (uc *UseCase) saveWithRetry(input CreateInput) (domain.Service, error) {
	svc := buildService(input)
	for i := 0; i < maxIDRetries; i++ {
		svc.ID = uuid.NewString()
		err := uc.repo.Save(svc)
		if err == nil {
			return svc, nil
		}
		if !errors.Is(err, ErrIDConflict) {
			return domain.Service{}, err
		}
	}
	return domain.Service{}, ErrIDConflict
}
