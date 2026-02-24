// Package data embeds all dictionary and frequency data files.
package data

import _ "embed"

//go:embed dict.txt
var MorphDict []byte

//go:embed spell_freq.txt
var SpellFreq []byte

//go:embed keywords_freq.txt
var KeywordsFreq []byte

//go:embed lexicon.txt
var SentimentLexicon string
