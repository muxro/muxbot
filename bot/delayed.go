package bot

import (
	"context"
	"log"
	"strings"
	"time"

	"errors"
)

type DelayedConfig struct {
	Name       string
	Animation  []string
	Wait       time.Duration
	UpdateRate time.Duration
	NoAnimate  bool
}

var defWaitAnim = []string{"🕛", "🕐", "🕑", "🕒", "🕓", "🕔", "🕕", "🕖", "🕗", "🕘", "🕙", "🕚"}

func Delayed(ctx context.Context, conf *DelayedConfig, fn func() Content) Content {
	var config DelayedConfig
	if conf != nil {
		config = *conf
	}

	config.Name = strings.TrimSpace(config.Name)
	if len(config.Name) == 0 {
		config.Name = "working"
	}

	if len(config.Animation) == 0 {
		config.Animation = defWaitAnim
	}

	if config.Wait <= 0 {
		config.UpdateRate = 1500 * time.Millisecond
	}

	if config.UpdateRate <= 0 {
		config.UpdateRate = 1500 * time.Millisecond
	}

	done := make(chan bool)

	go delayedProgress(ctx, &config, done)

	content := fn()
	log.Println("got content")
	close(done)

	return content
}

func delayedProgress(ctx context.Context, conf *DelayedConfig, done chan bool) {
	if conf.Wait > 0 {
		select {
		case <-time.After(conf.Wait):
		case <-done:
			return
		}
		log.Println("waited grace period")
	}

	if conf.NoAnimate {
		log.Println("not animated")
		Reply(ctx, Text{Content: conf.Name + "..."})
		return
	}

	tick := time.NewTicker(conf.UpdateRate)
	defer tick.Stop()

	for i := 0; ; i++ {
		log.Println("tick")
		anim := conf.Animation[i%len(conf.Animation)]
		points := "..."[:(i+1)%4]
		content := conf.Name + " " + anim + " " + points
		Reply(ctx, Text{Content: content})

		select {
		case <-ctx.Done():
			log.Println("canceled")
			err := ctx.Err()
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if errors.Is(err, context.DeadlineExceeded) {
				Reply(ctx, Text{Content: "timed out"})
			} else if errors.Is(err, context.Canceled) {
				Reply(ctx, Text{Content: "canceled"})
			}
			cancel()
			return

		case <-done:
			log.Println("exited")
			return

		case <-tick.C:
			// just continue with the loop
		}

	}
}
