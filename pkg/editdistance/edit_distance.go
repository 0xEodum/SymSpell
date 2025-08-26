package editdistance

type IEditDistance interface {
	Distance(a, b string) int
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

func (d EditDistance) Distance(a, b string) int {
	switch d.Type {
	case DamerauLevenshtein:
		return damerauLevenshteinDistance(a, b)
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

	prev2 := make([]int, n+1)
	prev := make([]int, n+1)
	curr := make([]int, n+1)

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
