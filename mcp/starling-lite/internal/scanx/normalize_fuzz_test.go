package scanx

import (
	"encoding/json"
	"testing"
)

// FuzzNormalize feeds arbitrary scanner output to Normalize. ShrikeOps output
// is untrusted input for starling-lite, so Normalize must never panic, and an
// incomplete report must never carry a grade.
//
//	go test ./internal/scanx -run '^$' -fuzz FuzzNormalize -fuzztime 30s
func FuzzNormalize(f *testing.F) {
	f.Add([]byte(legacyNoTools))
	f.Add([]byte(`{"score":{"complete":false,"skipped_checks":[{"tool":"pluto"}]},"findings":[]}`))
	f.Add([]byte(`{"score":{"complete":true,"overall":91,"grade":"A"},"findings":[{"id":"X"}]}`))
	f.Add([]byte(`{"score":{"security":"high"},"findings":[{"id":"POLARIS-NOTRUN"},null,7]}`))
	f.Add([]byte(`{"score":null}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, raw []byte) {
		r, err := Normalize(raw)
		if err != nil || r == nil {
			return
		}
		if _, err := json.Marshal(r); err != nil {
			t.Fatalf("normalized report does not encode: %v", err)
		}
		if inc, _ := r["incomplete"].(bool); inc {
			score, _ := r["score"].(map[string]any)
			if score == nil {
				t.Fatalf("incomplete report without a score object")
			}
			if score["grade"] != "N/A" || score["complete"] != false {
				t.Fatalf("incomplete report still graded: grade=%v complete=%v", score["grade"], score["complete"])
			}
			if _, ok := r["warning"].(string); !ok {
				t.Fatalf("incomplete report without a warning")
			}
		}
	})
}
