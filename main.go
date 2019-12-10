package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"

	"github.com/Knetic/govaluate"

	"github.com/bwmarrin/discordgo"
)

func main() {
	tokenPtr := flag.String("token", "none", "Specify the token")
	flag.Parse()
	token := *tokenPtr

	if token == "none" {
		fmt.Println("You need to specify a token")
		return
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Could not open Discord session: ", err)
		return
	}

	fmt.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "MuxBot is Aliiive!")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {

	if message.Author.ID == session.State.User.ID {
		return
	}

	// It is a command
	if strings.HasPrefix(message.Content, ".") {
		commandMessage := strings.Join(strings.Split(message.Content, " ")[1:], " ")

		if strings.HasPrefix(message.Content, ".help") {
			sendMessageInChannel(`
.help - shows this
.ping - pong
.echo - echoes back whatever you send it
.calc - compute simple expression
			`, session, message.Message)
		} else if strings.HasPrefix(message.Content, ".g ") {
			res, err := ScrapeFirstWebRes(commandMessage)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Error: %v", err), session, message.Message)
				return
			}
			sendMessageInChannel(fmt.Sprintf("%s -- %s", res["url"], res["desc"]), session, message.Message)
		} else if strings.HasPrefix(message.Content, ".gis ") {
			res, err := ScrapeFirstImgRes(commandMessage)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Error: %v", err), session, message.Message)
				return
			}
			sendMessageInChannel(res, session, message.Message)
		} else if strings.HasPrefix(message.Content, ".ping") {
			sendMessageInChannel("pong", session, message.Message)
		} else if strings.HasPrefix(message.Content, ".echo") {
			if commandMessage != "" {
				sendMessageInChannel(commandMessage, session, message.Message)
			}
		} else if strings.HasPrefix(message.Content, ".calc") {
			expr, err := govaluate.NewEvaluableExpression(commandMessage)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Nu am putut calcula: Eroare %v", err), session, message.Message)
				return
			}
			result, err := expr.Evaluate(nil)
			if err != nil {
				sendMessageInChannel(fmt.Sprintf("Nu am putut calcula, eroare %v", err), session, message.Message)
				return
			}
			sendMessageInChannel(fmt.Sprintf("%v", result), session, message.Message)
		}
	}

}

func sendEmbedInChannel(embed *discordgo.MessageEmbed, session *discordgo.Session, channel string) {
	_, err := session.ChannelMessageSendEmbed(channel, embed)
	if err != nil {
		fmt.Println(err)
		return
	}

}

func sendMessageInChannel(message string, session *discordgo.Session, originalMessage *discordgo.Message) {
	session.ChannelMessageSend(originalMessage.ChannelID, fmt.Sprintf("(%s) %s", strings.Split(originalMessage.Author.String(), "#")[0], message))
}

func scrapeWeb(url string) (*goquery.Document, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 4.0.4; Galaxy Nexus Build/IMM76B) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.133 Mobile Safari/535.19")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// ScrapeFirstWebRes function scrapes dogpile.com for the first result of the query and returns it
func ScrapeFirstWebRes(q string) (map[string]string, error) {
	url := "https://www.dogpile.com/serp?qc=web&q=" + url.QueryEscape(q)
	doc, err := scrapeWeb(url)
	if err != nil {
		return nil, err
	}

	results := doc.Find(".layout__mainline").First()

	sel := results.Find(".web-bing__result").First()

	resURL := sel.Find(".web-bing__url").Text()
	resDesc := sel.Find(".web-bing__description").Text()

	return map[string]string{"url": resURL, "desc": resDesc}, nil
}

// ScrapeFirstImgRes function scrapes dogpile.com for the first result of the query and returns it
func ScrapeFirstImgRes(q string) (string, error) {
	url := "https://www.dogpile.com/serp?qc=images&q=" + url.QueryEscape(q)
	doc, err := scrapeWeb(url)
	if err != nil {
		return "", err
	}

	results := doc.Find(".layout__mainline").First()

	sel := results.Find(".image").First()
	link := sel.Find("a").First()
	url, _ = link.Attr("href")

	return url, nil
}
