package main

import "testing"

func TestCleanName(t *testing.T) {
	for _, c := range []struct{ answer, prompt, want string }{
		{"prod-7321-one-dialog-component\n", "Work on PROD-7321: dialogs", "prod-7321-one-dialog-component"},
		{"Here you go:\n`fix-galician-label`\n", "Fix PROD-7176 label", "prod-7176-fix-galician-label"},
		{"Outlook calendar series", "group Teams meetings", "outlook-calendar-series"},
		{"\n\n", "anything", ""},
	} {
		if got := cleanName(c.answer, c.prompt); got != c.want {
			t.Errorf("cleanName(%q) = %q, want %q", c.answer, got, c.want)
		}
	}
}
