package handler

import (
	"fmt"
	"math"
	"strings"
)

const (
	minFuzzySimilarity       = 0.50
	maxFuzzyCandidateWindows = 10000
	maxFuzzyRunes            = 4096
	maxFuzzyWorkCells        = 2_000_000
)

func tryFuzzyMatch(content, oldText, newText string, threshold float64) (bool, string, error) {
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < minFuzzySimilarity || threshold > 1 {
		return false, "", fmt.Errorf("similarity must be between %.2f and 1.0", minFuzzySimilarity)
	}
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	if len(oldLines) == 0 || len(contentLines) < len(oldLines) {
		return false, "", nil
	}
	windowCount := len(contentLines) - len(oldLines) + 1
	if windowCount > maxFuzzyCandidateWindows {
		return false, "", fmt.Errorf("fuzzy edit has %d candidate windows; limit is %d", windowCount, maxFuzzyCandidateWindows)
	}

	normalizedOld := normalizeFuzzyText(oldLines)
	oldRunes := []rune(normalizedOld)
	if len(oldRunes) > maxFuzzyRunes {
		return false, "", fmt.Errorf("fuzzy oldText has %d runes; limit is %d", len(oldRunes), maxFuzzyRunes)
	}

	bestIndex := -1
	bestScore := -1.0
	ambiguous := false
	workUsed := 0
	for index := 0; index < windowCount; index++ {
		candidate := normalizeFuzzyText(contentLines[index : index+len(oldLines)])
		candidateRunes := []rune(candidate)
		if len(candidateRunes) > maxFuzzyRunes {
			continue
		}
		maximumLength := max(len(oldRunes), len(candidateRunes))
		if maximumLength == 0 {
			continue
		}
		maximumDistance := int(math.Floor((1 - threshold) * float64(maximumLength)))
		estimatedWork := len(oldRunes) * (2*maximumDistance + 1)
		if estimatedWork < maximumLength {
			estimatedWork = maximumLength
		}
		workUsed += estimatedWork
		if workUsed > maxFuzzyWorkCells {
			return false, "", fmt.Errorf("fuzzy edit exceeds the deterministic comparison budget")
		}
		distance := boundedLevenshtein(oldRunes, candidateRunes, maximumDistance)
		if distance > maximumDistance {
			continue
		}
		score := 1 - float64(distance)/float64(maximumLength)
		if score > bestScore+1e-12 {
			bestIndex, bestScore, ambiguous = index, score, false
		} else if math.Abs(score-bestScore) <= 1e-12 {
			ambiguous = true
		}
	}
	if bestIndex < 0 {
		return false, "", nil
	}
	if ambiguous {
		return false, "", fmt.Errorf("fuzzy edit is ambiguous: multiple blocks match at similarity %.3f", bestScore)
	}
	return true, replaceLineWindow(contentLines, oldLines, newText, bestIndex), nil
}

func normalizeFuzzyText(lines []string) string {
	normalized := make([]string, len(lines))
	for index, line := range lines {
		normalized[index] = strings.Join(strings.Fields(line), " ")
	}
	return strings.Join(normalized, "\n")
}

func replaceLineWindow(contentLines, oldLines []string, newText string, start int) string {
	originalIndent := getLeadingWhitespace(contentLines[start])
	newLines := strings.Split(newText, "\n")
	for index := range newLines {
		if index == 0 {
			if newLines[index] != "" {
				newLines[index] = originalIndent + strings.TrimLeft(newLines[index], " \t")
			}
		} else {
			newLines[index] = adjustRelativeIndent(oldLines, newLines[index], index, originalIndent)
		}
	}
	result := make([]string, 0, len(contentLines)-len(oldLines)+len(newLines))
	result = append(result, contentLines[:start]...)
	result = append(result, newLines...)
	result = append(result, contentLines[start+len(oldLines):]...)
	return strings.Join(result, "\n")
}

func boundedLevenshtein(first, second []rune, maximum int) int {
	if absInt(len(first)-len(second)) > maximum {
		return maximum + 1
	}
	infinity := maximum + 1
	previous := make([]int, len(second)+1)
	current := make([]int, len(second)+1)
	for index := range previous {
		if index <= maximum {
			previous[index] = index
		} else {
			previous[index] = infinity
		}
	}
	for i := 1; i <= len(first); i++ {
		for index := range current {
			current[index] = infinity
		}
		if i <= maximum {
			current[0] = i
		}
		start := max(1, i-maximum)
		end := min(len(second), i+maximum)
		rowMinimum := infinity
		for j := start; j <= end; j++ {
			cost := 0
			if first[i-1] != second[j-1] {
				cost = 1
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
			rowMinimum = min(rowMinimum, current[j])
		}
		if rowMinimum > maximum {
			return infinity
		}
		previous, current = current, previous
	}
	return previous[len(second)]
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
