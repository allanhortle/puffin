package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"golang.org/x/term"

	"github.com/allanhortle/puffin/internal/config"
	"github.com/allanhortle/puffin/internal/git"
	"github.com/allanhortle/puffin/internal/github"
	"github.com/allanhortle/puffin/internal/paths"
	"github.com/allanhortle/puffin/internal/pr"
)

const (
	readTimeout = 30 * time.Second
	// The most a description's image may weigh, and the most rows it may take.
	imageLimit   = 10 << 20
	imageMaxRows = 30
	readMaxWidth = 100
)

// read shows a pull request's title, who is on it, how big it is, and its
// description, rendered: what to know before reading the diff.
func read(args []string) error {
	flags, rest, err := parse(args)
	if err != nil {
		return err
	}
	_, links := flags["--links"]
	target := ""
	if len(rest) > 0 {
		target = rest[0]
	}

	ctx, cancel := timeout(readTimeout)
	defer cancel()
	repo, number, status, err := readTarget(ctx, target)
	if err != nil {
		return err
	}
	gh := github.New()
	d, err := gh.Describe(ctx, repo, number)
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
	wrap := min(width-4, readMaxWidth)

	mode := noGraphics
	if tty && !links {
		mode = detectGraphics(ctx)
	}

	var hide []string
	if cfg, err := config.Load(paths.Config()); err == nil {
		hide = cfg.Read.Hide
	}
	body, images := prepare(d.Body, hide)
	rendered := paint("colour242", "  No description.") + "\n"
	if strings.TrimSpace(body) != "" {
		style := styles.DarkStyle
		if !tty {
			style = styles.NoTTYStyle
		}
		r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(wrap))
		if err != nil {
			return err
		}
		if rendered, err = r.Render(body); err != nil {
			return err
		}
	}

	var out bytes.Buffer
	out.WriteString(readHeader(d, status, mode != noGraphics))
	cell := cellSize(ctx)
	for _, line := range strings.SplitAfter(rendered, "\n") {
		if n, ok := pictureAt(line); ok && n < len(images) {
			drawImage(ctx, &out, gh, images[n], mode, cell, wrap)
			continue
		}
		out.WriteString(line)
	}

	// An image is drawn where the cursor is, so a pager, which redraws the
	// screen as text, would lose every one.
	if mode != noGraphics || !tty {
		_, err := os.Stdout.Write(out.Bytes())
		return err
	}
	return page(out.Bytes())
}

// readTarget resolves what to read: a number, a GitHub or Graphite link, or
// with nothing named, the pull request of the branch checked out here.
func readTarget(ctx context.Context, target string) (repo string, number int, status *pr.PR, err error) {
	if m := prLink.FindStringSubmatch(target); m != nil {
		n, _ := strconv.Atoi(m[2])
		return m[1], n, nil, nil
	}
	query := url.Values{"target": {target}}
	if target == "" {
		if query, err = here(ctx); err != nil {
			return "", 0, nil, err
		}
	}
	if found, err := daemonClient().PR(ctx, query); err == nil {
		p := found.PR
		return p.Repo, p.Number, &p, nil
	}
	// Not one the daemon holds: somebody else's that nobody asked me to
	// review. A number is then a number in the repository I am standing in,
	// or failing that the one the config names as the default.
	n, convErr := strconv.Atoi(strings.TrimPrefix(target, "#"))
	if convErr != nil {
		if target == "" {
			return "", 0, nil, errors.New("this branch has no pull request puffin knows of; name one")
		}
		return "", 0, nil, fmt.Errorf("no pull request %q", target)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", 0, nil, err
	}
	if info := git.Inspect(ctx, dir); info != nil {
		return git.RepoName(ctx, info.CommonDir), n, nil, nil
	}
	if cfg, err := config.Load(paths.Config()); err == nil && cfg.DefaultRepo != "" {
		return cfg.DefaultRepo, n, nil, nil
	}
	return "", 0, nil, fmt.Errorf("pull request %d of which repository? Run this in a clone, or give its link", n)
}

var prLink = regexp.MustCompile(`(?:github\.com/|/pr/)([\w.-]+/[\w.-]+?)(?:/pull)?/(\d+)`)

func readHeader(d github.Description, status *pr.PR, links bool) string {
	var b strings.Builder
	grey := func(s string) string { return paint("colour242", s) }
	b.WriteString("\n  " + paint("bold", d.Title) + "\n")

	id := fmt.Sprintf("%s#%d", d.Repo, d.Number)
	if links {
		id = hyperlink(prURL(pr.PR{Repo: d.Repo, Number: d.Number}), id)
	}
	parts := []string{id, d.Author}
	switch {
	case status != nil:
		label := status.Label()
		parts = append(parts, paint(label.Colour, label.Word))
	case d.State != "OPEN":
		parts = append(parts, strings.ToLower(d.State))
	case d.IsDraft:
		parts = append(parts, "draft")
	}
	parts = append(parts, "opened "+span(time.Since(d.CreatedAt))+" ago")
	b.WriteString("  " + strings.Join(parts, grey("  ·  ")) + "\n")

	b.WriteString("  " + grey(d.Head+" → "+d.Base) + "  " +
		paint("green", "+"+strconv.Itoa(d.Additions)) + " " + paint("red", "−"+strconv.Itoa(d.Deletions)) +
		grey("  "+plural(d.ChangedFiles, "file", "files")) + "\n")

	var who []string
	for _, r := range d.Reviews {
		colour, word := "colour242", "commented"
		switch r.State {
		case "APPROVED":
			colour, word = "green", "approved"
		case "CHANGES_REQUESTED":
			colour, word = "red", "wants changes"
		case "DISMISSED":
			word = "dismissed"
		}
		who = append(who, r.Login+" "+paint(colour, word))
	}
	if len(d.Asked) > 0 {
		who = append(who, grey("asked: ")+strings.Join(d.Asked, ", "))
	}
	if len(who) > 0 {
		b.WriteString("  " + strings.Join(who, grey("  ·  ")) + "\n")
	}
	if len(d.Labels) > 0 {
		b.WriteString("  " + grey(strings.Join(d.Labels, ", ")) + "\n")
	}
	return b.String()
}

func hyperlink(link, text string) string {
	return "\x1b]8;;" + link + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// page hands the text to the pager, which leaves at once when it fits.
func page(text []byte) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -RF"
	}
	cmd := exec.Command("sh", "-c", pager)
	cmd.Stdin = bytes.NewReader(text)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// A picture in a description, and where it was.
type picture struct {
	Alt, URL string
}

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	imgTag      = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	attr        = regexp.MustCompile(`(?i)\b(src|alt)\s*=\s*"([^"]*)"`)
	summaryTag  = regexp.MustCompile(`(?is)<summary>(.*?)</summary>`)
	breakTag    = regexp.MustCompile(`(?i)<br\s*/?>`)
	layoutTag   = regexp.MustCompile(`(?i)</?(details|p|div|sub|sup|picture|source|center|kbd)\b[^>]*>`)
	mdImage     = regexp.MustCompile(`!\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	// GitHub writes a pasted file as its bare link on a line of its own.
	bareAttachment = regexp.MustCompile(`^\s*(https://github\.com/user-attachments/assets/[\w-]+)\s*$`)
	markerText     = regexp.MustCompile(`PUFFINIMAGE(\d+)X`)
)

// prepare turns a description into Markdown the renderer draws well: blocks
// named in hide go, the HTML that pull request templates leave behind goes,
// and each picture is
// taken out and left as a marker on a line of its own, to be drawn in its
// place. A picture inside a table is drawn once the table ends.
func prepare(body string, hide []string) (string, []picture) {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	for _, name := range hide {
		n := regexp.QuoteMeta(strings.TrimSpace(name))
		block := regexp.MustCompile(`(?s)<!--\s*` + n + `\s*-->.*?<!--\s*/` + n + `\s*-->`)
		body = block.ReplaceAllString(body, "")
	}
	body = htmlComment.ReplaceAllString(body, "")
	body = imgTag.ReplaceAllStringFunc(body, func(tag string) string {
		var src, alt string
		for _, m := range attr.FindAllStringSubmatch(tag, -1) {
			if strings.EqualFold(m[1], "src") {
				src = m[2]
			} else {
				alt = m[2]
			}
		}
		if src == "" {
			return ""
		}
		return "![" + alt + "](" + src + ")"
	})
	body = summaryTag.ReplaceAllString(body, "**$1**\n")
	body = breakTag.ReplaceAllString(body, "\n")
	body = layoutTag.ReplaceAllString(body, "")

	var pictures []picture
	var held []int
	var out []string
	add := func(p picture) int {
		pictures = append(pictures, p)
		return len(pictures) - 1
	}
	place := func(n int) string { return "\nPUFFINIMAGE" + strconv.Itoa(n) + "X\n" }
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenced = !fenced
		}
		if fenced {
			out = append(out, line)
			continue
		}
		inTable := strings.HasPrefix(trimmed, "|")
		if !inTable && len(held) > 0 {
			for _, n := range held {
				out = append(out, place(n))
			}
			held = nil
		}
		if m := bareAttachment.FindStringSubmatch(line); m != nil {
			out = append(out, place(add(picture{Alt: "attachment", URL: m[1]})))
			continue
		}
		line = mdImage.ReplaceAllStringFunc(line, func(s string) string {
			m := mdImage.FindStringSubmatch(s)
			n := add(picture{Alt: m[1], URL: m[2]})
			if inTable {
				held = append(held, n)
				alt := m[1]
				if alt == "" {
					alt = "image " + strconv.Itoa(n+1)
				}
				return "[" + alt + "](" + m[2] + ")"
			}
			return place(n)
		})
		out = append(out, line)
	}
	for _, n := range held {
		out = append(out, place(n))
	}
	return strings.Join(out, "\n"), pictures
}

// pictureAt is the picture a rendered line stands in for, if it is one.
func pictureAt(line string) (int, bool) {
	m := markerText.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

type graphics int

const (
	noGraphics graphics = iota
	// Written as they are: a Kitty terminal with no tmux in between, or a
	// tmux that understands Kitty images itself and keeps them in its panes.
	directGraphics
	// Smuggled past a tmux that does not, straight to the terminal. tmux
	// does not know they are there: they do not stay inside the pane, and a
	// redraw loses them. Once tmux understands them, this goes.
	passthroughGraphics
)

func detectGraphics(ctx context.Context) graphics {
	if os.Getenv("TMUX") == "" {
		t := os.Getenv("TERM")
		if os.Getenv("TERM_PROGRAM") == "ghostty" || t == "xterm-ghostty" || t == "xterm-kitty" ||
			os.Getenv("KITTY_WINDOW_ID") != "" {
			return directGraphics
		}
		return noGraphics
	}
	out, err := exec.CommandContext(ctx, "tmux", "display", "-p", "-t", os.Getenv("TMUX_PANE"),
		"#{image_support}|#{client_termfeatures}|#{client_termname}|#{allow-passthrough}").Output()
	if err != nil {
		return noGraphics
	}
	f := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(f) != 4 {
		return noGraphics
	}
	if f[0] == "1" && strings.Contains(","+f[1]+",", ",kittygraphics,") {
		return directGraphics
	}
	kitty := strings.Contains(f[2], "ghostty") || strings.Contains(f[2], "kitty")
	if kitty && (f[3] == "on" || f[3] == "all") {
		return passthroughGraphics
	}
	return noGraphics
}

// cellSize is how many pixels one character cell is across and down.
func cellSize(ctx context.Context) [2]int {
	if os.Getenv("TMUX") != "" {
		out, err := exec.CommandContext(ctx, "tmux", "display", "-p", "-t", os.Getenv("TMUX_PANE"),
			"#{client_cell_width} #{client_cell_height}").Output()
		if err == nil {
			var w, h int
			if n, _ := fmt.Sscan(string(out), &w, &h); n == 2 && w > 0 && h > 0 {
				return [2]int{w, h}
			}
		}
	}
	return [2]int{8, 16}
}

// drawImage puts a picture where its marker was, or its link if it cannot be
// drawn here.
func drawImage(ctx context.Context, out *bytes.Buffer, gh *github.Client, p picture, mode graphics, cell [2]int, wrap int) {
	asLink := func() {
		alt := p.Alt
		if alt == "" {
			alt = "image"
		}
		out.WriteString("  " + paint("colour242", "▣ "+alt+"  "+p.URL) + "\n\n")
	}
	if mode == noGraphics {
		asLink()
		return
	}
	data, err := gh.Download(ctx, p.URL, imageLimit)
	if err != nil {
		asLink()
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width == 0 || config.Height == 0 {
		asLink() // a video, or a format there is no decoder for
		return
	}
	if format != "png" {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			asLink()
			return
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			asLink()
			return
		}
		data = buf.Bytes()
	}

	cols := min((config.Width+cell[0]-1)/cell[0], wrap)
	rows := func(cols int) int {
		return max(1, (cols*cell[0]*config.Height/config.Width+cell[1]-1)/cell[1])
	}
	r := rows(cols)
	if r > imageMaxRows {
		cols = max(1, cols*imageMaxRows/r)
		r = rows(cols)
	}
	writeImage(out, data, cols, r, mode)
}

// writeImage makes room for the picture with blank lines, which every layer
// in between understands, then goes back up and draws it there without
// moving the cursor, and comes down past it.
func writeImage(out io.Writer, data []byte, cols, rows int, mode graphics) {
	fmt.Fprint(out, strings.Repeat("\n", rows))
	fmt.Fprintf(out, "\x1b[%dA\r\x1b[2C", rows)
	encoded := base64.StdEncoding.EncodeToString(data)
	const chunk = 4096
	for i := 0; i < len(encoded); i += chunk {
		end := min(i+chunk, len(encoded))
		more := 0
		if end < len(encoded) {
			more = 1
		}
		control := fmt.Sprintf("m=%d", more)
		if i == 0 {
			control = fmt.Sprintf("a=T,f=100,q=2,C=1,c=%d,r=%d,m=%d", cols, rows, more)
		}
		seq := "\x1b_G" + control + ";" + encoded[i:end] + "\x1b\\"
		if mode == passthroughGraphics {
			seq = "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
		}
		io.WriteString(out, seq)
	}
	fmt.Fprintf(out, "\r\x1b[%dB\n", rows)
}
