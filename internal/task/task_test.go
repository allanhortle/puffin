package task

import "testing"

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"How does auth work here":                                       "how-does-auth-work",
		"PROD-123 build the thing now please":                           "prod-123-build-the-thing-now",
		"prod-5259: Build a team reporting page for admins":             "prod-5259-build-a-team-reporting",
		"  Fix   the/weird_bug!! ":                                      "fix-the-weird-bug",
		"internationalisation accessibility infrastructure reliability": "internationalisation-accessibility",
		"": "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
		if len(Slugify(in)) > 40 {
			t.Errorf("Slugify(%q) is longer than 40", in)
		}
	}
}
