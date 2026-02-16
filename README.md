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

## License

MIT
