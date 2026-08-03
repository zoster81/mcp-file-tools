package handler

import (
	"container/heap"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sortablePath struct {
	path    string
	name    string
	size    int64
	modTime time.Time
}

func normalizeSortBy(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case "name":
		return "name", nil
	case "mtime", "size":
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("sortBy must be name, mtime, or size")
	}
}

func statSortablePath(path string) (sortablePath, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sortablePath{}, err
	}
	return sortablePath{
		path:    path,
		name:    filepath.Base(path),
		size:    info.Size(),
		modTime: info.ModTime(),
	}, nil
}

func compareSortable(first, second sortablePath, sortBy string) int {
	var comparison int
	switch sortBy {
	case "mtime":
		switch {
		case first.modTime.Before(second.modTime):
			comparison = -1
		case first.modTime.After(second.modTime):
			comparison = 1
		}
	case "size":
		switch {
		case first.size < second.size:
			comparison = -1
		case first.size > second.size:
			comparison = 1
		}
	default:
		comparison = strings.Compare(first.name, second.name)
	}
	if comparison == 0 {
		comparison = strings.Compare(filepath.ToSlash(first.path), filepath.ToSlash(second.path))
	}
	return comparison
}

func sortSortablePaths(items []sortablePath, sortBy string, reverse bool) {
	sort.SliceStable(items, func(i, j int) bool {
		comparison := compareSortable(items[i], items[j], sortBy)
		if reverse {
			return comparison > 0
		}
		return comparison < 0
	})
}

// boundedPathHeap retains the globally best K paths without retaining every
// matching path. The root is the worst retained item, so a better candidate can
// replace it in O(log K) time.
type boundedPathHeap struct {
	items   []sortablePath
	sortBy  string
	reverse bool
}

func (h boundedPathHeap) Len() int { return len(h.items) }
func (h boundedPathHeap) Less(i, j int) bool {
	comparison := compareSortable(h.items[i], h.items[j], h.sortBy)
	if h.reverse {
		return comparison < 0
	}
	return comparison > 0
}
func (h boundedPathHeap) Swap(i, j int)   { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *boundedPathHeap) Push(value any) { h.items = append(h.items, value.(sortablePath)) }
func (h *boundedPathHeap) Pop() any {
	last := len(h.items) - 1
	value := h.items[last]
	h.items = h.items[:last]
	return value
}

func (h *boundedPathHeap) add(item sortablePath, limit int) {
	if limit <= 0 {
		return
	}
	if len(h.items) < limit {
		heap.Push(h, item)
		return
	}
	comparison := compareSortable(item, h.items[0], h.sortBy)
	better := comparison < 0
	if h.reverse {
		better = comparison > 0
	}
	if better {
		h.items[0] = item
		heap.Fix(h, 0)
	}
}

func (h *boundedPathHeap) sorted() []sortablePath {
	items := append([]sortablePath(nil), h.items...)
	sortSortablePaths(items, h.sortBy, h.reverse)
	return items
}
