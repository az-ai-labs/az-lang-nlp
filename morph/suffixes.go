package morph

// fsmState represents a position in the morphotactic chain.
// The engine starts at initial and transitions through states as suffixes are stripped.
type fsmState int

const (
	initial         fsmState = iota // entry point for both noun and verb chains
	afterCopula                     // after copula -dir (near-terminal for nouns)
	afterQuestion                   // after question particle -mi (terminal)
	nounAfterCase                   // after a case suffix (noun chain)
	nounAfterPoss                   // after a possessive suffix (noun chain)
	nounAfterPlural                 // after plural -lar/-ler (noun chain)
	nounAfterDeriv                  // after a derivational suffix (noun chain)
	verbAfterPerson                 // after a person suffix (verb chain)
	verbAfterTense                  // after a tense/mood/participle suffix (verb chain)
	verbAfterNeg                    // after negation -ma/-me (verb chain)
	verbAfterVoice                  // after a voice suffix (verb chain)
	stem                            // terminal state: remaining string is a stem candidate
)

// harmonyKind classifies the vowel harmony pattern of a suffix.
type harmonyKind int

const (
	noHarmony harmonyKind = iota // suffix does not alternate (e.g. -ki, -m, -n, -t)
	backFront                    // two-way: a/e alternation (lar/ler, dan/den)
	fourWay                      // four-way: i/i/u/u alternation (possessive, some verb suffixes)
)

// suffixRule describes one suffix (or group of allomorphs) with its grammatical
// tag, valid source/target states in the FSM, and vowel harmony class.
type suffixRule struct {
	surfaces     []string    // all surface allomorphs, longest first
	surfaceRunes [][]rune    // pre-computed rune slices for surfaces
	tag          MorphTag    // grammatical tag
	fromStates   []fsmState  // valid source states
	toState      fsmState    // target state after match
	harmony      harmonyKind // harmony validation type
}

// suffixRules is the core suffix table for Azerbaijani morphological analysis.
// The engine iterates this table at each FSM state, trying to match the longest
// surface form at the end of the remaining string. Surfaces are ordered longest
// first within each rule for greedy matching.
//
// All surface strings use lowercase Azerbaijani Latin characters with proper
// Unicode: \u0259 (ə), \u0131 (ı), \u00FC (ü), \u00F6 (ö), \u00E7 (ç),
// \u015F (ş), \u011F (ğ).
var suffixRules = []suffixRule{

	// ---------------------------------------------------------------
	// NOUN SUFFIXES
	// ---------------------------------------------------------------

	// Plural: -lar / -l\u0259r (back/front alternation)
	{
		surfaces:   []string{"lar", "l\u0259r"},
		tag:        Plural,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    nounAfterPlural,
		harmony:    backFront,
	},

	// Possessive 1sg: -\u0131m / -im / -um / -\u00FCm (after consonant), -m (after vowel)
	{
		surfaces:   []string{"\u0131m", "im", "um", "\u00FCm", "m"},
		tag:        Poss1Sg,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    fourWay,
	},

	// Possessive 2sg: -\u0131n / -in / -un / -\u00FCn (after consonant), -n (after vowel)
	{
		surfaces:   []string{"\u0131n", "in", "un", "\u00FCn", "n"},
		tag:        Poss2Sg,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    fourWay,
	},

	// Possessive 3sg: -s\u0131 / -si / -su / -s\u00FC (after vowel), -\u0131 / -i / -u / -\u00FC (after consonant)
	{
		surfaces:   []string{"s\u0131", "si", "su", "s\u00FC", "\u0131", "i", "u", "\u00FC"},
		tag:        Poss3Sg,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    fourWay,
	},

	// Possessive 1pl: -\u0131m\u0131z / -imiz / -umuz / -\u00FCm\u00FCz (after consonant),
	//                  -m\u0131z / -miz / -muz / -m\u00FCz (after vowel)
	{
		surfaces: []string{
			"\u0131m\u0131z", "imiz", "umuz", "\u00FCm\u00FCz",
			"m\u0131z", "miz", "muz", "m\u00FCz",
		},
		tag:        Poss1Pl,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    fourWay,
	},

	// Possessive 2pl: -\u0131n\u0131z / -iniz / -unuz / -\u00FCn\u00FCz (after consonant),
	//                  -n\u0131z / -niz / -nuz / -n\u00FCz (after vowel)
	{
		surfaces: []string{
			"\u0131n\u0131z", "iniz", "unuz", "\u00FCn\u00FCz",
			"n\u0131z", "niz", "nuz", "n\u00FCz",
		},
		tag:        Poss2Pl,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    fourWay,
	},

	// Possessive 3pl: -lar\u0131 / -l\u0259ri (back/front alternation)
	{
		surfaces:   []string{"lar\u0131", "l\u0259ri"},
		tag:        Poss3Pl,
		fromStates: []fsmState{initial, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterPoss,
		harmony:    backFront,
	},

	// Case Genitive: -n\u0131n / -nin / -nun / -n\u00FCn (after vowel/poss),
	//                -\u0131n / -in / -un / -\u00FCn (after consonant)
	{
		surfaces: []string{
			"n\u0131n", "nin", "nun", "n\u00FCn",
			"\u0131n", "in", "un", "\u00FCn",
		},
		tag:        CaseGen,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    fourWay,
	},

	// Case Dative: -ya / -y\u0259 (after vowel), -a / -\u0259 (after consonant),
	//              -na / -n\u0259 (after 3sg possessive)
	{
		surfaces:   []string{"ya", "y\u0259", "na", "n\u0259", "a", "\u0259"},
		tag:        CaseDat,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    backFront,
	},

	// Case Accusative: -n\u0131 / -ni / -nu / -n\u00FC (after vowel/poss),
	//                  -\u0131 / -i / -u / -\u00FC (after consonant)
	{
		surfaces: []string{
			"n\u0131", "ni", "nu", "n\u00FC",
			"\u0131", "i", "u", "\u00FC",
		},
		tag:        CaseAcc,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    fourWay,
	},

	// Case Locative: -nda / -nd\u0259 (after poss),
	//                -da / -d\u0259 (standard), -ta / -t\u0259 (d->t after voiceless)
	{
		surfaces:   []string{"nda", "nd\u0259", "da", "d\u0259", "ta", "t\u0259"},
		tag:        CaseLoc,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    backFront,
	},

	// Case Ablative: -ndan / -nd\u0259n (after poss),
	//                -dan / -d\u0259n (standard), -tan / -t\u0259n (d->t after voiceless)
	{
		surfaces:   []string{"ndan", "nd\u0259n", "dan", "d\u0259n", "tan", "t\u0259n"},
		tag:        CaseAbl,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    backFront,
	},

	// Case Instrumental: -la / -l\u0259 (back/front alternation)
	{
		surfaces:   []string{"la", "l\u0259"},
		tag:        CaseIns,
		fromStates: []fsmState{initial, nounAfterPoss, nounAfterPlural, nounAfterDeriv},
		toState:    nounAfterCase,
		harmony:    backFront,
	},

	// Derivational Agent: -\u00E7\u0131 / -\u00E7i / -\u00E7u / -\u00E7\u00FC (forms agent nouns)
	{
		surfaces:   []string{"\u00E7\u0131", "\u00E7i", "\u00E7u", "\u00E7\u00FC"},
		tag:        DerivAgent,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    nounAfterDeriv,
		harmony:    fourWay,
	},

	// Derivational Abstract: -l\u0131q / -lik / -luq / -l\u00FCk (forms abstract nouns)
	{
		surfaces:   []string{"l\u0131q", "lik", "luq", "l\u00FCk"},
		tag:        DerivAbstract,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    nounAfterDeriv,
		harmony:    fourWay,
	},

	// Derivational Privative: -s\u0131z / -siz / -suz / -s\u00FCz (forms privative adjectives)
	{
		surfaces:   []string{"s\u0131z", "siz", "suz", "s\u00FCz"},
		tag:        DerivPriv,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    nounAfterDeriv,
		harmony:    fourWay,
	},

	// Derivational Possessive: -l\u0131 / -li / -lu / -l\u00FC (forms possessive adjectives)
	{
		surfaces:   []string{"l\u0131", "li", "lu", "l\u00FC"},
		tag:        DerivPoss,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    nounAfterDeriv,
		harmony:    fourWay,
	},

	// Derivational Verb: -la\u015F / -l\u0259\u015F (denominal verb, triggers noun-to-verb transition)
	{
		surfaces:   []string{"la\u015F", "l\u0259\u015F"},
		tag:        DerivVerb,
		fromStates: []fsmState{initial, nounAfterDeriv},
		toState:    verbAfterVoice,
		harmony:    backFront,
	},

	// Copula: -d\u0131r / -dir / -dur / -d\u00FCr (standard),
	//         -t\u0131r / -tir / -tur / -t\u00FCr (d->t after voiceless)
	{
		surfaces: []string{
			"d\u0131r", "dir", "dur", "d\u00FCr",
			"t\u0131r", "tir", "tur", "t\u00FCr",
		},
		tag: Copula,
		fromStates: []fsmState{
			initial, nounAfterCase, nounAfterPoss, nounAfterPlural, nounAfterDeriv,
			verbAfterTense,
		},
		toState: afterCopula,
		harmony: fourWay,
	},

	// ---------------------------------------------------------------
	// VERB SUFFIXES
	// ---------------------------------------------------------------

	// Negation: -ma / -m\u0259 (back/front alternation)
	{
		surfaces:   []string{"ma", "m\u0259"},
		tag:        Negation,
		fromStates: []fsmState{initial, verbAfterVoice},
		toState:    verbAfterNeg,
		harmony:    backFront,
	},

	// Voice Passive: -\u0131l / -il / -ul / -\u00FCl
	{
		surfaces:   []string{"\u0131l", "il", "ul", "\u00FCl"},
		tag:        VoicePass,
		fromStates: []fsmState{initial},
		toState:    verbAfterVoice,
		harmony:    fourWay,
	},

	// Voice Reflexive: -\u0131n / -in / -un / -\u00FCn
	{
		surfaces:   []string{"\u0131n", "in", "un", "\u00FCn"},
		tag:        VoiceReflex,
		fromStates: []fsmState{initial},
		toState:    verbAfterVoice,
		harmony:    fourWay,
	},

	// Voice Reciprocal: -\u0131\u015F / -i\u015F / -u\u015F / -\u00FC\u015F
	{
		surfaces:   []string{"\u0131\u015F", "i\u015F", "u\u015F", "\u00FC\u015F"},
		tag:        VoiceRecip,
		fromStates: []fsmState{initial},
		toState:    verbAfterVoice,
		harmony:    fourWay,
	},

	// Voice Causative (short -t): invariant, lexically conditioned
	{
		surfaces:   []string{"t"},
		tag:        VoiceCaus,
		fromStates: []fsmState{initial},
		toState:    verbAfterVoice,
		harmony:    noHarmony,
	},

	// Voice Causative (vowel form): -\u0131r / -ir / -ur / -\u00FCr
	{
		surfaces:   []string{"\u0131r", "ir", "ur", "\u00FCr"},
		tag:        VoiceCaus,
		fromStates: []fsmState{initial},
		toState:    verbAfterVoice,
		harmony:    fourWay,
	},

	// Tense Past Definite: -d\u0131 / -di / -du / -d\u00FC (standard),
	//                       -t\u0131 / -ti / -tu / -t\u00FC (d->t after voiceless)
	{
		surfaces: []string{
			"d\u0131", "di", "du", "d\u00FC",
			"t\u0131", "ti", "tu", "t\u00FC",
		},
		tag:        TensePastDef,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice, verbAfterTense},
		toState:    verbAfterTense,
		harmony:    fourWay,
	},

	// Tense Past Indefinite: -m\u0131\u015F / -mi\u015F / -mu\u015F / -m\u00FC\u015F
	{
		surfaces:   []string{"m\u0131\u015F", "mi\u015F", "mu\u015F", "m\u00FC\u015F"},
		tag:        TensePastIndef,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    fourWay,
	},

	// Tense Present Continuous: -\u0131r / -ir / -ur / -\u00FCr
	{
		surfaces:   []string{"\u0131r", "ir", "ur", "\u00FCr"},
		tag:        TensePresent,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    fourWay,
	},

	// Tense Future: -acaq / -\u0259c\u0259k (back/front alternation)
	{
		surfaces:   []string{"acaq", "\u0259c\u0259k"},
		tag:        TenseFuture,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Tense Aorist: -ar / -\u0259r (back/front alternation)
	{
		surfaces:   []string{"ar", "\u0259r"},
		tag:        TenseAorist,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Mood Obligative: -mal\u0131 / -m\u0259li (back/front alternation)
	{
		surfaces:   []string{"mal\u0131", "m\u0259li"},
		tag:        MoodOblig,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Mood Conditional: -sa / -s\u0259 (back/front alternation)
	{
		surfaces:   []string{"sa", "s\u0259"},
		tag:        MoodCond,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Participle (present): -an / -\u0259n (back/front alternation)
	{
		surfaces:   []string{"an", "\u0259n"},
		tag:        Participle,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Gerund (manner): -araq / -\u0259r\u0259k (back/front alternation)
	{
		surfaces:   []string{"araq", "\u0259r\u0259k"},
		tag:        Gerund,
		fromStates: []fsmState{initial, verbAfterNeg, verbAfterVoice},
		toState:    verbAfterTense,
		harmony:    backFront,
	},

	// Person 1sg: -əm / -am (after consonant-final tense), -m (after vowel-final tense)
	{
		surfaces:   []string{"\u0259m", "am", "m"},
		tag:        Pers1Sg,
		fromStates: []fsmState{verbAfterTense},
		toState:    verbAfterPerson,
		harmony:    backFront,
	},

	// Person 2sg: -sən / -san (after consonant-final tense), -n (after vowel-final tense)
	{
		surfaces:   []string{"s\u0259n", "san", "n"},
		tag:        Pers2Sg,
		fromStates: []fsmState{verbAfterTense},
		toState:    verbAfterPerson,
		harmony:    backFront,
	},

	// Person 1pl: -ıq / -ik / -uq / -ük (after consonant), -q / -k (after vowel)
	{
		surfaces:   []string{"\u0131q", "ik", "uq", "\u00FCk", "q", "k"},
		tag:        Pers1Pl,
		fromStates: []fsmState{verbAfterTense},
		toState:    verbAfterPerson,
		harmony:    fourWay,
	},

	// Person 2pl: -sınız / -siniz / -sunuz / -sünüz (after consonant),
	//             -nız / -niz / -nuz / -nüz (after vowel)
	{
		surfaces: []string{
			"s\u0131n\u0131z", "siniz", "sunuz", "s\u00FCn\u00FCz",
			"n\u0131z", "niz", "nuz", "n\u00FCz",
		},
		tag:        Pers2Pl,
		fromStates: []fsmState{verbAfterTense},
		toState:    verbAfterPerson,
		harmony:    fourWay,
	},

	// Person 3 (plural marker on verbs): -lar / -lər (back/front alternation)
	// Same surface as noun Plural but distinguished by fromStates.
	// Only valid after tense/mood markers; bare stems use noun Plural instead.
	{
		surfaces:   []string{"lar", "l\u0259r"},
		tag:        Pers3,
		fromStates: []fsmState{verbAfterTense},
		toState:    verbAfterPerson,
		harmony:    backFront,
	},

	// Question particle: -m\u0131 / -mi / -mu / -m\u00FC (four-way harmony)
	{
		surfaces:   []string{"m\u0131", "mi", "mu", "m\u00FC"},
		tag:        Question,
		fromStates: []fsmState{initial, verbAfterPerson, afterCopula, verbAfterTense},
		toState:    afterQuestion,
		harmony:    fourWay,
	},
}
