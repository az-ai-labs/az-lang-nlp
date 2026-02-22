package keywords

// stopwords contains Azerbaijani function words and high-frequency auxiliaries
// that carry no discriminative value for keyword extraction.
var stopwords = map[string]struct{}{
	// Personal pronouns
	"mən": {}, "sən": {}, "o": {}, "biz": {}, "siz": {}, "onlar": {},
	// Demonstrative pronouns
	"bu": {}, "belə": {}, "elə": {}, "həmin": {},
	// Interrogative pronouns
	"kim": {}, "nə": {}, "hara": {}, "hansı": {}, "neçə": {},
	// Reflexive pronouns
	"öz": {},
	// Indefinite / negative pronouns
	"kimsə": {}, "nəsə": {}, "heç": {},
	// Conjunctions
	"və": {}, "ilə": {}, "da": {}, "də": {}, "həm": {},
	"ya": {}, "ancaq": {}, "lakin": {}, "amma": {}, "ki": {}, "çünki": {},
	// Postpositions
	"üçün": {}, "qarşı": {}, "doğru": {}, "qədər": {}, "dək": {},
	"görə": {}, "sonra": {}, "əvvəl": {}, "başqa": {},
	// Particles
	"yalnız": {}, "lap": {}, "ən": {},
	// Modal words
	"əlbəttə": {}, "bəlkə": {}, "yəqin": {}, "əslində": {},
	"həqiqətən": {}, "deməli": {}, "guya": {},
	// Auxiliaries
	"var": {}, "yox": {}, "deyil": {},
	// High-frequency verb stems
	"ol": {}, "et": {}, "ed": {}, "de": {}, "get": {}, "gəl": {}, "ver": {}, "al": {}, "qoy": {},
}

func isStopword(stem string) bool {
	_, ok := stopwords[stem]
	return ok
}
