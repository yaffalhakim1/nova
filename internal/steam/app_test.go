package steam

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	raw := []byte(`{"1091500":{"success":true,"data":{
		"name":"Cyberpunk 2077",
		"short_description":"An open-world RPG.",
		"detailed_description":"<p>A long <b>description</b> of the game, with enough words to be worth showing.</p>",
		"header_image":"https://x/header.jpg",
		"background_raw":"https://x/bg.jpg",
		"release_date":{"date":"9 Dec, 2020"},
		"developers":["CD PROJEKT RED"],
		"genres":[{"description":"RPG"}],
		"screenshots":[{"path_full":"https://x/s1.jpg"},{"path_full":"https://x/s2.jpg"}]
	}}}`)

	d, ok := parse(raw, "1091500")
	if !ok {
		t.Fatal("parse rejected a valid store response")
	}
	if d.Name != "Cyberpunk 2077" {
		t.Fatalf("name = %q", d.Name)
	}
	if d.Description == "" || containsTag(d.Description) {
		t.Fatalf("description not cleaned: %q", d.Description)
	}
	if len(d.Screenshots) != 2 {
		t.Fatalf("screenshots = %d, want 2", len(d.Screenshots))
	}
	if len(d.Genres) != 1 || d.Genres[0] != "RPG" {
		t.Fatalf("genres = %v", d.Genres)
	}
	if d.Background != "https://x/bg.jpg" {
		t.Fatalf("background = %q", d.Background)
	}
}

// TestParseUnknownApp checks a store that does not know the app reports it.
func TestParseUnknownApp(t *testing.T) {
	raw := []byte(`{"999":{"success":false}}`)
	if _, ok := parse(raw, "999"); ok {
		t.Fatal("parse accepted a failed response")
	}
	if _, ok := parse([]byte(`not json`), "1"); ok {
		t.Fatal("parse accepted junk")
	}
}

// TestCleanTextDropsHeadingsAndSlogans pins the real shape of a store page: the
// detailed description opens with banner markup and an <h1> slogan, and once the
// tags were stripped that slogan became the opening sentence of the About card,
// in capitals. The short description is real prose, so it wins.
func TestCleanTextDropsHeadingsAndSlogans(t *testing.T) {
	detailed := `<h1>GET READY FOR AN EPIC STORY FULL OF CHOICES</h1>` +
		`<p><p class="bb_paragraph"><span class="bb_img_ctn">` +
		`<img src="https://x/a.jpg" width=780 height=300 /></span></p></p>` +
		`<br><h1>About the Game</h1>` +
		`<p class="bb_paragraph">Become a monster slayer for hire and journey across a dark fantasy world.</p>`
	short := "The Witcher 3 modernizes the acclaimed role-playing game with a range of upgrades."

	got := cleanText(detailed, short)
	if got != short {
		t.Fatalf("cleanText = %q, want the short description", got)
	}
	for _, banned := range []string{"GET READY", "About the Game"} {
		if strings.Contains(got, banned) {
			t.Fatalf("cleanText kept the heading %q: %q", banned, got)
		}
	}
}

// TestStripHeadingsKeepsTheBody checks the heading is dropped but the paragraph
// under it survives, which is what makes the fallback worth having.
func TestStripHeadingsKeepsTheBody(t *testing.T) {
	got := stripHeadings(`<h2>About the Game</h2><p>Real prose about the game.</p>`)
	if got != "Real prose about the game." {
		t.Fatalf("stripHeadings = %q", got)
	}
}

// TestStripTagsTurnOneLine checks a description that is only markup falls back
// cleanly instead of becoming empty or a pile of spaces.
func TestStripTagsTurnOneLine(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`<h1>Title</h1><p>Body text here.</p>`, "Body text here."},
		{`<img src=x>`, ""},
		{`<p>One</p><p>Two</p>`, "One Two"},
	} {
		if got := stripHeadings(tc.in); got != tc.want {
			t.Errorf("stripHeadings(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestStripTags checks the markup of a description is dropped and its
// entities decoded.
func TestStripTags(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"<p>Hello <b>world</b></p>", "Hello world"},
		{"A &amp; B", "A & B"},
		{"<h1>Title</h1><p>Body  text</p>", "Title Body text"},
		{"", ""},
	} {
		if got := stripTags(tc.in); got != tc.want {
			t.Errorf("stripTags(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestCleanTextRejectsAnImageOnlyDetailedDescription pins the real page that
// broke the About card: The Witcher 3's detailed description is two h1 slogans
// and six images, with no prose at all. The short description must win.
func TestCleanTextRejectsAnImageOnlyDetailedDescription(t *testing.T) {
	detailed := `<h1>GET READY FOR AN EPIC WITCHER STORY FULL OF CHOICES AND CONSEQUENCES</h1>` +
		`<p><p class="bb_paragraph"><span class="bb_img_ctn"><img class="bb_img" src="https://x/1.avif" width=780 height=300 /></span></p></p><br>` +
		`<h1>About the Game</h1><p class="bb_paragraph"><span class="bb_img_ctn"><img src="https://x/2.avif" /></span></p>` +
		`<p><img src="https://x/3.avif" /></p>`
	short := "The Witcher 3 modernizes the acclaimed role-playing game with a range of upgrades and enhancements."

	if got := cleanText(detailed, short); got != short {
		t.Fatalf("cleanText = %q, want the short description", got)
	}
}

// TestCleanTextFallsBackToTheDetailed checks the fallback: when the short
// description is too brief to be a paragraph, the detailed one is used, and a
// detailed one that is only markup yields to the short text rather than going
// blank.
func TestCleanTextFallsBackToTheDetailed(t *testing.T) {
	long := "<p>This is a genuinely long description of what the game is about and how it plays.</p>"
	if got := cleanText(long, "Short."); got == "Short." {
		t.Fatalf("cleanText kept the short one: %q", got)
	}
	if got := cleanText("<img src=x>", "Short one."); got != "Short one." {
		t.Fatalf("cleanText = %q, want the short one", got)
	}
}

func containsTag(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			return true
		}
	}
	return false
}
