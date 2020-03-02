package web_search

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/urfave/cli/v2"
	"gitlab.com/muxro/muxbot/addons"
	"gitlab.com/muxro/muxbot/bot"
)

const userAgent = "Mozilla/5.0 (Linux; Android 4.0.4; Galaxy Nexus Build/IMM76B) AppleWebKit/535.19 (KHTML, like Gecko) Chrome/18.0.1025.133 Mobile Safari/535.19"

func init() {
	addons.Register("web-search", WebSearch{})
}

type WebSearch struct{}

func (_ WebSearch) Add(b *bot.Bot) error {
	b.RegisterCommand(&cli.Command{
		Name:    "search",
		Aliases: []string{"google", "g"},
		Action:  bot.CommandHandler(findWebPage),
	})

	b.RegisterCommand(&cli.Command{
		Name:    "images",
		Aliases: []string{"image", "img", "gis"},
		Action:  bot.CommandHandler(findImage),
	})

	b.RegisterCommand(&cli.Command{
		Name:    "videos",
		Aliases: []string{"video", "vid", "yt"},
		Action:  bot.CommandHandler(findVideo),
	})

	return nil
}

func findWebPage(q string) (string, error) {
	url := "https://www.dogpile.com/serp?qc=web&q=" + url.QueryEscape(q)
	doc, err := scrapeWeb(url)
	if err != nil {
		return "", err
	}

	results := doc.Find(".layout__mainline").First()

	sel := results.Find(".web-bing__result").First()

	resURL := sel.Find(".web-bing__url").Text()
	resDesc := sel.Find(".web-bing__description").Text()

	return fmt.Sprintf("%s -- %s", resURL, resDesc), nil
}

func findImage(q string) (string, error) {
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

func findVideo(q string) (string, error) {
	url := "https://www.dogpile.com/serp?qc=video&q=" + url.QueryEscape(q)
	doc, err := scrapeWeb(url)
	if err != nil {
		return "", err
	}

	results := doc.Find(".layout__mainline").First()

	first := results.Find(".video").First()
	anchor := first.Find("a").First()
	url, _ = anchor.Attr("href")

	return fmt.Sprintf("%s -- %s", strings.TrimSpace(anchor.Text()), url), nil
}

func scrapeWeb(url string) (*goquery.Document, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

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
