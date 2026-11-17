package llmfix

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

// FuzzParseAnswer feeds arbitrary model completions to parseAnswer. A
// completion is untrusted input, so the parser must never panic, and anything
// it accepts must be a non-nil patch object with a sanitized rationale.
//
//	go test ./internal/llmfix -run '^$' -fuzz FuzzParseAnswer -fuzztime 30s
func FuzzParseAnswer(f *testing.F) {
	f.Add(`{"patch":{"spec":{"replicas":2}},"rationale":"scale up"}`)
	f.Add("```json\n{\"patch\":{\"spec\":{}},\"rationale\":\"fenced\"}\n```")
	f.Add(`{"patch":{},"rationale":"line\nbreak\ttab\u0000nul"}`)
	f.Add(`{"patch":[1,2],"rationale":"array patch"}`)
	f.Add(`{"patch":null,"rationale":""}`)
	f.Add(`{"patch":{},"rationale":"x","extra":1}`)
	f.Add(`{"patch":{},"rationale":"x"} trailing`)
	f.Add("```")
	f.Add(``)

	f.Fuzz(func(t *testing.T, text string) {
		patch, rationale, err := parseAnswer(text)
		if err != nil {
			return
		}
		if patch == nil {
			t.Fatalf("accepted completion with a nil patch")
		}
		if n := utf8.RuneCountInString(rationale); n > maxRationaleChars+1 {
			t.Fatalf("rationale not truncated: %d runes", n)
		}
		for _, r := range rationale {
			if unicode.IsControl(r) {
				t.Fatalf("rationale keeps control character %U", r)
			}
		}
	})
}
