package steam

import "testing"

// TestParseSchema checks the response shape Steam serves for a game's
// achievement definitions, and that a missing or empty list is not data.
func TestParseSchema(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"missing wrapper", `{}`, 0},
		{"empty list", `{"game":{"availableGameStats":{"achievements":[]}}}`, 0},
		{"junk", `not json`, 0},
		{
			"names by api name",
			`{"game":{"availableGameStats":{"achievements":[
				{"name":"A_FIRST","displayName":"First steps","description":"Do it"},
				{"name":"B_SECOND","displayName":"Second steps","description":"Do more"}
			]}}}`,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSchema([]byte(tt.body))
			if len(got) != tt.want {
				t.Fatalf("entries = %d, want %d (ok=%v)", len(got), tt.want, ok)
			}
			if tt.want == 2 {
				if got["A_FIRST"].Name != "First steps" || got["A_FIRST"].Description != "Do it" {
					t.Errorf("first = %+v", got["A_FIRST"])
				}
				if got["B_SECOND"].Name != "Second steps" {
					t.Errorf("second = %+v", got["B_SECOND"])
				}
			}
		})
	}
}
