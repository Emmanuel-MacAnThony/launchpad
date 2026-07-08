package infra

import (
	"context"
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/jackc/pgx/v5/pgtype"
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
	var activeSlot pgtype.Text
	if svc.ActiveSlot != nil {
		activeSlot = pgtype.Text{String: string(*svc.ActiveSlot), Valid: true}
	}

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
		BluePort:       int32(svc.BluePort),
		GreenPort:      int32(svc.GreenPort),
		ContainerPort:  int32(svc.ContainerPort),
		ActiveSlot:     activeSlot,
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
	return rowToDomain(row.ID, row.Name, row.RepoUrl, row.Domain, row.HealthCheckUrl,
		row.WebhookSecret, row.Host, row.SshUser, row.SshKeyPath,
		row.BluePort, row.GreenPort, row.ContainerPort, row.ActiveSlot, row.CreatedAt), nil
}

func (r *PostgresServiceRepository) ListAll() ([]domain.Service, error) {
	rows, err := r.queries.ListServices(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	svcs := make([]domain.Service, len(rows))
	for i, row := range rows {
		svcs[i] = rowToDomain(row.ID, row.Name, row.RepoUrl, row.Domain, row.HealthCheckUrl,
			row.WebhookSecret, row.Host, row.SshUser, row.SshKeyPath,
			row.BluePort, row.GreenPort, row.ContainerPort, row.ActiveSlot, row.CreatedAt)
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

func rowToDomain(
	id, name, repoURL, svcDomain, healthCheckURL, webhookSecret, host, sshUser, sshKeyPath string,
	bluePort, greenPort, containerPort int32,
	activeSlot pgtype.Text,
	createdAt pgtype.Timestamptz,
) domain.Service {
	var slot *domain.Slot
	if activeSlot.Valid {
		s := domain.Slot(activeSlot.String)
		slot = &s
	}

	return domain.Service{
		ID:             id,
		Name:           name,
		RepoURL:        repoURL,
		Domain:         svcDomain,
		HealthCheckURL: healthCheckURL,
		WebhookSecret:  webhookSecret,
		Host:           host,
		SSHUser:        sshUser,
		SSHKeyPath:     sshKeyPath,
		BluePort:       int(bluePort),
		GreenPort:      int(greenPort),
		ContainerPort:  int(containerPort),
		ActiveSlot:     slot,
		CreatedAt:      createdAt.Time,
	}
}
