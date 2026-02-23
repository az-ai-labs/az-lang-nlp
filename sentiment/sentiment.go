// Package sentiment performs lexicon-based sentiment analysis of Azerbaijani text.
//
// The analyzer tokenizes input, normalizes diacritics, stems each word, and
// looks up the stem in an embedded sentiment lexicon. Word scores are averaged
// to produce an aggregate sentiment score.
//
// Three convenience functions are provided:
//
//   - Analyze returns a full Result with score, polarity, and word counts.
//   - Score returns the aggregate score (-1.0 to +1.0).
//   - IsPositive returns true when overall sentiment is positive.
//
// v1 limitations:
//   - No negation handling ("yaxşı deyil" scores as positive).
//   - No intensifier/diminisher support.
//   - Sarcasm is not detected.
//
// All functions are safe for concurrent use by multiple goroutines.
package sentiment

import (
	"encoding/json"
	"fmt"
)

// maxInputBytes is the maximum input size. Inputs exceeding this return a zero Result.
const maxInputBytes = 1 << 20 // 1 MiB

// Sentiment represents the sentiment polarity.
type Sentiment int

const (
	Negative Sentiment = -1
	Neutral  Sentiment = 0
	Positive Sentiment = 1
)

// sentimentNames maps Sentiment values to their string names.
var sentimentNames = map[Sentiment]string{
	Negative: "Negative",
	Neutral:  "Neutral",
	Positive: "Positive",
}

// sentimentFromName maps string names back to Sentiment values.
var sentimentFromName = map[string]Sentiment{
	"Negative": Negative,
	"Neutral":  Neutral,
	"Positive": Positive,
}

// String returns the name of the sentiment polarity.
func (s Sentiment) String() string {
	if name, ok := sentimentNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Sentiment(%d)", int(s))
}

// MarshalJSON encodes the sentiment as a JSON string.
func (s Sentiment) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON decodes a JSON string into a Sentiment.
func (s *Sentiment) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	v, ok := sentimentFromName[str]
	if !ok {
		return fmt.Errorf("sentiment: unknown sentiment: %q", str)
	}
	*s = v
	return nil
}

// Result holds the sentiment analysis output.
type Result struct {
	Sentiment Sentiment `json:"sentiment"`
	Score     float64   `json:"score"`    // -1.0 to +1.0
	Positive  int       `json:"positive"` // count of positive words
	Negative  int       `json:"negative"` // count of negative words
	Total     int       `json:"total"`    // total analyzed words
}

// String returns a debug representation of the result.
func (r Result) String() string {
	return fmt.Sprintf("%s(score=%.2f, pos=%d, neg=%d, total=%d)",
		r.Sentiment, r.Score, r.Positive, r.Negative, r.Total)
}

// Analyze returns detailed sentiment analysis of text.
// Returns a zero Result for empty or oversized input.
func Analyze(text string) Result {
	if text == "" || len(text) > maxInputBytes {
		return Result{}
	}
	return analyze(text)
}

// Score returns the aggregate sentiment score (-1.0 to +1.0).
func Score(text string) float64 {
	return Analyze(text).Score
}

// IsPositive returns true if overall sentiment is positive.
func IsPositive(text string) bool {
	return Analyze(text).Sentiment == Positive
}
