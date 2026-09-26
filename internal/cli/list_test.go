package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/VishalDalwadi/anchor/internal/model"
)

// movies → sci-fi → classics, plus a separate groceries list.
func testCollections() []model.Collection {
	return []model.Collection{
		{ID: "c_movies", Name: "movies-to-watch", Items: []string{"Dune", "Arrival", "1917"}},
		{ID: "c_scifi", Name: "sci-fi", Items: []string{"Solaris"}, Parent: "c_movies"},
		{ID: "c_classics", Name: "classics", Items: []string{"Stalker"}, Parent: "c_scifi"},
		{ID: "c_groceries", Name: "Groceries", Items: []string{}},
	}
}

func TestFindCollection(t *testing.T) {
	cols := testCollections()
	for ref, want := range map[string]string{
		"c_scifi":   "sci-fi",    // by id
		"sci-fi":    "sci-fi",    // by name, even a sublist
		"groceries": "Groceries", // ignoring case
	} {
		i, err := findCollection(cols, ref)
		if err != nil || cols[i].Name != want {
			t.Errorf("findCollection(%q) = %v, %v; want %s", ref, i, err, want)
		}
	}
	if _, err := findCollection(cols, "nope"); err == nil {
		t.Error("findCollection(nope): want an error")
	}
}

func TestValidateCollectionName(t *testing.T) {
	cols := testCollections()
	if err := validateCollectionName(cols, "SCI-FI", ""); err == nil {
		t.Error("duplicate name (different case) accepted for a new list")
	}
	if err := validateCollectionName(cols, "Sci-Fi", "c_scifi"); err != nil {
		t.Errorf("renaming a list to a new case of its own name: %v", err)
	}
	if err := validateCollectionName(cols, "", ""); err == nil {
		t.Error("empty name accepted")
	}
}

func TestValidateCollectionParent(t *testing.T) {
	cols := testCollections()
	// Moving an existing list under another, e.g. a newly created one.
	if err := validateCollectionParent(cols, "c_groceries", "c_classics"); err != nil {
		t.Errorf("groceries under classics: %v", err)
	}
	for _, tt := range []struct{ child, parent string }{
		{"c_movies", "c_movies"},   // itself
		{"c_movies", "c_classics"}, // its own grandchild
		{"c_scifi", "c_classics"},  // its own child
	} {
		if err := validateCollectionParent(cols, tt.child, tt.parent); err == nil || !strings.Contains(err.Error(), "own sublists") {
			t.Errorf("%s under %s: err = %v, want a loop error", tt.child, tt.parent, err)
		}
	}
}

func TestFindItem(t *testing.T) {
	items := []string{"Dune", "Arrival", "1917"}
	for ref, want := range map[string]int{
		"2":       1, // by number
		"Arrival": 1, // by text
		"arrival": 1, // ignoring case
		"1917":    2, // out-of-range number falls back to text
	} {
		if got, err := findItem(items, ref); err != nil || got != want {
			t.Errorf("findItem(%q) = %d, %v; want %d", ref, got, err, want)
		}
	}
	for _, ref := range []string{"0", "4", "Tenet"} {
		if _, err := findItem(items, ref); err == nil {
			t.Errorf("findItem(%q): want an error", ref)
		}
	}
}

func TestPrintCollectionNestsSublists(t *testing.T) {
	cols := testCollections()
	var out bytes.Buffer
	printCollection(&out, cols, cols[0], 0)
	want := `movies-to-watch
  1. Dune
  2. Arrival
  3. 1917
  sci-fi/
    1. Solaris
    classics/
      1. Stalker
`
	if out.String() != want {
		t.Errorf("show:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestPrintCollectionTree(t *testing.T) {
	var out bytes.Buffer
	printCollectionTree(&out, testCollections())
	want := "Groceries\t(0 items)\n" +
		"movies-to-watch\t(3 items)\n" +
		"  sci-fi\t(1 item)\n" +
		"    classics\t(1 item)\n"
	if out.String() != want {
		t.Errorf("all:\n%s\nwant:\n%s", out.String(), want)
	}
}
