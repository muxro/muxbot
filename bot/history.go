package bot

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

// msgHistory stores the last [no] messages per channel that the bot sent
type msgHistory struct {
	max int

	mu    sync.Mutex
	chans map[string]*chanHistory
}

type chanHistory struct {
	no   int
	msgs []*message
}

// newMessageHistory creates a msgHistory instance with `max` possible slots
func newMessageHistory(max int) *msgHistory {
	return &msgHistory{
		max:   max,
		chans: map[string]*chanHistory{},
	}
}

type message struct {
	ReplyTo *discordgo.Message

	mu      sync.Mutex
	Sent    *discordgo.Message
	Message Message
	removed bool
}

func (mh *msgHistory) Add(replyTo *discordgo.Message) *message {
	m := &message{
		ReplyTo: replyTo,
	}

	mh.mu.Lock()
	defer mh.mu.Unlock()

	ch, ok := mh.chans[replyTo.ChannelID]
	if !ok {
		ch = &chanHistory{
			msgs: make([]*message, 0, mh.max),
		}
		mh.chans[replyTo.ChannelID] = ch
	}

	for i, e := range ch.msgs {
		if e.ReplyTo.ID == replyTo.ID {
			m.Sent = e.Sent
			e.mu.Lock()
			e.removed = true
			e.mu.Unlock()
			ch.msgs[i] = m
			return m
		}
	}

	if len(ch.msgs) < mh.max {
		ch.msgs = append(ch.msgs, m)
		return m
	}

	ch.msgs[ch.no] = m
	ch.no = (ch.no + 1) % mh.max
	return m
}

func (mh *msgHistory) GetMessage(chanID, id string) *message {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	ch, ok := mh.chans[chanID]
	if !ok {
		return nil
	}

	for _, m := range ch.msgs {
		m.mu.Lock()
		mID := m.Sent.ID
		m.mu.Unlock()
		if mID == id {
			return m
		}
	}

	return nil
}

//func (mh *msgHistory) GetReplyTo(chanID, id string) *message {
//	mh.mu.Lock()
//	defer mh.mu.Unlock()
//
//	ch, ok := mh.chans[chanID]
//	if !ok {
//		return nil
//	}
//
//	for _, n := range ch.msgs {
//		if n.ReplyID == id {
//			return n
//		}
//	}
//
//	return nil
//}
