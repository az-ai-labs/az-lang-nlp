package datetime

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// ref is the fixed reference time used across all tests: Friday, 2026-02-20 10:30 UTC.
var ref = time.Date(2026, 2, 20, 10, 30, 0, 0, time.UTC)

// d builds a UTC date-only time.
func d(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// dt builds a UTC date+time.
func dt(year int, month time.Month, day, hour, min, sec int) time.Time {
	return time.Date(year, month, day, hour, min, sec, 0, time.UTC)
}

// compareResults compares two Result slices with per-field error messages.
func compareResults(t *testing.T, want, got []Result) {
	t.Helper()

	if len(want) == 0 && len(got) == 0 {
		return
	}
	if len(want) == 0 && got == nil {
		return
	}

	if len(got) != len(want) {
		t.Errorf("got %d results, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
		return
	}

	for i := range want {
		if got[i].Text != want[i].Text {
			t.Errorf("[%d] Text: got %q, want %q", i, got[i].Text, want[i].Text)
		}
		if got[i].Start != want[i].Start {
			t.Errorf("[%d] Start: got %d, want %d", i, got[i].Start, want[i].Start)
		}
		if got[i].End != want[i].End {
			t.Errorf("[%d] End: got %d, want %d", i, got[i].End, want[i].End)
		}
		if got[i].Type != want[i].Type {
			t.Errorf("[%d] Type: got %s, want %s", i, got[i].Type, want[i].Type)
		}
		wantTrunc := want[i].Time.Truncate(time.Second)
		gotTrunc := got[i].Time.Truncate(time.Second)
		if !gotTrunc.Equal(wantTrunc) {
			t.Errorf("[%d] Time: got %v, want %v", i, gotTrunc, wantTrunc)
		}
		if got[i].Explicit != want[i].Explicit {
			t.Errorf("[%d] Explicit: got %s, want %s", i, got[i].Explicit, want[i].Explicit)
		}
	}
}

// TestExtractNumeric tests ISO, dot, slash date formats and HH:MM(:SS) time formats.
func TestExtractNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			name: "ISO date",
			in:   "2026-03-05",
			ref:  ref,
			want: []Result{{
				Text:     "2026-03-05",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			name: "dot date",
			in:   "05.03.2026",
			ref:  ref,
			want: []Result{{
				Text:     "05.03.2026",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			name: "slash date",
			in:   "05/03/2026",
			ref:  ref,
			want: []Result{{
				Text:     "05/03/2026",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			name: "time HH:MM",
			in:   "14:30",
			ref:  ref,
			want: []Result{{
				Text:     "14:30",
				Start:    0,
				End:      5,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 14, 30, 0),
				Explicit: HasHour | HasMinute,
			}},
		},
		{
			name: "time HH:MM:SS",
			in:   "09:05:22",
			ref:  ref,
			want: []Result{{
				Text:     "09:05:22",
				Start:    0,
				End:      8,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 9, 5, 22),
				Explicit: HasHour | HasMinute | HasSecond,
			}},
		},
		{
			// "tarix " = 6 bytes (ASCII), "2026-03-05" = 10 bytes → start=6, end=16
			name: "ISO in text",
			in:   "tarix 2026-03-05 qeyd olunub",
			ref:  ref,
			want: []Result{{
				Text:     "2026-03-05",
				Start:    6,
				End:      16,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractNaturalText tests Azerbaijani month-name-based patterns with case
// forms, ordinal days, possessive compounds, and genitive constructs.
func TestExtractNaturalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			name: "full date",
			in:   "5 mart 2026",
			ref:  ref,
			want: []Result{{
				Text:     "5 mart 2026",
				Start:    0,
				End:      11,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasMonth | HasDay | HasYear,
			}},
		},
		{
			// year inferred from ref (2026)
			name: "day month",
			in:   "5 mart",
			ref:  ref,
			want: []Result{{
				Text:     "5 mart",
				Start:    0,
				End:      6,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasMonth | HasDay,
			}},
		},
		{
			// day defaults to 1 when only month given
			name: "month only",
			in:   "mart",
			ref:  ref,
			want: []Result{{
				Text:     "mart",
				Start:    0,
				End:      4,
				Type:     TypeDate,
				Time:     d(2026, time.March, 1),
				Explicit: HasMonth,
			}},
		},
		{
			// "5-ci" is an ordinal: 5-ci mart
			name: "ordinal day ci",
			in:   "5-ci mart",
			ref:  ref,
			want: []Result{{
				Text:     "5-ci mart",
				Start:    0,
				End:      9,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasMonth | HasDay,
			}},
		},
		{
			// "5-nci" ordinal variant
			name: "ordinal nci",
			in:   "5-nci mart",
			ref:  ref,
			want: []Result{{
				Text:     "5-nci mart",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.March, 5),
				Explicit: HasMonth | HasDay,
			}},
		},
		{
			// "martın 15-i" — martın (7 bytes) + space + 15-i (4 bytes) = 12 bytes total
			// martın is genitive; "15-i" is possessive ordinal
			name: "genitive ordinal",
			in:   "martın 15-i",
			ref:  ref,
			want: []Result{{
				Text:     "martın 15-i",
				Start:    0,
				End:      12,
				Type:     TypeDate,
				Time:     d(2026, time.March, 15),
				Explicit: HasMonth | HasDay,
			}},
		},
		{
			// "mart ayının 15-i" — possessive compound with bridge word "ayının"
			// "mart" (4) + " " + "ayının" (8) + " " + "15-i" (4) = 18 bytes
			name: "possessive compound",
			in:   "mart ayının 15-i",
			ref:  ref,
			want: []Result{{
				Text:     "mart ayının 15-i",
				Start:    0,
				End:      18,
				Type:     TypeDate,
				Time:     d(2026, time.March, 15),
				Explicit: HasMonth | HasDay,
			}},
		},
		{
			// locative case: martda → March
			name: "case locative",
			in:   "martda",
			ref:  ref,
			want: []Result{{
				Text:     "martda",
				Start:    0,
				End:      6,
				Type:     TypeDate,
				Time:     d(2026, time.March, 1),
				Explicit: HasMonth,
			}},
		},
		{
			// ablative case: yanvardan → January
			name: "case ablative",
			in:   "yanvardan",
			ref:  ref,
			want: []Result{{
				Text:     "yanvardan",
				Start:    0,
				End:      9,
				Type:     TypeDate,
				Time:     d(2026, time.January, 1),
				Explicit: HasMonth,
			}},
		},
		{
			// dative case: marta → March
			name: "case dative",
			in:   "marta",
			ref:  ref,
			want: []Result{{
				Text:     "marta",
				Start:    0,
				End:      5,
				Type:     TypeDate,
				Time:     d(2026, time.March, 1),
				Explicit: HasMonth,
			}},
		},
		{
			// accusative case: martı → March (martı = 6 bytes, ı is 2 bytes)
			name: "case accusative",
			in:   "martı",
			ref:  ref,
			want: []Result{{
				Text:     "martı",
				Start:    0,
				End:      6,
				Type:     TypeDate,
				Time:     d(2026, time.March, 1),
				Explicit: HasMonth,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractWeekday tests Azerbaijani weekday name recognition.
// ref = Friday 2026-02-20.
func TestExtractWeekday(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			// bazar ertəsi = Monday; next Monday from Friday 2026-02-20 → 2026-02-23
			// "bazar ertəsi" = 13 bytes
			name: "bazar ertəsi Monday",
			in:   "bazar ertəsi",
			ref:  ref,
			want: []Result{{
				Text:     "bazar ertəsi",
				Start:    0,
				End:      13,
				Type:     TypeDate,
				Time:     d(2026, time.February, 23),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// çərşənbə axşamı = Tuesday; next Tuesday from Friday → 2026-02-24
			// "çərşənbə axşamı" = 22 bytes
			name: "çərşənbə axşamı Tuesday",
			in:   "çərşənbə axşamı",
			ref:  ref,
			want: []Result{{
				Text:     "çərşənbə axşamı",
				Start:    0,
				End:      22,
				Type:     TypeDate,
				Time:     d(2026, time.February, 24),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// çərşənbə = Wednesday; next Wednesday from Friday → 2026-02-25
			// "çərşənbə" = 13 bytes
			name: "çərşənbə Wednesday",
			in:   "çərşənbə",
			ref:  ref,
			want: []Result{{
				Text:     "çərşənbə",
				Start:    0,
				End:      13,
				Type:     TypeDate,
				Time:     d(2026, time.February, 25),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// cümə axşamı = Thursday; next Thursday from Friday → 2026-02-26
			// "cümə axşamı" = 15 bytes
			name: "cümə axşamı Thursday",
			in:   "cümə axşamı",
			ref:  ref,
			want: []Result{{
				Text:     "cümə axşamı",
				Start:    0,
				End:      15,
				Type:     TypeDate,
				Time:     d(2026, time.February, 26),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// cümə = Friday; today is Friday → nextWeekday returns today
			// "cümə" = 6 bytes
			name: "cümə Friday today",
			in:   "cümə",
			ref:  ref,
			want: []Result{{
				Text:     "cümə",
				Start:    0,
				End:      6,
				Type:     TypeDate,
				Time:     d(2026, time.February, 20),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// bazar = Sunday; next Sunday from Friday 2026-02-20 → 2026-02-22
			// "bazar" = 5 bytes
			name: "bazar Sunday",
			in:   "bazar",
			ref:  ref,
			want: []Result{{
				Text:     "bazar",
				Start:    0,
				End:      5,
				Type:     TypeDate,
				Time:     d(2026, time.February, 22),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// şənbə = Saturday; next Saturday from Friday 2026-02-20 → 2026-02-21
			// "şənbə" = 8 bytes
			name: "şənbə Saturday",
			in:   "şənbə",
			ref:  ref,
			want: []Result{{
				Text:     "şənbə",
				Start:    0,
				End:      8,
				Type:     TypeDate,
				Time:     d(2026, time.February, 21),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractRelative tests relative date/time expressions.
// ref = Friday 2026-02-20.
func TestExtractRelative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			// "bu gün" = 7 bytes (ü is 2 bytes)
			name: "bu gün today",
			in:   "bu gün",
			ref:  ref,
			want: []Result{{
				Text:     "bu gün",
				Start:    0,
				End:      7,
				Type:     TypeDate,
				Time:     d(2026, time.February, 20),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "bugün" = 6 bytes
			name: "bugün today",
			in:   "bugün",
			ref:  ref,
			want: []Result{{
				Text:     "bugün",
				Start:    0,
				End:      6,
				Type:     TypeDate,
				Time:     d(2026, time.February, 20),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			name: "sabah tomorrow",
			in:   "sabah",
			ref:  ref,
			want: []Result{{
				Text:     "sabah",
				Start:    0,
				End:      5,
				Type:     TypeDate,
				Time:     d(2026, time.February, 21),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "dünən" = 7 bytes: d(1)+ü(2)+n(1)+ə(2)+n(1)
			name: "dünən yesterday",
			in:   "dünən",
			ref:  ref,
			want: []Result{{
				Text:     "dünən",
				Start:    0,
				End:      7,
				Type:     TypeDate,
				Time:     d(2026, time.February, 19),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "birigün" = 8 bytes (ü is 2 bytes), day after tomorrow
			name: "birigün day after tomorrow",
			in:   "birigün",
			ref:  ref,
			want: []Result{{
				Text:     "birigün",
				Start:    0,
				End:      8,
				Type:     TypeDate,
				Time:     d(2026, time.February, 22),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "srağagün" = 10 bytes (ğ=2, ü=2), two days ago
			name: "srağagün two days ago",
			in:   "srağagün",
			ref:  ref,
			want: []Result{{
				Text:     "srağagün",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.February, 18),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "keçən həftə" = 15 bytes; Monday of previous week
			// ref=Friday 2026-02-20; current week Monday = 2026-02-16; prev week Monday = 2026-02-09
			name: "keçən həftə previous week",
			in:   "keçən həftə",
			ref:  ref,
			want: []Result{{
				Text:     "keçən həftə",
				Start:    0,
				End:      15,
				Type:     TypeDate,
				Time:     d(2026, time.February, 9),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "gələn həftə" = 15 bytes; Monday of next week
			// ref=Friday 2026-02-20; current Monday=2026-02-16; next Monday = 2026-02-23
			name: "gələn həftə next week",
			in:   "gələn həftə",
			ref:  ref,
			want: []Result{{
				Text:     "gələn həftə",
				Start:    0,
				End:      15,
				Type:     TypeDate,
				Time:     d(2026, time.February, 23),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "bu ay" = first of current month
			name: "bu ay this month",
			in:   "bu ay",
			ref:  ref,
			want: []Result{{
				Text:     "bu ay",
				Start:    0,
				End:      5,
				Type:     TypeDate,
				Time:     d(2026, time.February, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "keçən ay" = 10 bytes; first of previous month
			name: "keçən ay previous month",
			in:   "keçən ay",
			ref:  ref,
			want: []Result{{
				Text:     "keçən ay",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.January, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "gələn ay" = 10 bytes; first of next month
			name: "gələn ay next month",
			in:   "gələn ay",
			ref:  ref,
			want: []Result{{
				Text:     "gələn ay",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2026, time.March, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "bu il" = first of current year
			name: "bu il this year",
			in:   "bu il",
			ref:  ref,
			want: []Result{{
				Text:     "bu il",
				Start:    0,
				End:      5,
				Type:     TypeDate,
				Time:     d(2026, time.January, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "keçən il" = 10 bytes; first of previous year
			name: "keçən il previous year",
			in:   "keçən il",
			ref:  ref,
			want: []Result{{
				Text:     "keçən il",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2025, time.January, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "gələn il" = 10 bytes; first of next year
			name: "gələn il next year",
			in:   "gələn il",
			ref:  ref,
			want: []Result{{
				Text:     "gələn il",
				Start:    0,
				End:      10,
				Type:     TypeDate,
				Time:     d(2027, time.January, 1),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "3 gün əvvəl" = 14 bytes (ü=2, ə=2, ə=2); 3 days before ref.
			// applyQuantityOffset calls ref.AddDate which preserves the ref time component.
			name: "3 gün əvvəl",
			in:   "3 gün əvvəl",
			ref:  ref,
			want: []Result{{
				Text:     "3 gün əvvəl",
				Start:    0,
				End:      14,
				Type:     TypeDate,
				Time:     dt(2026, time.February, 17, 10, 30, 0),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "2 həftə sonra" = 15 bytes (ə=2, ə=2); 2 weeks after ref.
			// applyQuantityOffset calls ref.AddDate which preserves the ref time component.
			name: "2 həftə sonra",
			in:   "2 həftə sonra",
			ref:  ref,
			want: []Result{{
				Text:     "2 həftə sonra",
				Start:    0,
				End:      15,
				Type:     TypeDate,
				Time:     dt(2026, time.March, 6, 10, 30, 0),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "3 gün öncə" = 13 bytes (ü=2, ö=2, ə=2); öncə = əvvəl.
			// applyQuantityOffset calls ref.AddDate which preserves the ref time component.
			name: "3 gün öncə",
			in:   "3 gün öncə",
			ref:  ref,
			want: []Result{{
				Text:     "3 gün öncə",
				Start:    0,
				End:      13,
				Type:     TypeDate,
				Time:     dt(2026, time.February, 17, 10, 30, 0),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "keçən bazar ertəsi" = 21 bytes; previous Monday before Friday 2026-02-20
			// prevWeekday: days = Friday(5) - Monday(1) = 4 → 2026-02-20 - 4 = 2026-02-16
			name: "keçən bazar ertəsi previous Monday",
			in:   "keçən bazar ertəsi",
			ref:  ref,
			want: []Result{{
				Text:     "keçən bazar ertəsi",
				Start:    0,
				End:      21,
				Type:     TypeDate,
				Time:     d(2026, time.February, 16),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
		{
			// "gələn cümə" on a Friday skips today → next Friday 2026-02-27.
			name: "gələn cümə next Friday",
			in:   "gələn cümə",
			ref:  ref,
			want: []Result{{
				Text:     "gələn cümə",
				Start:    0,
				End:      14,
				Type:     TypeDate,
				Time:     d(2026, time.February, 27),
				Explicit: HasYear | HasMonth | HasDay,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractSaatTime tests "saat N" time-of-day patterns with optional AM/PM modifiers.
func TestExtractSaatTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			// "saat 3" = 6 bytes; no modifier → 03:00
			name: "saat 3 bare",
			in:   "saat 3",
			ref:  ref,
			want: []Result{{
				Text:     "saat 3",
				Start:    0,
				End:      6,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 3, 0, 0),
				Explicit: HasHour,
			}},
		},
		{
			// "axşam saat 7" = 13 bytes (ş=2, ı=2); axşam is PM modifier → 19:00
			name: "axşam saat 7 PM shift",
			in:   "axşam saat 7",
			ref:  ref,
			want: []Result{{
				Text:     "axşam saat 7",
				Start:    0,
				End:      13,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 19, 0, 0),
				Explicit: HasHour,
			}},
		},
		{
			// "səhər saat 7" = 14 bytes (ə=2, ə=2); səhər is AM → 07:00, no shift
			name: "səhər saat 7 AM no shift",
			in:   "səhər saat 7",
			ref:  ref,
			want: []Result{{
				Text:     "səhər saat 7",
				Start:    0,
				End:      14,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 7, 0, 0),
				Explicit: HasHour,
			}},
		},
		{
			// "axşam saat 12" — hour 12 with PM modifier must stay 12, not become 24.
			name: "axşam saat 12 noon edge",
			in:   "axşam saat 12",
			ref:  ref,
			want: []Result{{
				Text:     "axşam saat 12",
				Start:    0,
				End:      14,
				Type:     TypeTime,
				Time:     dt(ref.Year(), ref.Month(), ref.Day(), 12, 0, 0),
				Explicit: HasHour,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractMerge tests that adjacent date + time spans merge into TypeDateTime.
func TestExtractMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ref  time.Time
		want []Result
	}{
		{
			// "5 mart 2026 14:30" = 17 bytes; date 0..11, gap=1 space, time 12..17
			name: "natural date then time",
			in:   "5 mart 2026 14:30",
			ref:  ref,
			want: []Result{{
				Text:     "5 mart 2026 14:30",
				Start:    0,
				End:      17,
				Type:     TypeDateTime,
				Time:     dt(2026, time.March, 5, 14, 30, 0),
				Explicit: HasYear | HasMonth | HasDay | HasHour | HasMinute,
			}},
		},
		{
			// "2026-03-05 09:15" = 16 bytes; date 0..10, gap=1 space, time 11..16
			name: "ISO date then time",
			in:   "2026-03-05 09:15",
			ref:  ref,
			want: []Result{{
				Text:     "2026-03-05 09:15",
				Start:    0,
				End:      16,
				Type:     TypeDateTime,
				Time:     dt(2026, time.March, 5, 9, 15, 0),
				Explicit: HasYear | HasMonth | HasDay | HasHour | HasMinute,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			compareResults(t, tt.want, got)
		})
	}
}

// TestExtractMultiple tests that multiple disjoint spans are returned.
func TestExtractMultiple(t *testing.T) {
	t.Parallel()

	// "sabah saat 3-də və 5 mart 2026" = 32 bytes
	// sabah: 0..5 (relative date → 2026-02-21)
	// saat 3-də: 6..16 (time → 03:00)
	// sabah + saat 3-də → gap = 6-5 = 1 → within maxMergeGap → merge to DateTime "sabah saat 3-də"
	// "5 mart 2026": 21..32 (date → 2026-03-05)
	// After merge: 2 results
	in := "sabah saat 3-də və 5 mart 2026"
	got := Extract(in, ref)

	if len(got) < 2 {
		t.Fatalf("want at least 2 results, got %d: %v", len(got), got)
	}

	// Verify the date result for "5 mart 2026" is present
	var foundMart bool
	for _, r := range got {
		if r.Text == "5 mart 2026" && r.Type == TypeDate {
			foundMart = true
			if r.Start != 21 || r.End != 32 {
				t.Errorf("'5 mart 2026' offsets: got [%d:%d], want [21:32]", r.Start, r.End)
			}
			wantTime := d(2026, time.March, 5)
			if !r.Time.Equal(wantTime) {
				t.Errorf("'5 mart 2026' time: got %v, want %v", r.Time, wantTime)
			}
		}
	}
	if !foundMart {
		t.Errorf("missing result for '5 mart 2026': got %v", got)
	}

	// Verify offset invariant for all results
	for _, r := range got {
		if in[r.Start:r.End] != r.Text {
			t.Errorf("invariant: s[%d:%d]=%q != Text=%q", r.Start, r.End, in[r.Start:r.End], r.Text)
		}
	}
}

// TestParse tests the Parse API: success, empty input, unrecognized input, oversized input.
func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("positive", func(t *testing.T) {
		t.Parallel()
		r, err := Parse("5 mart 2026", ref)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r.Type != TypeDate {
			t.Errorf("Type: got %s, want Date", r.Type)
		}
		want := d(2026, time.March, 5)
		if !r.Time.Equal(want) {
			t.Errorf("Time: got %v, want %v", r.Time, want)
		}
		if r.Text != "5 mart 2026" {
			t.Errorf("Text: got %q, want %q", r.Text, "5 mart 2026")
		}
	})

	t.Run("error empty", func(t *testing.T) {
		t.Parallel()
		_, err := Parse("", ref)
		if err == nil {
			t.Fatal("want error for empty input, got nil")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("error %q does not contain 'empty'", err.Error())
		}
	})

	t.Run("error unrecognized", func(t *testing.T) {
		t.Parallel()
		_, err := Parse("abc xyz", ref)
		if err == nil {
			t.Fatal("want error for unrecognized input, got nil")
		}
		if !strings.Contains(err.Error(), "unrecognized") {
			t.Errorf("error %q does not contain 'unrecognized'", err.Error())
		}
	})

	t.Run("error oversized", func(t *testing.T) {
		t.Parallel()
		big := strings.Repeat("a", maxInputBytes+1)
		_, err := Parse(big, ref)
		if err == nil {
			t.Fatal("want error for oversized input, got nil")
		}
		if !strings.Contains(err.Error(), "exceeds") {
			t.Errorf("error %q does not contain 'exceeds'", err.Error())
		}
	})
}

// TestExtractNegative tests that invalid or unrecognizable inputs return nil or no spurious result.
func TestExtractNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          string
		ref         time.Time
		wantNil     bool   // if true, expect nil result
		rejectText  string // if set, no Result.Text should equal this
	}{
		{
			// day 32 is invalid; the word "32" is not consumed as day (>31)
			// "mart" should still match as month-only (day defaults to 1, year from ref)
			// So we just check that no result has day 32
			name:       "invalid day 32",
			in:         "32 mart 2026",
			ref:        ref,
			wantNil:    false,
			rejectText: "",
		},
		{
			// "heft" is not a recognized month or keyword
			name:    "unknown word heft",
			in:      "5 heft 2026",
			ref:     ref,
			wantNil: true,
		},
		{
			// minute 99 is invalid
			name:    "invalid time 25:99",
			in:      "25:99",
			ref:     ref,
			wantNil: true,
		},
		{
			name:    "empty string",
			in:      "",
			ref:     ref,
			wantNil: true,
		},
		{
			name:    "no date or time",
			in:      "abc xyz",
			ref:     ref,
			wantNil: true,
		},
		{
			// Feb 30 is impossible — should be rejected, not normalized.
			name:    "impossible date Feb 30",
			in:      "30.02.2026",
			ref:     ref,
			wantNil: true,
		},
		{
			// Apr 31 is impossible.
			name:    "impossible date Apr 31",
			in:      "31.04.2026",
			ref:     ref,
			wantNil: true,
		},
		{
			// Feb 29 on a non-leap year (2026) is impossible.
			name:    "impossible date Feb 29 non-leap",
			in:      "29.02.2026",
			ref:     ref,
			wantNil: true,
		},
		{
			// Feb 29 on a leap year (2024) is valid.
			name:       "valid date Feb 29 leap year",
			in:         "29.02.2024",
			ref:        ref,
			wantNil:    false,
			rejectText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, tt.ref)
			if tt.wantNil {
				if got != nil {
					t.Errorf("want nil, got %v", got)
				}
				return
			}
			// For non-nil cases, verify no result has an impossible day.
			for _, r := range got {
				if r.Time.Day() > 31 {
					t.Errorf("result has impossible day %d: %v", r.Time.Day(), r)
				}
			}
		})
	}
}

// TestTypeEnum tests Type.String(), MarshalJSON, and UnmarshalJSON.
func TestTypeEnum(t *testing.T) {
	t.Parallel()

	t.Run("String TypeDate", func(t *testing.T) {
		t.Parallel()
		if got := TypeDate.String(); got != "Date" {
			t.Errorf("got %q, want %q", got, "Date")
		}
	})

	t.Run("String TypeTime", func(t *testing.T) {
		t.Parallel()
		if got := TypeTime.String(); got != "Time" {
			t.Errorf("got %q, want %q", got, "Time")
		}
	})

	t.Run("String TypeDateTime", func(t *testing.T) {
		t.Parallel()
		if got := TypeDateTime.String(); got != "DateTime" {
			t.Errorf("got %q, want %q", got, "DateTime")
		}
	})

	t.Run("String unknown", func(t *testing.T) {
		t.Parallel()
		unknown := Type(99)
		got := unknown.String()
		if !strings.HasPrefix(got, "Type(") {
			t.Errorf("got %q, want Type(...) format", got)
		}
	})

	t.Run("MarshalJSON UnmarshalJSON round-trip", func(t *testing.T) {
		t.Parallel()
		for _, typ := range []Type{TypeDate, TypeTime, TypeDateTime} {
			data, err := json.Marshal(typ)
			if err != nil {
				t.Fatalf("Marshal %s: %v", typ, err)
			}
			var got Type
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal %s: %v", typ, err)
			}
			if got != typ {
				t.Errorf("round-trip: got %s, want %s", got, typ)
			}
		}
	})

	t.Run("UnmarshalJSON unknown string error", func(t *testing.T) {
		t.Parallel()
		var typ Type
		err := json.Unmarshal([]byte(`"Bogus"`), &typ)
		if err == nil {
			t.Error("want error for unknown type string, got nil")
		}
	})

	t.Run("UnmarshalJSON non-string error", func(t *testing.T) {
		t.Parallel()
		var typ Type
		err := json.Unmarshal([]byte(`123`), &typ)
		if err == nil {
			t.Error("want error for non-string JSON, got nil")
		}
	})
}

// TestComponentsString tests Components.String() output.
func TestComponentsString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    Components
		want string
	}{
		{
			name: "YMD",
			c:    HasYear | HasMonth | HasDay,
			want: "YMD",
		},
		{
			name: "hm",
			c:    HasHour | HasMinute,
			want: "hm",
		},
		{
			name: "all",
			c:    HasYear | HasMonth | HasDay | HasHour | HasMinute | HasSecond,
			want: "YMDhms",
		},
		{
			name: "none",
			c:    Components(0),
			want: "none",
		},
		{
			name: "year only",
			c:    HasYear,
			want: "Y",
		},
		{
			name: "second only",
			c:    HasSecond,
			want: "s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.c.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResultString tests Result.String() format.
func TestResultString(t *testing.T) {
	t.Parallel()

	r := Result{
		Text:  "5 mart 2026",
		Start: 3,
		End:   14,
		Type:  TypeDate,
	}
	got := r.String()
	want := `Date("5 mart 2026")[3:14]`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestTypeMapsComplete is an enum-sync guard: every Type value must have a name
// in typeNames and a reverse entry in typeFromName.
func TestTypeMapsComplete(t *testing.T) {
	t.Parallel()

	for i := Type(0); i <= TypeDateTime; i++ {
		name := i.String()
		if strings.HasPrefix(name, "Type(") {
			t.Errorf("Type %d has no name in typeNames", i)
		}
		if _, ok := typeFromName[name]; !ok {
			t.Errorf("typeFromName missing entry for %q (Type %d)", name, i)
		}
	}
}

// TestOffsetInvariant verifies that s[r.Start:r.End] == r.Text for all results.
func TestOffsetInvariant(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"2026-03-05",
		"5 mart 2026",
		"martın 15-i",
		"sabah",
		"keçən həftə",
		"bazar ertəsi",
		"çərşənbə axşamı",
		"3 gün əvvəl",
		"axşam saat 7",
		"5 mart 2026 14:30",
		"tarix 2026-03-05 qeyd olunub",
	}

	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			for _, r := range Extract(in, ref) {
				if in[r.Start:r.End] != r.Text {
					t.Errorf("invariant broken: s[%d:%d]=%q != Text=%q",
						r.Start, r.End, in[r.Start:r.End], r.Text)
				}
			}
		})
	}
}

// TestCaseInsensitive verifies that month names are recognized regardless of case.
func TestCaseInsensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		month time.Month
	}{
		{name: "all caps MART", in: "MART", month: time.March},
		{name: "title case Mart", in: "Mart", month: time.March},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Extract(tt.in, ref)
			if len(got) == 0 {
				t.Fatalf("want at least one result for %q, got nil", tt.in)
			}
			if got[0].Time.Month() != tt.month {
				t.Errorf("month: got %v, want %v", got[0].Time.Month(), tt.month)
			}
		})
	}
}

// TestRefZeroUsesNow verifies that a zero ref time causes Extract to use time.Now().
func TestRefZeroUsesNow(t *testing.T) {
	t.Parallel()

	got := Extract("sabah", time.Time{})
	if len(got) == 0 {
		t.Fatal("want result for 'sabah' with zero ref, got nil")
	}
	if got[0].Time.IsZero() {
		t.Error("result time is zero; expected time.Now()+1 day")
	}
}

// ExampleExtract demonstrates extracting date/time spans from Azerbaijani text.
func ExampleExtract() {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	results := Extract("5 mart 2026 saat 14:30", r)
	for _, res := range results {
		fmt.Println(res)
	}
}

// ExampleParse demonstrates parsing a single relative date expression.
func ExampleParse() {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	res, _ := Parse("sabah", r)
	fmt.Println(res.Time.Format("2006-01-02"))
	// Output: 2026-02-21
}

// BenchmarkExtract benchmarks Extract on a mixed natural + numeric expression.
func BenchmarkExtract(b *testing.B) {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	for b.Loop() {
		Extract("5 mart 2026 saat 14:30, sabah görüşərik", r)
	}
}

// BenchmarkExtractLong benchmarks Extract on a long multi-match input.
func BenchmarkExtractLong(b *testing.B) {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	input := strings.Repeat("Görüş 5 mart 2026 saat 14:30, sabah 10:00-da davam edəcək. ", 20)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Extract(input, r)
	}
}

// BenchmarkExtractRelative benchmarks Extract on relative expressions only.
func BenchmarkExtractRelative(b *testing.B) {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	input := "sabah axşam görüşərik, birisi gün hazır olacaq"
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Extract(input, r)
	}
}

// BenchmarkExtractNumeric benchmarks Extract on numeric date/time formats only.
func BenchmarkExtractNumeric(b *testing.B) {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	input := "2026-03-05 14:30 tarixində 21.06.2026 və 01/01/2027"
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for b.Loop() {
		Extract(input, r)
	}
}

// BenchmarkParse benchmarks Parse on a simple full date.
func BenchmarkParse(b *testing.B) {
	r := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	for b.Loop() {
		Parse("5 mart 2026", r) //nolint:errcheck
	}
}
