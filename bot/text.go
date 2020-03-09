package bot

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Text struct {
	Header     string
	Content    string
	Footer     string
	Raw        bool
	Quoted     bool
	QuoteType  string
	Pagination bool
	Pastebin   bool
}

var _ Content = Text{}

func (s Text) ToMessage(ctx context.Context, replyTo *discordgo.Message) Message {
	s.Content = strings.TrimSpace(s.Content)
	s.Header = strings.TrimSpace(s.Header)
	s.Footer = strings.TrimSpace(s.Footer)
	s.QuoteType = Escape(strings.TrimSpace(s.QuoteType))

	if replyTo != nil {
		s.Header = addContext("("+Escape(replyTo.Author.Username)+")", s.Header)
	}

	//orig := s.Content
	if !s.Raw {
		s.Content = Escape(s.Content)
	}

	// string is small enough to just sent as-is
	if s.calcSize() <= 2000 {
		return textMessage(s.compose())
	}

	if !s.Pagination {
		s.Header = addContext(s.Header, "message was trimmed")
	}

	// if the string is too big, pastebin it if we're asked to
	//var pasteURL string
	//if s.Pastebin {
	//	return Delayed(&DelayedConfig{Name: "uploading", ReplyTo: replyTo.Author.Username}, func() Content {
	//		pasteURL, _ = pastebin(orig)
	//		if len(pasteURL) > 0 {
	//			s.Header = addContext(s.Header, "full version @ <"+pasteURL+">")
	//		} else {
	//			s.Header = addContext(s.Header, "failed to upload paste")
	//		}

	//		return s.trimmedOrPaginated(replyTo)
	//	})
	//}

	return s.trimmedOrPaginated(replyTo)
}

func (s *Text) trimmedOrPaginated(replyTo *discordgo.Message) Message {
	// paginate response
	//if s.Pagination {
	//	return paginateText(replyTo, s)
	//}

	// something bad could happen when trimming a raw string
	if s.Raw {
		log.Println("WARNING: trimmed raw string")
	}

	// set trim size
	maxSize := len(s.Content) - (s.calcSize() - 2000)

	// trim possible escapes
	s.Content = strings.TrimRight(s.Content[:maxSize], "\\")

	return textMessage(s.compose())

}

func (s *Text) calcSize() int {
	size := len(s.Header) + len(s.Content) + len(s.Footer)
	if size == 0 {
		return 0
	}

	// quoted strings don't require separators
	if s.Quoted {
		// quote type + separator
		if len(s.QuoteType) > 0 {
			size += len(s.QuoteType) + 1
		}

		// 6 backticks
		return size + 6
	}

	// do we need a header separator?
	if len(s.Header) > 0 {
		size += 1
	}

	// do we need a footer separator
	if len(s.Footer) > 0 {
		size += 1
	}

	return size
}

func (s *Text) compose() string {
	header := s.Header
	content := s.Content
	footer := s.Footer

	if s.Quoted {
		if len(s.QuoteType) > 0 {
			content = s.QuoteType + "\n" + content
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
