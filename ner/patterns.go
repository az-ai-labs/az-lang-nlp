package ner

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
)

// Compiled regexes for each entity type.
// Order matters: more specific patterns (IBAN, URL, Email) are matched first
// so they take priority over generic ones (FIN bare, VOEN bare) in overlap resolution.
var (
	// Phone: international format +994 XX XXX XX XX (spaces optional)
	rePhoneIntl = regexp.MustCompile(`\+994\s?(\d{2})\s?(\d{3})\s?(\d{2})\s?(\d{2})`)
	// Phone: local format 0XX XXX XX XX (spaces optional)
	rePhoneLocal = regexp.MustCompile(`\b0(\d{2})\s?(\d{3})\s?(\d{2})\s?(\d{2})\b`)

	// Email: standard pattern
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

	// URL: http or https prefixed, restricted to RFC 3986 characters
	reURL = regexp.MustCompile(`https?://[A-Za-z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+`)

	// IBAN: AZ + 2 digits + 4 uppercase letters + 20 alphanumeric chars = 28 total
	reIBAN = regexp.MustCompile(`\bAZ\d{2}[A-Z]{4}[A-Z0-9]{20}\b`)

	// LicensePlate: XX-YY-ZZZ format
	reLicensePlate = regexp.MustCompile(`\b\d{2}-[A-Z]{2}-\d{3}\b`)

	// FIN labeled: preceded by keyword "FIN" with optional colon/space
	reFINLabeled = regexp.MustCompile(`(?i)\bFIN[:\s]\s?([A-HJ-NP-Z0-9]{7})\b`)
	// FIN bare: 7 uppercase alphanumeric chars (no I, no O)
	reFINBare = regexp.MustCompile(`\b[A-HJ-NP-Z0-9]{7}\b`)

	// VOEN labeled: preceded by keyword "VOEN" or "VÖEN" with optional colon/space.
	// Bare VOEN is not matched — 10 bare digits are too ambiguous.
	reVOENLabeled = regexp.MustCompile(`(?i)\bV[ÖO]EN[:\s]\s?(\d{10})\b`)
)

// maxEmailLen is the maximum length of an email address per RFC 5321.
const maxEmailLen = 254

// maxEntities is the maximum number of entities returned per call.
const maxEntities = 10000

// recognize is the internal implementation of Recognize.
func recognize(s string) []Entity {
	// Pre-allocate with a heuristic: ~1 entity per 200 bytes.
	const minCap = 8
	all := make([]Entity, 0, len(s)/200+minCap)

	// High-specificity patterns first
	all = appendURL(all, s)
	all = appendEmail(all, s)
	all = appendIBAN(all, s)
	all = appendLicensePlate(all, s)
	all = appendPhone(all, s)

	// Ambiguous patterns last (FIN/VOEN labeled, then bare)
	all = appendFIN(all, s)
	all = appendVOEN(all, s)

	if len(all) == 0 {
		return nil
	}

	// resolveOverlaps returns entities already sorted by Start offset.
	return resolveOverlaps(all)
}

// appendPhone appends phone numbers in both international and local formats.
func appendPhone(all []Entity, s string) []Entity {
	for _, m := range rePhoneIntl.FindAllStringIndex(s, -1) {
		all = append(all, Entity{
			Text:  s[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Type:  Phone,
		})
	}
	for _, m := range rePhoneLocal.FindAllStringIndex(s, -1) {
		all = append(all, Entity{
			Text:  s[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Type:  Phone,
		})
	}
	return all
}

// appendEmail appends email addresses, skipping those exceeding RFC 5321 length.
func appendEmail(all []Entity, s string) []Entity {
	for _, m := range reEmail.FindAllStringIndex(s, -1) {
		if m[1]-m[0] > maxEmailLen {
			continue
		}
		all = append(all, Entity{
			Text:  s[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Type:  Email,
		})
	}
	return all
}

// appendURL appends HTTP/HTTPS URLs, trimming trailing punctuation.
func appendURL(all []Entity, s string) []Entity {
	for _, m := range reURL.FindAllStringIndex(s, -1) {
		text := s[m[0]:m[1]]
		// Trim trailing punctuation that is likely not part of the URL.
		text = strings.TrimRight(text, ".,;:!?)]}>")
		end := m[0] + len(text)
		all = append(all, Entity{
			Text:  text,
			Start: m[0],
			End:   end,
			Type:  URL,
		})
	}
	return all
}

// appendIBAN appends Azerbaijani IBAN numbers.
func appendIBAN(all []Entity, s string) []Entity {
	for _, m := range reIBAN.FindAllStringIndex(s, -1) {
		all = append(all, Entity{
			Text:  s[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Type:  IBAN,
		})
	}
	return all
}

// appendLicensePlate appends Azerbaijani license plates.
func appendLicensePlate(all []Entity, s string) []Entity {
	for _, m := range reLicensePlate.FindAllStringIndex(s, -1) {
		all = append(all, Entity{
			Text:  s[m[0]:m[1]],
			Start: m[0],
			End:   m[1],
			Type:  LicensePlate,
		})
	}
	return all
}

// appendFIN appends FIN codes. Labeled matches (preceded by "FIN" keyword)
// take priority; bare matches at the same position are skipped.
func appendFIN(all []Entity, s string) []Entity {
	// Track labeled positions to skip bare duplicates.
	labeled := make(map[int]struct{})

	// Labeled matches: "FIN: XXXXXXX" or "FIN XXXXXXX"
	for _, sub := range reFINLabeled.FindAllStringSubmatchIndex(s, -1) {
		// sub[2]:sub[3] is the capture group (the 7-char code)
		labeled[sub[2]] = struct{}{}
		all = append(all, Entity{
			Text:    s[sub[2]:sub[3]],
			Start:   sub[2],
			End:     sub[3],
			Type:    FIN,
			Labeled: true,
		})
	}

	// Bare matches: 7-char [A-HJ-NP-Z0-9] with word boundaries.
	// Must contain at least one letter and one digit to avoid matching
	// pure words (PRODUCT, VERSION) or pure numbers (1234567).
	for _, m := range reFINBare.FindAllStringIndex(s, -1) {
		if _, ok := labeled[m[0]]; ok {
			continue
		}
		text := s[m[0]:m[1]]
		if isMixedAlphanumeric(text) {
			all = append(all, Entity{
				Text:  text,
				Start: m[0],
				End:   m[1],
				Type:  FIN,
			})
		}
	}

	return all
}

// appendVOEN appends VOEN codes. Only labeled matches (preceded by "VOEN"/"VÖEN")
// are recognized — bare 10-digit sequences are too ambiguous.
func appendVOEN(all []Entity, s string) []Entity {
	for _, sub := range reVOENLabeled.FindAllStringSubmatchIndex(s, -1) {
		all = append(all, Entity{
			Text:    s[sub[2]:sub[3]],
			Start:   sub[2],
			End:     sub[3],
			Type:    VOEN,
			Labeled: true,
		})
	}
	return all
}

// isMixedAlphanumeric returns true if s contains at least one ASCII letter
// and at least one ASCII digit. Used to filter bare FIN candidates.
func isMixedAlphanumeric(s string) bool {
	var hasLetter, hasDigit bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			hasLetter = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// resolveOverlaps removes overlapping entities. When two entities overlap:
//   - The longer (more specific) match wins.
//   - If equal length, labeled wins over unlabeled.
//   - If still tied, the first one encountered wins.
//
// Returns entities sorted by Start offset.
func resolveOverlaps(entities []Entity) []Entity {
	if len(entities) <= 1 {
		return entities
	}

	// Sort by start offset, then by length descending, then labeled first.
	slices.SortFunc(entities, func(a, b Entity) int {
		if c := cmp.Compare(a.Start, b.Start); c != 0 {
			return c
		}
		la := a.End - a.Start
		lb := b.End - b.Start
		if c := cmp.Compare(lb, la); c != 0 {
			return c
		}
		if a.Labeled != b.Labeled {
			if a.Labeled {
				return -1
			}
			return 1
		}
		return 0
	})

	result := make([]Entity, 0, len(entities))
	maxEnd := 0

	for _, e := range entities {
		if e.Start >= maxEnd {
			result = append(result, e)
			if len(result) >= maxEntities {
				break
			}
			maxEnd = e.End
		}
	}

	return result
}
