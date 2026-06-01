package generator

import "math/rand"

// Random utility functions inspired by EET's dice system (d6, d12, d20, d42, d100)
// These provide controlled randomness for SQL generation decisions

// d6: roll a 6-sided die (1-6)
func (g *QueryGenerator) d6() int {
	return g.Rand.Intn(6) + 1
}

// d12: roll a 12-sided die (1-12)
func (g *QueryGenerator) d12() int {
	return g.Rand.Intn(12) + 1
}

// d20: roll a 20-sided die (1-20)
func (g *QueryGenerator) d20() int {
	return g.Rand.Intn(20) + 1
}

// d42: roll a 42-sided die (1-42)
func (g *QueryGenerator) d42() int {
	return g.Rand.Intn(42) + 1
}

// d100: roll a 100-sided die (1-100)
func (g *QueryGenerator) d100() int {
	return g.Rand.Intn(100) + 1
}

// pickRandom: pick a random element from a slice
func pickRandom[T any](r *rand.Rand, items []T) T {
	return items[r.Intn(len(items))]
}

// pickRandomN: pick N random elements from a slice (may repeat)
func pickRandomN[T any](r *rand.Rand, items []T, n int) []T {
	result := make([]T, n)
	for i := 0; i < n; i++ {
		result[i] = items[r.Intn(len(items))]
	}
	return result
}

// shuffle: shuffle a slice in place
func shuffle[T any](r *rand.Rand, items []T) {
	for i := len(items) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		items[i], items[j] = items[j], items[i]
	}
}

// randBool: return a random boolean
func (g *QueryGenerator) randBool() bool {
	return g.Rand.Intn(2) == 0
}

// randInt: return a random integer in range [min, max]
func (g *QueryGenerator) randInt(min, max int) int {
	if min >= max {
		return min
	}
	return min + g.Rand.Intn(max - min + 1)
}

// randIntN: return a random integer in range [1, n]
func (g *QueryGenerator) randIntN(n int) int {
	if n <= 0 {
		return 1
	}
	return g.Rand.Intn(n) + 1
}