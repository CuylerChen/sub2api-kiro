//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnthropicKiroGatewayCacheUsageSplitsLikeKiroGateway(t *testing.T) {
	system := strings.Repeat("stable system token ", 3000)
	body := []byte(`{"model":"claude-opus-5-5","system":[{"type":"text","text":"` + system + `","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":"current turn"}]}`)
	first := ClaudeUsage{InputTokens: 2000, CacheReadInputTokens: 60000, CacheCreationInputTokens: 10000, OutputTokens: 25}
	require.True(t, applyAnthropicKiroGatewayCacheUsage(41, "session-a", "claude-opus-5-5", body, &first))
	require.Equal(t, 72000, first.InputTokens+first.CacheReadInputTokens+first.CacheCreationInputTokens)
	require.Equal(t, 25, first.OutputTokens)
	require.Zero(t, first.CacheReadInputTokens)
	require.Greater(t, first.CacheCreationInputTokens, 0)
	require.Equal(t, first.CacheCreationInputTokens, first.CacheCreation5mTokens)

	second := ClaudeUsage{InputTokens: 2000, CacheReadInputTokens: 60000, CacheCreationInputTokens: 10000, OutputTokens: 25}
	require.True(t, applyAnthropicKiroGatewayCacheUsage(41, "session-a", "claude-opus-5-5", body, &second))
	require.Equal(t, 72000, second.InputTokens+second.CacheReadInputTokens+second.CacheCreationInputTokens)
	require.Greater(t, second.CacheReadInputTokens, 0)
	require.Zero(t, second.CacheCreationInputTokens)

	unchanged := ClaudeUsage{InputTokens: 10, CacheReadInputTokens: 20, OutputTokens: 3}
	plain := []byte(`{"model":"claude-opus-5-5","messages":[{"role":"user","content":"hi"}]}`)
	require.False(t, applyAnthropicKiroGatewayCacheUsage(41, "session-a", "claude-opus-5-5", plain, &unchanged))
	require.Equal(t, 20, unchanged.CacheReadInputTokens)
	require.False(t, applyAnthropicKiroGatewayCacheUsage(41, "session-a", "gpt-5.4", body, &unchanged))
}
