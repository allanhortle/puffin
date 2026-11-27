package main

import (
	"github.com/allanhortle/puffin/internal/config"
	"reflect"
	"strings"
	"testing"
)

func TestPrepare(t *testing.T) {
	body := strings.Join([]string{
		"<!-- template: delete this -->",
		"## What",
		`Before <img width="400" alt="old" src="https://x/old.png"> and after.`,
		"![new](https://x/new.png)",
		"https://github.com/user-attachments/assets/1234-abcd",
		"| before | after |",
		"|---|---|",
		"| ![b](https://x/b.png) | ![a](https://x/a.png) |",
		"",
		"```",
		"![not an image](https://x/code.png)",
		"```",
		"<details><summary>Logs</summary>",
		"ok<br>",
		"</details>",
	}, "\n")
	got, pictures := prepare(body, config.Read{})

	want := []picture{
		{"old", "https://x/old.png"},
		{"new", "https://x/new.png"},
		{"attachment", "https://github.com/user-attachments/assets/1234-abcd"},
		{"b", "https://x/b.png"},
		{"a", "https://x/a.png"},
	}
	if !reflect.DeepEqual(pictures, want) {
		t.Fatalf("pictures = %+v", pictures)
	}
	for _, gone := range []string{"template", "<img", "<details>", "<summary>", "<br>"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q left in:\n%s", gone, got)
		}
	}
	if !strings.Contains(got, "**Logs**") || !strings.Contains(got, "![not an image](https://x/code.png)") {
		t.Errorf("summary or fenced code mangled:\n%s", got)
	}
	// The table keeps its links, and its pictures follow it.
	table := strings.Index(got, "| [b](https://x/b.png) | [a](https://x/a.png) |")
	after := strings.Index(got, "PUFFINIMAGE3X")
	if table < 0 || after < table {
		t.Errorf("table pictures not held until after the table:\n%s", got)
	}
	if n, ok := pictureAt("  \x1b[0mPUFFINIMAGE4X\x1b[0m  "); !ok || n != 4 {
		t.Errorf("marker = %d, %v", n, ok)
	}
}

func TestPRLink(t *testing.T) {
	for link, want := range map[string]string{
		"https://github.com/Tactiq-HQ/main/pull/9604":              "Tactiq-HQ/main 9604",
		"https://app.graphite.com/github/pr/Tactiq-HQ/main/9604":   "Tactiq-HQ/main 9604",
		"https://app.graphite.com/github/pr/Tactiq-HQ/main/9604/x": "Tactiq-HQ/main 9604",
	} {
		m := prLink.FindStringSubmatch(link)
		if m == nil || m[1]+" "+m[2] != want {
			t.Errorf("%s -> %v", link, m)
		}
	}
	if prLink.MatchString("9604") {
		t.Error("a bare number read as a link")
	}
}

func TestPrepareHides(t *testing.T) {
	body := "Keep this.\n<!-- CURSOR_SUMMARY -->\n> [!NOTE]\n> Low risk\n<!-- /CURSOR_SUMMARY -->\nAnd this."
	got, _ := prepare(body, config.Read{Hide: []string{`(?s)<!--\s*CURSOR_SUMMARY\s*-->.*?<!--\s*/CURSOR_SUMMARY\s*-->`}})
	if strings.Contains(got, "Low risk") || !strings.Contains(got, "Keep this.") || !strings.Contains(got, "And this.") {
		t.Errorf("hidden block not removed cleanly:\n%s", got)
	}
	if kept, _ := prepare(body, config.Read{}); !strings.Contains(kept, "Low risk") {
		t.Errorf("a block was hidden with nothing named:\n%s", kept)
	}
}

func TestPrepareStripsABotsFooter(t *testing.T) {
	body := "### Menus lose stacking order\n\nThe menu paints under the dialog.\n" +
		`<div><a href="https://cursor.com/open?link=abc" target="_blank"><picture><img alt="Fix in Cursor" src="https://cursor.com/a.png"></picture></a>&nbsp;<a href="https://cursor.com/agents?link=def"><img alt="Fix in Web" src="https://cursor.com/b.png"></a></div>` +
		"\n\n<sup>Reviewed by [Cursor Bugbot](https://cursor.com/bugbot) for commit 7385ba4. Configure [here](https://www.cursor.com/dashboard/bugbot).</sup>\n"
	got, pictures := prepare(body, config.Read{Hide: []string{
		`(?s)<div><a href="https://cursor\.com/.*?</div>`,
		`(?s)<sup>Reviewed by \[Cursor Bugbot\].*?</sup>`,
	}})
	if strings.Contains(got, "Cursor") || len(pictures) != 0 || !strings.Contains(got, "paints under the dialog") {
		t.Fatalf("footer not stripped: %q, %d pictures", got, len(pictures))
	}
}
