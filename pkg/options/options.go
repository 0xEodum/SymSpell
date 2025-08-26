package options

var DefaultOptions = SymspellOptions{
	MaxDictionaryEditDistance: 2,
	PrefixLength:              7,
	CountThreshold:            1,
	SplitItemThreshold:        1,
	PreserveCase:              false,
	SplitWordBySpace:          false,
	SplitWordAndNumber:        false,
	MinimumCharacterToChange:  1,
	FrequencyThreshold:        1000, // Новая опция: минимальная частота для точных совпадений
	FrequencyMultiplier:       10,   // Во сколько раз должна быть больше частота альтернативы
}

type SymspellOptions struct {
	MaxDictionaryEditDistance int
	PrefixLength              int
	CountThreshold            int
	SplitItemThreshold        int
	PreserveCase              bool
	SplitWordBySpace          bool
	SplitWordAndNumber        bool
	MinimumCharacterToChange  int
	FrequencyThreshold        int // Минимальная частота для принятия точного совпадения
	FrequencyMultiplier       int // Во сколько раз альтернатива должна быть частотнее
}

type Options interface {
	Apply(options *SymspellOptions)
}

type FuncConfig struct {
	ops func(options *SymspellOptions)
}

func (w FuncConfig) Apply(conf *SymspellOptions) {
	w.ops(conf)
}

func NewFuncOption(f func(options *SymspellOptions)) *FuncConfig {
	return &FuncConfig{ops: f}
}

func WithMaxDictionaryEditDistance(maxDictionaryEditDistance int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.MaxDictionaryEditDistance = maxDictionaryEditDistance
	})
}

func WithPrefixLength(prefixLength int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.PrefixLength = prefixLength
	})
}

func WithCountThreshold(countThreshold int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.CountThreshold = countThreshold
	})
}

func WithSplitItemThreshold(splitThreshold int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.SplitItemThreshold = splitThreshold
	})
}

func WithPreserveCase() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.PreserveCase = true
	})
}

func WithSplitWordBySpace() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.SplitWordBySpace = true
	})
}

func WithMinimumCharacterToChange(charLength int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.MinimumCharacterToChange = charLength
	})
}

func WithSplitWordAndNumbers() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.SplitWordAndNumber = true
	})
}

// Новые опции для настройки частотности

// WithFrequencyThreshold устанавливает минимальную частоту для принятия точного совпадения
// Если точное совпадение имеет частоту ниже этого порога, будут искаться альтернативы
func WithFrequencyThreshold(threshold int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.FrequencyThreshold = threshold
	})
}

// WithFrequencyMultiplier устанавливает, во сколько раз должна быть больше частота
// альтернативы, чтобы она была предпочтена точному совпадению
func WithFrequencyMultiplier(multiplier int) Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.FrequencyMultiplier = multiplier
	})
}

// WithSmartFrequencyCorrection включает умную коррекцию на основе частотности
// Эквивалентно WithFrequencyThreshold(1000) + WithFrequencyMultiplier(10)
func WithSmartFrequencyCorrection() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.FrequencyThreshold = 1000
		options.FrequencyMultiplier = 10
	})
}

// WithStrictFrequencyCorrection включает строгую коррекцию на основе частотности
// Эквивалентно WithFrequencyThreshold(5000) + WithFrequencyMultiplier(5)
func WithStrictFrequencyCorrection() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.FrequencyThreshold = 5000
		options.FrequencyMultiplier = 5
	})
}

// WithLenientFrequencyCorrection включает мягкую коррекцию на основе частотности
// Эквивалентно WithFrequencyThreshold(100) + WithFrequencyMultiplier(20)
func WithLenientFrequencyCorrection() Options {
	return NewFuncOption(func(options *SymspellOptions) {
		options.FrequencyThreshold = 100
		options.FrequencyMultiplier = 20
	})
}
