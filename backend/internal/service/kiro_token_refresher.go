package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	kiroRefreshWindow            = 15 * time.Minute
	kiroMissingExpiresAtGrace    = 10 * time.Minute
	kiroProvisionalExpiresAtKey  = "_kiro_expires_at_provisional"
	kiroRefreshedTokenDefaultTTL = time.Hour
)

type kiroAccountTokenRefresherService interface {
	RefreshAccountToken(ctx context.Context, account *Account) (*KiroTokenInfo, error)
	BuildAccountCredentials(tokenInfo *KiroTokenInfo) map[string]any
}

type KiroTokenRefresher struct {
	kiroOAuthService kiroAccountTokenRefresherService
	now              func() time.Time
}

func NewKiroTokenRefresher(kiroOAuthService *KiroOAuthService) *KiroTokenRefresher {
	return &KiroTokenRefresher{
		kiroOAuthService: kiroOAuthService,
	}
}

func (r *KiroTokenRefresher) currentTime() time.Time {
	if r != nil && r.now != nil {
		return r.now()
	}
	return time.Now()
}

func (r *KiroTokenRefresher) CacheKey(account *Account) string {
	return KiroTokenCacheKey(account)
}

func (r *KiroTokenRefresher) CanRefresh(account *Account) bool {
	return account != nil && account.Platform == PlatformKiro && account.Type == AccountTypeOAuth
}

func (r *KiroTokenRefresher) NeedsRefresh(account *Account, _ time.Duration) bool {
	if !r.CanRefresh(account) {
		return false
	}
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return true
	}
	until := expiresAt.Sub(r.currentTime())
	if kiroHasProvisionalExpiresAt(account) {
		return until <= 0
	}
	return until <= kiroRefreshWindow
}

func (r *KiroTokenRefresher) Refresh(ctx context.Context, account *Account) (map[string]any, error) {
	if !r.CanRefresh(account) {
		return nil, fmt.Errorf("not a kiro oauth account")
	}
	if account.GetCredentialAsTime("expires_at") == nil {
		return MergeCredentials(account.Credentials, map[string]any{
			"expires_at":                r.currentTime().Add(kiroMissingExpiresAtGrace).UTC().Format(time.RFC3339),
			kiroProvisionalExpiresAtKey: true,
		}), nil
	}
	if r.kiroOAuthService == nil {
		return nil, fmt.Errorf("kiro oauth service is nil")
	}

	tokenInfo, err := r.kiroOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return nil, err
	}

	return applyKiroRefreshedCredentials(account.Credentials, r.kiroOAuthService.BuildAccountCredentials(tokenInfo), tokenInfo, r.currentTime()), nil
}

func kiroHasProvisionalExpiresAt(account *Account) bool {
	if account == nil || account.Credentials == nil {
		return false
	}
	value, ok := account.Credentials[kiroProvisionalExpiresAtKey]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func applyKiroRefreshedCredentials(oldCredentials, builtCredentials map[string]any, tokenInfo *KiroTokenInfo, now time.Time) map[string]any {
	merged := MergeCredentials(oldCredentials, builtCredentials)
	expiresAt := ""
	if tokenInfo != nil {
		expiresAt = strings.TrimSpace(tokenInfo.ExpiresAt)
	}
	if expiresAt == "" {
		expiresAt = now.UTC().Add(kiroRefreshedTokenDefaultTTL).Format(time.RFC3339)
	}
	merged["expires_at"] = expiresAt
	delete(merged, kiroProvisionalExpiresAtKey)
	return merged
}
