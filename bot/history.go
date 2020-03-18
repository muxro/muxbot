package bot

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

// guildHistory stores the last [no] messages per channel that the bot sent
type history struct {
	maxPerChannel int

	mu    sync.Mutex
	chans map[string]*chanHistory
}

type chanHistory struct {
	no   int
	msgs []*message
}

type message struct {
	sync.Mutex

	Message Message
	ReplyTo *discordgo.Message
	Sent    *discordgo.Message

	removed bool
	cancel  func()
}

// newMessageHistory creates a history instance with `maxPerChannel` possible slots
func newMessageHistory(maxPerChannel int) *history {
	return &history{
		maxPerChannel: maxPerChannel,
		chans:         map[string]*chanHistory{},
	}
}

func (mh *history) channel(id string) *chanHistory {
	ch, ok := mh.chans[id]
	if !ok {
		ch = &chanHistory{
			msgs: make([]*message, 0, mh.maxPerChannel),
		}
		mh.chans[id] = ch
	}

	return ch
}

func (mh *history) Add(msg *message) *message {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	if msg.ReplyTo != nil {
		return mh.addReply(msg)
	} else if msg.Message != nil {
		return mh.addMessage(msg)
	}

	panic("invalid message")
}

func (mh *history) addReply(msg *message) *message {
	ch := mh.channel(msg.ReplyTo.ChannelID)

	// check if the message already exists, and cancel the old one
	for i, e := range ch.msgs {
		if e.ReplyTo.ID == msg.ReplyTo.ID {
			e.Lock()
			msg.Sent = e.Sent
			msg.Message = e.Message
			e.removed = true
			e.cancel()
			e.Unlock()

			ch.msgs[i] = msg
			return e
		}
	}

	addHistory(mh, ch, msg)
	return nil
}

func (mh *history) addMessage(msg *message) *message {
	ch := mh.channel(msg.Sent.ChannelID)

	// check if the message already exists, and cancel the old one
	for i, e := range ch.msgs {
		if e.Sent.ID == msg.Sent.ID {
			e.Lock()
			msg.ReplyTo = e.ReplyTo
			msg.Message = e.Message
			e.removed = true
			e.cancel()
			e.Unlock()

			ch.msgs[i] = msg
			return e
		}
	}

	addHistory(mh, ch, msg)
	return nil
}

func addHistory(mh *history, ch *chanHistory, m *message) {
	if len(ch.msgs) < mh.maxPerChannel {
		ch.msgs = append(ch.msgs, m)
		return
	}

	ch.msgs[ch.no] = m
	ch.no = (ch.no + 1) % mh.maxPerChannel
}

func (mh *history) GetMessage(chanID, id string) *message {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	ch, ok := mh.chans[chanID]
	if !ok {
		return nil
	}

	for _, m := range ch.msgs {
		m.Lock()
		mID := m.Sent.ID
		m.Unlock()
		if mID == id {
			return m
		}
	}

	return nil
}

//func (mh *history) GetReplyTo(chanID, id string) *message {
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
