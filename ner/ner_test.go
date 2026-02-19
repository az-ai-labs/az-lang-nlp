package ner

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecognizePhones(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "international format no spaces",
			in:   "Zəng edin: +994501234567",
			want: []Entity{{Text: "+994501234567", Start: 12, End: 25, Type: Phone}},
		},
		{
			name: "international format with spaces",
			in:   "+994 50 123 45 67 nömrəsi",
			want: []Entity{{Text: "+994 50 123 45 67", Start: 0, End: 17, Type: Phone}},
		},
		{
			name: "local format",
			in:   "tel: 0501234567",
			want: []Entity{{Text: "0501234567", Start: 5, End: 15, Type: Phone}},
		},
		{
			name: "local format with spaces",
			in:   "050 123 45 67",
			want: []Entity{{Text: "050 123 45 67", Start: 0, End: 13, Type: Phone}},
		},
		{
			name: "multiple phones",
			in:   "+994501234567 və 0551234567",
			// "və" contains ə (2 bytes), so offset shifts by 1
			want: []Entity{
				{Text: "+994501234567", Start: 0, End: 13, Type: Phone},
				{Text: "0551234567", Start: 18, End: 28, Type: Phone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeEmails(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "simple email",
			in:   "info@example.com",
			want: []Entity{{Text: "info@example.com", Start: 0, End: 16, Type: Email}},
		},
		{
			name: "email in text",
			in:   "Yazın: user.name+tag@mail.gov.az məktub",
			// "Yazın" has ı (2 bytes) → prefix = 8 bytes
			want: []Entity{{Text: "user.name+tag@mail.gov.az", Start: 8, End: 33, Type: Email}},
		},
		{
			name: "multiple emails",
			in:   "a@b.co və c@d.az",
			// "və" has ə (2 bytes) → second email at byte 11
			want: []Entity{
				{Text: "a@b.co", Start: 0, End: 6, Type: Email},
				{Text: "c@d.az", Start: 11, End: 17, Type: Email},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeFIN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "labeled with colon",
			in:   "FIN: 5ARPXK2",
			want: []Entity{{Text: "5ARPXK2", Start: 5, End: 12, Type: FIN, Labeled: true}},
		},
		{
			name: "labeled with space",
			in:   "FIN 5ARPXK2 yazılıb",
			want: []Entity{{Text: "5ARPXK2", Start: 4, End: 11, Type: FIN, Labeled: true}},
		},
		{
			name: "labeled case insensitive",
			in:   "fin: 5ARPXK2",
			want: []Entity{{Text: "5ARPXK2", Start: 5, End: 12, Type: FIN, Labeled: true}},
		},
		{
			name: "bare FIN",
			in:   "sənəddə 5ARPXK2 var",
			// "sənəddə" has 3x ə (2 bytes each) → prefix = 11 bytes
			want: []Entity{{Text: "5ARPXK2", Start: 11, End: 18, Type: FIN}},
		},
		{
			name: "excludes I and O",
			in:   "IOIOIOI",
			want: nil,
		},
		{
			name: "bare pure letters not matched",
			in:   "PRODUCT VERSION SECTION",
			want: nil,
		},
		{
			name: "bare pure digits not matched",
			in:   "1234567",
			want: nil,
		},
		{
			name: "bare mixed alphanumeric matched",
			in:   "код ABC1234",
			want: []Entity{{Text: "ABC1234", Start: 7, End: 14, Type: FIN}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeVOEN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "labeled VOEN",
			in:   "VOEN: 1234567890",
			want: []Entity{{Text: "1234567890", Start: 6, End: 16, Type: VOEN, Labeled: true}},
		},
		{
			name: "labeled with O-umlaut",
			in:   "VÖEN: 1234567890",
			want: []Entity{{Text: "1234567890", Start: 7, End: 17, Type: VOEN, Labeled: true}},
		},
		{
			name: "bare digits not matched as VOEN",
			in:   "nömrə 1234567890 yazılıb",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeIBAN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "valid IBAN",
			in:   "Hesab: AZ21NABZ00000000137010001944",
			want: []Entity{{Text: "AZ21NABZ00000000137010001944", Start: 7, End: 35, Type: IBAN}},
		},
		{
			name: "IBAN in text",
			in:   "Köçürmə AZ77AIIB38060019441234567890 hesabına",
			// "Köçürmə" has ö(2)+ç(2)+ü(2)+ə(2) → prefix = 12 bytes
			want: []Entity{{Text: "AZ77AIIB38060019441234567890", Start: 12, End: 40, Type: IBAN}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeLicensePlate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "standard plate",
			in:   "maşın 10-AA-123",
			// "maşın" has ş(2)+ı(2) → prefix = 8 bytes
			want: []Entity{{Text: "10-AA-123", Start: 8, End: 17, Type: LicensePlate}},
		},
		{
			name: "plate in sentence",
			in:   "90-BZ-456 nömrəli avtomobil",
			want: []Entity{{Text: "90-BZ-456", Start: 0, End: 9, Type: LicensePlate}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Entity
	}{
		{
			name: "https URL",
			in:   "sayt: https://gov.az/services",
			want: []Entity{{Text: "https://gov.az/services", Start: 6, End: 29, Type: URL}},
		},
		{
			name: "http URL",
			in:   "bax http://example.com",
			want: []Entity{{Text: "http://example.com", Start: 4, End: 22, Type: URL}},
		},
		{
			name: "URL with trailing punctuation trimmed",
			in:   "Bura bax: https://example.com.",
			want: []Entity{{Text: "https://example.com", Start: 10, End: 29, Type: URL}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Recognize(tt.in)
			compareEntities(t, tt.want, got)
		})
	}
}

func TestRecognizeMixed(t *testing.T) {
	in := "FIN: 5ARPXK2, tel +994501234567, email info@gov.az"
	got := Recognize(in)

	if len(got) != 3 {
		t.Fatalf("want 3 entities, got %d: %v", len(got), got)
	}

	wantTypes := []EntityType{FIN, Phone, Email}
	for i, e := range got {
		if e.Type != wantTypes[i] {
			t.Errorf("entity[%d]: want type %s, got %s", i, wantTypes[i], e.Type)
		}
		if in[e.Start:e.End] != e.Text {
			t.Errorf("entity[%d]: invariant broken: s[%d:%d]=%q != Text=%q",
				i, e.Start, e.End, in[e.Start:e.End], e.Text)
		}
	}
}

func TestRecognizeEmpty(t *testing.T) {
	if got := Recognize(""); got != nil {
		t.Errorf("Recognize empty: want nil, got %v", got)
	}
}

func TestRecognizeNoEntities(t *testing.T) {
	if got := Recognize("Bu sadə cümlədir."); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	in := "FIN: 5ARPXK2, tel +994501234567, VOEN: 1234567890, email info@gov.az, IBAN AZ21NABZ00000000137010001944, plate 10-AA-123, url https://gov.az"

	assertStrings(t, "Phones", Phones(in), []string{"+994501234567"})
	assertStrings(t, "Emails", Emails(in), []string{"info@gov.az"})
	assertStrings(t, "FINs", FINs(in), []string{"5ARPXK2"})
	assertStrings(t, "VOENs", VOENs(in), []string{"1234567890"})
	assertStrings(t, "IBANs", IBANs(in), []string{"AZ21NABZ00000000137010001944"})
	assertStrings(t, "LicensePlates", LicensePlates(in), []string{"10-AA-123"})
	assertStrings(t, "URLs", URLs(in), []string{"https://gov.az"})
}

func TestEntityTypeJSON(t *testing.T) {
	e := Entity{Text: "test", Start: 0, End: 4, Type: Phone, Labeled: false}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Entity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Type != Phone {
		t.Errorf("round-trip: want Phone, got %s", decoded.Type)
	}
}

func TestEntityTypeStringUnknown(t *testing.T) {
	var et EntityType = 99
	got := et.String()
	want := "EntityType(99)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEntityTypeUnmarshalUnknown(t *testing.T) {
	var et EntityType
	err := et.UnmarshalJSON([]byte(`"Bogus"`))
	if err == nil {
		t.Error("want error for unknown type, got nil")
	}
}

func TestOverlapResolution(t *testing.T) {
	// When a VOEN bare match overlaps with a phone number, the phone wins (longer).
	in := "+994501234567"
	got := Recognize(in)
	if len(got) != 1 {
		t.Fatalf("want 1 entity, got %d: %v", len(got), got)
	}
	if got[0].Type != Phone {
		t.Errorf("want Phone, got %s", got[0].Type)
	}
}

func TestOffsetInvariant(t *testing.T) {
	inputs := []string{
		"FIN: 5ARPXK2",
		"+994501234567",
		"info@gov.az",
		"AZ21NABZ00000000137010001944",
		"10-AA-123",
		"https://gov.az",
		"VOEN: 1234567890",
	}
	for _, in := range inputs {
		for _, e := range Recognize(in) {
			if in[e.Start:e.End] != e.Text {
				t.Errorf("invariant broken for %s: s[%d:%d]=%q != %q",
					e.Type, e.Start, e.End, in[e.Start:e.End], e.Text)
			}
		}
	}
}

func TestEntityString(t *testing.T) {
	e := Entity{Text: "+994501234567", Start: 0, End: 13, Type: Phone}
	got := e.String()
	want := `Phone("+994501234567")[0:13]`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	e = Entity{Text: "5ARPXK2", Start: 5, End: 12, Type: FIN, Labeled: true}
	got = e.String()
	want = `FIN("5ARPXK2")[5:12,labeled]`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEntityTypeUnmarshalNonString(t *testing.T) {
	var et EntityType
	err := et.UnmarshalJSON([]byte("123"))
	if err == nil {
		t.Error("want error for non-string JSON, got nil")
	}
}

func TestEntityTypeMapsComplete(t *testing.T) {
	for i := EntityType(0); i <= URL; i++ {
		name := i.String()
		if strings.HasPrefix(name, "EntityType(") {
			t.Errorf("EntityType %d has no name in entityTypeNames", i)
		}
		if _, ok := entityTypeFromName[name]; !ok {
			t.Errorf("entityTypeFromName missing entry for %q", name)
		}
	}
}

// compareEntities compares two entity slices with helpful error messages.
func compareEntities(t *testing.T, want, got []Entity) {
	t.Helper()

	if len(want) == 0 && len(got) == 0 {
		return
	}
	if len(want) == 0 && got == nil {
		return
	}

	if len(got) != len(want) {
		t.Errorf("got %d entities, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
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
		if got[i].Labeled != want[i].Labeled {
			t.Errorf("[%d] Labeled: got %v, want %v", i, got[i].Labeled, want[i].Labeled)
		}
	}
}

// assertStrings compares string slices.
func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %d items %v, want %d items %v", label, len(got), got, len(want), want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d]: got %q, want %q", label, i, got[i], want[i])
		}
	}
}
