// Package morph performs morphological analysis on Azerbaijani words,
// decomposing inflected forms into stem + suffix chain.
//
// The package provides two API layers:
//
//   - Structured: Analyze returns []Analysis with full morpheme breakdown
//     and grammatical tags for each suffix.
//
//   - Convenience: Stem returns just the base form string, and Stems
//     is a batch wrapper for use with tokenizer.Words().
//
// The analyzer uses a table-driven morphotactic state machine with
// backtracking. It validates vowel harmony, consonant assimilation,
// and suffix ordering constraints without requiring a dictionary.
//
// All functions are safe for concurrent use by multiple goroutines.
//
// Known limitations (v1.0):
//
//   - Dictionary lookup is soft (ranking only). Cannot resolve all homographic stems.
//   - The -t causative suffix may produce false positives.
//   - k/q restoration at morpheme boundaries is best-effort.
//   - Reciprocal -ış/-iş/-uş/-üş may over-stem verb roots like danış-.
//   - Composite tenses (pluperfect -mışdı-, etc.) are not supported.
//   - Vowel dropping (ogul -> oglu) is not handled without a dictionary.
//   - Azerbaijani Latin only. Use translit.CyrillicToLatin for Cyrillic input.
//   - Input is expected in NFC Unicode normalization form.
package morph

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	nounBase   = 100
	derivBase  = 200
	copBase    = 250
	vvoiceBase = 300
	vnegBase   = 310
	vtenseBase = 320
	vmoodBase  = 330
	vpartBase  = 340
	vpersBase  = 350
	questBase  = 400
)

// MorphTag classifies morphemes by grammatical category.
type MorphTag int

const (
	Plural  MorphTag = nounBase + iota // -lar/-ler
	Poss1Sg                            // -m, -im/-im/-um/-um
	Poss2Sg                            // -n, -in/-in/-un/-un
	Poss3Sg                            // -i/-i/-u/-u, -si/-si/-su/-su
	Poss1Pl                            // -miz/-miz/-muz/-muz
	Poss2Pl                            // -niz/-niz/-nuz/-nuz
	Poss3Pl                            // -lari/-leri
)

const (
	CaseGen MorphTag = nounBase + 10 + iota // -in/-in/-un/-un
	CaseDat                                 // -a/-e, -ya/-ye
	CaseAcc                                 // -i/-i/-u/-u, -ni/-ni/-nu/-nu
	CaseLoc                                 // -da/-de, -ta/-te
	CaseAbl                                 // -dan/-den, -tan/-ten
	CaseIns                                 // -la/-le
)

const (
	DerivAgent    MorphTag = derivBase + iota // -ci/-ci, -cu/-cu (agent noun)
	DerivAbstract                             // -liq/-lik, -luq/-luk (abstract noun)
	DerivPriv                                 // -siz/-siz (privative)
	DerivPoss                                 // -li/-li (possessive adjective)
	DerivVerb                                 // -la/-le (denominal verb)
)

const (
	Copula MorphTag = copBase // -dir/-dir, -dur/-dur (copula)
)

const (
	VoicePass   MorphTag = vvoiceBase + iota // -il/-il, -ul/-ul, -n (passive)
	VoiceReflex                              // -in/-in, -un/-un (reflexive)
	VoiceRecip                               // -is/-is, -us/-us (reciprocal)
	VoiceCaus                                // -t/-dir/-dir (causative)
)

const (
	Negation MorphTag = vnegBase // -ma/-me (verbal negation)
)

const (
	TensePastDef   MorphTag = vtenseBase + iota // -di/-di, -du/-du
	TensePastIndef                              // -mis/-mis, -mus/-mus
	TensePresent                                // -ir/-ir, -ur/-ur, -ar/-er
	TenseFuture                                 // -acaq/-ecek
	TenseAorist                                 // -ar/-er (aorist/habitual)
)

const (
	MoodOblig MorphTag = vmoodBase + iota // -mali/-meli (obligative)
	MoodCond                              // -sa/-se (conditional)
	MoodImper                             // imperative (unmarked 2sg, -in 2pl)
)

const (
	Participle    MorphTag = vpartBase + iota // -an/-en (present participle)
	ParticipleAdj                             // -mish/-mish (past participle adjective)
	Gerund                                    // -maq/-mek (verbal noun/infinitive)
)

const (
	Pers1Sg MorphTag = vpersBase + iota // -m (1sg), -am/-em (1sg after vowel)
	Pers2Sg                             // -san/-sen (2sg)
	Pers1Pl                             // -iq/-ik (1pl)
	Pers2Pl                             // -siniz/-siniz (2pl)
	Pers3                               // unmarked or -dir/-dir
)

const (
	Question MorphTag = questBase // -mi/-mi, -mu/-mu (question particle)
)

// morphTagNames maps MorphTag values to their string names.
var morphTagNames = map[MorphTag]string{
	Plural:  "Plural",
	Poss1Sg: "Poss1Sg",
	Poss2Sg: "Poss2Sg",
	Poss3Sg: "Poss3Sg",
	Poss1Pl: "Poss1Pl",
	Poss2Pl: "Poss2Pl",
	Poss3Pl: "Poss3Pl",

	CaseGen: "CaseGen",
	CaseDat: "CaseDat",
	CaseAcc: "CaseAcc",
	CaseLoc: "CaseLoc",
	CaseAbl: "CaseAbl",
	CaseIns: "CaseIns",

	DerivAgent:    "DerivAgent",
	DerivAbstract: "DerivAbstract",
	DerivPriv:     "DerivPriv",
	DerivPoss:     "DerivPoss",
	DerivVerb:     "DerivVerb",

	Copula: "Copula",

	VoicePass:   "VoicePass",
	VoiceReflex: "VoiceReflex",
	VoiceRecip:  "VoiceRecip",
	VoiceCaus:   "VoiceCaus",

	Negation: "Negation",

	TensePastDef:   "TensePastDef",
	TensePastIndef: "TensePastIndef",
	TensePresent:   "TensePresent",
	TenseFuture:    "TenseFuture",
	TenseAorist:    "TenseAorist",

	MoodOblig: "MoodOblig",
	MoodCond:  "MoodCond",
	MoodImper: "MoodImper",

	Participle:    "Participle",
	ParticipleAdj: "ParticipleAdj",
	Gerund:        "Gerund",

	Pers1Sg: "Pers1Sg",
	Pers2Sg: "Pers2Sg",
	Pers1Pl: "Pers1Pl",
	Pers2Pl: "Pers2Pl",
	Pers3:   "Pers3",

	Question: "Question",
}

// morphTagFromName maps string names back to MorphTag values.
var morphTagFromName = map[string]MorphTag{
	"Plural":  Plural,
	"Poss1Sg": Poss1Sg,
	"Poss2Sg": Poss2Sg,
	"Poss3Sg": Poss3Sg,
	"Poss1Pl": Poss1Pl,
	"Poss2Pl": Poss2Pl,
	"Poss3Pl": Poss3Pl,

	"CaseGen": CaseGen,
	"CaseDat": CaseDat,
	"CaseAcc": CaseAcc,
	"CaseLoc": CaseLoc,
	"CaseAbl": CaseAbl,
	"CaseIns": CaseIns,

	"DerivAgent":    DerivAgent,
	"DerivAbstract": DerivAbstract,
	"DerivPriv":     DerivPriv,
	"DerivPoss":     DerivPoss,
	"DerivVerb":     DerivVerb,

	"Copula": Copula,

	"VoicePass":   VoicePass,
	"VoiceReflex": VoiceReflex,
	"VoiceRecip":  VoiceRecip,
	"VoiceCaus":   VoiceCaus,

	"Negation": Negation,

	"TensePastDef":   TensePastDef,
	"TensePastIndef": TensePastIndef,
	"TensePresent":   TensePresent,
	"TenseFuture":    TenseFuture,
	"TenseAorist":    TenseAorist,

	"MoodOblig": MoodOblig,
	"MoodCond":  MoodCond,
	"MoodImper": MoodImper,

	"Participle":    Participle,
	"ParticipleAdj": ParticipleAdj,
	"Gerund":        Gerund,

	"Pers1Sg": Pers1Sg,
	"Pers2Sg": Pers2Sg,
	"Pers1Pl": Pers1Pl,
	"Pers2Pl": Pers2Pl,
	"Pers3":   Pers3,

	"Question": Question,
}

// String returns the name of the morpheme tag.
func (t MorphTag) String() string {
	if name, ok := morphTagNames[t]; ok {
		return name
	}
	return fmt.Sprintf("MorphTag(%d)", int(t))
}

// MarshalJSON encodes the morph tag as a JSON string (e.g. "Plural").
func (t MorphTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON decodes a JSON string (e.g. "Plural") into a MorphTag.
func (t *MorphTag) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tag, ok := morphTagFromName[s]
	if !ok {
		return fmt.Errorf("unknown morph tag: %q", s)
	}
	*t = tag
	return nil
}

// Morpheme represents a single morpheme with its surface form and grammatical tag.
type Morpheme struct {
	Surface string   `json:"surface"` // The morpheme text (e.g. "lar")
	Tag     MorphTag `json:"tag"`     // Classification of the morpheme
}

// Analysis represents a complete morphological analysis with stem and suffix chain.
type Analysis struct {
	Stem      string     `json:"stem"`      // The base form
	Morphemes []Morpheme `json:"morphemes"` // Ordered list of suffixes
}

// String returns a debug representation, e.g. kitab[Plural:lar|Poss1Pl:imiz|CaseAbl:dan].
func (a Analysis) String() string {
	if len(a.Morphemes) == 0 {
		return a.Stem
	}
	var sb strings.Builder
	sb.WriteString(a.Stem)
	sb.WriteByte('[')
	for i, m := range a.Morphemes {
		if i > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(m.Tag.String())
		sb.WriteByte(':')
		sb.WriteString(m.Surface)
	}
	sb.WriteByte(']')
	return sb.String()
}

const maxWordBytes = 256

// Stem extracts the stem (base form) from an inflected Azerbaijani word.
// Returns the original word if it cannot be analyzed or exceeds maxWordBytes.
// Handles hyphens by stemming each part separately and rejoining.
// Handles apostrophes by returning the part before the first apostrophe.
func Stem(word string) string {
	if word == "" || len(word) > maxWordBytes {
		return word
	}

	// Handle hyphens: split, stem each part, rejoin
	if idx := strings.Index(word, "-"); idx > 0 && idx < len(word)-1 {
		parts := strings.Split(word, "-")
		for i, p := range parts {
			parts[i] = Stem(p)
		}
		return strings.Join(parts, "-")
	}

	// Handle apostrophes: split at first apostrophe, return pre-apostrophe part
	for i, r := range word {
		if r == '\'' || r == '\u2019' || r == '\u02BC' {
			if i > 0 {
				return word[:i]
			}
			return word // apostrophe at start, return unchanged
		}
	}

	results := Analyze(word)

	// Three-pass dictionary-aware stem selection.
	wordKnown := isKnownStem(toLower(word))
	// Pass 1: prefer analysis with morphemes AND known dictionary stem,
	// but skip when the whole word is also known (avoids stripping real
	// stems like ana->an where both are dictionary entries).
	if !wordKnown {
		for _, a := range results {
			if len(a.Morphemes) > 0 && isKnownStem(toLower(a.Stem)) {
				return a.Stem
			}
		}
	}
	// Pass 2: if the whole word is a known dictionary stem, don't strip.
	if wordKnown {
		return word
	}
	// Pass 3: fall back to any analysis with morphemes (pre-dictionary behavior).
	for _, a := range results {
		if len(a.Morphemes) > 0 {
			return a.Stem
		}
	}
	return word
}

// Analyze performs morphological analysis on an Azerbaijani word.
// Returns all possible analyses (stems with suffix chains).
// Returns nil for empty input.
// Returns a single-element slice with the original word as stem if analysis fails.
func Analyze(word string) []Analysis {
	if word == "" {
		return nil
	}
	if len(word) > maxWordBytes {
		return []Analysis{{Stem: word}}
	}

	results := analyze(word)
	// Always include bare-stem interpretation.
	if isValidStem(toLower(word)) {
		results = append(results, Analysis{Stem: word})
	}
	if len(results) == 0 {
		return []Analysis{{Stem: word}}
	}
	return results
}

// Stems extracts stems from a slice of words.
// Designed to be used with tokenizer.Words().
// Returns nil if the input is nil.
func Stems(words []string) []string {
	if words == nil {
		return nil
	}
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = Stem(w)
	}
	return out
}
