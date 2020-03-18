package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type Pager interface {
	Page(page uint) Content
	TotalPages() uint
}

type Paginator struct {
	Pager    Pager
	Page     uint
	OnlyUser string
}

var _ Content = Paginator{}

func (pg Paginator) ToMessage(ctx context.Context, replyTo *discordgo.Message) Message {
	p := &paginated{
		Paginator:  &pg,
		totalPages: pg.Pager.TotalPages(),
		replyTo:    replyTo,
	}

	p.setPage(ctx, 0)
	return p
}

type paginated struct {
	*Paginator
	totalPages uint
	replyTo    *discordgo.Message
	Message
}

var _ OnSender = &paginated{}
var _ OnReactAdder = &paginated{}

func (p *paginated) OnSend(ctx context.Context, msg *discordgo.Message) error {
	emojis := []string{"◀️", "▶️"}

	b := FromContext(ctx)
	for _, emoji := range emojis {
		b.Disco.MessageReactionAdd(msg.ChannelID, msg.ID, emoji)
	}

	return nil
}

func (p *paginated) OnReactAdd(ctx context.Context, react *discordgo.MessageReaction) error {
	page := p.Page
	switch react.Emoji.APIName() {
	case "◀️":
		page -= 1
	case "▶️":
		page += 1
	default:
		return nil
	}

	b := FromContext(ctx)
	b.Disco.MessageReactionRemove(react.ChannelID, react.MessageID, react.Emoji.APIName(), react.UserID)

	if len(p.OnlyUser) > 0 && react.UserID != p.OnlyUser {
		return nil
	}

	if page < 0 || (p.totalPages > 0 && page >= p.totalPages) {
		return nil
	}

	p.setPage(ctx, page)
	return SendMessage(ctx, p)
}

func (p *paginated) setPage(ctx context.Context, page uint) {
	content := p.Pager.Page(page)
	p.Message = content.ToMessage(ctx, p.replyTo)
	p.Page = page
}

type ContentSlice []Content

var _ Pager = ContentSlice{}

func (cs ContentSlice) Page(page uint) Content {
	return cs[int(page)]
}

func (cs ContentSlice) TotalPages() uint {
	return uint(len(cs))
}
