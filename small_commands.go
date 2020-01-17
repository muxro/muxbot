package main

import (
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
)

func helpHandler(args []string) (string, error) {
	return "Head over to https://gitlab.com/muxro/muxbot/blob/master/commands.md for information regarding available commands.", nil
}

func pingHandler(args []string) (string, error) {
	return "pong", nil
}

func echoHandler(args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func evalHandler(args []string) (string, error) {
	commandMessage := strings.Join(args, " ")
	expr, err := govaluate.NewEvaluableExpression(commandMessage)
	if err != nil {
		return "", err
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", result), nil
}

func gHandler(args []string) (string, error) {
	commandMessage := strings.Join(args, " ")
	res, err := scrapeFirstWebRes(commandMessage)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s -- %s", res["url"], res["desc"]), nil
}

func gisHandler(args []string) (string, error) {
	commandMessage := strings.Join(args, " ")
	res, err := scrapeFirstImgRes(commandMessage)
	if err != nil {
		return "", err
	}

	return res, nil
}

func ytHandler(args []string) (string, error) {
	commandMessage := strings.Join(args, " ")
	res, err := getFirstYTResult(commandMessage)
	if err != nil {
		return "", err
	}

	return res, nil
}

func nonExistentHandler(args []string) (string, error) {
	return "This command has been disabled because the bot maintainer didn't specify the required key", nil
}
