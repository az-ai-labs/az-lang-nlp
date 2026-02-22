package keywords

import (
	"bytes"
	_ "embed"
	"math"
	"strconv"
)

//go:embed freq.txt
var freqRaw []byte

// Corpus frequency data (populated in init, read-only after).
var (
	corpusFreq  map[string]int64
	totalTokens int64
)

const maxCandidates = 10000 // internal processing cap on unique stems

func init() {
	lines := bytes.Split(freqRaw, []byte("\n"))
	corpusFreq = make(map[string]int64, len(lines))

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

		corpusFreq[word] = freq
		totalTokens += freq
	}
}

func computeIDF(stem string) float64 {
	freq, ok := corpusFreq[stem]
	if !ok {
		return math.Log(float64(totalTokens))
	}
	return math.Log(float64(totalTokens) / float64(1+freq))
}

func scoreTFIDF(stems []string) []Keyword {
	tf := make(map[string]int, len(stems))
	for _, s := range stems {
		if _, exists := tf[s]; !exists && len(tf) >= maxCandidates {
			continue
		}
		tf[s]++
	}

	docLen := float64(len(stems))
	result := make([]Keyword, 0, len(tf))
	for stem, count := range tf {
		normalizedTF := float64(count) / docLen
		idf := computeIDF(stem)
		score := normalizedTF * idf
		result = append(result, Keyword{Stem: stem, Score: score, Count: count})
	}

	return result
}
