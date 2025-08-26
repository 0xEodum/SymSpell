package internal

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"symspell/pkg/editdistance"
	"symspell/pkg/options"
)

const maxUint32 = ^uint32(0)

// SymSpell represents the Symmetric Delete spelling correction algorithm.
type SymSpell struct {
	MaxDictionaryEditDistance int
	PrefixLength              int
	CountThreshold            int
	SplitThreshold            int
	PreserveCase              bool
	SplitWordBySpace          bool
	SplitWordAndNumber        bool
	MinimumCharToChange       int
	FrequencyThreshold        int // Новое поле: минимальная частота для точных совпадений
	FrequencyMultiplier       int // Новое поле: множитель для сравнения частот
	Words                     map[string]uint32
	BelowThresholdWords       map[string]uint32
	Deletes                   map[string][]uint32
	ExactTransform            map[string]string
	words                     []string
	counts                    []uint32
	maxLength                 int
	distanceComparer          editdistance.IEditDistance
	// lookup compound
	N              float64
	Bigrams        map[string]uint32
	BigramCountMin uint32
}

// NewSymSpell is the constructor for the SymSpell struct.
func NewSymSpell(opt ...options.Options) (*SymSpell, error) {
	opts := options.DefaultOptions
	for _, config := range opt {
		config.Apply(&opts)
	}
	if opts.MaxDictionaryEditDistance < 0 {
		return nil, errors.New("maxDictionaryEditDistance cannot be negative")
	}
	if opts.PrefixLength < 1 {
		return nil, errors.New("prefixLength cannot be less than 1")
	}
	if opts.PrefixLength <= opts.MaxDictionaryEditDistance {
		return nil, errors.New("prefixLength must be greater than maxDictionaryEditDistance")
	}
	if opts.CountThreshold < 0 {
		return nil, errors.New("countThreshold cannot be negative")
	}
	if opts.FrequencyThreshold < 0 {
		return nil, errors.New("frequencyThreshold cannot be negative")
	}
	if opts.FrequencyMultiplier <= 1 {
		return nil, errors.New("frequencyMultiplier must be greater than 1")
	}

	return &SymSpell{
		MaxDictionaryEditDistance: opts.MaxDictionaryEditDistance,
		PrefixLength:              opts.PrefixLength,
		CountThreshold:            opts.CountThreshold,
		SplitThreshold:            opts.SplitItemThreshold,
		PreserveCase:              opts.PreserveCase,
		SplitWordBySpace:          opts.SplitWordBySpace,
		SplitWordAndNumber:        opts.SplitWordAndNumber,
		MinimumCharToChange:       opts.MinimumCharacterToChange,
		FrequencyThreshold:        opts.FrequencyThreshold,
		FrequencyMultiplier:       opts.FrequencyMultiplier,
		Words:                     make(map[string]uint32),
		BelowThresholdWords:       make(map[string]uint32),
		Deletes:                   make(map[string][]uint32),
		ExactTransform:            nil,
		words:                     make([]string, 0),
		counts:                    make([]uint32, 0),
		distanceComparer:          editdistance.NewEditDistance(editdistance.DamerauLevenshtein),
		maxLength:                 0,
		Bigrams:                   nil,
		N:                         1024908267229,
		BigramCountMin:            maxUint32,
	}, nil
}

// createDictionaryEntry creates or updates an entry in the dictionary.
func (s *SymSpell) createDictionaryEntry(key string, count uint32) bool {
	if count == 0 {
		if s.CountThreshold > 0 {
			return false
		}
	}

	if s.CountThreshold > 1 {
		if countPrev, found := s.BelowThresholdWords[key]; found {
			count = incrementCount(count, countPrev)
			if int(count) < s.CountThreshold {
				s.BelowThresholdWords[key] = count
				return false
			}
			delete(s.BelowThresholdWords, key)
		}
	} else if idx, found := s.Words[key]; found {
		s.counts[idx] = incrementCount(count, s.counts[idx])
		return false
	}
	if int(count) < s.CountThreshold {
		s.BelowThresholdWords[key] = count
		return false
	}

	index := uint32(len(s.words))
	s.words = append(s.words, key)
	s.counts = append(s.counts, count)
	s.Words[key] = index

	if len(key) > s.maxLength {
		s.maxLength = len(key)
	}

	edits := s.editsPrefix(key)
	for deleteWord := range edits {
		s.Deletes[deleteWord] = append(s.Deletes[deleteWord], index)
	}

	return true
}

func (s *SymSpell) edits(word string, editDistance int, deleteWords map[string]bool, currentDistance int) {
	editDistance++
	runes := []rune(word)
	if len(runes) == 0 {
		if utf8.RuneCountInString(word) <= s.MaxDictionaryEditDistance {
			deleteWords[""] = true
		}
		return
	}
	for i := currentDistance; i < len(runes); i++ {
		deleteWord := string(runes[:i]) + string(runes[i+1:])
		if !deleteWords[deleteWord] {
			deleteWords[deleteWord] = true
		}
		if editDistance < s.MaxDictionaryEditDistance {
			s.edits(deleteWord, editDistance, deleteWords, i)
		}
	}
}

// editsPrefix function corresponds to _edits_prefix in Python, handling Unicode characters correctly
func (s *SymSpell) editsPrefix(key string) map[string]bool {
	hashSet := make(map[string]bool)
	if utf8.RuneCountInString(key) <= s.MaxDictionaryEditDistance {
		hashSet[""] = true
	}
	runes := []rune(key)
	if len(runes) > s.PrefixLength {
		key = string(runes[:s.PrefixLength])
	}
	hashSet[key] = true
	s.edits(key, 0, hashSet, 0)
	return hashSet
}

// LoadDictionary loads dictionary entries from a file.
func (s *SymSpell) LoadDictionary(corpusPath string, termIndex int, countIndex int, separator string) (bool, error) {
	if corpusPath == "" {
		return false, errors.New("corpus path cannot be empty")
	}

	// Check if the file exists
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		log.Printf("Dictionary file not found at %s.\n", corpusPath)
		return false, nil
	}

	// Open the file
	file, err := os.Open(corpusPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Load dictionary data from file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, separator)
		if len(fields) <= max(termIndex, countIndex) {
			continue // Skip invalid lines
		}

		term := fields[termIndex]
		c, err := strconv.ParseUint(fields[countIndex], 10, 32)
		if err != nil {
			continue // Skip invalid counts
		}
		s.createDictionaryEntry(term, uint32(c))
	}

	if err = scanner.Err(); err != nil {
		return false, err
	}

	s.BelowThresholdWords = nil

	return true, nil
}

func incrementCount(count, countPrevious uint32) uint32 {
	if maxUint32-countPrevious > count {
		return countPrevious + count
	}
	return maxUint32
}

func (s *SymSpell) LoadExactDictionary(
	corpusPath string,
	separator string,
) (bool, error) {
	if corpusPath == "" {
		return false, fmt.Errorf("corpus path cannot be empty")
	}
	// Check if the file exists
	file, err := os.Open(corpusPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Use the stream-based loading function
	return s.LoadExactDictionaryStream(file, separator), nil
}

func (s *SymSpell) LoadExactDictionaryStream(corpusStream *os.File, separator string) bool {
	if s.ExactTransform == nil {
		s.ExactTransform = make(map[string]string)
	}
	scanner := bufio.NewScanner(corpusStream)
	// Define minimum parts depending on the separator
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Split line by the separator
		var parts []string
		if separator == "" {
			parts = strings.Fields(line)
		} else {
			parts = strings.Split(line, separator)
		}
		if len(parts) < 2 {
			continue
		}
		// Parse count
		exactMatch := parts[1]
		// Create the key
		key := parts[0]
		// Add to Exact Transform dictionary
		s.ExactTransform[key] = exactMatch
	}
	return true
}

// ClearTransformData releases memory used by optional bigram and transform maps.
func (s *SymSpell) ClearTransformData() {
	s.Bigrams = nil
	s.ExactTransform = nil
}
