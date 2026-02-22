package spell

import (
	"bytes"
	_ "embed"
	"hash/fnv"
	"strconv"
	"unicode/utf8"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
)

const (
	maxEditDistance = 2       // maximum pre-computed edit distance
	prefixLength    = 7       // prefix length for delete generation (memory optimization)
	maxWordBytes    = 256     // maximum word length in bytes
	maxInputBytes   = 1 << 20 // 1 MiB limit for Correct
	minWordRunes    = 2       // minimum runes for a word to be spell-checked
	deletesPerWord  = 4       // estimated delete variants per word for initial map capacity
	maxHyphenParts  = 8       // maximum hyphen-separated parts to check independently
)

//go:embed freq.txt
var freqRaw []byte

// Core SymSpell index (populated in init, read-only after).
var (
	words      map[string]int64    // word -> frequency
	deletes    map[uint32][]uint32 // hash(delete) -> []index into wordList
	wordList   []string            // indexed word list (saves memory vs storing strings in deletes)
	maxWordLen int                 // longest word in dictionary (in runes)
)

func init() {
	lines := bytes.Split(freqRaw, []byte("\n"))
	words = make(map[string]int64, len(lines))
	wordList = make([]string, 0, len(lines))
	deletes = make(map[uint32][]uint32, len(lines)*deletesPerWord)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		sp := bytes.LastIndexByte(line, ' ')
		if sp <= 0 {
			continue
		}

		word := string(line[:sp])
		freq, err := strconv.ParseInt(string(line[sp+1:]), 10, 64)
		if err != nil || freq < 0 {
			continue
		}

		words[word] = freq
		idx := uint32(len(wordList)) //nolint:gosec // dictionary size is bounded well below uint32 max
		wordList = append(wordList, word)

		n := utf8.RuneCountInString(word)
		if n > maxWordLen {
			maxWordLen = n
		}

		// Generate delete variants for the prefix and add to the index.
		prefix := truncateToRunes(word, prefixLength)
		edits := generateDeletes(prefix, maxEditDistance)
		for _, del := range edits {
			h := fnvHash(del)
			deletes[h] = append(deletes[h], idx)
		}
	}
}

// truncateToRunes returns s truncated to at most n runes.
func truncateToRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// fnvHash returns the FNV-1a 32-bit hash of s.
func fnvHash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s)) // cannot fail per hash.Hash contract
	return h.Sum32()
}

// generateDeletes returns all unique strings obtainable by deleting 1 to dist
// characters from s. The original string itself is not included.
func generateDeletes(s string, dist int) []string {
	if dist == 0 {
		return nil
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return nil
	}

	type item struct {
		word  string
		depth int
	}

	seen := make(map[string]struct{})
	var results []string

	queue := []item{{s, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		r := []rune(current.word)
		if len(r) == 0 {
			continue
		}

		for i := range r {
			del := string(r[:i]) + string(r[i+1:])
			if _, exists := seen[del]; exists {
				continue
			}
			seen[del] = struct{}{}
			results = append(results, del)
			if current.depth+1 < dist {
				queue = append(queue, item{del, current.depth + 1})
			}
		}
	}

	return results
}

// lookup finds spelling correction candidates for the input word within maxDist
// edit distance. Returns candidates sorted by distance ascending, then frequency
// descending. Returns nil if input is empty or exceeds maxWordLen + maxDist.
func lookup(input string, maxDist int) []Suggestion {
	if input == "" {
		return nil
	}
	if maxDist > maxEditDistance {
		maxDist = maxEditDistance
	}

	inputLower := azcase.ToLower(input)
	inputLen := utf8.RuneCountInString(inputLower)

	// Exact match: distance 0.
	if freq, ok := words[inputLower]; ok {
		return []Suggestion{{Term: inputLower, Distance: 0, Frequency: freq}}
	}

	// No possible match if every dictionary word is too short.
	if inputLen-maxDist > maxWordLen {
		return nil
	}

	var results []Suggestion
	seen := make(map[string]struct{})

	// Generate delete variants of the input prefix, and include the prefix
	// itself so that distance-0 prefix collisions are detected.
	inputPrefix := truncateToRunes(inputLower, prefixLength)
	inputDeletes := generateDeletes(inputPrefix, maxDist)
	inputDeletes = append(inputDeletes, inputPrefix)

	for _, del := range inputDeletes {
		h := fnvHash(del)
		candidates, ok := deletes[h]
		if !ok {
			continue
		}

		for _, idx := range candidates {
			candidate := wordList[idx]
			if _, already := seen[candidate]; already {
				continue
			}
			seen[candidate] = struct{}{}

			candidateLen := utf8.RuneCountInString(candidate)

			// Quick length-based filter before expensive distance computation.
			lenDiff := inputLen - candidateLen
			if lenDiff < 0 {
				lenDiff = -lenDiff
			}
			if lenDiff > maxDist {
				continue
			}

			dist := damerauLevenshtein(inputLower, candidate)
			if dist <= maxDist {
				freq := words[candidate]
				results = append(results, Suggestion{Term: candidate, Distance: dist, Frequency: freq})
			}
		}
	}

	sortSuggestions(results)
	return results
}

// sortSuggestions sorts candidates by distance ascending, then frequency
// descending. Uses insertion sort because result sets are small (typically < 20).
func sortSuggestions(s []Suggestion) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && suggestionLess(key, s[j]) {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

func suggestionLess(a, b Suggestion) bool {
	if a.Distance != b.Distance {
		return a.Distance < b.Distance
	}
	return a.Frequency > b.Frequency
}

// damerauLevenshtein computes the optimal string alignment distance between a
// and b. This restricted variant handles transpositions of adjacent characters
// but does not allow a substring to be edited more than once.
func damerauLevenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la := len(ra)
	lb := len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	diff := la - lb
	if diff < 0 {
		diff = -diff
	}
	if diff > maxEditDistance {
		return diff
	}

	// Three-row approach: prev2, prev, curr for transposition support.
	prev2 := make([]int, lb+1)
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}

			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost

			best := del
			if ins < best {
				best = ins
			}
			if sub < best {
				best = sub
			}

			// Transposition of two adjacent characters.
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				trans := prev2[j-2] + cost
				if trans < best {
					best = trans
				}
			}

			curr[j] = best
		}

		prev2, prev, curr = prev, curr, prev2
	}

	return prev[lb]
}
