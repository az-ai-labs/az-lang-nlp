package sentiment

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/az-ai-labs/az-lang-nlp/data"
	"github.com/az-ai-labs/az-lang-nlp/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/normalize"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

// negationWord is the Azerbaijani copula used to negate predicates.
const negationWord = "deyil"

// lexicon maps stems to sentiment scores, built once at init.
var lexicon map[string]float64

func init() {
	lexicon = parseLexicon(data.SentimentLexicon)
}

// parseLexicon parses tab-separated "stem\tscore" lines.
func parseLexicon(raw string) map[string]float64 {
	m := make(map[string]float64, 256) //nolint:mnd
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "\t", 2) //nolint:mnd
		if len(parts) != 2 {                   //nolint:mnd
			continue
		}
		stem := strings.TrimSpace(parts[0])
		score, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}
		m[stem] = score
	}
	return m
}

// analyze implements the core sentiment analysis pipeline.
func analyze(text string) Result {
	text = azcase.ComposeNFC(text)
	words := tokenizer.Words(text)
	if len(words) == 0 {
		return Result{}
	}

	// Pre-compute stems to avoid double stemming during negation lookahead.
	stems := make([]string, len(words))
	for i, word := range words {
		if isNonLinguistic(word) {
			continue
		}
		stems[i] = azcase.ToLower(morph.Stem(normalize.NormalizeWord(word)))
	}

	var (
		sum      float64
		scored   int
		posCount int
		negCount int
	)

	for i, word := range words {
		if isNonLinguistic(word) {
			continue
		}

		stem := stems[i]

		// Skip the negation word itself.
		if stem == negationWord {
			continue
		}

		score, ok := lexicon[stem]
		if !ok {
			continue
		}

		// Negate score when the next meaningful word is "deyil".
		if followedByNeg(stems, i) {
			score = -score
		}

		sum += score
		scored++
		if score > 0 {
			posCount++
		} else if score < 0 {
			negCount++
		}
	}

	if scored == 0 {
		return Result{
			Sentiment: Neutral,
			Total:     len(words),
		}
	}

	avg := sum / float64(scored)

	var polarity Sentiment
	switch {
	case avg > 0:
		polarity = Positive
	case avg < 0:
		polarity = Negative
	default:
		polarity = Neutral
	}

	return Result{
		Sentiment: polarity,
		Score:     avg,
		Positive:  posCount,
		Negative:  negCount,
		Total:     len(words),
	}
}

// followedByNeg reports whether the next non-empty stem after position idx
// is the negation word "deyil".
func followedByNeg(stems []string, idx int) bool {
	for j := idx + 1; j < len(stems); j++ {
		if stems[j] == "" {
			continue
		}
		return stems[j] == negationWord
	}
	return false
}

// isNonLinguistic reports whether a word token is non-linguistic
// (all digits, or contains no letters).
func isNonLinguistic(word string) bool {
	for _, r := range word {
		if unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
