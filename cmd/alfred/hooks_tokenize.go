package main

import (
	"strings"
	"sync"
	"unicode"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/kljensen/snowball"
)

// ---------------------------------------------------------------------------
// Kagome tokenizer (lazy singleton)
// ---------------------------------------------------------------------------

// kagomeTokenizer is lazily initialized on first use.
var (
	kagomeOnce sync.Once
	kagomeTok  *tokenizer.Tokenizer
)

func getKagome() *tokenizer.Tokenizer {
	kagomeOnce.Do(func() {
		t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
		if err != nil {
			debugf("kagome init failed: %v", err)
			return
		}
		kagomeTok = t
	})
	return kagomeTok
}

// ---------------------------------------------------------------------------
// CJK detection
// ---------------------------------------------------------------------------

// hasCJK reports whether s contains any CJK characters.
func hasCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Tokenization
// ---------------------------------------------------------------------------

// tokenizePrompt splits a prompt into searchable tokens.
// Uses kagome (IPA dictionary) for Japanese text, simple word splitting for ASCII.
//
//	"hookの設定方法を教えて" → ["hook", "の", "設定", "方法", "を", "教え", "て"]
//	"how to configure hooks" → ["how", "to", "configure", "hooks"]
func tokenizePrompt(s string) []string {
	if !hasCJK(s) {
		return tokenizeASCII(s)
	}
	tok := getKagome()
	if tok == nil {
		return tokenizeASCII(s)
	}
	seg := tok.Tokenize(s)
	tokens := make([]string, 0, len(seg))
	for _, t := range seg {
		surface := strings.TrimSpace(t.Surface)
		if surface == "" {
			continue
		}
		// Skip pure punctuation/symbols.
		hasLetterOrDigit := false
		for _, r := range surface {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				hasLetterOrDigit = true
				break
			}
		}
		if !hasLetterOrDigit {
			continue
		}
		tokens = append(tokens, surface)
	}
	return tokens
}

// tokenizeASCII splits ASCII text on non-letter/digit boundaries.
func tokenizeASCII(s string) []string {
	var tokens []string
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(r)
		} else if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}

// ---------------------------------------------------------------------------
// POS filtering (kagome)
// ---------------------------------------------------------------------------

// isContentPOS reports whether a kagome token is a content word (noun, verb, adjective).
// Filters out particles (助詞), auxiliaries (助動詞), symbols (記号), and fillers.
func isContentPOS(t tokenizer.Token) bool {
	pos := t.POS()
	if len(pos) == 0 {
		return false
	}
	switch pos[0] {
	case "名詞": // noun
		// Filter pronouns and non-independent nouns
		if len(pos) > 1 && (pos[1] == "代名詞" || pos[1] == "非自立") {
			return false
		}
		return true
	case "動詞": // verb
		// Only independent verbs, not auxiliaries like する/いる
		if len(pos) > 1 && pos[1] == "非自立" {
			return false
		}
		return true
	case "形容詞": // i-adjective
		return true
	case "副詞": // adverb — sometimes useful
		return true
	default:
		// 助詞, 助動詞, 記号, 接続詞, フィラー, 連体詞 → skip
		return false
	}
}

// ---------------------------------------------------------------------------
// Token filtering & stemming
// ---------------------------------------------------------------------------

// isMeaningfulToken reports whether a token string is worth scoring.
// For CJK: filters single-character particles and common auxiliaries.
// For ASCII: requires 3+ characters.
func isMeaningfulToken(w string) bool {
	runeLen := len([]rune(w))
	isCJK := len(w) > runeLen // multi-byte = CJK
	if isCJK {
		if runeLen <= 1 {
			return false // single CJK char (particles)
		}
		// Filter common Japanese auxiliaries/copulas
		switch w {
		case "する", "いる", "ある", "なる", "できる",
			"です", "ます", "ない", "たい", "れる", "られる",
			"ください", "ている", "ておく", "てある":
			return false
		}
		return true
	}
	// ASCII: 3+ chars
	return runeLen >= 3
}

// stemEnglish applies Snowball stemming to an English word.
// Returns the stem, or the original word if stemming fails.
func stemEnglish(word string) string {
	stemmed, err := snowball.Stem(word, "english", true)
	if err != nil || stemmed == "" {
		return word
	}
	return stemmed
}

// englishStopWords are common English words that add no search value.
var englishStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "can": true, "this": true, "that": true,
	"these": true, "those": true, "with": true, "from": true, "into": true,
	"for": true, "and": true, "but": true, "not": true, "what": true,
	"how": true, "when": true, "where": true, "which": true, "who": true,
	"about": true, "some": true, "want": true, "need": true, "like": true,
	"make": true, "just": true, "also": true, "more": true, "very": true,
	"please": true, "help": true, "using": true, "used": true, "use": true,
}

// ---------------------------------------------------------------------------
// Keyword extraction for FTS search
// ---------------------------------------------------------------------------

// extractSearchKeywords extracts meaningful keywords from a prompt for FTS search.
// Uses kagome POS for Japanese, stop word filtering for English.
func extractSearchKeywords(prompt string, maxWords int) string {
	if hasCJK(prompt) {
		return extractSearchKeywordsCJK(prompt, maxWords)
	}
	return extractSearchKeywordsASCII(prompt, maxWords)
}

// extractSearchKeywordsCJK uses kagome POS to extract content words from CJK text.
func extractSearchKeywordsCJK(prompt string, maxWords int) string {
	tok := getKagome()
	if tok == nil {
		return ""
	}
	seg := tok.Tokenize(prompt)
	var keywords []string
	for _, t := range seg {
		if !isContentPOS(t) {
			continue
		}
		surface := strings.TrimSpace(t.Surface)
		if surface == "" || len([]rune(surface)) < 2 {
			continue
		}
		keywords = append(keywords, surface)
		if len(keywords) >= maxWords {
			break
		}
	}
	return strings.Join(keywords, " OR ")
}

// extractSearchKeywordsASCII extracts keywords from English text using stop word filtering.
func extractSearchKeywordsASCII(prompt string, maxWords int) string {
	var keywords []string
	for _, word := range strings.Fields(strings.ToLower(prompt)) {
		word = strings.Trim(word, ".,!?;:\"'`()[]{}/-")
		if len(word) < 3 || englishStopWords[word] {
			continue
		}
		keywords = append(keywords, word)
		if len(keywords) >= maxWords {
			break
		}
	}
	return strings.Join(keywords, " ")
}

// ---------------------------------------------------------------------------
// Content token extraction for relevance scoring
// ---------------------------------------------------------------------------

// contentTokensForScoring extracts meaningful content tokens from a prompt
// for relevance scoring. Uses kagome POS for CJK, stop word filtering for ASCII.
// Returns both surface forms and base forms (for cross-lingual matching).
func contentTokensForScoring(prompt string) []string {
	if !hasCJK(prompt) {
		// ASCII path: use tokenizeASCII + isMeaningfulToken + stemming
		tokens := tokenizeASCII(strings.ToLower(prompt))
		seen := make(map[string]bool)
		var result []string
		for _, w := range tokens {
			if !isMeaningfulToken(w) || englishStopWords[w] {
				continue
			}
			if !seen[w] {
				seen[w] = true
				result = append(result, w)
			}
			// Add stem for broader matching ("configuring" → "configur" matches "configuration")
			stemmed := stemEnglish(w)
			if stemmed != w && !seen[stemmed] {
				seen[stemmed] = true
				result = append(result, stemmed)
			}
		}
		return result
	}
	tok := getKagome()
	if tok == nil {
		tokens := tokenizeASCII(strings.ToLower(prompt))
		var result []string
		for _, w := range tokens {
			if isMeaningfulToken(w) {
				result = append(result, w)
			}
		}
		return result
	}
	seg := tok.Tokenize(prompt)
	seen := make(map[string]bool)
	var result []string
	for _, t := range seg {
		if !isContentPOS(t) {
			continue
		}
		surface := strings.ToLower(strings.TrimSpace(t.Surface))
		if surface == "" || len([]rune(surface)) < 2 {
			continue
		}
		if !seen[surface] {
			seen[surface] = true
			result = append(result, surface)
		}
		// Add base form if different (helps matching: 教え → 教える)
		if base, ok := t.BaseForm(); ok {
			baseLower := strings.ToLower(base)
			if baseLower != surface && baseLower != "*" && !seen[baseLower] {
				seen[baseLower] = true
				result = append(result, baseLower)
			}
		}
		// For ASCII tokens within CJK text, also add English stem
		if len(surface) == len([]rune(surface)) && len(surface) >= 3 {
			stemmed := stemEnglish(surface)
			if stemmed != surface && !seen[stemmed] {
				seen[stemmed] = true
				result = append(result, stemmed)
			}
		}
	}
	return result
}

// significantWords extracts significant words from text for duplicate detection.
// Uses tokenizePrompt for proper Japanese word segmentation, then filters
// by isMeaningfulToken to keep only content words.
func significantWords(text string) []string {
	tokens := tokenizePrompt(text)
	var result []string
	for _, w := range tokens {
		w = strings.Trim(w, ".,!?;:\"'`()[]{}/-")
		if isMeaningfulToken(w) {
			result = append(result, w)
		}
	}
	return result
}
