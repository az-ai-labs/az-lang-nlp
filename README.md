# BarsNLP

NLP toolkit for Azerbaijani language. Pure Go, zero dependencies.

## Install

```
go get github.com/BarsNLP/barsnlp
```

## Transliteration

Convert Azerbaijani text between Latin and Cyrillic scripts.

```go
package main

import (
	"fmt"
	"github.com/BarsNLP/barsnlp/translit"
)

func main() {
	fmt.Println(translit.CyrillicToLatin("Азәрбајҹан"))
	// Azərbaycan

	fmt.Println(translit.LatinToCyrillic("Həyat gözəldir"))
	// Һәјат ҝөзәлдир
}
```

Contextual rules handle Cyrillic Г/г disambiguation automatically. Non-Azerbaijani characters (digits, punctuation, emoji) pass through unchanged.

All functions are safe for concurrent use.

## Tokenizer

Split Azerbaijani text into words and sentences with byte offsets.

```go
package main

import (
	"fmt"
	"github.com/BarsNLP/barsnlp/tokenizer"
)

func main() {
	// Word tokenization
	fmt.Println(tokenizer.Words("Bakı'nın küçələri gözəldir."))
	// [Bakı'nın küçələri gözəldir]

	// Structured tokens with byte offsets
	for _, t := range tokenizer.WordTokens("Salam, dünya!") {
		fmt.Printf("%s: %q\n", t.Type, t.Text)
	}
	// Word: "Salam"
	// Punctuation: ","
	// Space: " "
	// Word: "dünya"
	// Punctuation: "!"

	// Sentence splitting
	fmt.Println(tokenizer.Sentences("Birinci cümlə. İkinci cümlə."))
	// [Birinci cümlə.  İkinci cümlə.]
}
```

Handles URLs, emails, Azerbaijani abbreviations (Prof., Az.R.), thousand-separator dots (1.000.000), decimal commas (3,14), hyphens (sosial-iqtisadi), and apostrophe suffixes (Bakı'nın).

All functions are safe for concurrent use.

## Morphological Analysis

Decompose inflected Azerbaijani words into stem and suffix chain.

```go
package main

import (
	"fmt"
	"github.com/BarsNLP/barsnlp/morph"
)

func main() {
	// Extract stem from inflected word
	fmt.Println(morph.Stem("kitablarımızdan"))
	// kitab

	// Full morphological analysis
	for _, a := range morph.Analyze("kitablar") {
		fmt.Println(a)
	}
	// kitab[Plural:lar]
	// kitabl[TenseAorist:ar]
	// kitablar

	// Batch stemming (pairs with tokenizer.Words)
	fmt.Println(morph.Stems([]string{"kitablarımızdan", "evlərdə", "gəlmişdir"}))
	// [kitab ev gəl]
}
```

Uses a table-driven morphotactic state machine with backtracking. Validates vowel harmony, consonant assimilation, and suffix ordering. Includes an embedded dictionary (~12K stems from Wiktionary) for stem validation.

All functions are safe for concurrent use.

## Number-to-Text

Convert between numbers and Azerbaijani text representations.

```go
package main

import (
	"fmt"
	"github.com/BarsNLP/barsnlp/numtext"
)

func main() {
	// Cardinal number
	fmt.Println(numtext.Convert(123))
	// yüz iyirmi üç

	// Ordinal number with vowel-harmony suffix
	fmt.Println(numtext.ConvertOrdinal(5))
	// beşinci

	// Decimal: math mode
	fmt.Println(numtext.ConvertFloat("3.14", numtext.MathMode))
	// üç tam yüzdə on dörd

	// Decimal: digit-by-digit mode
	fmt.Println(numtext.ConvertFloat("3.14", numtext.DigitMode))
	// üç vergül bir dörd

	// Parse text back to number
	n, _ := numtext.Parse("iki milyon üç yüz min doxsan beş")
	fmt.Println(n)
	// 2300095
}
```

Supports integers up to ±10^18, negative numbers, ordinals, and decimals with dot or comma separator. Parse is case-insensitive and accepts both canonical ("yüz") and explicit ("bir yüz") forms.

All functions are safe for concurrent use.

## Named Entity Recognition

Extract structured entities from Azerbaijani text: FIN, VOEN, phone numbers, emails, IBANs, license plates, and URLs.

```go
package main

import (
	"fmt"
	"github.com/BarsNLP/barsnlp/ner"
)

func main() {
	// Extract all entities with byte offsets
	for _, e := range ner.Recognize("FIN: 5ARPXK2, tel +994501234567") {
		fmt.Printf("%s: %q (labeled=%v)\n", e.Type, e.Text, e.Labeled)
	}
	// FIN: "5ARPXK2" (labeled=true)
	// Phone: "+994501234567" (labeled=false)

	// Convenience functions return []string
	fmt.Println(ner.Phones("+994501234567 və 0551234567"))
	// [+994501234567 0551234567]

	fmt.Println(ner.Emails("info@gov.az"))
	// [info@gov.az]

	fmt.Println(ner.IBANs("AZ21NABZ00000000137010001944"))
	// [AZ21NABZ00000000137010001944]
}
```

FIN and VOEN patterns are ambiguous in isolation. When preceded by a keyword (e.g. "FIN:", "VOEN:"), `Entity.Labeled` is true, indicating higher confidence. Overlapping entities are resolved by preferring longer matches.

All functions are safe for concurrent use.

## License

MIT
