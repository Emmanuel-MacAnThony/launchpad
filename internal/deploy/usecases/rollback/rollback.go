package rollback

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
	KeyBytes []byte
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
	GetByID(serviceID string) (servicedomain.Service, error)
	UpdateActiveSlot(serviceID string, slot servicedomain.Slot) error
}

type DeployRepo interface {
	GetLatestOnSlot(serviceID string, slot deploydomain.Slot) (deploydomain.Deploy, error)
	SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error
}

type RollbackInput struct {
	ServiceID string
}

type UseCase struct {
	serviceRepo ServiceRepo
	deployRepo  DeployRepo
	sshFactory  SSHExecutorFactory
}

func New(serviceRepo ServiceRepo, deployRepo DeployRepo, sshFactory SSHExecutorFactory) *UseCase {
	return &UseCase{serviceRepo: serviceRepo, deployRepo: deployRepo, sshFactory: sshFactory}
}

func (uc *UseCase) Execute(input RollbackInput) result.Result[struct{}] {
	if input.ServiceID == "" {
		return result.Fail[struct{}](ErrValidation)
	}

	svc, err := uc.serviceRepo.GetByID(input.ServiceID)
	if err != nil {
		if errors.Is(err, servicedomain.ErrNotFound) {
			return result.Fail[struct{}](ErrServiceNotFound)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	if svc.ActiveSlot == nil {
		return result.Fail[struct{}](ErrNoActiveDeployment)
	}

	activeSlot := deploydomain.Slot(*svc.ActiveSlot)
	currentActive, err := uc.deployRepo.GetLatestOnSlot(input.ServiceID, activeSlot)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrNoActiveDeployment)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	inactiveSlot := inactive(*svc.ActiveSlot)

	target, err := uc.deployRepo.GetLatestOnSlot(input.ServiceID, inactiveSlot)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrNoPreviousDeployment)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	// Determine the port for the slot we are rolling back to.
	inactivePort := svc.GreenPort
	if inactiveSlot == deploydomain.SlotBlue {
		inactivePort = svc.BluePort
	}

	config := nginxConfig(svc.Domain, inactivePort)

	tmp, err := os.CreateTemp("", fmt.Sprintf("launchpad-%s-*.conf", svc.ID))
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
		Host:     svc.Host,
		User:     svc.SSHUser,
		KeyBytes: []byte(svc.SSHKey),
	})
	if err != nil {
		return result.Fail[struct{}](fmt.Errorf("%w: %s", ErrSSHFailed, err))
	}
	defer ex.Close()

	remotePath := fmt.Sprintf("/etc/nginx/launchpad/%s.conf", svc.ID)

	if err := ex.Upload(tmp.Name(), remotePath); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	// Validate before reloading. If invalid, remove the uploaded file so nginx
	// stays on its current working config.
	if _, err := ex.Run("nginx -t"); err != nil {
		ex.Run(fmt.Sprintf("rm -f %s", remotePath))
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if _, err := ex.Run("nginx -s reload"); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	newActiveSlot := servicedomain.Slot(inactiveSlot)
	if err := uc.serviceRepo.UpdateActiveSlot(input.ServiceID, newActiveSlot); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.deployRepo.SetStatus(currentActive.ID, deploydomain.StatusRolledBack, nil); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.deployRepo.SetStatus(target.ID, deploydomain.StatusActive, nil); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	return result.Ok(struct{}{})
}

func inactive(slot servicedomain.Slot) deploydomain.Slot {
	if slot == servicedomain.SlotBlue {
		return deploydomain.SlotGreen
	}
	return deploydomain.SlotBlue
}

// nginxConfig generates the nginx server block that proxies to the target slot's port.
func nginxConfig(domain string, port int) string {
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
`, domain, port)
}
