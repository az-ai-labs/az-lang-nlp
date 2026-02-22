package morph

import (
	"sort"

	"github.com/az-ai-labs/az-lang-nlp/internal/azcase"
)

const maxDepth = 10

// walker holds the state for a single backtracking morphological analysis run.
type walker struct {
	origRunes  []rune     // original-cased word as runes
	lowerRunes []rune     // lowercased word as runes
	results    []Analysis // accumulated analyses
}

// terminalStates caches all unique toState values from suffixRules.
// These represent states that can be the "outermost" (rightmost) suffix outcome.
// Computed once at init time.
var terminalStates []fsmState

// minSuffixLen maps each toState to the shortest suffix (in runes) that
// produces that state. Used for early termination in walk().
var minSuffixLen map[fsmState]int

func init() {
	seen := make(map[fsmState]bool)
	for i := range suffixRules {
		seen[suffixRules[i].toState] = true
	}
	terminalStates = make([]fsmState, 0, len(seen))
	for s := range seen {
		terminalStates = append(terminalStates, s)
	}

	// Pre-compute rune forms for all suffix surfaces.
	for i := range suffixRules {
		rule := &suffixRules[i]
		rule.surfaceRunes = make([][]rune, len(rule.surfaces))
		for j, s := range rule.surfaces {
			rule.surfaceRunes[j] = []rune(s)
		}
	}

	// Compute minimum suffix length for each toState.
	minSuffixLen = make(map[fsmState]int)
	for i := range suffixRules {
		rule := &suffixRules[i]
		for _, sr := range rule.surfaceRunes {
			l := len(sr)
			if prev, ok := minSuffixLen[rule.toState]; !ok || l < prev {
				minSuffixLen[rule.toState] = l
			}
		}
	}
}

// analyze performs morphological analysis on word, returning all valid parses
// sorted by morpheme count descending (deepest analysis first), deduplicated.
func analyze(word string) []Analysis {
	low := azcase.ToLower(word)
	origRunes := []rune(word)
	lowerRunes := []rune(low)

	w := &walker{
		origRunes:  origRunes,
		lowerRunes: lowerRunes,
	}

	// The suffix table uses left-to-right morphotactic semantics:
	//   fromStates = valid predecessor states, toState = successor state.
	// Since we strip right-to-left, we start from terminal states and
	// work backward: match rule.toState == currentState, then recurse
	// into each rule.fromStates entry. Base case: state == initial.
	runeLen := len(lowerRunes)
	for _, ts := range terminalStates {
		w.walk(runeLen, ts, nil, 0)
	}

	w.results = dedup(w.results)

	// Sort analyses by plausibility. Known dictionary stems rank first.
	// Among known stems, prefer longer stems (less stripping) and simpler
	// analyses (fewer morphemes) — Occam's razor. Among unknown stems,
	// prefer shorter stems (deeper stripping found the real root), then
	// simpler analyses (fewer morphemes) for same-length stems.
	sort.Slice(w.results, func(i, j int) bool {
		ki := isKnownStem(azcase.ToLower(w.results[i].Stem))
		kj := isKnownStem(azcase.ToLower(w.results[j].Stem))
		if ki != kj {
			return ki
		}
		si, sj := len([]rune(w.results[i].Stem)), len([]rune(w.results[j].Stem))
		if ki && kj {
			// Both known: prefer longer stem (less aggressive stripping).
			if si != sj {
				return si > sj
			}
			// Same-length known stems: prefer fewer morphemes (simpler parse).
			mi, mj := len(w.results[i].Morphemes), len(w.results[j].Morphemes)
			if mi != mj {
				return mi < mj
			}
		} else {
			// Unknown stems: prefer shorter stem (deeper stripping), then
			// simpler parse (fewer morphemes) for same-length stems.
			if si != sj {
				return si < sj
			}
			mi, mj := len(w.results[i].Morphemes), len(w.results[j].Morphemes)
			if mi != mj {
				return mi < mj
			}
		}
		return tagsKey(w.results[i].Morphemes) < tagsKey(w.results[j].Morphemes)
	})
	return w.results
}

// walk recursively strips suffixes from the right, building morpheme chains.
// pos is a rune index: runes [0..pos) are the remaining candidate stem.
// state is the expected toState of the next suffix to strip (going right-to-left).
// When state == initial, we've traced back to the stem boundary.
func (w *walker) walk(pos int, state fsmState, morphemes []Morpheme, depth int) {
	// Base case: traced back to initial → check stem validity.
	if state == initial {
		if pos > 0 && isValidStem(string(w.lowerRunes[:pos])) {
			w.results = append(w.results, Analysis{
				Stem:      string(w.origRunes[:pos]),
				Morphemes: cloneMorphemes(morphemes),
			})
		}
		return
	}

	if depth >= maxDepth {
		return
	}

	// Early termination: remaining runes too short for any suffix producing this state.
	if minLen, ok := minSuffixLen[state]; ok && pos < minLen {
		return
	}

	for ri := range suffixRules {
		rule := &suffixRules[ri]
		if rule.toState != state {
			continue
		}

		for si, surface := range rule.surfaces {
			surfRunes := rule.surfaceRunes[si]
			surfLen := len(surfRunes)
			if surfLen > pos {
				continue
			}

			stemEnd := pos - surfLen
			if !runesEqual(w.lowerRunes[stemEnd:pos], surfRunes) {
				continue
			}

			// Vowel harmony validation against the remaining stem AFTER stripping.
			stemPart := string(w.lowerRunes[:stemEnd])
			stemLV := lastVowel(stemPart)
			suffFV := firstVowel(surface)

			switch rule.harmony {
			case backFront:
				if stemLV != 0 && suffFV != 0 && !matchesBackFront(stemLV, suffFV) {
					continue
				}
			case fourWay:
				if stemLV != 0 && suffFV != 0 && !matchesFourWay(stemLV, suffFV) {
					continue
				}
			}

			// Consonant assimilation for d/t alternation.
			// Only reject t-form after non-voiceless consonants. The d-form
			// is accepted after all consonants because real-world Azerbaijani
			// text commonly uses -d even after voiceless consonants (e.g.
			// "danışdı", "kitabçılıqda"). Dictionary-aware ranking handles
			// disambiguation. Copula skips the check entirely — both forms
			// are standard (e.g. "gəlmişdir" and "gəlmiştir").
			if surfRunes[0] == 't' {
				if hasDTVariants(rule) && stemEnd > 0 && rule.tag != Copula {
					preceding := w.lowerRunes[stemEnd-1]
					if !isVoiceless(preceding) {
						continue
					}
				}
			}

			// Build new morpheme list (prepend for left-to-right order).
			origSurface := string(w.origRunes[stemEnd:pos])
			newMorphemes := make([]Morpheme, len(morphemes)+1)
			newMorphemes[0] = Morpheme{Surface: origSurface, Tag: rule.tag}
			copy(newMorphemes[1:], morphemes)

			// Recurse into each valid predecessor state.
			for _, fromState := range rule.fromStates {
				w.walk(stemEnd, fromState, newMorphemes, depth+1)

				// k/q softening: if suffix starts with a vowel and the stem
				// ends with y or ğ, try restoring the underlying k or q.
				if stemEnd > 0 && isVowel(surfRunes[0]) {
					lastStemRune := w.lowerRunes[stemEnd-1]
					if lastStemRune == 'y' {
						w.tryRestoredStem(stemEnd, 'k', fromState, newMorphemes, depth)
					}
					if lastStemRune == '\u011F' { // ğ
						w.tryRestoredStem(stemEnd, 'q', fromState, newMorphemes, depth)
					}
				}
			}
		}
	}
}

// tryRestoredStem temporarily replaces the last rune of the stem at newPos-1
// with restoredRune (k or q), checks if the restored form is a valid stem,
// and recurses. The original runes are restored afterward.
func (w *walker) tryRestoredStem(newPos int, restoredRune rune, state fsmState, morphemes []Morpheme, depth int) {
	idx := newPos - 1

	savedLower := w.lowerRunes[idx]
	savedOrig := w.origRunes[idx]

	w.lowerRunes[idx] = restoredRune
	w.origRunes[idx] = restoredRune

	w.walk(newPos, state, morphemes, depth+1)

	w.lowerRunes[idx] = savedLower
	w.origRunes[idx] = savedOrig
}

// firstVowel returns the first vowel rune in s, or 0 if none found.
func firstVowel(s string) rune {
	for _, r := range s {
		if isVowel(r) {
			return r
		}
	}
	return 0
}

// hasDTVariants reports whether a suffix rule has both d-starting and
// t-starting surface forms, indicating consonant assimilation applies.
func hasDTVariants(rule *suffixRule) bool {
	hasD := false
	hasT := false
	for _, surf := range rule.surfaces {
		runes := []rune(surf)
		if len(runes) > 0 {
			switch runes[0] {
			case 'd':
				hasD = true
			case 't':
				hasT = true
			}
		}
		if hasD && hasT {
			return true
		}
	}
	return false
}

// runesEqual reports whether two rune slices are identical.
func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// cloneMorphemes returns a copy of the morpheme slice.
func cloneMorphemes(ms []Morpheme) []Morpheme {
	out := make([]Morpheme, len(ms))
	copy(out, ms)
	return out
}

// dedup removes duplicate analyses. Two analyses are equal if they have the
// same stem and identical morpheme tag sequences.
func dedup(results []Analysis) []Analysis {
	if len(results) <= 1 {
		return results
	}
	type key struct {
		stem string
		tags string
	}
	seen := make(map[key]struct{}, len(results))
	out := make([]Analysis, 0, len(results))
	for _, a := range results {
		k := key{stem: a.Stem, tags: tagsKey(a.Morphemes)}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, a)
	}
	return out
}

// tagsKey builds a string key from morpheme tags for deduplication.
const avgTagNameLen = 8

func tagsKey(ms []Morpheme) string {
	if len(ms) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(ms)*avgTagNameLen)
	for i, m := range ms {
		if i > 0 {
			buf = append(buf, '|')
		}
		buf = append(buf, m.Tag.String()...)
	}
	return string(buf)
}
