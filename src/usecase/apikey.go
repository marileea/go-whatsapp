package usecase

import (
	"context"
	"fmt"

	domainAPIKey "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/apikey"
	domainTenant "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/tenant"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/apikeyutil"
	"github.com/google/uuid"
)

type apiKeyService struct {
	repo       domainAPIKey.IAPIKeyRepository
	tenantRepo domainTenant.ITenantRepository
	salt       string
	keyPrefix  string
}

func NewAPIKeyService(repo domainAPIKey.IAPIKeyRepository, tenantRepo domainTenant.ITenantRepository, salt, keyPrefix string) domainAPIKey.IAPIKeyUsecase {
	if repo == nil || tenantRepo == nil {
		panic("apikey usecase: repositories are required")
	}
	if salt == "" {
		panic("apikey usecase: hashing salt is required")
	}
	return &apiKeyService{
		repo:       repo,
		tenantRepo: tenantRepo,
		salt:       salt,
		keyPrefix:  keyPrefix,
	}
}

func (s *apiKeyService) CreateKey(ctx context.Context, req *domainAPIKey.CreateAPIKeyRequest) (*domainAPIKey.CreateAPIKeyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.Scopes) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}
	if tenant.Status != domainTenant.TenantStatusActive {
		return nil, fmt.Errorf("tenant is not active")
	}

	plainKey, preview, err := apikeyutil.GeneratePlainKey(s.keyPrefix, 32)
	if err != nil {
		return nil, err
	}
	hash, err := apikeyutil.HashKey(plainKey, s.salt)
	if err != nil {
		return nil, err
	}

	keyPrefix := s.keyPrefix
	if keyPrefix == "" {
		keyPrefix = preview
	}

	key := &domainAPIKey.APIKey{
		ID:        uuid.NewString(),
		TenantID:  req.TenantID,
		Name:      req.Name,
		KeyHash:   hash,
		KeyPrefix: keyPrefix,
		Scopes:    append([]string{}, req.Scopes...),
		Status:    domainAPIKey.APIKeyStatusActive,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}

	return &domainAPIKey.CreateAPIKeyResponse{
		APIKey:   key,
		PlainKey: plainKey,
	}, nil
}

func (s *apiKeyService) RotateKey(ctx context.Context, apiKeyID string) (*domainAPIKey.CreateAPIKeyResponse, error) {
	if apiKeyID == "" {
		return nil, fmt.Errorf("api key id is required")
	}

	key, err := s.repo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	if key.Status != domainAPIKey.APIKeyStatusActive {
		return nil, fmt.Errorf("only active keys can be rotated")
	}

	resp, err := s.CreateKey(ctx, &domainAPIKey.CreateAPIKeyRequest{
		TenantID:  key.TenantID,
		Name:      key.Name,
		Scopes:    key.Scopes,
		ExpiresAt: key.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	if err := s.DisableKey(ctx, apiKeyID); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *apiKeyService) DisableKey(ctx context.Context, apiKeyID string) error {
	if apiKeyID == "" {
		return fmt.Errorf("api key id is required")
	}

	status := domainAPIKey.APIKeyStatusRevoked
	return s.repo.Update(ctx, apiKeyID, &domainAPIKey.UpdateAPIKeyRequest{
		Status: &status,
	})
}

func (s *apiKeyService) DescribeKey(ctx context.Context, apiKeyID string) (*domainAPIKey.APIKey, error) {
	if apiKeyID == "" {
		return nil, fmt.Errorf("api key id is required")
	}
	return s.repo.GetByID(ctx, apiKeyID)
}

func (s *apiKeyService) ListKeys(ctx context.Context, tenantID string, limit, offset int) ([]*domainAPIKey.APIKey, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByTenantID(ctx, tenantID, limit, offset)
}
