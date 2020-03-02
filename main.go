package main

import (
	"context"
	"flag"

	"gitlab.com/muxro/muxbot/addons"
	_ "gitlab.com/muxro/muxbot/addons/simple-commands"
	_ "gitlab.com/muxro/muxbot/addons/web-search"
	"gitlab.com/muxro/muxbot/bot"
)

var token = flag.String("token", "", "Specify the token")

func main() {
	ctx := context.Background()
	bot, err := bot.New(ctx, bot.Config{
		Token:  *token,
		Prefix: ",",
	})
	if err != nil {
		panic(err)
	}

	addons.Add(bot)

	bot.Start()

	select {}
}
