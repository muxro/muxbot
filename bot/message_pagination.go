package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type Pager interface {
	Page(page int) Content
}

type Paginator struct {
	pager      Pager
	page       int
	totalPages int

	onlyUser string

	Message
}

var _ OnSender = &Paginator{}
var _ OnReactAdder = &Paginator{}

func (p *Paginator) OnSend(ctx context.Context, b *Bot, msg *discordgo.Message) error {
	emojis := []string{"◀️", "▶️"}

	for _, emoji := range emojis {
		b.Disco.MessageReactionAdd(msg.ChannelID, msg.ID, emoji)
	}

	return nil
}

func (p *Paginator) OnReactAdd(ctx context.Context, b *Bot, react *discordgo.MessageReaction) error {
	page := p.page
	switch react.Emoji.APIName() {
	case "◀️":
		page -= 1
	case "▶️":
		page += 1
	default:
		return nil
	}

	b.Disco.MessageReactionRemove(react.ChannelID, react.MessageID, react.Emoji.APIName(), react.UserID)

	if len(p.onlyUser) > 0 && react.UserID != p.onlyUser {
		return nil
	}

	if page < 0 || (p.totalPages > 0 && page >= p.totalPages) {
		return nil
	}

	content := p.pager.Page(page)
	// TODO: need replyTo here
	msg := content.ToMessage(ctx, nil)
	p.Message = msg
	p.page = page

	//b.EditMessage(ctx, react.ChannelID, react.MessageID, p)
	return nil
}

type stringPager []*Text

func (sp stringPager) Page(page int) Content {
	return sp[page]
}

func paginateText(replyTo string, s *Text) *Paginator {
	parts := make([]string, 0, len(s.Content)/1000)
	for i := 0; i < len(s.Content); i += 1000 {
		upper := i + 1000
		if upper > len(s.Content) {
			upper = len(s.Content)
		}

		parts = append(parts, s.Content[i:upper])
	}

	sparts := make(stringPager, 0, len(parts))
	for i := range parts {
		sparts = append(sparts, &Text{
			Header:    s.Header,
			Content:   parts[i],
			Footer:    fmt.Sprintf("Page %d/%d", i+1, len(parts)),
			Quoted:    s.Quoted,
			QuoteType: s.QuoteType,
		})
	}

	p := &Paginator{
		pager:      sparts,
		page:       0,
		totalPages: len(parts),
	}
	// TODO: need replyTo here
	ctx := context.TODO()
	p.Message = sparts[0].ToMessage(ctx, nil)

	return p
}
