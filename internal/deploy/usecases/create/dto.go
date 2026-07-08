package create

import (
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
)

type CreateInput struct {
	ServiceID     string
	CommitSHA     string
	CommitMessage string
	PushedAt      time.Time
}

type CreateOutput struct {
	Deploy domain.Deploy
	Result CreateResult
}

// CreateResult describes what the incoming push event did to this service's deploy queue.
type CreateResult string

const (
	// DeployQueued means the queue slot was empty; a fresh pending deploy was created.
	DeployQueued CreateResult = "queued"

	// PendingPromoted means a pending deploy was already in the queue, but the incoming
	// push was newer (higher pushed_at) and was promoted into the pending slot, overwriting
	// the older commit. The older commit will never be deployed.
	PendingPromoted CreateResult = "promoted"

	// PushDiscarded means a pending deploy was already in the queue and is newer than
	// the incoming push. This webhook arrived out of order — the queue already holds
	// a more recent commit, so this push was ignored and the queue was left unchanged.
	PushDiscarded CreateResult = "discarded"
)
