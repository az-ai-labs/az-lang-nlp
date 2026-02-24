//go:build ignore

// buildlexicon evaluates the current sentiment lexicon against a review dataset
// and finds candidate words for lexicon expansion using Log-Likelihood Ratio.
// Run from the project root:
//
//	go run scripts/buildlexicon.go
//
// Outputs:
//   - data/lexicon_candidates.tsv — top candidate stems sorted by LLR
//   - data/lexicon_evaluation.txt — human-readable evaluation report
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/az-ai-labs/az-lang-nlp/data"
	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/normalize"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

const (
	reviewsPath    = "az_data/reviews_clean.tsv"
	candidatesPath = "data/lexicon_candidates.tsv"
	evaluationPath = "data/lexicon_evaluation.txt"
	scannerBufSize = 4 * 1024 * 1024
	minDocs        = 10
	minClassFreq   = 5
	llrThreshold   = 10.83
	maxCandidates  = 500
)

// reviewClass represents the sentiment class of a review.
type reviewClass int

const (
	classNegative reviewClass = iota
	classNeutral
	classPositive
)

// review holds a processed review.
type review struct {
	stems []string // deduplicated stems
	class reviewClass
}

// stemStats tracks per-stem document frequencies across review classes.
type stemStats struct {
	posCount int
	negCount int
	neuCount int
}

// candidate represents a lexicon expansion candidate.
type candidate struct {
	stem     string
	score    float64
	posCount int
	negCount int
	neuCount int
	llr      float64
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("[buildlexicon] ")

	lex := parseLexicon(data.SentimentLexicon)
	log.Printf("loaded lexicon with %d entries", len(lex))

	reviews, err := loadReviews(reviewsPath)
	if err != nil {
		log.Fatalf("cannot load reviews: %v", err)
	}

	totalPos, totalNeg, totalNeu := countClasses(reviews)
	log.Printf("loaded %d reviews (%d positive, %d negative, %d neutral)",
		len(reviews), totalPos, totalNeg, totalNeu)

	// Evaluate current lexicon — build confusion matrix.
	// Rows: actual class; columns: predicted class. Order: pos, neu, neg.
	var confusion [3][3]int
	for _, r := range reviews {
		predicted := predictClass(r.stems, lex)
		confusion[r.class][predicted]++
	}

	// Compute per-stem document frequencies.
	stats := make(map[string]*stemStats, 4096)
	for _, r := range reviews {
		for _, stem := range r.stems {
			s, ok := stats[stem]
			if !ok {
				s = &stemStats{}
				stats[stem] = s
			}
			switch r.class {
			case classPositive:
				s.posCount++
			case classNegative:
				s.negCount++
			case classNeutral:
				s.neuCount++
			}
		}
	}

	// Select candidates: stems not in lexicon, passing frequency and LLR filters.
	var candidates []candidate
	for stem, s := range stats {
		if _, inLex := lex[stem]; inLex {
			continue
		}
		totalStemDocs := s.posCount + s.negCount + s.neuCount
		if totalStemDocs < minDocs {
			continue
		}

		// Determine dominant class between positive and negative only.
		dominant := max(s.posCount, s.negCount)
		if dominant < minClassFreq {
			continue
		}

		// LLR 2x2 contingency table: positive vs negative reviews only.
		a := float64(s.posCount)
		b := float64(totalPos - s.posCount)
		c := float64(s.negCount)
		d := float64(totalNeg - s.negCount)
		g := llr(a, b, c, d)
		if g < llrThreshold {
			continue
		}

		// Score: ratio of (pos - neg) / (pos + neg), scaled to [-0.9, 0.9].
		posneg := s.posCount + s.negCount
		var score float64
		if posneg > 0 {
			ratio := float64(s.posCount-s.negCount) / float64(posneg)
			score = math.Round(ratio*0.9/0.05) * 0.05
		}

		candidates = append(candidates, candidate{
			stem:     stem,
			score:    score,
			posCount: s.posCount,
			negCount: s.negCount,
			neuCount: s.neuCount,
			llr:      g,
		})
	}

	// Sort by LLR descending.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].llr > candidates[j].llr
	})
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}
	log.Printf("found %d candidates", len(candidates))

	if err := writeCandidates(candidatesPath, candidates); err != nil {
		log.Fatalf("cannot write candidates: %v", err)
	}
	log.Printf("wrote candidates to %s", candidatesPath)

	if err := writeEvaluation(evaluationPath, confusion, candidates,
		len(reviews), totalPos, totalNeg, totalNeu, len(lex)); err != nil {
		log.Fatalf("cannot write evaluation: %v", err)
	}
	log.Printf("wrote evaluation to %s", evaluationPath)
}

// parseLexicon parses tab-separated "stem\tscore" lines.
// Lines starting with # and empty lines are ignored.
func parseLexicon(raw string) map[string]float64 {
	m := make(map[string]float64, 256)
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
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

// loadReviews reads the TSV file and returns processed reviews.
// Format: content\tscore (no header). Scores 1-2 = negative, 3 = neutral, 4-5 = positive.
func loadReviews(path string) ([]review, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, scannerBufSize)
	sc.Buffer(buf, scannerBufSize)

	var reviews []review
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		content := parts[0]
		scoreStr := strings.TrimSpace(parts[1])
		rating, err := strconv.Atoi(scoreStr)
		if err != nil {
			continue
		}

		class := scoreToClass(rating)
		stems := extractStems(content)
		reviews = append(reviews, review{stems: stems, class: class})

		if lineNum%10_000 == 0 {
			log.Printf("processed %d reviews", lineNum)
		}
	}
	return reviews, sc.Err()
}

// scoreToClass maps a 1-5 rating to a review class.
func scoreToClass(rating int) reviewClass {
	switch {
	case rating <= 2:
		return classNegative
	case rating >= 4:
		return classPositive
	default:
		return classNeutral
	}
}

// extractStems applies the analysis pipeline to a text and returns deduplicated stems.
// Pipeline: ComposeNFC -> Words -> skip isNonLinguistic -> NormalizeWord -> Stem -> ToLower.
func extractStems(text string) []string {
	text = azcase.ComposeNFC(text)
	words := tokenizer.Words(text)

	seen := make(map[string]struct{}, len(words))
	var stems []string
	for _, word := range words {
		if isNonLinguistic(word) {
			continue
		}
		stem := azcase.ToLower(morph.Stem(normalize.NormalizeWord(word)))
		if stem == "" {
			continue
		}
		if _, exists := seen[stem]; exists {
			continue
		}
		seen[stem] = struct{}{}
		stems = append(stems, stem)
	}
	return stems
}

// isNonLinguistic reports whether a word token contains no letters.
func isNonLinguistic(word string) bool {
	for _, r := range word {
		if unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// predictClass predicts the sentiment class for a set of stems using the lexicon.
func predictClass(stems []string, lex map[string]float64) reviewClass {
	var sum float64
	var scored int
	for _, stem := range stems {
		score, ok := lex[stem]
		if !ok {
			continue
		}
		sum += score
		scored++
	}
	if scored == 0 {
		return classNeutral
	}
	avg := sum / float64(scored)
	switch {
	case avg > 0:
		return classPositive
	case avg < 0:
		return classNegative
	default:
		return classNeutral
	}
}

// countClasses counts reviews by class.
func countClasses(reviews []review) (pos, neg, neu int) {
	for _, r := range reviews {
		switch r.class {
		case classPositive:
			pos++
		case classNegative:
			neg++
		case classNeutral:
			neu++
		}
	}
	return
}

// llr computes the Log-Likelihood Ratio (G-test) for a 2x2 contingency table.
// a = positive reviews containing stem
// b = positive reviews not containing stem
// c = negative reviews containing stem
// d = negative reviews not containing stem
func llr(a, b, c, d float64) float64 {
	n := a + b + c + d
	if n == 0 {
		return 0
	}
	var g float64
	for _, cell := range []struct{ obs, row, col float64 }{
		{a, a + b, a + c},
		{b, a + b, b + d},
		{c, c + d, a + c},
		{d, c + d, b + d},
	} {
		if cell.obs == 0 {
			continue
		}
		expected := cell.row * cell.col / n
		if expected == 0 {
			continue
		}
		g += cell.obs * math.Log(cell.obs/expected)
	}
	return 2 * g
}

// writeCandidates writes the top candidates to a TSV file.
func writeCandidates(path string, candidates []candidate) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriterSize(f, 4*1024*1024)
	fmt.Fprintln(bw, "# Lexicon expansion candidates — generated by buildlexicon.go")
	fmt.Fprintln(bw, "# stem\tscore\tpos_docs\tneg_docs\tneu_docs\ttotal_docs\tllr")
	for _, c := range candidates {
		total := c.posCount + c.negCount + c.neuCount
		fmt.Fprintf(bw, "%s\t%.2f\t%d\t%d\t%d\t%d\t%.4f\n",
			c.stem, c.score, c.posCount, c.negCount, c.neuCount, total, c.llr)
	}
	return bw.Flush()
}

// writeEvaluation writes the human-readable evaluation report.
func writeEvaluation(path string, confusion [3][3]int, candidates []candidate,
	total, totalPos, totalNeg, totalNeu, lexSize int) error {

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriterSize(f, 512*1024)

	fmt.Fprintln(bw, "Lexicon Evaluation Report")
	fmt.Fprintln(bw, "=========================")
	fmt.Fprintf(bw, "Dataset: %d reviews (%d positive, %d negative, %d neutral)\n",
		total, totalPos, totalNeg, totalNeu)
	fmt.Fprintf(bw, "Lexicon: %d entries\n", lexSize)
	fmt.Fprintln(bw)

	// Confusion matrix.
	// Rows are actual class (pos=2, neu=1, neg=0 in our const, so we reorder for display).
	// Display order: Pos, Neu, Neg (indices 2, 1, 0 mapped from classPositive=2, classNeutral=1, classNegative=0).
	fmt.Fprintln(bw, "Confusion Matrix:")
	fmt.Fprintln(bw, "              Predicted")
	fmt.Fprintln(bw, "              Pos   Neu   Neg")
	displayOrder := []struct {
		label string
		idx   reviewClass
	}{
		{"Actual Pos", classPositive},
		{"       Neu", classNeutral},
		{"       Neg", classNegative},
	}
	for _, row := range displayOrder {
		fmt.Fprintf(bw, "%s    %5d %5d %5d\n",
			row.label,
			confusion[row.idx][classPositive],
			confusion[row.idx][classNeutral],
			confusion[row.idx][classNegative],
		)
	}
	fmt.Fprintln(bw)

	// Accuracy.
	correct := confusion[classPositive][classPositive] +
		confusion[classNeutral][classNeutral] +
		confusion[classNegative][classNegative]
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total) * 100
	}
	fmt.Fprintf(bw, "Accuracy: %.2f%%\n", accuracy)

	// Per-class precision, recall, F1.
	classes := []struct {
		label string
		idx   reviewClass
	}{
		{"Positive", classPositive},
		{"Negative", classNegative},
		{"Neutral ", classNeutral},
	}
	for _, cls := range classes {
		tp := confusion[cls.idx][cls.idx]
		var fpSum, fnSum int
		for i := range 3 {
			if i != int(cls.idx) {
				fpSum += confusion[reviewClass(i)][cls.idx]
				fnSum += confusion[cls.idx][reviewClass(i)]
			}
		}
		p := precision(tp, fpSum)
		r := recall(tp, fnSum)
		f1 := f1score(p, r)
		fmt.Fprintf(bw, "%s: P=%.2f R=%.2f F1=%.2f\n", cls.label, p, r, f1)
	}

	// Top 20 candidates.
	fmt.Fprintln(bw)
	fmt.Fprintln(bw, "Top 20 candidates:")
	fmt.Fprintf(bw, "%-20s %6s %6s %6s %10s\n", "stem", "score", "pos", "neg", "llr")
	limit := min(20, len(candidates))
	for _, c := range candidates[:limit] {
		fmt.Fprintf(bw, "%-20s %6.2f %6d %6d %10.4f\n",
			c.stem, c.score, c.posCount, c.negCount, c.llr)
	}

	return bw.Flush()
}

func precision(tp, fp int) float64 {
	if tp+fp == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fp)
}

func recall(tp, fn int) float64 {
	if tp+fn == 0 {
		return 0
	}
	return float64(tp) / float64(tp+fn)
}

func f1score(p, r float64) float64 {
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}
