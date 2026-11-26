package main

import (
	"strings"
	"testing"

	"github.com/allanhortle/puffin/internal/github"
)

func sha(c string) string { return strings.Repeat(c, 40) }

func TestResolveOrigin(t *testing.T) {
	h := github.History{
		Head: sha("c"), BaseRef: "main", Reviewed: sha("a"),
		Versions: []github.Version{{SHA: sha("a")}, {SHA: sha("b")}, {SHA: sha("c")}},
		Commits:  []github.Commit{{SHA: sha("d")}, {SHA: sha("c")}},
	}
	for since, want := range map[string]string{
		"":            sha("a"), // my last review
		"reviewed":    sha("a"),
		"base":        "",
		"v2":          sha("b"),
		"ddddddd":     sha("d"),
		"some-branch": "some-branch",
		sha("e")[:12]: sha("e")[:12],
	} {
		got, err := resolveOrigin(h, originsOf(h), since)
		if err != nil || got.SHA != want {
			t.Errorf("--since %q: %q, %v; want %q", since, got.SHA, err, want)
		}
	}
	if _, err := resolveOrigin(h, originsOf(h), "v3"); err == nil {
		t.Error("v3 is the head: there is nothing to diff from it")
	}

	// Reviewed, with nothing pushed since: the whole change.
	h.Reviewed = h.Head
	if got, _ := resolveOrigin(h, originsOf(h), ""); got.Name != sinceBase {
		t.Errorf("reviewed at the head: from %q, want the base", got.Name)
	}
	if _, err := resolveOrigin(h, originsOf(h), "reviewed"); err == nil {
		t.Error("reviewed at the head should say nothing was pushed")
	}
}
