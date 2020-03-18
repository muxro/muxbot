package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"gitlab.com/muxro/muxbot/addons"
	//_ "gitlab.com/muxro/muxbot/addons/simple-commands"
	_ "gitlab.com/muxro/muxbot/addons/eval"
	_ "gitlab.com/muxro/muxbot/addons/test"

	//_ "gitlab.com/muxro/muxbot/addons/web-search"
	"gitlab.com/muxro/muxbot/bot"
)

var token = flag.String("token", "", "Specify the token")

func main() {
	flag.Parse()

	bot, err := bot.New(bot.Config{
		Token:  *token,
		Prefix: ",",
	})
	if err != nil {
		panic(err)
	}

	err = addons.Add(bot)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	err = bot.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected")

	select {}
}
