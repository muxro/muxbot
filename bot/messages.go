package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// SendReplyComplex is the base for SendReply and SendReplyEmbed
func (b *Bot) SendReplyComplex(msg *discordgo.Message, data *discordgo.MessageSend) (*discordgo.Message, error) {
	var existing *discordgo.Message
	for _, pair := range b.replyHist.msgs {
		if pair != nil && pair[0].ID == msg.ID {
			existing = pair[1]
			break
		}
	}

	if data.Content != "" {
		author := strings.Split(msg.Author.String(), "#")[0]
		data.Content = fmt.Sprintf("(%s) %s", author, data.Content)
		data.Embed = nil
	}
	var replyMsg *discordgo.Message
	var err error
	if existing != nil {
		replyMsg, err = b.Disco.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID: existing.ID, Channel: existing.ChannelID,
			Content: &data.Content,
			Embed:   data.Embed,
		})
	} else {
		replyMsg, err = b.Disco.ChannelMessageSendComplex(msg.ChannelID, data)
	}
	if err != nil {
		return nil, err
	}

	b.replyHist.Add(msg, replyMsg)
	return replyMsg, err
}

// SendReply sends a msg with the username in parentheses at the start
func (b *Bot) SendReply(msg *discordgo.Message, reply string) (*discordgo.Message, error) {
	return b.SendReplyComplex(msg, &discordgo.MessageSend{Content: reply})
}
