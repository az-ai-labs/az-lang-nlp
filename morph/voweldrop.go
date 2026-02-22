// Vowel drop restoration for Azerbaijani morphological analysis.
//
// This module reconstructs canonical dictionary stems from contracted
// forms produced by vowel dropping during inflection (e.g. oğul→oğl-um).
//
// Analyze() output is NOT modified. Only Stem() ranking uses this —
// the reconstruction is safe and does not affect the morpheme chain
// returned to the caller.
package morph

import "github.com/az-ai-labs/az-lang-nlp/internal/azcase"

// minRestoreLen is the minimum number of runes in a stem for vowel drop
// restoration to be attempted. A contracted stem must have at least a vowel
// prefix and a two-consonant cluster.
const minRestoreLen = 3

// tryRestoreVowelDrop attempts to restore a dropped vowel in a contracted
// stem by inserting vowels into the final consonant cluster and checking
// the dictionary. Returns the restored form or "" if restoration fails.
//
// Examples: oğlu → oğul, burnu → burun, ağzı → ağız
func tryRestoreVowelDrop(stem string) string {
	runes := []rune(stem)
	if len(runes) < minRestoreLen {
		return ""
	}

	// Find rightmost pair of consecutive non-vowel runes.
	insertPos := -1
	for i := len(runes) - 1; i > 0; i-- {
		if !isVowel(runes[i]) && !isVowel(runes[i-1]) {
			insertPos = i
			break
		}
	}
	if insertPos < 1 {
		return ""
	}

	// Prefix before insertion must contain a vowel (ensures we're not
	// operating on an all-consonant prefix, which would be invalid).
	prefix := string(runes[:insertPos])
	stemVowel := lastVowel(prefix)
	if stemVowel == 0 {
		return ""
	}

	// Try all 9 Azerbaijani vowels at the insertion point.
	azVowels := []rune{'a', 'e', '\u0259', 'i', '\u0131', 'o', '\u00F6', 'u', '\u00FC'}
	var matches []string
	for _, v := range azVowels {
		candidate := prefix + string(v) + string(runes[insertPos:])
		if isKnownStem(candidate) {
			matches = append(matches, candidate)
		}
	}

	switch len(matches) {
	case 0:
		return ""
	case 1:
		return matches[0]
	default:
		// Multiple dictionary hits — use four-way vowel harmony to pick
		// the correct one (e.g. aln → alın not alan).
		target := fourWayTarget(azcase.Lower(stemVowel))
		for _, m := range matches {
			if []rune(m)[insertPos] == target {
				return m
			}
		}
		return ""
	}
}
