package internal

import (
	"errors"
	"sort"
	"strings"

	"symspell/pkg/items"
	verbositypkg "symspell/pkg/verbosity"
)

func (s *SymSpell) Lookup(
	phrase string,
	verbosity verbositypkg.Verbosity,
	maxEditDistance int,
) ([]items.SuggestItem, error) {
	if maxEditDistance > s.MaxDictionaryEditDistance {
		return nil, errors.New("distance too large")
	}
	cp := newCandidateProcessor(maxEditDistance, verbosity, phrase)
	// Early exit - word too big to match any words
	if cp.phraseLen-maxEditDistance > s.maxLength {
		return cp.suggestions, nil
	}

	exactMatch := s.checkExactMatch(phrase, verbosity, &cp)

	if exactMatch.shouldStop {
		return cp.suggestions, nil
	}

	if maxEditDistance == 0 {
		return cp.suggestions, nil
	}
	cp.consideredSuggestions[phrase] = true
	// Add original prefix
	phrasePrefixRunes := s.getOriginPrefix(&cp)
	cp.candidates = append(cp.candidates, string(phrasePrefixRunes))
	// Process candidates
	s.processCandidate(maxEditDistance, &cp)

	// Финальная обработка с учетом относительной частотности
	s.finalizeWithFrequencyCheck(&cp, exactMatch.exactItem)

	cp.sortCandidate()

	return cp.suggestions, nil
}

type ExactMatchResult struct {
	shouldStop bool
	exactItem  *items.SuggestItem
}

func (s *SymSpell) getOriginPrefix(cp *candidateProcessor) []rune {
	phrase := cp.phrase
	if cp.phraseLen > s.PrefixLength {
		phrase = cp.phrase[:s.PrefixLength]
	}
	return []rune(phrase)
}

func (s *SymSpell) checkExactMatch(phrase string, verbosity verbositypkg.Verbosity, cp *candidateProcessor) ExactMatchResult {
	if idx, found := s.Words[phrase]; found {
		count := s.counts[idx]
		exactItem := items.SuggestItem{Term: phrase, Distance: 0, Count: int(count)}
		cp.suggestions = append(cp.suggestions, exactItem)

		if verbosity != verbositypkg.All && int(count) >= s.FrequencyThreshold {
			return ExactMatchResult{shouldStop: true, exactItem: &exactItem}
		}

		return ExactMatchResult{shouldStop: false, exactItem: &exactItem}
	}
	return ExactMatchResult{shouldStop: false, exactItem: nil}
}

func (s *SymSpell) finalizeWithFrequencyCheck(cp *candidateProcessor, exactMatch *items.SuggestItem) {
	if exactMatch == nil || len(cp.suggestions) <= 1 {
		return
	}

	// Ищем лучший вариант с учетом расстояния и частоты
	var bestAlternative *items.SuggestItem

	for i := range cp.suggestions {
		suggestion := &cp.suggestions[i]

		// Пропускаем точное совпадение
		if suggestion.Distance == 0 {
			continue
		}

		// Учитываем только близкие варианты (расстояние 1-2)
		if suggestion.Distance <= 2 {
			// Используем настройки частотности
			requiredFrequency := exactMatch.Count * s.FrequencyMultiplier

			if suggestion.Count >= requiredFrequency {
				if bestAlternative == nil ||
					suggestion.Count > bestAlternative.Count ||
					(suggestion.Count == bestAlternative.Count && suggestion.Distance < bestAlternative.Distance) {
					bestAlternative = suggestion
				}
			}
		}
	}

	// Если нашли лучшую альтернативу, удаляем точное совпадение
	if bestAlternative != nil {
		// Опциональный лог для отладки
		// fmt.Printf("Заменяем '%s' (%d) на '%s' (%d) из-за низкой частотности\n",
		//     exactMatch.Term, exactMatch.Count, bestAlternative.Term, bestAlternative.Count)

		// Удаляем точное совпадение из результатов
		newSuggestions := make([]items.SuggestItem, 0, len(cp.suggestions))
		for _, suggestion := range cp.suggestions {
			if suggestion.Distance != 0 { // Оставляем только не точные совпадения
				newSuggestions = append(newSuggestions, suggestion)
			}
		}
		cp.suggestions = newSuggestions
	}
}

func (s *SymSpell) processCandidate(maxEditDistance int, cp *candidateProcessor) {
	for cp.candidatePointer < len(cp.candidates) {
		candidate := s.preProcessCandidate(cp)

		if cp.lenDiff > cp.maxEditDistance2 {
			if cp.verbosity == verbositypkg.All {
				continue
			}
			break
		}

		// Check suggestions for the candidate
		if dictSuggestions, found := s.Deletes[candidate]; found {
			for _, idx := range dictSuggestions {
				suggestion := s.words[idx]
				if suggestion == cp.phrase {
					continue
				}
				cp.updateSuggestion(suggestion)
				skip := s.checkSuggestionToSkip(cp, suggestion, candidate)
				if skip {
					continue
				}
				cp.resetDistance()
				if cp.candidateLen == 0 {
					cp.distance = max(cp.phraseLen, cp.suggestionLen)
					if cp.distance > cp.maxEditDistance2 || cp.consideredSuggestions[suggestion] {
						continue
					}
				} else if cp.suggestionLen == 1 {
					skip = s.checkFirstRuneDistance(cp, suggestion)
					if skip {
						continue
					}
				} else {
					s.updateMinDistance(maxEditDistance, cp)
					skip = s.checkDistanceToSkip(maxEditDistance, cp, suggestion)
					if skip {
						continue
					}
				}
				if cp.distance <= cp.maxEditDistance2 {
					s.updateSuggestions(suggestion, cp)
				}
			}
		}
		if cp.lenDiff <= maxEditDistance && cp.candidateLen <= s.PrefixLength {
			if cp.verbosity != verbositypkg.All && cp.lenDiff >= cp.maxEditDistance2 {
				continue
			}
			s.addEditDistance(candidate, cp)
		}
	}
}

func (s *SymSpell) preProcessCandidate(cp *candidateProcessor) string {
	candidate := cp.candidates[cp.candidatePointer]
	cp.candidatePointer++
	cp.candidateLen = len(candidate)
	cp.lenDiff = cp.phraseLen - cp.candidateLen
	return candidate
}

func (s *SymSpell) checkSuggestionToSkip(cp *candidateProcessor, suggestion string, candidate string) bool {
	if abs(cp.suggestionLen-cp.phraseLen) > cp.maxEditDistance2 || cp.suggestionLen < cp.candidateLen ||
		(cp.suggestionLen == cp.candidateLen && suggestion != candidate) {
		return true
	}
	suggestionPrefixLen := min(cp.suggestionLen, s.PrefixLength)
	if suggestionPrefixLen > cp.phraseLen && suggestionPrefixLen-cp.candidateLen > cp.maxEditDistance2 {
		return true
	}
	return false
}

func (s *SymSpell) checkDistanceToSkip(maxEditDistance int, cp *candidateProcessor, suggestion string) bool {
	if s.PrefixLength-maxEditDistance == cp.candidateLen {
		skip := s.checkProcessShouldSkip(cp, suggestion)
		if skip {
			return true
		}
	}
	// delete in suggestion prefix is somewhat expensive, and
	// only pays off when verbosity is TOP or CLOSEST
	if cp.consideredSuggestions[suggestion] {
		return true
	}
	cp.consideredSuggestions[suggestion] = true
	cp.distance = s.distanceCompare(cp.phrase, suggestion, cp.maxEditDistance2)
	return cp.distance < 0
}

func (s *SymSpell) updateMinDistance(maxEditDistance int, cp *candidateProcessor) {
	if s.PrefixLength-maxEditDistance == cp.candidateLen {
		cp.minDistance = min(cp.phraseLen, cp.suggestionLen) - s.PrefixLength
	} else {
		cp.minDistance = 0
	}
}

func (s *SymSpell) checkFirstRuneDistance(cp *candidateProcessor, suggestion string) bool {
	first := suggestion[0]
	if strings.IndexByte(cp.phrase, first) >= 0 {
		cp.distance = cp.phraseLen - 1
	} else {
		cp.distance = cp.phraseLen
	}
	if cp.distance > cp.maxEditDistance2 || cp.consideredSuggestions[suggestion] {
		return true
	}
	return false
}

func (s *SymSpell) checkProcessShouldSkip(cp *candidateProcessor, suggestion string) bool {
	phrase := cp.phrase
	if cp.minDistance > 1 && phrase[cp.phraseLen+1-cp.minDistance:] != suggestion[cp.suggestionLen+1-cp.minDistance:] {
		return true
	}
	if cp.minDistance > 0 && phrase[cp.phraseLen-cp.minDistance] != suggestion[cp.suggestionLen-cp.minDistance] {
		if phrase[cp.phraseLen-cp.minDistance-1] != suggestion[cp.suggestionLen-cp.minDistance] ||
			phrase[cp.phraseLen-cp.minDistance] != suggestion[cp.suggestionLen-cp.minDistance-1] {
			return true
		}
	}
	return false
}

func (s *SymSpell) updateSuggestions(suggestion string, cp *candidateProcessor) {
	idx := s.Words[suggestion]
	suggestionCount := s.counts[idx]
	item := items.SuggestItem{Term: suggestion, Distance: cp.distance, Count: int(suggestionCount)}

	if len(cp.suggestions) > 0 {
		if shouldContinue := s.updateBestSuggestion(cp, int(suggestionCount), item); shouldContinue {
			return
		}
	}
	if cp.verbosity != verbositypkg.All {
		cp.maxEditDistance2 = cp.distance
	}
	cp.suggestions = append(cp.suggestions, items.SuggestItem{Term: suggestion, Distance: cp.distance, Count: int(suggestionCount)})
}

func (s *SymSpell) updateBestSuggestion(cp *candidateProcessor, suggestionCount int, item items.SuggestItem) bool {
	if cp.verbosity == verbositypkg.Closest {
		// Keep only the closest suggestions
		if cp.distance < cp.maxEditDistance2 {
			cp.suggestions = []items.SuggestItem{}
		}
	} else if cp.verbosity == verbositypkg.Top {
		// Keep the top suggestion based on count or distance
		if cp.distance < cp.maxEditDistance2 || suggestionCount > cp.suggestions[0].Count {
			cp.maxEditDistance2 = cp.distance
			cp.suggestions[0] = item
		}
		return true
	}
	return false
}

func (s *SymSpell) addEditDistance(candidate string, cp *candidateProcessor) {
	runes := []rune(candidate)
	for i := 0; i < len(runes); i++ {
		deleteItem := string(runes[:i]) + string(runes[i+1:])
		if !cp.consideredDeletes[deleteItem] {
			cp.consideredDeletes[deleteItem] = true
			cp.candidates = append(cp.candidates, deleteItem)
		}
	}
}

func (s *SymSpell) distanceCompare(a, b string, maxDistance int) int {
	distance := s.distanceComparer.Distance(a, b)

	// Check if the distance exceeds the maxDistance
	if distance > maxDistance {
		return -1
	}
	return distance
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

type candidateProcessor struct {
	candidates            []string
	consideredDeletes     map[string]bool
	consideredSuggestions map[string]bool
	maxEditDistance2      int
	candidatePointer      int
	verbosity             verbositypkg.Verbosity
	phraseLen             int
	phrase                string
	candidateLen          int
	distance              int
	minDistance           int
	suggestions           []items.SuggestItem
	suggestionLen         int
	lenDiff               int
}

func newCandidateProcessor(maxEditDistance int, verbosity verbositypkg.Verbosity, phrase string) candidateProcessor {
	return candidateProcessor{
		candidates:            make([]string, 0),
		consideredDeletes:     make(map[string]bool),
		consideredSuggestions: make(map[string]bool),
		maxEditDistance2:      maxEditDistance,
		candidatePointer:      0,
		verbosity:             verbosity,
		phraseLen:             len(phrase),
		phrase:                phrase,
		candidateLen:          0,
		distance:              0,
		minDistance:           0,
		suggestions:           []items.SuggestItem{},
		lenDiff:               0,
	}
}

func (c *candidateProcessor) resetDistance() {
	c.distance, c.minDistance = 0, 0
}

func (c *candidateProcessor) updateSuggestion(suggestion string) {
	c.suggestionLen = len(suggestion)
}

func (c *candidateProcessor) sortCandidate() {
	if len(c.suggestions) > 1 {
		sort.Slice(c.suggestions, func(i, j int) bool {
			if c.suggestions[i].Distance == c.suggestions[j].Distance {
				return c.suggestions[i].Count > c.suggestions[j].Count
			}
			return c.suggestions[i].Distance < c.suggestions[j].Distance
		})
	}
}
