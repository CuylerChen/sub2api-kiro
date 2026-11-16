//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubKiroTokenRefresherOAuth struct {
	tokenInfo *KiroTokenInfo
	err       error
	calls     int
}

func (s *stubKiroTokenRefresherOAuth) RefreshAccountToken(context.Context, *Account) (*KiroTokenInfo, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.tokenInfo, nil
}

func (s *stubKiroTokenRefresherOAuth) BuildAccountCredentials(tokenInfo *KiroTokenInfo) map[string]any {
	if tokenInfo == nil {
		return map[string]any{}
	}
	creds := map[string]any{}
	if tokenInfo.AccessToken != "" {
		creds["access_token"] = tokenInfo.AccessToken
	}
	if tokenInfo.RefreshToken != "" {
		creds["refresh_token"] = tokenInfo.RefreshToken
	}
	if tokenInfo.ExpiresAt != "" {
		creds["expires_at"] = tokenInfo.ExpiresAt
	}
	return creds
}

func kiroOAuthAccount(creds map[string]any) *Account {
	return &Account{
		ID:          38,
		Platform:    PlatformKiro,
		Type:        AccountTypeOAuth,
		Credentials: creds,
	}
}

func TestKiroTokenRefresher_NeedsRefreshMissingExpiresAt(t *testing.T) {
	refresher := NewKiroTokenRefresher(nil)
	require.True(t, refresher.NeedsRefresh(kiroOAuthAccount(map[string]any{
		"refresh_token": "rt",
	}), 0))
}

func TestKiroTokenRefresher_NeedsRefreshProvisionalWaitsUntilExpiry(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	refresher := &KiroTokenRefresher{now: func() time.Time { return now }}
	account := kiroOAuthAccount(map[string]any{
		"refresh_token":             "rt",
		"expires_at":                now.Add(10 * time.Minute).Format(time.RFC3339),
		kiroProvisionalExpiresAtKey: true,
	})
	require.False(t, refresher.NeedsRefresh(account, 0))

	refresher.now = func() time.Time { return now.Add(10 * time.Minute) }
	require.True(t, refresher.NeedsRefresh(account, 0))
}

func TestKiroTokenRefresher_NeedsRefreshRealExpiryUsesWindow(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	refresher := &KiroTokenRefresher{now: func() time.Time { return now }}
	require.False(t, refresher.NeedsRefresh(kiroOAuthAccount(map[string]any{
		"refresh_token": "rt",
		"expires_at":    now.Add(time.Hour).Format(time.RFC3339),
	}), 0))
	require.True(t, refresher.NeedsRefresh(kiroOAuthAccount(map[string]any{
		"refresh_token": "rt",
		"expires_at":    now.Add(10 * time.Minute).Format(time.RFC3339),
	}), 0))
}

func TestKiroTokenRefresher_RefreshBootstrapsMissingExpiresAt(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	oauth := &stubKiroTokenRefresherOAuth{}
	refresher := &KiroTokenRefresher{
		kiroOAuthService: oauth,
		now:              func() time.Time { return now },
	}
	creds, err := refresher.Refresh(context.Background(), kiroOAuthAccount(map[string]any{
		"refresh_token": "rt",
		"access_token":  "at",
	}))
	require.NoError(t, err)
	require.Equal(t, 0, oauth.calls)
	require.Equal(t, now.Add(10*time.Minute).UTC().Format(time.RFC3339), creds["expires_at"])
	require.Equal(t, true, creds[kiroProvisionalExpiresAtKey])
	require.Equal(t, "rt", creds["refresh_token"])
	require.Equal(t, "at", creds["access_token"])
}

func TestKiroTokenRefresher_RefreshPersistsUpstreamExpiresAt(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	upstreamExpiry := now.Add(55 * time.Minute).UTC().Format(time.RFC3339)
	oauth := &stubKiroTokenRefresherOAuth{
		tokenInfo: &KiroTokenInfo{
			AccessToken:  "new-at",
			RefreshToken: "new-rt",
			ExpiresAt:    upstreamExpiry,
		},
	}
	refresher := &KiroTokenRefresher{
		kiroOAuthService: oauth,
		now:              func() time.Time { return now },
	}
	creds, err := refresher.Refresh(context.Background(), kiroOAuthAccount(map[string]any{
		"refresh_token":             "old-rt",
		"access_token":              "old-at",
		"expires_at":                now.Add(-time.Minute).Format(time.RFC3339),
		kiroProvisionalExpiresAtKey: true,
		"profile_arn":               "arn:example",
	}))
	require.NoError(t, err)
	require.Equal(t, 1, oauth.calls)
	require.Equal(t, "new-at", creds["access_token"])
	require.Equal(t, "new-rt", creds["refresh_token"])
	require.Equal(t, upstreamExpiry, creds["expires_at"])
	require.Equal(t, "arn:example", creds["profile_arn"])
	_, hasProvisional := creds[kiroProvisionalExpiresAtKey]
	require.False(t, hasProvisional)
}

func TestKiroTokenRefresher_RefreshDefaultsExpiresAtWhenUpstreamOmitsIt(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	oauth := &stubKiroTokenRefresherOAuth{
		tokenInfo: &KiroTokenInfo{
			AccessToken:  "new-at",
			RefreshToken: "new-rt",
		},
	}
	refresher := &KiroTokenRefresher{
		kiroOAuthService: oauth,
		now:              func() time.Time { return now },
	}
	creds, err := refresher.Refresh(context.Background(), kiroOAuthAccount(map[string]any{
		"refresh_token": "old-rt",
		"expires_at":    now.Add(-time.Second).Format(time.RFC3339),
	}))
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Hour).UTC().Format(time.RFC3339), creds["expires_at"])
}
