package editdistance

import (
	"sync"
	"unicode/utf8"
)

type IEditDistance interface {
	Distance(a, b string) int
	DistanceMax(a, b string, maxDistance int) int
}

func NewEditDistance(Type string) *EditDistance {
	return &EditDistance{Type: Type}
}

const (
	DamerauLevenshtein = "DamerauLevenshtein"
)

type EditDistance struct {
	Type string
}

var intSlicePool = sync.Pool{New: func() any { return make([]int, 0) }}

func getIntSlice(size int) []int {
	v := intSlicePool.Get()
	if v == nil {
		return make([]int, size)
	}
	s := v.([]int)
	if cap(s) < size {
		return make([]int, size)
	}
	return s[:size]
}

func putIntSlice(s []int) {
	intSlicePool.Put(s)
}

func (d EditDistance) Distance(a, b string) int {
	switch d.Type {
	case DamerauLevenshtein:
		if isASCII(a) && isASCII(b) {
			return damerauLevenshteinDistance(a, b)
		}
		return damerauLevenshteinDistanceRunes([]rune(a), []rune(b))
	}
	return 0
}

func (d EditDistance) DistanceMax(a, b string, maxDistance int) int {
	switch d.Type {
	case DamerauLevenshtein:
		if isASCII(a) && isASCII(b) {
			return damerauLevenshteinDistanceMax(a, b, maxDistance)
		}
		return damerauLevenshteinDistanceMaxRunes([]rune(a), []rune(b), maxDistance)
	}
	return 0
}

func damerauLevenshteinDistance(a, b string) int {
	m := len(a)
	n := len(b)

	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	prev2 := getIntSlice(n + 1)
	prev := getIntSlice(n + 1)
	curr := getIntSlice(n + 1)
	defer putIntSlice(prev2)
	defer putIntSlice(prev)
	defer putIntSlice(curr)

	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			dist := min(del, min(ins, sub))
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				trans := prev2[j-2] + cost
				if trans < dist {
					dist = trans
				}
			}
			curr[j] = dist
		}
		prev2, prev, curr = prev, curr, prev2
	}

	return prev[n]
}

func damerauLevenshteinDistanceMax(a, b string, k int) int {
	m := len(a)
	n := len(b)

	if m == 0 {
		if n <= k {
			return n
		}
		return k + 1
	}
	if n == 0 {
		if m <= k {
			return m
		}
		return k + 1
	}

	if d := m - n; d > k || d < -k {
		return k + 1
	}

	prev2 := getIntSlice(n + 1)
	prev := getIntSlice(n + 1)
	curr := getIntSlice(n + 1)
	defer putIntSlice(prev2)
	defer putIntSlice(prev)
	defer putIntSlice(curr)

	limit := k + 1
	for j := 0; j <= n; j++ {
		if j <= k {
			prev[j] = j
		} else {
			prev[j] = limit
		}
	}

	for i := 1; i <= m; i++ {
		curr[0] = i

		jStart := 1
		if i > k {
			jStart = i - k
		}
		jEnd := n
		if jEnd > i+k {
			jEnd = i + k
		}

		if jStart > 1 {
			curr[jStart-1] = limit
		}

		rowMin := limit
		for j := jStart; j <= jEnd; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			dist := min(del, min(ins, sub))
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				trans := prev2[j-2] + cost
				if trans < dist {
					dist = trans
				}
			}
			curr[j] = dist
			if dist < rowMin {
				rowMin = dist
			}
		}
		if rowMin > k {
			return k + 1
		}
		if jEnd < n {
			curr[jEnd+1] = limit
		}
		prev2, prev, curr = prev, curr, prev2
	}

	if prev[n] > k {
		return k + 1
	}
	return prev[n]
}

func damerauLevenshteinDistanceRunes(a, b []rune) int {
	m := len(a)
	n := len(b)

	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	prev2 := getIntSlice(n + 1)
	prev := getIntSlice(n + 1)
	curr := getIntSlice(n + 1)
	defer putIntSlice(prev2)
	defer putIntSlice(prev)
	defer putIntSlice(curr)

	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			dist := min(del, min(ins, sub))
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				trans := prev2[j-2] + cost
				if trans < dist {
					dist = trans
				}
			}
			curr[j] = dist
		}
		prev2, prev, curr = prev, curr, prev2
	}

	return prev[n]
}

func damerauLevenshteinDistanceMaxRunes(a, b []rune, k int) int {
	m := len(a)
	n := len(b)

	if m == 0 {
		if n <= k {
			return n
		}
		return k + 1
	}
	if n == 0 {
		if m <= k {
			return m
		}
		return k + 1
	}

	if d := m - n; d > k || d < -k {
		return k + 1
	}

	prev2 := getIntSlice(n + 1)
	prev := getIntSlice(n + 1)
	curr := getIntSlice(n + 1)
	defer putIntSlice(prev2)
	defer putIntSlice(prev)
	defer putIntSlice(curr)

	limit := k + 1
	for j := 0; j <= n; j++ {
		if j <= k {
			prev[j] = j
		} else {
			prev[j] = limit
		}
	}

	for i := 1; i <= m; i++ {
		curr[0] = i

		jStart := 1
		if i > k {
			jStart = i - k
		}
		jEnd := n
		if jEnd > i+k {
			jEnd = i + k
		}

		if jStart > 1 {
			curr[jStart-1] = limit
		}

		rowMin := limit
		for j := jStart; j <= jEnd; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			dist := min(del, min(ins, sub))
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				trans := prev2[j-2] + cost
				if trans < dist {
					dist = trans
				}
			}
			curr[j] = dist
			if dist < rowMin {
				rowMin = dist
			}
		}
		if rowMin > k {
			return k + 1
		}
		if jEnd < n {
			curr[jEnd+1] = limit
		}
		prev2, prev, curr = prev, curr, prev2
	}

	if prev[n] > k {
		return k + 1
	}
	return prev[n]
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
