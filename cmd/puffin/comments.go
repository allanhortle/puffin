package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"golang.org/x/term"

	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/github"
	"github.com/allanhortle/puffin/internal/paths"
	"github.com/allanhortle/puffin/internal/pr"
)

// How much of the diff a thread was started on is shown above it: the lines
// nearest the comment, which sits under the last.
const hunkLines = 4

// comments lists what has been said on a pull request: the conversation
// under it, then each thread on its diff with the lines it is about.
func comments(args []string) error {
	flags, rest, err := parse(args)
	if err != nil {
		return err
	}
	_, unresolved := flags["--unresolved"]
	_, all := flags["--all"]
	target := ""
	if len(rest) > 0 {
		target = rest[0]
	}
	ctx, cancel := timeout(readTimeout)
	defer cancel()
	repo, number, _, err := readTarget(ctx, target)
	if err != nil {
		return err
	}
	d, err := github.New().Discuss(ctx, repo, number)
	if err != nil {
		return err
	}

	width := 80
	tty := term.IsTerminal(int(os.Stdout.Fd()))
	if tty {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		}
	}
	style := styles.DarkStyle
	if !tty {
		style = styles.NoTTYStyle
	}
	renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style),
		glamour.WithWordWrap(min(width-4, readMaxWidth)))
	if err != nil {
		return err
	}
	cfg, _ := config.Load(paths.Config())
	hide := cfg.Read
	grey := func(s string) string { return paint("colour242", s) }
	markdown := func(text string) string {
		out, err := renderer.Render(text)
		if err != nil {
			return text + "\n"
		}
		return strings.Trim(out, "\n") + "\n"
	}

	var out bytes.Buffer
	said := func(c github.Comment) {
		parts := []string{paint("bold", or(c.Author, "ghost"))}
		switch c.Verdict {
		case "APPROVED":
			parts = append(parts, paint("green", "approved"))
		case "CHANGES_REQUESTED":
			parts = append(parts, paint("red", "wants changes"))
		case "DISMISSED":
			parts = append(parts, grey("dismissed"))
		}
		parts = append(parts, grey(span(time.Since(c.At))+" ago"))
		out.WriteString("  " + strings.Join(parts, "  ") + "\n")
		if body := plainBody(c.Body, hide); body != "" {
			out.WriteString(markdown(body))
		}
		out.WriteString("\n")
	}

	open := 0
	for _, t := range d.Threads {
		if !t.Resolved {
			open++
		}
	}
	id := fmt.Sprintf("%s#%d", d.Repo, d.Number)
	if tty {
		id = hyperlink(prURL(pr.PR{Repo: d.Repo, Number: d.Number}), id)
	}
	out.WriteString("\n  " + paint("bold", d.Title) + "\n  " + id + grey("  ·  ") +
		plural(open, "open thread", "open threads") + grey("  ·  ") +
		strconv.Itoa(len(d.Threads)-open) + " resolved\n\n")

	machines := 0
	if !unresolved {
		for _, c := range d.Conversation {
			// What a machine posts under a pull request is a build link or a
			// badge. One that reviews does it in the threads, which stay.
			if !all && (c.Bot || stackComment.MatchString(c.Body)) {
				machines++
				continue
			}
			said(c)
		}
	}
	for _, t := range d.Threads {
		if unresolved && t.Resolved {
			continue
		}
		where := t.Path
		if t.Line != 0 {
			where += ":" + strconv.Itoa(t.Line)
		}
		state := paint("red", "open")
		if t.Resolved {
			state = paint("green", "resolved")
		}
		if t.Outdated {
			state += grey("  outdated")
		}
		out.WriteString("  " + grey("── ") + paint("blue", where) + "  " + state + "\n")
		if hunk := hunkTail(t.Hunk); hunk != "" {
			out.WriteString(markdown("```diff\n" + hunk + "\n```"))
		}
		out.WriteString("\n")
		for _, c := range t.Comments {
			said(c)
		}
	}
	if len(d.Conversation) == machines && len(d.Threads) == 0 {
		out.WriteString(grey("  Nothing said yet.") + "\n")
	}
	if machines > 0 {
		out.WriteString(grey("  "+plural(machines, "comment", "comments")+" from bots left out; --all shows them") + "\n")
	}

	if !tty {
		_, err := os.Stdout.Write(out.Bytes())
		return err
	}
	return page(out.Bytes())
}

// Graphite posts the stack a pull request is in as a comment by its author.
var stackComment = regexp.MustCompile(`(?i)this stack of pull requests is managed by`)

func or(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// plainBody is a comment as Markdown the renderer draws well, with its
// pictures as links: a list of comments is read in the pager, which cannot
// draw them.
func plainBody(body string, hide config.Read) string {
	text, pictures := prepare(body, hide)
	text = markerText.ReplaceAllStringFunc(text, func(m string) string {
		n, _ := strconv.Atoi(markerText.FindStringSubmatch(m)[1])
		if n >= len(pictures) {
			return ""
		}
		return "[" + or(pictures[n].Alt, "image") + "](" + pictures[n].URL + ")"
	})
	return strings.TrimSpace(text)
}

// hunkTail is the end of a diff hunk, without the header when there are
// lines enough without it.
func hunkTail(hunk string) string {
	lines := strings.Split(strings.TrimRight(hunk, "\n"), "\n")
	if len(lines) > 1 && strings.HasPrefix(lines[0], "@@") {
		lines = lines[1:]
	}
	if len(lines) > hunkLines {
		lines = lines[len(lines)-hunkLines:]
	}
	return strings.Join(lines, "\n")
}
