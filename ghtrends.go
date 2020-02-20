package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

var (
	errEmptyTrendsResponse = errors.New("empty response from github")
	ghTrendsBaseURL        = "https://github-trending-api.now.sh/"
)

func ghTrends(bot *Bot, msg *discordgo.Message, args string) error {
	var trendsFields []*discordgo.MessageEmbedField
	var parts []string
	if len(args) > 1 {
		parts = strings.Split(args[1:], " ")
	}
	// get send params
	base, _ := url.Parse(ghTrendsBaseURL)
	base.Path += "repositories"
	params := url.Values{}
	for _, part := range parts {
		if part == "daily" || part == "weekly" || part == "monthly" {
			params.Set("since", part)
		} else {
			params.Add("language", part)
		}
	}
	base.RawQuery = params.Encode()

	resp, err := http.Get(base.String())
	if err != nil {
		return errEmptyTrendsResponse
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errEmptyTrendsResponse
	}
	if len(body) < 2 {
		return errEmptyTrendsResponse
	}

	var data []interface{}
	json.Unmarshal(body, &data)
	for _, project := range data {
		field := &discordgo.MessageEmbedField{}
		field.Name = fmt.Sprintf("%s: %s",
			project.(map[string]interface{})["name"].(string),
			project.(map[string]interface{})["url"].(string))
		if len(field.Name) > 255 {
			field.Name = project.(map[string]interface{})["url"].(string)
			if len(field.Name) > 1023 {
				field.Value = "name too long"
				trendsFields = append(trendsFields, field)
				continue
			}
			field.Value = "Name: " + project.(map[string]interface{})["name"].(string) + "\n"
		}
		field.Value += project.(map[string]interface{})["description"].(string)
		for len(field.Value) > 1024 {
			lastSpace := strings.LastIndexAny(field.Value, " \t\n")
			if lastSpace == -1 {
				field.Value = field.Value[:1000] + "..."
			} else {
				field.Value = field.Value[:lastSpace]
				field.Value += "..."
			}
		}
		if field.Value == "" {
			field.Value = "No description provided"
		}
		// field.Name = "A"
		// field.Value = "B"
		trendsFields = append(trendsFields, field)
		if len(trendsFields) > 9 {
			break
		}
	}
	spew.Dump(trendsFields)
	_, err = bot.SendReplyEmbed(msg, &discordgo.MessageEmbed{Fields: trendsFields})
	return err
}
