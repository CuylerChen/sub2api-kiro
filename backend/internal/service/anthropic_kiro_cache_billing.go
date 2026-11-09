package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/kiro/anthropictokenizer"
)

const anthropicKiroCacheMaxPoints = 4

type anthropicKiroCachePoint struct {
	fingerprint string
	cumulative  int
	kind        string
}

type anthropicKiroCacheStore struct {
	mu      sync.Mutex
	hits    map[string]time.Time
	history map[string]string
}

var anthropicKiroCaches = &anthropicKiroCacheStore{
	hits:    map[string]time.Time{},
	history: map[string]string{},
}

// applyAnthropicKiroGatewayCacheUsage 用 kiro-gateway 的断点规则重算缓存读写 token。
// 输入侧 token 总数保持上游原值，只改 input / cache_read / cache_creation 的划分。
// 请求没有 cache_control 时不改，返回 false。
func applyAnthropicKiroGatewayCacheUsage(accountID int64, sessionKey, model string, body []byte, usage *ClaudeUsage) bool {
	if usage == nil || accountID <= 0 || len(body) == 0 || !isClaudeBillingModel(model) {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	if !anthropicRequestHasCacheControl(payload) {
		return false
	}
	minTokens := kiroMinimumCacheableTokens(model)
	points, totalTokens, ttl := buildAnthropicKiroCachePoints(accountID, sessionKey, model, payload, minTokens)
	local := consumeAnthropicKiroCache(accountID, points, totalTokens, minTokens, ttl)
	upstreamTotal := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
	localTotal := local.InputTokens + local.CacheReadInputTokens + local.CacheCreationInputTokens
	if upstreamTotal <= 0 {
		upstreamTotal = localTotal
	}
	if localTotal <= 0 || upstreamTotal <= 0 {
		return false
	}
	read := int(math.Round(float64(upstreamTotal) * float64(local.CacheReadInputTokens) / float64(localTotal)))
	write := int(math.Round(float64(upstreamTotal) * float64(local.CacheCreationInputTokens) / float64(localTotal)))
	if read < 0 {
		read = 0
	}
	if write < 0 {
		write = 0
	}
	if read+write > upstreamTotal {
		overflow := read + write - upstreamTotal
		if write >= overflow {
			write -= overflow
		} else {
			overflow -= write
			write = 0
			read = max(0, read-overflow)
		}
	}
	usage.CacheReadInputTokens = read
	usage.CacheCreationInputTokens = write
	usage.InputTokens = upstreamTotal - read - write
	if write > 0 && ttl >= kiroCacheOneHourTTL {
		usage.CacheCreation1hTokens = write
		usage.CacheCreation5mTokens = 0
	} else if write > 0 {
		usage.CacheCreation5mTokens = write
		usage.CacheCreation1hTokens = 0
	} else {
		usage.CacheCreation5mTokens = 0
		usage.CacheCreation1hTokens = 0
	}
	return true
}

func isClaudeBillingModel(model string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(model)), "claude")
}

func anthropicRequestHasCacheControl(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	for _, block := range normalizeKiroSystemBlocks(payload["system"]) {
		if extractKiroCacheTTL(block) != nil {
			return true
		}
	}
	if tools, ok := payload["tools"].([]any); ok {
		for _, tool := range tools {
			if extractKiroCacheTTL(tool) != nil {
				return true
			}
		}
	}
	messages, _ := payload["messages"].([]any)
	for _, raw := range messages {
		message, _ := raw.(map[string]any)
		switch content := message["content"].(type) {
		case string:
			continue
		case []any:
			for _, block := range content {
				if extractKiroCacheTTL(block) != nil {
					return true
				}
			}
		}
	}
	return false
}

func buildAnthropicKiroCachePoints(accountID int64, sessionKey, model string, payload map[string]any, minTokens int) ([]anthropicKiroCachePoint, int, time.Duration) {
	ttl := anthropicMaxCacheTTL(payload)
	toolCache := false
	var toolTokens int
	if tools, ok := payload["tools"].([]any); ok {
		for _, tool := range tools {
			if extractKiroCacheTTL(tool) != nil {
				toolCache = true
			}
			toolTokens += kiroTokensPerTool
		}
	}
	segments, dynamic := splitAnthropicSystemCacheSegments(payload["system"])
	reserved := 2
	if toolCache {
		reserved++
	}
	slots := max(0, anthropicKiroCacheMaxPoints-reserved)
	if len(segments) > slots {
		segments = segments[:slots]
	}

	var points []anthropicKiroCachePoint
	systemTokens := 0
	var systemParts []string
	for _, segment := range segments {
		systemParts = append(systemParts, segment)
		systemTokens += anthropictokenizer.CountTokens(segment)
		if systemTokens < minTokens {
			continue
		}
		points = append(points, anthropicKiroCachePoint{
			fingerprint: anthropicKiroFingerprint("system", systemParts),
			cumulative:  systemTokens,
			kind:        "system",
		})
	}

	history, currentTokens := anthropicHistoryAndCurrent(payload["messages"])
	if dynamic != "" {
		currentTokens += anthropictokenizer.CountTokens(dynamic)
	}
	checkpoint := anthropicKiroCaches.historyCheckpoint(accountID, sessionKey, model)
	historyTokens := systemTokens
	var historyParts []string
	var historyPoints []anthropicKiroCachePoint
	for _, text := range history {
		historyParts = append(historyParts, text)
		historyTokens += anthropictokenizer.CountTokens(text) + kiroTokensPerMessage
		if historyTokens < minTokens {
			continue
		}
		historyPoints = append(historyPoints, anthropicKiroCachePoint{
			fingerprint: anthropicKiroFingerprint("history", append(append([]string{}, systemParts...), historyParts...)),
			cumulative:  historyTokens,
			kind:        "history",
		})
	}
	if len(historyPoints) > 0 {
		write := historyPoints[len(historyPoints)-1]
		if checkpoint != "" && checkpoint != write.fingerprint {
			for _, point := range historyPoints {
				if point.fingerprint == checkpoint {
					points = append(points, point)
					break
				}
			}
		}
		points = append(points, write)
		anthropicKiroCaches.setHistoryCheckpoint(accountID, sessionKey, model, write.fingerprint, ttl)
	}

	if toolCache && toolTokens >= minTokens {
		points = append(points, anthropicKiroCachePoint{
			fingerprint: anthropicKiroFingerprint("tools", []string{strconv.Itoa(len(payload["tools"].([]any)))}),
			cumulative:  toolTokens,
			kind:        "tools",
		})
	}
	total := historyTokens + currentTokens
	if toolCache {
		total += toolTokens
	}
	return points, total, ttl
}

func consumeAnthropicKiroCache(accountID int64, points []anthropicKiroCachePoint, totalTokens, minTokens int, ttl time.Duration) kiroCacheEmulationUsage {
	usage := kiroCacheEmulationUsage{InputTokens: max(0, totalTokens)}
	if len(points) == 0 {
		return usage
	}
	hits := anthropicKiroCaches.lookup(accountID, points)
	readPrefix := 0
	toolsRead := 0
	for _, point := range points {
		if !hits[point.fingerprint] || point.cumulative < minTokens {
			continue
		}
		if point.kind == "tools" {
			toolsRead = max(toolsRead, point.cumulative)
		} else if point.cumulative > readPrefix {
			readPrefix = point.cumulative
		}
	}
	latest := 0
	toolsTokens := 0
	for _, point := range points {
		if point.kind == "tools" {
			toolsTokens = max(toolsTokens, point.cumulative)
			continue
		}
		latest = max(latest, point.cumulative)
	}
	writable := 0
	if latest >= minTokens {
		writable = latest
	}
	creation := max(0, writable-readPrefix)
	if creation > 0 && creation < minTokens {
		creation = 0
	}
	toolsCreation := 0
	if toolsRead == 0 && toolsTokens >= minTokens {
		toolsCreation = toolsTokens
	}
	usage.CacheReadInputTokens = readPrefix + toolsRead
	usage.CacheCreationInputTokens = creation + toolsCreation
	if usage.CacheReadInputTokens+usage.CacheCreationInputTokens > totalTokens {
		overflow := usage.CacheReadInputTokens + usage.CacheCreationInputTokens - totalTokens
		if usage.CacheCreationInputTokens >= overflow {
			usage.CacheCreationInputTokens -= overflow
		} else {
			overflow -= usage.CacheCreationInputTokens
			usage.CacheCreationInputTokens = 0
			usage.CacheReadInputTokens = max(0, usage.CacheReadInputTokens-overflow)
		}
	}
	usage.InputTokens = max(0, totalTokens-usage.CacheReadInputTokens-usage.CacheCreationInputTokens)
	var store []anthropicKiroCachePoint
	for _, point := range points {
		if point.cumulative < minTokens {
			continue
		}
		if point.kind == "history" {
			delta := point.cumulative - readPrefix
			if delta != 0 && delta < minTokens {
				continue
			}
		}
		store = append(store, point)
	}
	anthropicKiroCaches.store(accountID, store, ttl)
	return usage
}

func splitAnthropicSystemCacheSegments(system any) (stable []string, dynamic string) {
	blocks := normalizeKiroSystemBlocks(system)
	var pending []string
	for _, block := range blocks {
		text := anthropicBlockText(block)
		if text != "" {
			pending = append(pending, text)
		}
		if extractKiroCacheTTL(block) != nil && len(pending) > 0 {
			stable = append(stable, strings.Join(pending, "\n"))
			pending = nil
		}
	}
	return stable, strings.Join(pending, "\n")
}

func anthropicHistoryAndCurrent(raw any) (history []string, currentTokens int) {
	messages, _ := raw.([]any)
	if len(messages) == 0 {
		return nil, 0
	}
	last := len(messages) - 1
	for i, rawMessage := range messages {
		message, _ := rawMessage.(map[string]any)
		tokens := anthropicMessageTokens(message)
		if i == last {
			currentTokens = tokens
			continue
		}
		history = append(history, anthropicMessageText(message))
	}
	return history, currentTokens
}

func anthropicMessageTokens(message map[string]any) int {
	if message == nil {
		return kiroTokensPerMessage
	}
	switch content := message["content"].(type) {
	case string:
		return anthropictokenizer.CountTokens(content) + kiroTokensPerMessage
	case []any:
		total := kiroTokensPerMessage
		for _, block := range content {
			total += countKiroMessageContentTokens(block)
		}
		return total
	default:
		return kiroTokensPerMessage
	}
}

func anthropicMessageText(message map[string]any) string {
	switch content := message["content"].(type) {
	case string:
		return content
	case []any:
		var parts []string
		for _, block := range content {
			if text := anthropicBlockText(block); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func anthropicBlockText(block any) string {
	switch typed := block.(type) {
	case string:
		return typed
	case map[string]any:
		if text, ok := typed["text"].(string); ok {
			return text
		}
	}
	return ""
}

func anthropicMaxCacheTTL(payload map[string]any) time.Duration {
	maxTTL := kiroCacheDefaultTTL
	consider := func(value any) {
		if ttl := extractKiroCacheTTL(value); ttl != nil && *ttl > maxTTL {
			maxTTL = *ttl
		}
	}
	for _, block := range normalizeKiroSystemBlocks(payload["system"]) {
		consider(block)
	}
	if tools, ok := payload["tools"].([]any); ok {
		for _, tool := range tools {
			consider(tool)
		}
	}
	messages, _ := payload["messages"].([]any)
	for _, raw := range messages {
		message, _ := raw.(map[string]any)
		if blocks, ok := message["content"].([]any); ok {
			for _, block := range blocks {
				consider(block)
			}
		}
	}
	return maxTTL
}

func anthropicKiroFingerprint(kind string, parts []string) string {
	raw, _ := json.Marshal(map[string]any{"kind": kind, "parts": parts})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *anthropicKiroCacheStore) lookup(accountID int64, points []anthropicKiroCachePoint) map[string]bool {
	now := time.Now()
	prefix := strconv.FormatInt(accountID, 10) + "|"
	out := map[string]bool{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, point := range points {
		key := prefix + point.fingerprint
		expiry, ok := s.hits[key]
		if ok && expiry.After(now) {
			out[point.fingerprint] = true
		}
	}
	return out
}

func (s *anthropicKiroCacheStore) store(accountID int64, points []anthropicKiroCachePoint, ttl time.Duration) {
	if ttl <= 0 {
		ttl = kiroCacheDefaultTTL
	}
	expiry := time.Now().Add(ttl)
	prefix := strconv.FormatInt(accountID, 10) + "|"
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, point := range points {
		s.hits[prefix+point.fingerprint] = expiry
	}
}

func (s *anthropicKiroCacheStore) historyCheckpoint(accountID int64, sessionKey, model string) string {
	if strings.TrimSpace(sessionKey) == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.history[anthropicKiroSessionKey(accountID, sessionKey, model)]
}

func (s *anthropicKiroCacheStore) setHistoryCheckpoint(accountID int64, sessionKey, model, fingerprint string, _ time.Duration) {
	if strings.TrimSpace(sessionKey) == "" || fingerprint == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[anthropicKiroSessionKey(accountID, sessionKey, model)] = fingerprint
}

func anthropicKiroSessionKey(accountID int64, sessionKey, model string) string {
	return strconv.FormatInt(accountID, 10) + "|" + sessionKey + "|" + strings.ToLower(strings.TrimSpace(model))
}
