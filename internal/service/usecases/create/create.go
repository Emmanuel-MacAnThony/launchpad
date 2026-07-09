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
	Domain    string
	Host      string
	BluePort  int
	GreenPort int
}

// SSHClient checks port availability on a remote host.
// The factory pattern is used because connection details vary per service —
// we cannot inject a single shared client at startup.
type SSHClient interface {
	AreFree(ports ...int) (bool, error)
}

type SSHClientFactory interface {
	New(host, user, keyPath string) SSHClient
}

type UseCase struct {
	repo       Repo
	nginx      Nginx
	sshFactory SSHClientFactory
}

func New(repo Repo, nginx Nginx, sshFactory SSHClientFactory) *UseCase {
	return &UseCase{repo: repo, nginx: nginx, sshFactory: sshFactory}
}

func (uc *UseCase) Execute(input CreateInput) result.Result[CreateOutput] {
	if input.Name == "" || input.RepoURL == "" || input.Domain == "" ||
		input.HealthCheckURL == "" || input.WebhookSecret == "" ||
		input.Host == "" || input.SSHUser == "" || input.SSHKeyPath == "" ||
		input.BluePort == 0 || input.GreenPort == 0 || input.ContainerPort == 0 {
		return result.Fail[CreateOutput](ErrInvalidInput)
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

	ssh := uc.sshFactory.New(input.Host, input.SSHUser, input.SSHKeyPath)
	free, err := ssh.AreFree(input.BluePort, input.GreenPort)
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

	if err := uc.nginx.WriteConfig(svc.ID, withDomain(svc.Domain), withHost(svc.Host), withBluePort(svc.BluePort), withGreenPort(svc.GreenPort)); err != nil {
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
		BluePort:       svc.BluePort,
		GreenPort:      svc.GreenPort,
		ContainerPort:  svc.ContainerPort,
		ActiveSlot:     svc.ActiveSlot,
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
		BluePort:       input.BluePort,
		GreenPort:      input.GreenPort,
		ContainerPort:  input.ContainerPort,
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

func withDomain(d string) func(*NginxConfig) {
	return func(c *NginxConfig) { c.Domain = d }
}

func withHost(h string) func(*NginxConfig) {
	return func(c *NginxConfig) { c.Host = h }
}

func withBluePort(p int) func(*NginxConfig) {
	return func(c *NginxConfig) { c.BluePort = p }
}

func withGreenPort(p int) func(*NginxConfig) {
	return func(c *NginxConfig) { c.GreenPort = p }
}
