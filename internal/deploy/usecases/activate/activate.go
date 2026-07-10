package activate

import (
	"errors"
	"fmt"
	"os"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

// SSHConfig holds connection parameters for the customer server.
type SSHConfig struct {
	Host    string
	User    string
	KeyPath string
}

// SSHResult holds the output of a remote command.
type SSHResult struct {
	Stdout string
	Stderr string
}

// SSHExecutor is a persistent connection to the customer server.
// Defined here so this use case owns its dependency contract.
type SSHExecutor interface {
	Run(cmd string) (SSHResult, error)
	Upload(localPath, remotePath string) error
	Close() error
}

// SSHExecutorFactory dials the customer server and returns an Executor for
// the lifetime of a single Execute call.
type SSHExecutorFactory interface {
	NewExecutor(cfg SSHConfig) (SSHExecutor, error)
}

type ServiceRepo interface {
	UpdateActiveSlot(serviceID string, slot servicedomain.Slot) error
}

type DeployRepo interface {
	GetByID(deployID string) (deploydomain.Deploy, error)
	SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error
}

type LockRepo interface {
	ReleaseLock(deployID string) error
}

type ActivateInput struct {
	DeployID  string
	ServiceID string
	Slot      deploydomain.Slot

	// SSH + nginx fields — provided by the agent worker which already fetched
	// the service from DB. Passing them here avoids a second DB lookup in activate.
	Host       string
	SSHUser    string
	SSHKeyPath string
	Domain     string
	ActivePort int // resolved port for the target slot (blue or green)
}

type UseCase struct {
	sshFactory  SSHExecutorFactory
	serviceRepo ServiceRepo
	deployRepo  DeployRepo
	lockRepo    LockRepo
}

func New(sshFactory SSHExecutorFactory, serviceRepo ServiceRepo, deployRepo DeployRepo, lockRepo LockRepo) *UseCase {
	return &UseCase{
		sshFactory:  sshFactory,
		serviceRepo: serviceRepo,
		deployRepo:  deployRepo,
		lockRepo:    lockRepo,
	}
}

func (uc *UseCase) Execute(input ActivateInput) result.Result[struct{}] {
	if input.DeployID == "" || input.ServiceID == "" || input.Slot == "" ||
		input.Host == "" || input.SSHUser == "" || input.SSHKeyPath == "" ||
		input.Domain == "" || input.ActivePort == 0 {
		return result.Fail[struct{}](ErrValidation)
	}

	deploy, err := uc.deployRepo.GetByID(input.DeployID)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrDeployNotFound)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	if deploy.Status != deploydomain.StatusBuilding {
		return result.Fail[struct{}](ErrInvalidState)
	}

	// Generate the nginx config that proxies to the newly deployed container.
	// The config is regenerated on every activation — no stored metadata needed.
	config := nginxConfig(input.Domain, input.ActivePort)

	// Write config to a temp file on the Launchpad server so we can Upload it.
	tmp, err := os.CreateTemp("", fmt.Sprintf("launchpad-%s-*.conf", input.ServiceID))
	if err != nil {
		return result.Fail[struct{}](ErrInternal)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(config); err != nil {
		tmp.Close()
		return result.Fail[struct{}](ErrInternal)
	}
	tmp.Close()

	ex, err := uc.sshFactory.NewExecutor(SSHConfig{
		Host:    input.Host,
		User:    input.SSHUser,
		KeyPath: input.SSHKeyPath,
	})
	if err != nil {
		return result.Fail[struct{}](fmt.Errorf("%w: %s", ErrSSHFailed, err))
	}
	defer ex.Close()

	remotePath := fmt.Sprintf("/etc/nginx/launchpad/%s.conf", input.ServiceID)

	if err := ex.Upload(tmp.Name(), remotePath); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	// Validate the config before reloading. If invalid, delete the uploaded file
	// so nginx stays on its current working config.
	if _, err := ex.Run("nginx -t"); err != nil {
		ex.Run(fmt.Sprintf("rm -f %s", remotePath))
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if _, err := ex.Run("nginx -s reload"); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if err := uc.serviceRepo.UpdateActiveSlot(input.ServiceID, servicedomain.Slot(input.Slot)); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.deployRepo.SetStatus(input.DeployID, deploydomain.StatusActive, nil); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.lockRepo.ReleaseLock(input.DeployID); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	return result.Ok(struct{}{})
}

// nginxConfig generates the nginx server block that proxies to the active container.
// The entire file is regenerated on each deployment — no partial updates or patching.
func nginxConfig(domain string, activePort int) string {
	return fmt.Sprintf(`# Managed by Launchpad — do not edit manually
server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://127.0.0.1:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, activePort)
}
