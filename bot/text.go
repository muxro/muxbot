package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

type Text struct {
	Header  string
	Content string
	Footer  string

	Raw        bool
	Quoted     bool
	QuoteType  string
	Pagination bool
	Pastebin   bool
}

type textMessage string

func (tm textMessage) Content() string                { return string(tm) }
func (tm textMessage) Embed() *discordgo.MessageEmbed { return nil }

var _ Content = Text{}

func (t Text) ToMessage(ctx context.Context, replyTo *discordgo.Message) Message {
	t.Content = strings.TrimSpace(t.Content)
	t.Header = strings.TrimSpace(t.Header)
	t.Footer = strings.TrimSpace(t.Footer)
	t.QuoteType = Escape(strings.TrimSpace(t.QuoteType), true)

	if replyTo != nil {
		t.Header = addContext("("+Escape(replyTo.Author.Username, false)+")", t.Header)
	}

	orig := t.Content
	if !t.Raw {
		t.Content = Escape(t.Content, t.Quoted)
	}

	// string is small enough to just sent as-is
	if t.calcSize() <= 2000 {
		return textMessage(t.compose())
	}

	if !t.Pagination {
		t.Header = addContext(t.Header, "message was trimmed")
	}

	//// if the string is too big, pastebin it if we're asked to
	var pasteURL string
	if t.Pastebin {
		fmt.Println("pastebin")
		//return DelayedMessage(ctx, &DelayedConfig{Name: "uploading"}, func(replyTo *discordgo.Message) Message {
		pasteURL, _ = pastebin(ctx, orig)
		if len(pasteURL) > 0 {
			t.Header = addContext(t.Header, "full version @ <"+pasteURL+">")
		} else {
			t.Header = addContext(t.Header, "failed to upload paste")
		}

		//return t.trimmedOrPaginated(replyTo)
		//})
	}

	// paginate response
	if t.Pagination {
		return t.paginated(ctx, replyTo)
	}

	// something bad could happen when trimming a raw string
	if t.Raw {
		log.Println("WARNING: trimmed raw string")
	}

	maxSize := 2000 - (t.calcSize() - len(t.Content))
	parts := splitText(maxSize, t.Content)
	t.Content = parts[0]

	return textMessage(t.compose())
}

func (t Text) paginated(ctx context.Context, replyTo *discordgo.Message) Message {
	origFoot := t.Footer
	t.Footer = addContext("Page 00/00", origFoot)

	maxSize := 2000 - (t.calcSize() - len(t.Content))
	parts := splitText(maxSize, t.Content)

	pages := make([]Content, 0, len(parts))
	for i, part := range parts {
		text := Text{
			Header:    t.Header,
			Content:   part,
			Footer:    addContext(fmt.Sprintf("Page %d/%d", i+1, len(parts)), origFoot),
			Quoted:    t.Quoted,
			QuoteType: t.QuoteType,
		}
		content := text.compose()

		page := MessageFunc(func(ctx context.Context, replyTo *discordgo.Message) Message {
			return textMessage(content)
		})

		pages = append(pages, page)
	}

	p := Paginator{
		Pager: ContentSlice(pages),
	}

	return p.ToMessage(ctx, replyTo)
}

func (t *Text) calcSize() int {
	size := len(t.Header) + len(t.Content) + len(t.Footer)
	if size == 0 {
		return 0
	}

	// quoted strings don't require separators
	if t.Quoted {
		// quote type + separator
		if len(t.QuoteType) > 0 {
			size += len(t.QuoteType) + 1
		}

		// 6 backticks
		return size + 6
	}

	// do we need a header separator?
	if len(t.Header) > 0 {
		size += 1
	}

	// do we need a footer separator
	if len(t.Footer) > 0 {
		size += 1
	}

	return size
}

func (t *Text) compose() string {
	header := t.Header
	content := t.Content
	footer := t.Footer

	if t.Quoted {
		if len(t.QuoteType) > 0 {
			content = t.QuoteType + "\n" + content
		}
		content = "```" + content + "```"

		return header + content + footer
	}

	if len(header) > 0 {
		if strings.ContainsRune(content, '\n') || len(footer) > 0 {
			header += "\n"
		} else {
			header += " "
		}
	}

	if len(footer) > 0 {
		footer = "\n" + footer
	}

	return header + content + footer
}

func addContext(ctx string, msg string) string {
	if len(ctx) == 0 {
		return msg
	}
	if len(msg) == 0 {
		return ctx
	}

	if ctx[len(ctx)-1] == ')' {
		return ctx + " " + msg
	}

	return ctx + " | " + msg
}

func splitText(max int, text string) []string {
	var parts []string
	for len(text) > 0 {
		split := findSplit(max, text)
		part := strings.TrimSpace(text[:split])
		text = strings.TrimSpace(text[split:])

		if len(part) == 0 {
			continue
		}

		parts = append(parts, part)
	}

	return parts
}

func findSplit(max int, text string) int {
	if len(text) <= max {
		return len(text)
	}

	maxBreak := (max * 90) / 100

	bestPrio, pos := 0, max
	for i := max; i >= maxBreak; i-- {
		// skip extra utf8 bytes
		if text[i]&0xC0 == 0x80 {
			continue
		}

		if isEscaped(text, i) {
			continue
		}

		r, size := utf8.DecodeRune([]byte(text[i:]))
		if i+size > max {
			continue
		}

		prio := splitPrio(r)
		if prio > bestPrio {
			bestPrio, pos = prio, i+size
		}

		if prio == 100 {
			break
		}
	}

	return pos
}

func splitPrio(r rune) int {
	if r == '\n' || r == '\r' {
		return 100
	}

	if r == '\t' {
		return 3
	}

	if unicode.IsSpace(r) {
		return 2
	}

	return 1
}

func isEscaped(text string, pos int) bool {
	r := text[pos]
	if r == '\n' || r == '\r' {
		return false
	}

	var count int
	for i := pos - 1; i >= 0; i-- {
		if text[i] != '\\' {
			break
		}

		count++
	}

	return count%2 == 1
}
