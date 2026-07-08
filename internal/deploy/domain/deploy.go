package domain

import "time"

type DeployStatus string

const (
	StatusPending    DeployStatus = "pending"
	StatusBuilding   DeployStatus = "building"
	StatusActive     DeployStatus = "active"
	StatusFailed     DeployStatus = "failed"
	StatusRolledBack DeployStatus = "rolled_back"
)

type Slot string

const (
	SlotBlue  Slot = "blue"
	SlotGreen Slot = "green"
)

type Deploy struct {
	ID            string
	ServiceID     string
	Slot          *Slot
	Status        DeployStatus
	CommitSHA     string
	CommitMessage string
	RollbackOf    *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
}

type DeployLock struct {
	DeployID   string
	LockedAt   time.Time
	ExpiresAt  time.Time
	ReleasedAt *time.Time
}
