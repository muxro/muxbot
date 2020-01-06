package main

import (
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

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

func scrapeFirstWebRes(q string) (map[string]string, error) {
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

func scrapeFirstImgRes(q string) (string, error) {
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

func getFirstYTResult(q string) (string, error) {
	client := &http.Client{
		Transport: &transport.APIKey{Key: *googleDevKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		return "", err
	}

	call := service.Search.List("id,snippet").Q(q)
	response, err := call.Do()
	if err != nil {
		return "", err
	}

	first := response.Items[0]
	switch first.Id.Kind {
	case "youtube#video":
		url := "https://youtube.com/watch?v=" + first.Id.VideoId
		return url, nil
	case "youtube#channel":
		url := "https://youtube.com/channel/" + first.Id.ChannelId
		return url, nil
	case "youtube#playlist":
		url := "https://youtube.com/playlist?list=" + first.Id.PlaylistId
		return url, nil
	}

	return "Error: Something broke", nil
}
