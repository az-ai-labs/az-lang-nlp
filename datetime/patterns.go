package datetime

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// wordSpan represents a word in the source text with its byte offsets.
type wordSpan struct {
	text  string // original case from source
	lower string // Azerbaijani-aware lowercase for matching
	start int    // byte offset (inclusive)
	end   int    // byte offset (exclusive)
}

// extract is the internal implementation of Extract.
func extract(s string, ref time.Time) []Result {
	const minCap = 4
	all := make([]Result, 0, len(s)/100+minCap)

	words := splitWords(s)
	lower := azLower(s)

	all = appendNumeric(all, s, ref)
	all = appendText(all, s, lower, words, ref)
	all = appendRelative(all, s, words, ref)

	if len(all) == 0 {
		return nil
	}

	all = resolveOverlaps(all)
	all = mergeAdjacent(all, s)
	return all
}

// ---------- appendNumeric ----------

// Capture group indices for date regexes (1-based submatch positions).
const (
	grpFirst  = 1 // first capture group
	grpSecond = 2 // second capture group
	grpThird  = 3 // third capture group
)

// appendNumeric matches ISO, dot, slash date formats and HH:MM(:SS) times.
func appendNumeric(all []Result, s string, ref time.Time) []Result {
	all = appendRegexDate(all, s, reISO, grpFirst, grpSecond, grpThird)   // YYYY-MM-DD
	all = appendRegexDate(all, s, reDot, grpThird, grpSecond, grpFirst)   // DD.MM.YYYY
	all = appendRegexDate(all, s, reSlash, grpThird, grpSecond, grpFirst) // DD/MM/YYYY
	all = appendTimeFmt(all, s, ref)
	return all
}

// appendRegexDate extracts dates from s using re whose capture groups at
// yearIdx, monthIdx, dayIdx (1-based) hold year, month, and day strings.
func appendRegexDate(all []Result, s string, re *regexp.Regexp, yearIdx, monthIdx, dayIdx int) []Result {
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		yearStr := s[m[yearIdx*2]:m[yearIdx*2+1]]
		monthStr := s[m[monthIdx*2]:m[monthIdx*2+1]]
		dayStr := s[m[dayIdx*2]:m[dayIdx*2+1]]

		year, month, day, ok := parseDateParts(yearStr, monthStr, dayStr)
		if !ok {
			continue
		}

		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		all = append(all, Result{
			Text:     s[m[0]:m[1]],
			Start:    m[0],
			End:      m[1],
			Type:     TypeDate,
			Time:     t,
			Explicit: HasYear | HasMonth | HasDay,
		})
	}
	return all
}

func appendTimeFmt(all []Result, s string, ref time.Time) []Result {
	for _, m := range reTime.FindAllStringSubmatchIndex(s, -1) {
		hourStr := s[m[2]:m[3]]
		minStr := s[m[4]:m[5]]

		hour, err := strconv.Atoi(hourStr)
		if err != nil || hour < minHour || hour > maxHour {
			continue
		}
		mn, err := strconv.Atoi(minStr)
		if err != nil || mn > maxMinute {
			continue
		}

		sec := 0
		explicit := HasHour | HasMinute
		if m[6] != -1 {
			secStr := s[m[6]:m[7]]
			sec, err = strconv.Atoi(secStr)
			if err != nil || sec > maxSecond {
				continue
			}
			explicit |= HasSecond
		}

		t := time.Date(ref.Year(), ref.Month(), ref.Day(), hour, mn, sec, 0, time.UTC)
		all = append(all, Result{
			Text:     s[m[0]:m[1]],
			Start:    m[0],
			End:      m[1],
			Type:     TypeTime,
			Time:     t,
			Explicit: explicit,
		})
	}
	return all
}

// parseDateParts validates and converts year/month/day strings to integers.
// Rejects impossible calendar dates like Feb 30 by checking time.Date normalization.
func parseDateParts(yearStr, monthStr, dayStr string) (year, month, day int, ok bool) {
	var err error
	year, err = strconv.Atoi(yearStr)
	if err != nil || year < minYear || year > maxYear {
		return 0, 0, 0, false
	}
	month, err = strconv.Atoi(monthStr)
	if err != nil || month < minMonth || month > maxMonth {
		return 0, 0, 0, false
	}
	day, err = strconv.Atoi(dayStr)
	if err != nil || day < minDay || day > maxDay {
		return 0, 0, 0, false
	}
	// Reject impossible calendar dates (e.g. Feb 30): time.Date normalizes
	// overflows, so a mismatch means the input date does not exist.
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day || t.Month() != time.Month(month) {
		return 0, 0, 0, false
	}
	return year, month, day, true
}

// ---------- appendText ----------

// appendText matches natural Azerbaijani text patterns:
// month names (with optional day/year), weekday names, and "saat" + number.
func appendText(all []Result, s, lower string, words []wordSpan, ref time.Time) []Result {
	used := make([]bool, len(words))

	// Pass 1: month-based patterns (highest priority for text matching)
	for i, w := range words {
		if used[i] {
			continue
		}
		mo, ok := months[w.lower]
		if !ok {
			continue
		}

		// Try to build the widest possible date span around this month.
		spanStart := w.start
		spanEnd := w.end
		explicit := HasMonth
		day := 0
		year := 0

		// Check for possessive compound: genitive-month + "ayının" + ordinal-day
		// e.g. "mart ayının 15-i"
		if i+2 < len(words) && words[i+1].lower == bridgeWord {
			if d, ok := parseOrdinalWord(words[i+2].lower); ok && d >= minDay && d <= maxDay {
				day = d
				spanEnd = words[i+2].end
				explicit |= HasDay
				used[i] = true
				used[i+1] = true
				used[i+2] = true
			}
		}

		// Check for genitive month + possessive day: "martın 15-i"
		if day == 0 && genitiveMonths[w.lower] && i+1 < len(words) {
			if d, ok := parseOrdinalWord(words[i+1].lower); ok && d >= minDay && d <= maxDay {
				day = d
				spanEnd = words[i+1].end
				explicit |= HasDay
				used[i] = true
				used[i+1] = true
			}
		}

		// Look backward for a day number: "5 mart" or "5-ci mart"
		if day == 0 && i > 0 && !used[i-1] {
			if d, ok := parseOrdinalWord(words[i-1].lower); ok && d >= minDay && d <= maxDay {
				day = d
				spanStart = words[i-1].start
				explicit |= HasDay
				used[i-1] = true
			} else if d, ok := parseBareNumber(words[i-1].lower); ok && d >= minDay && d <= maxDay {
				day = d
				spanStart = words[i-1].start
				explicit |= HasDay
				used[i-1] = true
			}
		}

		// Look forward for a 4-digit year: "mart 2026" or "5 mart 2026"
		nextIdx := i + 1
		for nextIdx < len(words) && used[nextIdx] {
			nextIdx++
		}
		if nextIdx < len(words) {
			if y, ok := parse4DigitYear(words[nextIdx].lower); ok {
				year = y
				spanEnd = words[nextIdx].end
				explicit |= HasYear
				used[nextIdx] = true
			}
		}

		used[i] = true

		// Resolve missing components from ref.
		if year == 0 {
			year = ref.Year()
		}
		if day == 0 {
			day = 1
		}

		t := time.Date(year, mo, day, 0, 0, 0, 0, time.UTC)
		all = append(all, Result{
			Text:     s[spanStart:spanEnd],
			Start:    spanStart,
			End:      spanEnd,
			Type:     TypeDate,
			Time:     t,
			Explicit: explicit,
		})
	}

	// Pass 2: weekday names
	all = appendWeekdays(all, s, lower, words, used, ref)

	// Pass 3: "saat" + number (time-of-day, not duration context)
	all = appendSaatTime(all, s, words, used, ref)

	return all
}

// appendWeekdays matches Azerbaijani weekday names in the word list.
func appendWeekdays(all []Result, s, lower string, words []wordSpan, used []bool, ref time.Time) []Result {
	for _, wd := range weekdays {
		// Search for each weekday name in the lowered full string.
		offset := 0
		for {
			idx := strings.Index(lower[offset:], wd.name)
			if idx < 0 {
				break
			}
			matchStart := offset + idx
			matchEnd := matchStart + len(wd.name)
			offset = matchEnd

			// Verify word boundaries: character before must not be a letter,
			// character after must not be a letter.
			if matchStart > 0 {
				r, _ := utf8.DecodeLastRuneInString(s[:matchStart])
				if unicode.IsLetter(r) {
					continue
				}
			}
			if matchEnd < len(s) {
				r, _ := utf8.DecodeRuneInString(s[matchEnd:])
				if unicode.IsLetter(r) {
					continue
				}
			}

			// Check that no word in the matched range is already used.
			overlaps := false
			for j := range words {
				if used[j] && words[j].start < matchEnd && words[j].end > matchStart {
					overlaps = true
					break
				}
			}
			if overlaps {
				continue
			}

			// Resolve to next occurrence of this weekday (bare weekday includes today).
			t := nextWeekday(ref, wd.weekday, false)
			all = append(all, Result{
				Text:     s[matchStart:matchEnd],
				Start:    matchStart,
				End:      matchEnd,
				Type:     TypeDate,
				Time:     t,
				Explicit: HasYear | HasMonth | HasDay,
			})

			// Mark overlapping words as used.
			for j := range words {
				if words[j].start >= matchStart && words[j].end <= matchEnd {
					used[j] = true
				}
			}
		}
	}
	return all
}

// appendSaatTime matches "saat N" patterns as time-of-day expressions.
// Skips when the number precedes "saat" (duration context, handled by appendRelative).
func appendSaatTime(all []Result, s string, words []wordSpan, used []bool, ref time.Time) []Result {
	for i, w := range words {
		if used[i] || w.lower != "saat" {
			continue
		}

		// "saat" must be followed by a number.
		if i+1 >= len(words) || used[i+1] {
			continue
		}

		// If preceding word is a number, this is duration context ("2 saat") — skip.
		if i > 0 && !used[i-1] {
			if _, ok := parseBareNumber(words[i-1].lower); ok {
				continue
			}
		}

		hourWord := words[i+1].lower
		// Strip ordinal/possessive suffix from the hour number (e.g. "3-də" → "3").
		hour, ok := parseNumberWithSuffix(hourWord)
		if !ok {
			continue
		}
		if hour < minHour || hour > maxHour {
			continue
		}

		spanStart := w.start
		spanEnd := words[i+1].end
		explicit := HasHour

		// Check for time-of-day modifier before "saat" (e.g. "axşam saat 7").
		if i > 0 && !used[i-1] {
			if shift, ok := timeOfDayWords[words[i-1].lower]; ok {
				if shift == shiftPM && hour < 12 && hour > 0 {
					hour += 12
				}
				spanStart = words[i-1].start
				used[i-1] = true
			}
		}

		used[i] = true
		used[i+1] = true

		t := time.Date(ref.Year(), ref.Month(), ref.Day(), hour, 0, 0, 0, time.UTC)
		all = append(all, Result{
			Text:     s[spanStart:spanEnd],
			Start:    spanStart,
			End:      spanEnd,
			Type:     TypeTime,
			Time:     t,
			Explicit: explicit,
		})
	}
	return all
}

// ---------- appendRelative ----------

// appendRelative matches relative date expressions:
// single keywords ("bu gün", "sabah"), period keywords ("keçən həftə"),
// and quantity-direction ("3 gün əvvəl").
func appendRelative(all []Result, s string, words []wordSpan, ref time.Time) []Result {
	used := make([]bool, len(words))

	// Pass 1: quantity-direction ("3 gün əvvəl", "2 həftə sonra")
	// Must run before keyword matching so "3 gün" isn't consumed as partial.
	for i := range words {
		if used[i] || i+2 >= len(words) {
			continue
		}

		qty, ok := parseBareNumber(words[i].lower)
		if !ok || qty <= 0 {
			continue
		}

		unit, ok := quantityUnits[words[i+1].lower]
		if !ok {
			continue
		}

		dir, ok := directionWords[words[i+2].lower]
		if !ok {
			continue
		}

		t := applyQuantityOffset(ref, qty, unit, dir)
		explicit := HasYear | HasMonth | HasDay
		typ := TypeDate
		if unit == qtyHour || unit == qtyMinute || unit == qtySecond {
			explicit |= HasHour | HasMinute | HasSecond
			typ = TypeDateTime
		}

		used[i] = true
		used[i+1] = true
		used[i+2] = true

		all = append(all, Result{
			Text:     s[words[i].start:words[i+2].end],
			Start:    words[i].start,
			End:      words[i+2].end,
			Type:     typ,
			Time:     t,
			Explicit: explicit,
		})
	}

	// Pass 2: two-word period keywords ("keçən həftə", "bu ay", "gələn il")
	for i := range words {
		if used[i] || i+1 >= len(words) || used[i+1] {
			continue
		}

		offset, ok := periodPrefix[words[i].lower]
		if !ok {
			continue
		}

		pk, ok := periodUnits[words[i+1].lower]
		if !ok {
			continue
		}

		t := resolvePeriod(ref, offset, pk)
		used[i] = true
		used[i+1] = true

		all = append(all, Result{
			Text:     s[words[i].start:words[i+1].end],
			Start:    words[i].start,
			End:      words[i+1].end,
			Type:     TypeDate,
			Time:     t,
			Explicit: HasYear | HasMonth | HasDay,
		})
	}

	// Pass 3: two-word day offsets ("bu gün")
	for i := range words {
		if used[i] || i+1 >= len(words) || used[i+1] {
			continue
		}
		twoWord := words[i].lower + " " + words[i+1].lower
		if offset, ok := dayOffsets[twoWord]; ok {
			t := ref.AddDate(0, 0, offset)
			used[i] = true
			used[i+1] = true
			all = append(all, Result{
				Text:     s[words[i].start:words[i+1].end],
				Start:    words[i].start,
				End:      words[i+1].end,
				Type:     TypeDate,
				Time:     time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
				Explicit: HasYear | HasMonth | HasDay,
			})
		}
	}

	// Pass 4: single-word day offsets ("sabah", "dünən", "bugün")
	for i := range words {
		if used[i] {
			continue
		}
		if offset, ok := dayOffsets[words[i].lower]; ok {
			t := ref.AddDate(0, 0, offset)
			used[i] = true
			all = append(all, Result{
				Text:     s[words[i].start:words[i].end],
				Start:    words[i].start,
				End:      words[i].end,
				Type:     TypeDate,
				Time:     time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
				Explicit: HasYear | HasMonth | HasDay,
			})
		}
	}

	// Pass 5: weekday with keçən/gələn prefix ("keçən bazar ertəsi", "gələn cümə")
	all = appendPrefixedWeekday(all, s, words, used, ref)

	return all
}

// appendPrefixedWeekday matches "keçən/gələn + weekday" patterns.
func appendPrefixedWeekday(all []Result, s string, words []wordSpan, used []bool, ref time.Time) []Result {
	for i := range words {
		if used[i] {
			continue
		}
		offset, ok := periodPrefix[words[i].lower]
		if !ok || offset == 0 {
			// "bu" + weekday is ambiguous; skip for now.
			continue
		}

		// Try matching weekday starting at i+1.
		if i+1 >= len(words) || used[i+1] {
			continue
		}

		remaining := len(words) - (i + 1)
		for _, wd := range weekdays {
			wdWords := strings.Fields(wd.name)
			if len(wdWords) > remaining {
				continue
			}

			match := true
			for j, ww := range wdWords {
				if words[i+1+j].lower != ww {
					match = false
					break
				}
			}
			if !match {
				continue
			}

			endIdx := i + 1 + len(wdWords) - 1
			for j := i; j <= endIdx; j++ {
				used[j] = true
			}

			var t time.Time
			if offset < 0 {
				t = prevWeekday(ref, wd.weekday)
			} else {
				// "gələn" skips today even if it matches the weekday.
				t = nextWeekday(ref, wd.weekday, true)
			}

			all = append(all, Result{
				Text:     s[words[i].start:words[endIdx].end],
				Start:    words[i].start,
				End:      words[endIdx].end,
				Type:     TypeDate,
				Time:     t,
				Explicit: HasYear | HasMonth | HasDay,
			})
			break
		}
	}
	return all
}

// ---------- overlap resolution and merging ----------

// resolveOverlaps removes overlapping results. When two results overlap,
// the longer (more specific) match wins. Ties broken by earlier start position.
// Returns results sorted by Start offset.
func resolveOverlaps(results []Result) []Result {
	if len(results) <= 1 {
		return results
	}

	slices.SortFunc(results, func(a, b Result) int {
		if c := cmp.Compare(a.Start, b.Start); c != 0 {
			return c
		}
		la := a.End - a.Start
		lb := b.End - b.Start
		return cmp.Compare(lb, la)
	})

	out := make([]Result, 0, len(results))
	maxEnd := 0
	for _, r := range results {
		if r.Start >= maxEnd {
			out = append(out, r)
			if len(out) >= maxResults {
				break
			}
			maxEnd = r.End
		}
	}
	return out
}

// mergeAdjacent combines adjacent TypeDate + TypeTime results into TypeDateTime
// when they are separated by at most maxMergeGap bytes.
func mergeAdjacent(results []Result, s string) []Result {
	if len(results) < 2 { //nolint:mnd
		return results
	}

	out := make([]Result, 0, len(results))
	i := 0
	for i < len(results) {
		if i+1 < len(results) {
			a, b := results[i], results[i+1]
			gap := b.Start - a.End
			if gap >= 0 && gap <= maxMergeGap {
				if merged, ok := tryMerge(a, b, s); ok {
					out = append(out, merged)
					i += 2
					continue
				}
			}
		}
		out = append(out, results[i])
		i++
	}
	return out
}

// tryMerge merges a date result and a time result into a datetime result.
func tryMerge(a, b Result, s string) (Result, bool) {
	var dateR, timeR Result
	switch {
	case a.Type == TypeDate && b.Type == TypeTime:
		dateR, timeR = a, b
	case a.Type == TypeTime && b.Type == TypeDate:
		timeR, dateR = a, b
	default:
		return Result{}, false
	}

	start := min(dateR.Start, timeR.Start)
	end := max(dateR.End, timeR.End)

	merged := Result{
		Text:     s[start:end],
		Start:    start,
		End:      end,
		Type:     TypeDateTime,
		Explicit: dateR.Explicit | timeR.Explicit,
	}

	merged.Time = time.Date(
		dateR.Time.Year(), dateR.Time.Month(), dateR.Time.Day(),
		timeR.Time.Hour(), timeR.Time.Minute(), timeR.Time.Second(),
		0, time.UTC,
	)

	return merged, true
}

// ---------- time computation helpers ----------

// nextWeekday returns the next occurrence of the given weekday.
// When skipToday is false, today is returned if it matches.
// When skipToday is true (e.g. "gələn cümə" on a Friday), today is skipped.
func nextWeekday(ref time.Time, wd time.Weekday, skipToday bool) time.Time {
	days := int(wd) - int(ref.Weekday())
	if days < 0 {
		days += daysPerWeek
	}
	if days == 0 {
		if skipToday {
			days = daysPerWeek
		} else {
			return time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	t := ref.AddDate(0, 0, days)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// prevWeekday returns the most recent past occurrence of the given weekday before ref.
func prevWeekday(ref time.Time, wd time.Weekday) time.Time {
	days := int(ref.Weekday()) - int(wd)
	if days <= 0 {
		days += daysPerWeek
	}
	t := ref.AddDate(0, 0, -days)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// resolvePeriod computes the start of a period offset from ref.
func resolvePeriod(ref time.Time, offset int, pk periodKind) time.Time {
	switch pk {
	case periodWeek:
		// Go to Monday of the current week, then add offset weeks.
		daysToMonday := int(ref.Weekday()) - int(time.Monday)
		if daysToMonday < 0 {
			daysToMonday += daysPerWeek
		}
		monday := ref.AddDate(0, 0, -daysToMonday)
		t := monday.AddDate(0, 0, offset*daysPerWeek)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

	case periodMonth:
		t := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, time.UTC)
		return t.AddDate(0, offset, 0)

	case periodYear:
		return time.Date(ref.Year()+offset, time.January, 1, 0, 0, 0, 0, time.UTC)

	default:
		return ref
	}
}

// applyQuantityOffset applies a quantity+unit+direction offset to ref.
func applyQuantityOffset(ref time.Time, qty int, unit qtyUnit, dir dirKind) time.Time {
	if dir == dirBefore {
		qty = -qty
	}
	switch unit {
	case qtyDay:
		return ref.AddDate(0, 0, qty)
	case qtyWeek:
		return ref.AddDate(0, 0, qty*daysPerWeek)
	case qtyMonth:
		return ref.AddDate(0, qty, 0)
	case qtyYear:
		return ref.AddDate(qty, 0, 0)
	case qtyHour:
		return ref.Add(time.Duration(qty) * time.Hour)
	case qtyMinute:
		return ref.Add(time.Duration(qty) * time.Minute)
	case qtySecond:
		return ref.Add(time.Duration(qty) * time.Second)
	default:
		return ref
	}
}

// ---------- word scanning helpers ----------

// splitWords scans s and returns all contiguous runs of letters/digits
// with their byte offsets. Hyphenated tokens like "5-ci" are kept as one word
// if the hyphen connects a digit sequence to a letter sequence.
func splitWords(s string) []wordSpan {
	var words []wordSpan
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isWordChar(r) {
			i += size
			continue
		}

		start := i
		for i < len(s) {
			r, size = utf8.DecodeRuneInString(s[i:])
			if isWordChar(r) {
				i += size
				continue
			}
			// Allow hyphens and dots within words (e.g. "5-ci", "bazar ertəsi" is 2 words).
			if (r == '-' || r == '.') && i+size < len(s) {
				next, _ := utf8.DecodeRuneInString(s[i+size:])
				if isWordChar(next) {
					i += size
					continue
				}
			}
			break
		}

		text := s[start:i]
		words = append(words, wordSpan{
			text:  text,
			lower: azLower(text),
			start: start,
			end:   i,
		})
	}
	return words
}

// isWordChar returns true if r is a letter or digit.
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// azLower lowercases a string with Azerbaijani-specific rules:
// İ (U+0130) → i, I (U+0049) → ı (U+0131).
func azLower(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u0130': // İ → i
			b.WriteByte('i')
		case 'I': // I → ı (Azerbaijani convention)
			b.WriteRune('ı')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// parseOrdinalWord extracts the numeric value from an ordinal or possessive word.
// Matches patterns like "5-ci", "1-inci", "5-nci", "15-i", "3-ü".
func parseOrdinalWord(s string) (int, bool) {
	m := reOrdinalDay.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseBareNumber parses a string as a plain integer (1-2 digits).
func parseBareNumber(s string) (int, bool) {
	if s == "" || len(s) > 2 {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parse4DigitYear parses a string as a 4-digit year.
func parse4DigitYear(s string) (int, bool) {
	if len(s) != 4 { //nolint:mnd
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < minYear || n > maxYear {
		return 0, false
	}
	return n, true
}

// parseNumberWithSuffix extracts the leading number from a word that may
// have a locative or other suffix attached (e.g. "3-də" → 3, "14" → 14).
func parseNumberWithSuffix(s string) (int, bool) {
	// Try ordinal first ("3-cü", "5-ci").
	if n, ok := parseOrdinalWord(s); ok {
		return n, true
	}
	// Try bare number.
	if n, ok := parseBareNumber(s); ok {
		return n, true
	}
	// Try extracting leading digits from hyphenated suffix ("3-də").
	idx := strings.IndexByte(s, '-')
	if idx > 0 {
		if n, ok := parseBareNumber(s[:idx]); ok {
			return n, true
		}
	}
	return 0, false
}
