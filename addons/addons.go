package addons

import (
	"fmt"

	"gitlab.com/muxro/muxbot/bot"
)

var addons = map[string]Addon{}

type Addon interface {
	Add(*bot.Bot) error
}

func Register(name string, addon Addon) {
	if _, ok := addons[name]; ok {
		panic(fmt.Sprintf("addon with name %s already exists", name))
	}

	addons[name] = addon
}

func Add(b *bot.Bot) error {
	for _, addon := range addons {
		err := addon.Add(b)
		if err != nil {
			return err
		}
	}

	return nil
}
