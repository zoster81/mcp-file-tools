package handler

import "github.com/zoster81/mcp-file-tools/internal/config"

func (h *Handler) maxFileBytes() int64 {
	if h != nil && h.config != nil && h.config.Limits.MaxFileBytes > 0 {
		return h.config.Limits.MaxFileBytes
	}
	return config.DefaultMaxFileBytes
}

func (h *Handler) maxDecodedCharacters() int {
	if h != nil && h.config != nil && h.config.Limits.MaxDecodedCharacters > 0 {
		return h.config.Limits.MaxDecodedCharacters
	}
	return config.DefaultMaxDecodedCharacters
}

func (h *Handler) maxLineBytes() int {
	if h != nil && h.config != nil && h.config.Limits.MaxLineBytes > 0 {
		return h.config.Limits.MaxLineBytes
	}
	return config.DefaultMaxLineBytes
}

func (h *Handler) maxBatchFiles() int {
	if h != nil && h.config != nil && h.config.Limits.MaxBatchFiles > 0 {
		return h.config.Limits.MaxBatchFiles
	}
	return config.DefaultMaxBatchFiles
}

func (h *Handler) maxMatches() int {
	if h != nil && h.config != nil && h.config.Limits.MaxMatches > 0 {
		return h.config.Limits.MaxMatches
	}
	return config.DefaultMaxMatches
}

func (h *Handler) maxOutputBytes() int64 {
	if h != nil && h.config != nil && h.config.Limits.MaxOutputBytes > 0 {
		return h.config.Limits.MaxOutputBytes
	}
	return config.DefaultMaxOutputBytes
}

func clampBudgetToInt(budget int64) int {
	maxInt := int64(^uint(0) >> 1)
	if budget > maxInt {
		return int(maxInt)
	}
	if budget < 0 {
		return 0
	}
	return int(budget)
}
