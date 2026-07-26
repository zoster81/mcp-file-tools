package handler

import "github.com/zoster81/mcp-file-tools/internal/config"

func (h *Handler) memoryBudget() int64 {
	if h != nil && h.config != nil && h.config.MemoryThreshold > 0 {
		return h.config.MemoryThreshold
	}
	return config.DefaultMaxSize
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
