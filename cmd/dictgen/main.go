// Command dictgen generates morph/dict.txt from kaikki.org Azerbaijani
// dictionary dump (JSONL format).
//
// Download the dump from https://kaikki.org/dictionary/Azerbaijani/
// then run:
//
//	go run ./cmd/dictgen -input kaikki.org-dictionary-Azerbaijani.jsonl
//
// Output: morph/dict.txt (commit this file). Regenerate when a new
// Wiktionary dump is available.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
)

const (
	defaultInput   = "data/dictionary/kaikki.org-dictionary-Azerbaijani.jsonl"
	defaultOutput  = "morph/dict.txt"
	scannerBufSize = 1 << 20 // 1 MB
	minLemmaRunes  = 2
)

// kaikkiEntry holds only the fields needed from each JSONL line.
type kaikkiEntry struct {
	Word string `json:"word"`
	POS  string `json:"pos"`
}

func main() {
	inputPath := flag.String("input", defaultInput, "path to kaikki.org JSONL dump")
	outputPath := flag.String("output", defaultOutput, "output path for dict.txt")
	flag.Parse()

	if *inputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: dictgen -input <file> [-output <file>]\n")
		os.Exit(1)
	}

	f, err := os.Open(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: open input: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(f)
	buf := make([]byte, scannerBufSize)
	scanner.Buffer(buf, scannerBufSize)

	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry kaikkiEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines silently; they are rare in kaikki dumps.
			continue
		}

		posByte, ok := mapPOS(entry.POS)
		if !ok {
			continue
		}

		lemma := azcase.ToLower(entry.Word)

		if !isAcceptable(lemma) {
			continue
		}

		if posByte == 'V' {
			lemma = stripInfinitive(lemma)
		}

		if len([]rune(lemma)) < minLemmaRunes {
			continue
		}

		key := string(posByte) + lemma
		seen[key] = struct{}{}
	}

	scanErr := scanner.Err()

	// Close input file explicitly after scanning (no defer, avoids exitAfterDefer).
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: close input: %v\n", err)
		os.Exit(1)
	}

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "dictgen: scan error: %v\n", scanErr)
		os.Exit(1)
	}

	filterInflected(seen)

	lines := make([]string, 0, len(seen))
	for key := range seen {
		lines = append(lines, key)
	}
	// Sort by lemma (from index 1), not by POS+lemma, so that the embedded
	// dictLemmas slice in morph/dict.go is already sorted after stripping POS.
	// Ties broken by POS byte for deterministic output.
	sort.Slice(lines, func(i, j int) bool {
		li, lj := lines[i][1:], lines[j][1:]
		if li != lj {
			return li < lj
		}
		return lines[i][0] < lines[j][0]
	})

	out, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: create output: %v\n", err)
		os.Exit(1)
	}

	w := bufio.NewWriter(out)
	posCounts := make(map[byte]int)

	for _, l := range lines {
		if _, writeErr := fmt.Fprintln(w, l); writeErr != nil {
			fmt.Fprintf(os.Stderr, "dictgen: write error: %v\n", writeErr)
			os.Exit(1)
		}
		posCounts[l[0]]++
	}

	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: flush error: %v\n", err)
		os.Exit(1)
	}

	info, err := out.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: stat output: %v\n", err)
		os.Exit(1)
	}

	if err := out.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "dictgen: close output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Total entries: %d\n", len(lines))
	fmt.Fprintf(os.Stderr, "  N (noun/name/pron/num/det): %d\n", posCounts['N'])
	fmt.Fprintf(os.Stderr, "  V (verb):                   %d\n", posCounts['V'])
	fmt.Fprintf(os.Stderr, "  A (adjective):              %d\n", posCounts['A'])
	fmt.Fprintf(os.Stderr, "  D (adv/intj/conj/postp/particle): %d\n", posCounts['D'])
	fmt.Fprintf(os.Stderr, "  X (other):                  %d\n", posCounts['X'])
	fmt.Fprintf(os.Stderr, "Output file: %s (%d bytes)\n", *outputPath, info.Size())
}

// mapPOS maps a kaikki POS tag to a single-byte category.
// Returns false if the POS should be skipped entirely.
func mapPOS(pos string) (byte, bool) {
	switch pos {
	case "suffix", "prefix", "character":
		return 0, false
	case "noun", "name", "pron", "num", "det":
		return 'N', true
	case "verb":
		return 'V', true
	case "adj":
		return 'A', true
	case "adv", "intj", "conj", "postp", "particle":
		return 'D', true
	default:
		return 'X', true
	}
}

// isAcceptable reports whether a lowercased word is suitable for the dictionary.
// Rejects words with spaces, hyphens, digits, non-letter runes, Cyrillic
// characters, or fewer than minLemmaRunes runes.
func isAcceptable(word string) bool {
	runes := []rune(word)
	if len(runes) < minLemmaRunes {
		return false
	}
	for _, r := range runes {
		if r == ' ' || r == '-' {
			return false
		}
		if unicode.IsDigit(r) {
			return false
		}
		if !unicode.IsLetter(r) {
			return false
		}
		// Reject Cyrillic block (U+0400–U+04FF).
		if r >= '\u0400' && r <= '\u04FF' {
			return false
		}
		// Reject Arabic block (U+0600–U+06FF).
		if r >= '\u0600' && r <= '\u06FF' {
			return false
		}
	}
	return true
}

// stripInfinitive removes the Azerbaijani infinitive suffix (-maq or -mək)
// from a verb lemma. If the resulting stem passes isValidStem, the stem is
// returned; otherwise the original form is returned unchanged.
func stripInfinitive(word string) string {
	for _, suffix := range []string{"maq", "mək"} {
		if strings.HasSuffix(word, suffix) {
			stem := word[:len(word)-len([]byte(suffix))]
			if isValidStem(stem) {
				return stem
			}
		}
	}
	return word
}

// isValidStem reports whether s can be a valid Azerbaijani stem:
// at least minLemmaRunes runes and at least one Azerbaijani vowel.
func isValidStem(s string) bool {
	runes := 0
	hasVowel := false
	for _, r := range s {
		runes++
		if isAzVowel(r) {
			hasVowel = true
		}
	}
	return runes >= minLemmaRunes && hasVowel
}

// isAzVowel reports whether r is an Azerbaijani vowel (lowercase expected).
func isAzVowel(r rune) bool {
	switch r {
	case 'a', 'e',
		'\u0259', // ə
		'\u0131', // ı
		'i', 'o',
		'\u00F6', // ö
		'u',
		'\u00FC': // ü
		return true
	}
	return false
}

// filterInflected removes noun entries that are inflected forms of other nouns.
// An entry is removed if stripping a common suffix yields another noun in the set.
// Also handles consonant restoration (k↔y, q↔ğ) before vowel suffixes.
// This prevents the dictionary from containing kitablar, evlər, ürəyi, etc.
// alongside their base forms kitab, ev, ürək.
func filterInflected(seen map[string]struct{}) {
	// Suffixes removable by plain stripping.
	plainSuffixes := []string{
		"lar", "lər",
		"ları", "ləri",
		"da", "də", "ta", "tə",
		"dan", "dən", "tan", "tən",
	}
	// Suffixes that may co-occur with k→y / q→ğ consonant alternation.
	// Only applied when the remaining stem ends in y or ğ.
	restoreSuffixes := []string{
		"i", "ı", "u", "ü",
		"in", "ın", "un", "ün",
	}

	var toDelete []string
	for key := range seen {
		if key[0] != 'N' {
			continue
		}
		lemma := key[1:]
		filtered := false

		for _, suf := range plainSuffixes {
			if !strings.HasSuffix(lemma, suf) {
				continue
			}
			stem := lemma[:len(lemma)-len(suf)]
			if len([]rune(stem)) < minLemmaRunes {
				continue
			}
			if _, ok := seen["N"+stem]; ok {
				toDelete = append(toDelete, key)
				filtered = true
				break
			}
		}
		if filtered {
			continue
		}

		// Consonant restoration: ürəyi → ürək (y→k), otağı → otaq (ğ→q).
		for _, suf := range restoreSuffixes {
			if !strings.HasSuffix(lemma, suf) {
				continue
			}
			stem := lemma[:len(lemma)-len(suf)]
			runes := []rune(stem)
			if len(runes) < minLemmaRunes {
				continue
			}
			switch runes[len(runes)-1] {
			case 'y':
				runes[len(runes)-1] = 'k'
			case '\u011F': // ğ
				runes[len(runes)-1] = 'q'
			default:
				continue
			}
			if _, ok := seen["N"+string(runes)]; ok {
				toDelete = append(toDelete, key)
				filtered = true
				break
			}
		}
	}
	for _, key := range toDelete {
		delete(seen, key)
	}
}
