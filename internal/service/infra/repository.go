package infra

import (
	"context"
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/crypto"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresServiceRepository struct {
	queries *Queries
	ctx     context.Context
	crypter crypto.Crypter
}

func NewPostgresServiceRepository(ctx context.Context, db *pgxpool.Pool, crypter crypto.Crypter) *PostgresServiceRepository {
	return &PostgresServiceRepository{
		queries: New(db),
		ctx:     ctx,
		crypter: crypter,
	}
}

func (r *PostgresServiceRepository) Save(svc domain.Service) error {
	encryptedSecret, err := r.crypter.Encrypt(svc.WebhookSecret)
	if err != nil {
		return fmt.Errorf("encrypting webhook secret: %w", err)
	}

	encryptedSSHKey, err := r.crypter.Encrypt(svc.SSHKey)
	if err != nil {
		return fmt.Errorf("encrypting ssh key: %w", err)
	}

	var activeSlot pgtype.Text
	if svc.ActiveSlot != nil {
		activeSlot = pgtype.Text{String: string(*svc.ActiveSlot), Valid: true}
	}

	err = r.queries.SaveService(r.ctx, SaveServiceParams{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoUrl:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckUrl: svc.HealthCheckURL,
		WebhookSecret:  encryptedSecret,
		Host:           svc.Host,
		SshUser:        svc.SSHUser,
		SshPrivateKey:  encryptedSSHKey,
		BluePort:       int32(svc.BluePort),
		GreenPort:      int32(svc.GreenPort),
		ContainerPort:  int32(svc.ContainerPort),
		ComposeService: svc.ComposeSvc,
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
	return r.rowToDomain(row.ID, row.Name, row.RepoUrl, row.Domain, row.HealthCheckUrl,
		row.WebhookSecret, row.Host, row.SshUser, row.SshPrivateKey,
		row.BluePort, row.GreenPort, row.ContainerPort, row.ComposeService, row.ActiveSlot, row.CreatedAt)
}

func (r *PostgresServiceRepository) ListAll() ([]domain.Service, error) {
	rows, err := r.queries.ListServices(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	svcs := make([]domain.Service, len(rows))
	for i, row := range rows {
		svc, err := r.rowToDomain(row.ID, row.Name, row.RepoUrl, row.Domain, row.HealthCheckUrl,
			row.WebhookSecret, row.Host, row.SshUser, row.SshPrivateKey,
			row.BluePort, row.GreenPort, row.ContainerPort, row.ComposeService, row.ActiveSlot, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		svcs[i] = svc
	}
	return svcs, nil
}

func (r *PostgresServiceRepository) UpdateActiveSlot(serviceID string, slot domain.Slot) error {
	if err := r.queries.UpdateServiceActiveSlot(r.ctx, UpdateServiceActiveSlotParams{
		ID:         serviceID,
		ActiveSlot: pgtype.Text{String: string(slot), Valid: true},
	}); err != nil {
		return fmt.Errorf("updating active slot: %w", err)
	}
	return nil
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

func (r *PostgresServiceRepository) rowToDomain(
	id, name, repoURL, svcDomain, healthCheckURL, webhookSecret, host, sshUser, sshPrivateKey string,
	bluePort, greenPort, containerPort int32,
	composeSvc string,
	activeSlot pgtype.Text,
	createdAt pgtype.Timestamptz,
) (domain.Service, error) {
	decryptedSecret, err := r.crypter.Decrypt(webhookSecret)
	if err != nil {
		return domain.Service{}, fmt.Errorf("decrypting webhook secret: %w", err)
	}

	decryptedSSHKey, err := r.crypter.Decrypt(sshPrivateKey)
	if err != nil {
		return domain.Service{}, fmt.Errorf("decrypting ssh key: %w", err)
	}

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
		WebhookSecret:  decryptedSecret,
		Host:           host,
		SSHUser:        sshUser,
		SSHKey:         decryptedSSHKey,
		BluePort:       int(bluePort),
		GreenPort:      int(greenPort),
		ContainerPort:  int(containerPort),
		ComposeSvc:     composeSvc,
		ActiveSlot:     slot,
		CreatedAt:      createdAt.Time,
	}, nil
}
