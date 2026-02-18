package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BarsNLP/barsnlp/tokenizer"
	"github.com/BarsNLP/barsnlp/translit"
)

const (
	chunkSize      = 4 << 20 // 4 MB per read chunk
	maxWorkers     = 4
	expectedArgs   = 2
	bytesToMBShift = 20
)

type fileRatio struct {
	path       string
	sentences  int
	paragraphs int
	ratio      float64
}

type Stats struct {
	mu                sync.Mutex
	filesScanned      int
	totalBytes        int64
	reconOK           int
	reconFail         int
	sentenceOutliers  int
	cyrillicFiles     int
	translitReconOK   int
	translitReconFail int
	tokenTypeCounts   map[tokenizer.TokenType]int
	fileRatios        []fileRatio
}

type fileState struct {
	path            string
	tokenCounts     map[tokenizer.TokenType]int
	totalBytes      int64
	reconFailed     bool
	reconFailLogged bool
	sentences       int
	paragraphs      int
	hasCyrillic     bool
	translitFailed  bool
	translitLogged  bool
}

func main() {
	if len(os.Args) != expectedArgs {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		os.Exit(1)
	}

	dirPath := os.Args[1]
	stats := &Stats{
		tokenTypeCounts: make(map[tokenizer.TokenType]int),
	}

	var filePaths []string
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".txt") {
			return nil
		}
		filePaths = append(filePaths, path)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Found %d files to process\n", len(filePaths))
	start := time.Now()

	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, path := range filePaths {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			processFile(p, stats)
		}(path)
	}

	wg.Wait()

	flagSentenceOutliers(stats)

	fmt.Fprintf(os.Stderr, "\nCompleted in %s\n\n", time.Since(start).Round(time.Millisecond))
	printStats(stats)
}

func processFile(path string, stats *Stats) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", path, err)
		return
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stat %s: %v\n", path, err)
		return
	}
	fileSize := info.Size()
	fmt.Fprintf(os.Stderr, "START %s (%d MB)\n", path, fileSize>>bytesToMBShift)
	fileStart := time.Now()

	state := &fileState{
		path:        path,
		tokenCounts: make(map[tokenizer.TokenType]int),
	}

	buf := make([]byte, chunkSize)
	var leftover []byte

	for {
		n, err := f.Read(buf)
		if n > 0 {
			leftover = append(leftover, buf[:n]...)
			chunk := leftover

			if err == nil {
				if idx := bytes.LastIndexByte(chunk, '\n'); idx > 0 {
					leftover = make([]byte, len(chunk)-idx-1)
					copy(leftover, chunk[idx+1:])
					chunk = chunk[:idx+1]
				} else {
					leftover = chunk
					continue
				}
			} else {
				leftover = nil
			}

			state.processChunk(chunk)
		}

		if err != nil {
			break
		}
	}

	if len(leftover) > 0 {
		state.processChunk(leftover)
	}

	state.paragraphs++

	fmt.Fprintf(os.Stderr, "DONE  %s in %s (%d MB processed)\n",
		filepath.Base(path), time.Since(fileStart).Round(time.Millisecond), state.totalBytes>>bytesToMBShift)

	mergeFileState(state, stats)
}

func (fs *fileState) processChunk(chunk []byte) {
	text := string(chunk)
	fs.totalBytes += int64(len(chunk))

	tokens := tokenizer.WordTokens(text)

	var sb strings.Builder
	if !fs.reconFailed {
		sb.Grow(len(text))
	}
	for _, token := range tokens {
		fs.tokenCounts[token.Type]++
		if !fs.reconFailed {
			sb.WriteString(token.Text)
		}
	}
	if !fs.reconFailed {
		if sb.String() != text {
			fs.reconFailed = true
			if !fs.reconFailLogged {
				logReconstructionFailure(fs.path, text, sb.String())
				fs.reconFailLogged = true
			}
		}
	}

	fs.sentences += len(tokenizer.SentenceTokens(text))

	fs.paragraphs += strings.Count(text, "\n\n")

	if !fs.hasCyrillic && containsCyrillic(text) {
		fs.hasCyrillic = true
	}

	if fs.hasCyrillic && !fs.translitFailed {
		latinText := translit.CyrillicToLatin(text)
		ltokens := tokenizer.WordTokens(latinText)
		var lsb strings.Builder
		lsb.Grow(len(latinText))
		for _, token := range ltokens {
			lsb.WriteString(token.Text)
		}
		if lsb.String() != latinText {
			fs.translitFailed = true
			if !fs.translitLogged {
				logTranslitReconstructionFailure(fs.path, latinText, lsb.String())
				fs.translitLogged = true
			}
		}
	}
}

func mergeFileState(fs *fileState, stats *Stats) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.filesScanned++
	stats.totalBytes += fs.totalBytes

	if fs.reconFailed {
		stats.reconFail++
	} else {
		stats.reconOK++
	}

	for tokenType, count := range fs.tokenCounts {
		stats.tokenTypeCounts[tokenType] += count
	}

	ratio := float64(fs.sentences) / float64(fs.paragraphs)
	stats.fileRatios = append(stats.fileRatios, fileRatio{
		path:       fs.path,
		sentences:  fs.sentences,
		paragraphs: fs.paragraphs,
		ratio:      ratio,
	})

	if fs.hasCyrillic {
		stats.cyrillicFiles++
		if fs.translitFailed {
			stats.translitReconFail++
		} else {
			stats.translitReconOK++
		}
	}
}

func logReconstructionFailure(path, original, reconstructed string) {
	pos, got, want := firstDivergence(original, reconstructed)
	fmt.Fprintf(os.Stderr, "RECON_FAIL: %s: first divergence at byte %d (got 0x%02x, want 0x%02x)\n",
		path, pos, got, want)
}

// flagSentenceOutliers computes the median sentence/paragraph ratio across all
// files and flags any file whose ratio exceeds 3x the median.
func flagSentenceOutliers(stats *Stats) {
	if len(stats.fileRatios) == 0 {
		return
	}

	ratios := make([]float64, len(stats.fileRatios))
	for i, fr := range stats.fileRatios {
		ratios[i] = fr.ratio
	}
	med := computeMedian(ratios)

	for _, fr := range stats.fileRatios {
		if med > 0 && fr.ratio > 3*med {
			stats.sentenceOutliers++
			fmt.Fprintf(os.Stderr, "SENTENCE_OUTLIER: %s: %d sentences / %d paragraphs (ratio %.2f, median %.2f)\n",
				fr.path, fr.sentences, fr.paragraphs, fr.ratio, med)
		}
	}
}

func containsCyrillic(text string) bool {
	for _, r := range text {
		if r >= '\u0400' && r <= '\u04FF' {
			return true
		}
	}
	return false
}

func logTranslitReconstructionFailure(path, original, reconstructed string) {
	pos, got, want := firstDivergence(original, reconstructed)
	fmt.Fprintf(os.Stderr, "TRANSLIT_RECON_FAIL: %s: first divergence at byte %d (got 0x%02x, want 0x%02x)\n",
		path, pos, got, want)
}

// firstDivergence finds the byte position where two strings first differ.
// Returns the position and the differing bytes from each string.
func firstDivergence(original, reconstructed string) (pos int, got, want byte) {
	n := min(len(original), len(reconstructed))
	for i := range n {
		if original[i] != reconstructed[i] {
			return i, reconstructed[i], original[i]
		}
	}
	pos = n
	if pos < len(reconstructed) {
		got = reconstructed[pos]
	}
	if pos < len(original) {
		want = original[pos]
	}
	return pos, got, want
}

func computeMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2 //nolint:mnd // arithmetic mean of two middle values
	}
	return sorted[mid]
}

func printStats(stats *Stats) {
	fmt.Printf("Files scanned:           %d\n", stats.filesScanned)
	fmt.Printf("Total bytes:             %d\n", stats.totalBytes)
	fmt.Printf("Reconstruction OK:       %d\n", stats.reconOK)
	fmt.Printf("Reconstruction FAIL:     %d\n", stats.reconFail)
	fmt.Printf("Sentence outliers:       %d\n", stats.sentenceOutliers)
	fmt.Printf("Cyrillic files:          %d\n", stats.cyrillicFiles)
	fmt.Printf("Translit recon OK:       %d\n", stats.translitReconOK)
	fmt.Printf("Translit recon FAIL:     %d\n", stats.translitReconFail)
	fmt.Println()

	totalTokens := 0
	for _, count := range stats.tokenTypeCounts {
		totalTokens += count
	}

	fmt.Println("Token type distribution:")
	printTokenTypeStats("Word", tokenizer.Word, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("Number", tokenizer.Number, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("Punctuation", tokenizer.Punctuation, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("Space", tokenizer.Space, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("Symbol", tokenizer.Symbol, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("URL", tokenizer.URL, stats.tokenTypeCounts, totalTokens)
	printTokenTypeStats("Email", tokenizer.Email, stats.tokenTypeCounts, totalTokens)
}

func printTokenTypeStats(label string, tokenType tokenizer.TokenType, counts map[tokenizer.TokenType]int, total int) {
	count := counts[tokenType]
	percentage := 0.0
	if total > 0 {
		percentage = float64(count) / float64(total) * 100
	}
	fmt.Printf("  %-15s %d  (%.1f%%)\n", label+":", count, percentage)
}
