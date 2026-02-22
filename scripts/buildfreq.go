//go:build ignore

// buildfreq generates spell/freq.txt — a word frequency dictionary for the
// Azerbaijani spell checker. Run from the project root:
//
//	go run scripts/buildfreq.go
//
// Output format: one entry per line, "word frequency\n", sorted descending by frequency.
// All stems from morph/dict.txt are guaranteed at least frequency 1.
// Inflected (non-stem) forms are included only when frequency >= 50.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
	"github.com/az-ai-labs/az-lang-nlp/morph"
	"github.com/az-ai-labs/az-lang-nlp/tokenizer"
)

const (
	morphDictPath    = "morph/dict.txt"
	corpusAzPath     = "data/corpus/az-corpus/sentences.txt"
	corpusWikiPath   = "data/corpus/azwiki/articles.txt"
	outputPath       = "spell/freq.txt"
	inflectedMinFreq = 50
	scannerBufSize   = 4 * 1024 * 1024 // 4 MB — handles very long lines
)

type freqEntry struct {
	word string
	freq int
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("[buildfreq] ")

	// Load all stems from morph dict so every stem gets at least freq 1.
	stems, err := loadMorphStems(morphDictPath)
	if err != nil {
		log.Fatalf("cannot load morph dict: %v", err)
	}
	log.Printf("loaded %d stems from morph dict", len(stems))

	// freq tracks occurrence counts for every token we encounter.
	freq := make(map[string]int, len(stems)*4)

	// Seed freq with every morph stem at 0 so they appear in output.
	for stem := range stems {
		freq[stem] = 0
	}

	// Process each corpus file.
	corpora := []string{corpusAzPath, corpusWikiPath}
	for _, path := range corpora {
		n, err := processCorpus(path, freq)
		if err != nil {
			log.Printf("warning: skipping corpus %q: %v", path, err)
			continue
		}
		log.Printf("processed %d lines from %s", n, path)
	}

	// Build output entries.
	var entries []freqEntry
	for word, f := range freq {
		isStem := stems[word]
		if isStem {
			// Include all stems; ensure minimum freq 1.
			if f < 1 {
				f = 1
			}
			entries = append(entries, freqEntry{word, f})
		} else if f >= inflectedMinFreq {
			// Include inflected forms only when sufficiently frequent.
			entries = append(entries, freqEntry{word, f})
		}
	}

	// Sort descending by frequency, then alphabetically for stability.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].freq != entries[j].freq {
			return entries[i].freq > entries[j].freq
		}
		return entries[i].word < entries[j].word
	})

	// Write output.
	if err := writeOutput(outputPath, entries); err != nil {
		log.Fatalf("cannot write output: %v", err)
	}
	log.Printf("wrote %d entries to %s", len(entries), outputPath)
}

// loadMorphStems reads morph/dict.txt and returns a set of lowercased stems.
// Each line has a single POS byte prefix followed by the lemma.
func loadMorphStems(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stems := make(map[string]bool, 12000)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if len(line) < 2 {
			continue
		}
		// First byte is the POS tag; the rest is the lemma.
		lemma := line[1:]
		lower := azcase.ToLower(lemma)
		stems[lower] = true
	}
	return stems, sc.Err()
}

// processCorpus reads a plain-text corpus file line by line, tokenizes each
// line, and accumulates word and stem frequencies into freq.
// Returns the number of lines processed.
func processCorpus(path string, freq map[string]int) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	buf := make([]byte, scannerBufSize)
	sc.Buffer(buf, scannerBufSize)

	lines := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		words := tokenizer.Words(line)
		for _, w := range words {
			lower := azcase.ToLower(w)
			if lower == "" {
				continue
			}
			// Count the whole lowercased word form.
			freq[lower]++

			// Count the stem separately (if different from the word form).
			stem := azcase.ToLower(morph.Stem(lower))
			if stem != lower && stem != "" {
				freq[stem]++
			}
		}
		lines++
		if lines%100_000 == 0 {
			fmt.Fprintf(os.Stderr, "[buildfreq] %s: %d lines processed\n", path, lines)
		}
	}
	return lines, sc.Err()
}

// writeOutput writes sorted frequency entries to path.
func writeOutput(path string, entries []freqEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriterSize(f, 4*1024*1024)
	for _, e := range entries {
		fmt.Fprintf(bw, "%s %d\n", e.word, e.freq)
	}
	return bw.Flush()
}
