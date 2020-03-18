package bot

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/bwmarrin/discordgo"
)

type DelayedConfig struct {
	Name       string
	Animation  []string
	Wait       time.Duration
	UpdateRate time.Duration
	NoAnimate  bool
}

var defWaitAnim = []string{"🕛", "🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚"}

func Delayed(ctx context.Context, config *DelayedConfig, fn func() Content) Content {
	var conf DelayedConfig
	if config != nil {
		conf = *config
	}

	conf.Name = strings.TrimSpace(conf.Name)
	if len(conf.Name) == 0 {
		conf.Name = "working"
	}

	if len(conf.Animation) == 0 {
		conf.Animation = defWaitAnim
	}

	if conf.Wait <= 0 {
		conf.Wait = 500 * time.Millisecond
	}

	if conf.UpdateRate <= 0 {
		conf.UpdateRate = 1500 * time.Millisecond
	}

	cchan := make(chan Content)
	go func() { cchan <- fn() }()

	if conf.Wait > 0 {
		select {
		case <-time.After(conf.Wait):
			log.Println("waited grace period")

		case content := <-cchan:
			log.Println("returned before grace period")
			return content
		}
	}

	dm := &delayedMessage{
		conf:  &conf,
		cchan: cchan,
	}
	dm.genMessage(0)

	var once sync.Once
	return MessageFunc(func(ctx context.Context, replyTo *discordgo.Message) Message {
		once.Do(func() {
			dm.ctx = ctx

			go dm.update()
		})
		return dm
	})
}

//func DelayedMessage(ctx context.Context, config *DelayedConfig, fn func() Message) Message {
//	var conf DelayedConfig
//	if config != nil {
//		conf = *config
//	}
//
//	conf.Name = strings.TrimSpace(conf.Name)
//	if len(conf.Name) == 0 {
//		conf.Name = "working"
//	}
//
//	if len(conf.Animation) == 0 {
//		conf.Animation = defWaitAnim
//	}
//
//	if conf.Wait <= 0 {
//		conf.UpdateRate = 1500 * time.Millisecond
//	}
//
//	if conf.UpdateRate <= 0 {
//		conf.UpdateRate = 1500 * time.Millisecond
//	}
//
//	cchan := make(chan Message)
//	go func() { cchan <- fn() }()
//
//	if conf.Wait > 0 {
//		select {
//		case <-time.After(conf.Wait):
//			log.Println("waited grace period")
//
//		case content := <-cchan:
//			log.Println("returned before grace period")
//			return content
//		}
//	}
//
//	dm := &delayedMessage{
//		conf:  &conf,
//		cchan: cchan,
//	}
//	dm.genMessage(0)
//
//	var once sync.Once
//	return MessageFunc(func(ctx context.Context, replyTo *discordgo.Message) Message {
//		once.Do(func() {
//			dm.ctx = ctx
//
//			go dm.update()
//		})
//		return dm
//	})
//}

type delayedMessage struct {
	ctx     context.Context
	replyTo *discordgo.Message
	conf    *DelayedConfig
	cchan   chan Content

	Message
}

var _ OnEditer = &delayedMessage{}

func (dm *delayedMessage) OnEdit(ctx context.Context, dmsg *discordgo.Message, cur Message) error {
	if cur == dm {
		return nil
	}

	close(dm.cchan)
	return nil
}

func (dm *delayedMessage) genMessage(seq int) {
	conf := dm.conf
	if dm.conf.NoAnimate {
		dm.setContent(Text{Content: conf.Name + "..."})
		return
	}

	anim := conf.Animation[seq%len(conf.Animation)]
	points := "..."[:(seq+1)%4]
	content := conf.Name + " " + anim + " " + points
	dm.setContent(Text{Content: content})
}

func (dm *delayedMessage) setContent(content Content) {
	dm.Message = content.ToMessage(dm.ctx, dm.replyTo)
}

func (dm *delayedMessage) update() {
	if dm.conf.NoAnimate {
		return
	}

	tick := time.NewTicker(dm.conf.UpdateRate)
	defer tick.Stop()

	ctx := dm.ctx

	for i := 1; ; i++ {
		select {
		case <-ctx.Done():
			log.Println("canceled")
			err := ctx.Err()
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if errors.Is(err, context.DeadlineExceeded) {
				Send(ctx, Text{Content: "timed out"})
			} else if errors.Is(err, context.Canceled) {
				Send(ctx, Text{Content: "canceled"})
			}
			cancel()
			return

		case content, ok := <-dm.cchan:
			if !ok {
				log.Println("canceled")
				return
			}

			log.Println("got content")
			Send(ctx, content)
			return

		case <-tick.C:
			log.Println("tick")
			dm.genMessage(i)
			SendMessage(ctx, dm)
		}

	}
}
