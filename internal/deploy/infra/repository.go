package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/create"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type PostgresDeployRepository struct {
	queries *Queries
	pool    *pgxpool.Pool
	ctx     context.Context
}

func NewPostgresDeployRepository(ctx context.Context, pool *pgxpool.Pool) *PostgresDeployRepository {
	return &PostgresDeployRepository{
		queries: New(pool),
		pool:    pool,
		ctx:     ctx,
	}
}

func (r *PostgresDeployRepository) EnqueueDeploy(serviceID, commitSHA, commitMessage string, pushedAt time.Time) (deploydomain.Deploy, create.CreateResult, error) {
	tx, err := r.pool.Begin(r.ctx)
	if err != nil {
		return deploydomain.Deploy{}, "", fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(r.ctx)

	qtx := r.queries.WithTx(tx)

	// Lock the service row to serialise concurrent webhook enqueues for this service.
	_, err = qtx.LockServiceRow(r.ctx, serviceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return deploydomain.Deploy{}, "", deploydomain.ErrServiceNotFound
		}
		return deploydomain.Deploy{}, "", fmt.Errorf("locking service row: %w", err)
	}

	pending, err := qtx.GetPendingDeploy(r.ctx, serviceID)

	var row Deploy
	var queueResult create.CreateResult

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		row, err = qtx.InsertDeploy(r.ctx, InsertDeployParams{
			ID:            uuid.NewString(),
			ServiceID:     serviceID,
			CommitSha:     commitSHA,
			CommitMessage: commitMessage,
			PushedAt:      pgtype.Timestamptz{Time: pushedAt.UTC(), Valid: true},
		})
		if err != nil {
			return deploydomain.Deploy{}, "", fmt.Errorf("inserting deploy: %w", err)
		}
		queueResult = create.DeployQueued

	case err != nil:
		return deploydomain.Deploy{}, "", fmt.Errorf("getting pending deploy: %w", err)

	case pushedAt.After(pending.PushedAt.Time):
		row, err = qtx.UpgradePendingDeploy(r.ctx, UpgradePendingDeployParams{
			ID:            pending.ID,
			CommitSha:     commitSHA,
			CommitMessage: commitMessage,
			PushedAt:      pgtype.Timestamptz{Time: pushedAt.UTC(), Valid: true},
		})
		if err != nil {
			return deploydomain.Deploy{}, "", fmt.Errorf("upgrading pending deploy: %w", err)
		}
		queueResult = create.PendingPromoted

	default:
		// Incoming push is stale — the queue already holds a newer commit.
		// No writes needed; let the deferred rollback release the lock.
		return rowToDomain(pending), create.PushDiscarded, nil
	}

	if err := tx.Commit(r.ctx); err != nil {
		return deploydomain.Deploy{}, "", fmt.Errorf("committing transaction: %w", err)
	}

	return rowToDomain(row), queueResult, nil
}

func (r *PostgresDeployRepository) ListPending() ([]deploydomain.Deploy, error) {
	rows, err := r.queries.ListPendingDeploys(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("listing pending deploys: %w", err)
	}
	deploys := make([]deploydomain.Deploy, len(rows))
	for i, row := range rows {
		deploys[i] = rowToDomain(row)
	}
	return deploys, nil
}

func rowToDomain(row Deploy) deploydomain.Deploy {
	var slot *deploydomain.Slot
	if row.Slot.Valid {
		s := deploydomain.Slot(row.Slot.String)
		slot = &s
	}

	var startedAt *time.Time
	if row.StartedAt.Valid {
		t := row.StartedAt.Time
		startedAt = &t
	}

	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		finishedAt = &t
	}

	return deploydomain.Deploy{
		ID:            row.ID,
		ServiceID:     row.ServiceID,
		Slot:          slot,
		Status:        deploydomain.DeployStatus(row.Status),
		CommitSHA:     row.CommitSha,
		CommitMessage: row.CommitMessage,
		PushedAt:      row.PushedAt.Time,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		CreatedAt:     row.CreatedAt.Time,
	}
}
