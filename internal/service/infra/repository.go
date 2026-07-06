package infra

import (
	"context"
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresServiceRepository struct {
	queries *Queries
	ctx     context.Context
}

func NewPostgresServiceRepository(ctx context.Context, db *pgxpool.Pool) *PostgresServiceRepository {
	return &PostgresServiceRepository{
		queries: New(db),
		ctx:     ctx,
	}
}

func (r *PostgresServiceRepository) Save(svc domain.Service) error {
	err := r.queries.SaveService(r.ctx, SaveServiceParams{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoUrl:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckUrl: svc.HealthCheckURL,
		WebhookSecret:  svc.WebhookSecret,
		Host:           svc.Host,
		SshUser:        svc.SSHUser,
		SshKeyPath:     svc.SSHKeyPath,
	})
	if err != nil {
		return fmt.Errorf("saving service: %w", err)
	}
	return nil
}

func (r *PostgresServiceRepository) ExistsByDomain(d string) (bool, error) {
	exists, err := r.queries.ExistsByDomain(r.ctx, d)
	if err != nil {
		return false, fmt.Errorf("checking domain existence: %w", err)
	}
	return exists, nil
}

func (r *PostgresServiceRepository) Delete(id string) error {
	if err := r.queries.DeleteService(r.ctx, id); err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}
	return nil
}

func (r *PostgresServiceRepository) GetByID(id string) (domain.Service, error) {
	row, err := r.queries.GetService(r.ctx, id)
	if err != nil {
		return domain.Service{}, fmt.Errorf("getting service: %w", err)
	}
	return toDomain(row), nil
}

func (r *PostgresServiceRepository) ListAll() ([]domain.Service, error) {
	rows, err := r.queries.ListServices(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	svcs := make([]domain.Service, len(rows))
	for i, row := range rows {
		svcs[i] = toDomain(row)
	}
	return svcs, nil
}

func (r *PostgresServiceRepository) Update(id, name, healthCheckURL string) error {
	if err := r.queries.UpdateService(r.ctx, UpdateServiceParams{
		ID:             id,
		Name:           name,
		HealthCheckUrl: healthCheckURL,
	}); err != nil {
		return fmt.Errorf("updating service: %w", err)
	}
	return nil
}

func toDomain(row Service) domain.Service {
	return domain.Service{
		ID:             row.ID,
		Name:           row.Name,
		RepoURL:        row.RepoUrl,
		Domain:         row.Domain,
		HealthCheckURL: row.HealthCheckUrl,
		WebhookSecret:  row.WebhookSecret,
		Host:           row.Host,
		SSHUser:        row.SshUser,
		SSHKeyPath:     row.SshKeyPath,
		CreatedAt:      row.CreatedAt.Time,
	}
}
