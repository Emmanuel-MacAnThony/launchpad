package create

import (
	"errors"
	"fmt"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
	"github.com/google/uuid"
)

const maxIDRetries = 3

type Repo interface {
	Save(service domain.Service) error
	ExistsByDomain(domain string) (bool, error)
	Delete(id string) error
}

type Nginx interface {
	WriteConfig(serviceID string, opts ...func(*NginxConfig)) error
	ReloadNginx() error
	DeleteConfig(serviceID string) error
}

type NginxConfig struct {
	Domain string
	Host   string
	Port   int
}

type UseCase struct {
	repo  Repo
	nginx Nginx
}

func New(repo Repo, nginx Nginx) *UseCase {
	return &UseCase{repo: repo, nginx: nginx}
}

func (uc *UseCase) Execute(input CreateInput) result.Result[CreateOutput] {
	if input.Name == "" || input.RepoURL == "" || input.Domain == "" ||
		input.HealthCheckURL == "" || input.WebhookSecret == "" ||
		input.Host == "" || input.SSHUser == "" || input.SSHKeyPath == "" {
		return result.Fail[CreateOutput](ErrInvalidInput)
	}

	exists, err := uc.repo.ExistsByDomain(input.Domain)
	if err != nil {
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrPersistFailed, err))
	}
	if exists {
		return result.Fail[CreateOutput](ErrDomainTaken)
	}

	svc, err := uc.saveWithRetry(input)
	if err != nil {
		// ErrIDConflict is not the client's fault — UUID collision is a server-side
		// transient issue. Wrap as ErrPersistFailed so the caller can retry.
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrPersistFailed, err))
	}

	if err := uc.nginx.WriteConfig(svc.ID, withDomain(svc.Domain), withHost(svc.Host)); err != nil {
		uc.repo.Delete(svc.ID)
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrNginxConfigFailed, err))
	}

	if err := uc.nginx.ReloadNginx(); err != nil {
		uc.nginx.DeleteConfig(svc.ID)
		uc.repo.Delete(svc.ID)
		return result.Fail[CreateOutput](fmt.Errorf("%w: %s", ErrNginxReloadFailed, err))
	}

	return result.Ok(CreateOutput{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoURL:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckURL: svc.HealthCheckURL,
		Host:           svc.Host,
		SSHUser:        svc.SSHUser,
		SSHKeyPath:     svc.SSHKeyPath,
		CreatedAt:      svc.CreatedAt,
	})
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
		SSHKeyPath:     input.SSHKeyPath,
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

func withDomain(d string) func(*NginxConfig) {
	return func(c *NginxConfig) { c.Domain = d }
}

func withHost(h string) func(*NginxConfig) {
	return func(c *NginxConfig) { c.Host = h }
}
