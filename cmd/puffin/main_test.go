package main

import "testing"

func TestTouchedGitHub(t *testing.T) {
	bash := func(command string) []byte {
		return []byte(`{"tool_name":"Bash","tool_input":{"command":` + quoteJSON(command) + `}}`)
	}
	for command, want := range map[string]bool{
		"gh pr review 12 --approve":         true,
		"gh pr create --fill":               true,
		"cd repo && git push -u origin x":   true,
		"gt submit --stack":                 true,
		"yarn test; gh api repos/o/r/pulls": true,
		"git status":                        false,
		"git pushd":                         false,
		"echo gh":                           false,
		"ls weight":                         false,
		"./light pr":                        false,
	} {
		if got := touchedGitHub(bash(command)); got != want {
			t.Errorf("%q: %v, want %v", command, got, want)
		}
	}
	if touchedGitHub([]byte(`{"tool_name":"Edit","tool_input":{"command":"gh pr merge"}}`)) {
		t.Error("a tool that is not the shell counted")
	}
	if touchedGitHub(nil) || touchedGitHub([]byte("not json")) {
		t.Error("no input counted")
	}
}

func quoteJSON(s string) string {
	out := []byte{'"'}
	for _, r := range s {
		if r == '"' || r == '\\' {
			out = append(out, '\\')
		}
		out = append(out, string(r)...)
	}
	return string(append(out, '"'))
}
