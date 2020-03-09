package bot

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func Escape(text string) string {
	escaped := make([]rune, 0, len(text))
	for _, r := range text {
		switch r {
		case '`', '*', '_', '>', '~', '|':
			escaped = append(escaped, '\\')
		}

		escaped = append(escaped, r)

	}

	return string(escaped)
}

func pastebin(text string) (string, error) {
	for i := 0; i < 3; i++ {
		resp, err := http.PostForm("http://ix.io", url.Values{"f:1": {text}})
		if err != nil {
			log.Printf("failed to upload to pastebin: %s", err)
			continue
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read pastebin url: %s", err)
			continue
		}

		return strings.TrimSpace(string(content)), nil
	}

	return "", errors.New("failed to upload to pastebin")
}
