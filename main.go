package main

import (
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/xanzy/go-gitlab"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"

	"github.com/PuerkitoBio/goquery"

	"github.com/Knetic/govaluate"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

var token = flag.String("token", "none", "Specify the token")
var googleDevKey = flag.String("gkey", "none", "Specify the google dev key")
var gitlabToken = flag.String("glt", "none", "Specify the Gitlab Token")
var prefix = flag.String("prefix", ".", "Specify the bot prefix")

// Category stores a ToDo category
type Category struct {
	name  string
	todos []Todo
}

// Todo stores the date for a Todo
type Todo struct {
	content   string
	completed bool
}

func main() {
	flag.Parse()

	envVal, exists := os.LookupEnv("token")
	if exists {
		*token = envVal
	}

	envVal, exists = os.LookupEnv("gkey")
	if exists {
		*googleDevKey = envVal
	}

	envVal, exists = os.LookupEnv("glt")
	if exists {
		*gitlabToken = envVal
	}

	if *token == "none" || *googleDevKey == "none" || *gitlabToken == "none" {
		fmt.Println("You need to specify a token, google dev and gitlab key")
		return
	}

	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Println("Could not create Discord session: ", err)
		return
	}

	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gitlabKeys (dtag varchar(512) UNIQUE, key varchar(512));")
	if err != nil {
		log.Fatal(err)
	}
	db.Close()

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	fmt.Println("Running MuxBot. Press Ctrl+C to exit")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "MuxBot is Aliiive!")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {

	if message.Author.ID == session.State.User.ID {
		return
	}

	// It is a command
	if strings.HasPrefix(message.Content, *prefix) {
		commandMessage := strings.Join(strings.Split(message.Content[1:], " ")[1:], " ")

		if startsCommand(message.Content, "help") {
			sendReplyInChannel(strings.ReplaceAll(`
^help - shows this
^ping - pong
^echo - echoes back whatever you send it
^eval - compute simple expression
^g - searches something on google and returns the first result
^gis - searches something on google image search and returns the first result
^yt - searches something on youtube and returns the first result
^glkey - associates a gitlab personal access key with your account
			`, "^", *prefix),
				session, message.Message)
		} else if startsCommand(message.Content, "g ") {
			res, err := scrapeFirstWebRes(commandMessage)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			sendReplyInChannel(fmt.Sprintf("%s -- %s", res["url"], res["desc"]), session, message.Message)
		} else if startsCommand(message.Content, "gis ") {
			res, err := scrapeFirstImgRes(commandMessage)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			sendReplyInChannel(res, session, message.Message)
		} else if startsCommand(message.Content, "yt") {
			res, err := getFirstYTResult(commandMessage)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			sendReplyInChannel(res, session, message.Message)
		} else if startsCommand(message.Content, "ping") {
			sendReplyInChannel("pong", session, message.Message)
		} else if startsCommand(message.Content, "echo") {
			if commandMessage != "" {
				sendReplyInChannel(commandMessage, session, message.Message)
			}
		} else if startsCommand(message.Content, "eval") {
			expr, err := govaluate.NewEvaluableExpression(commandMessage)
			if err != nil {

				sendErrorInChannel(err, session, message.Message)
				return
			}
			result, err := expr.Evaluate(nil)
			if err != nil {

				sendErrorInChannel(err, session, message.Message)
				return
			}
			sendReplyInChannel(fmt.Sprintf("%v", result), session, message.Message)
		} else if startsCommand(message.Content, "encode") {
			params := strings.SplitN(commandMessage, " ", 3)
			sendReplyInChannel("TODO", session, message.Message)
			if len(params) != 3 {
				sendReplyInChannel("Error: Trebuie specificata baza, tipul (int/string) si ce trebuie encodat", session, message.Message)
				return
			}
			if strings.Contains(params[0], "64") {
				sendReplyInChannel(base64.StdEncoding.EncodeToString([]byte(params[2])), session, message.Message)
			}
		} else if startsCommand(message.Content, "decode") {
			sendReplyInChannel("TODO", session, message.Message)
		} else if startsCommand(message.Content, "todo") {
			pinnedMessages, err := session.ChannelMessagesPinned(message.ChannelID)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			todoPin := &discordgo.Message{}
			if len(pinnedMessages) < 1 || pinnedMessages[0].Author.ID != session.State.User.ID {
				sendReplyInChannel("Primul mesaj pinned nu e al botului, rezolvam asta...", session, message.Message)
				result := sendMessageInChannel("TODOS:", session, message.Message)
				err := session.ChannelMessagePin(message.ChannelID, result.ID)
				todoPin = result
				if err != nil {
					sendErrorInChannel(err, session, message.Message)
					return
				}
			} else {
				todoPin = pinnedMessages[0]
			}

			params := strings.Split(commandMessage, " ")
			if len(params) < 1 {
				sendReplyInChannel("Usage: "+*prefix+"todo <add/remove/clean/move/rename/done>", session, message.Message)
				return
			}
			contents := ParseTodoMessage(todoPin.Content)
			switch params[0] {
			case "add":
				if len(params) < 3 {
					sendReplyInChannel("Usage: "+*prefix+"todo add <category letter> <text>", session, message.Message)
					return
				}
				categoryIndex := 0
				if len(params[1]) == 1 || params[1][0] >= 'A' || params[1][0] <= 'Z' {
					categoryIndex = int(params[1][0] - 'A')
				}
				if categoryIndex >= len(contents) {
					sendReplyInChannel("You can't add to a todo non-existent category", session, message.Message)
					return
				}
				todoText := strings.Join(params[2:], " ")
				contents[categoryIndex].todos = append(contents[categoryIndex].todos, Todo{content: todoText, completed: false})
			case "create":
				if len(params) < 2 {
					sendReplyInChannel("Usage: "+*prefix+"todo create <category name>", session, message.Message)
					return
				}
				categoryName := strings.Join(params[1:], " ")
				contents = append(contents, Category{name: categoryName})
			case "done":
				if len(params) < 3 {
					sendReplyInChannel("Usage: "+*prefix+"todo done <category> <todo index>", session, message.Message)
					return
				}
				categoryIndex := 0
				if len(params[1]) == 1 && params[1][0] >= 'A' && params[1][0] <= 'Z' {
					categoryIndex = int(params[1][0] - 'A')
				}
				todoIndex, err := strconv.Atoi(params[2])
				if err != nil {
					sendReplyInChannel("Error: Invalid todo index", session, message.Message)
				}
				todoIndex--
				contents[categoryIndex].todos[todoIndex].content = "~~" + contents[categoryIndex].todos[todoIndex].content + "~~"
			case "clean":
				if len(params) > 1 && params[1] == "sure" {
					sendReplyInChannel("Deleting todos", session, message.Message)
					contents = []Category{}
				} else {
					sendReplyInChannel("This might be dangerous, if you are sure you want to do this type `"+*prefix+"todo clean sure`", session, message.Message)
				}
			}
			_, err = session.ChannelMessageEdit(message.ChannelID, todoPin.ID, RenderTodoMessage(contents))
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
			}

		} else if startsCommand(message.Content, "issues") {
			git := gitlab.NewClient(nil, *gitlabToken)
			opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
			projects, _, err := git.Projects.ListProjects(opt)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			params := strings.Split(commandMessage, " ")
			if len(params) < 1 {
				sendReplyInChannel("Usage: "+*prefix+"issues <list|create|modify>", session, message.Message)
				return
			}
			switch params[0] {
			case "list":
				if len(params) < 2 {
					sendReplyInChannel("Usage: "+*prefix+"issues list <project name>", session, message.Message)
					return
				}
				found := false
				for _, project := range projects {
					if project.PathWithNamespace == params[1] || project.Name == params[1] {
						found = true
						issues, _, err := git.Issues.ListProjectIssues(project.ID, &gitlab.ListProjectIssuesOptions{Sort: gitlab.String("asc")})
						if err != nil {
							sendErrorInChannel(err, session, message.Message)
						}
						issueData := ""
						for _, issue := range issues {
							issueData += fmt.Sprintf("[%s#%d] %s\n", project.PathWithNamespace, issue.IID, issue.Title)
						}
						if issueData == "" {
							issueData = "No issue found."
						}
						sendMessageInChannel(issueData, session, message.Message)
						break
					}
				}
				git.Issues.CreateIssue(1, &gitlab.CreateIssueOptions{})
				if found == false {
					sendReplyInChannel("Could not find project", session, message.Message)
				}
			case "add":
				asTok, exists, err := associatedKey(message.Author.ID)
				if err != nil {
					sendErrorInChannel(err, session, message.Message)
					return
				}
				if exists == false {
					sendReplyInChannel("Eroare: Nu ai asociat un Personal Access Token gitlab cu contul tau", session, message.Message)
					return
				}
			}
		} else if startsCommand(message.Content, "glkey") {
			key := commandMessage
			err := session.ChannelMessageDelete(message.ChannelID, message.ID)
			if err != nil {
				sendErrorInChannel(err, session, message.Message)
				return
			}
			result, ok := testKey(key)
			if ok == true {
				err := associateUserToToken(message.Author.ID, key)

				if err != nil {
					sendErrorInChannel(err, session, message.Message)
					return
				}
				sendReplyInChannel("Associated user with gitlab user "+result.Name, session, message.Message)
			} else {
				sendReplyInChannel("Invalid key", session, message.Message)
			}
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

func sendMessageInChannel(message string, session *discordgo.Session, originalMessage *discordgo.Message) *discordgo.Message {
	resulting, _ := session.ChannelMessageSend(originalMessage.ChannelID, message)
	return resulting
}

func sendReplyInChannel(message string, session *discordgo.Session, originalMessage *discordgo.Message) *discordgo.Message {
	return sendMessageInChannel(fmt.Sprintf("(%s) %s", strings.Split(originalMessage.Author.String(), "#")[0], message), session, originalMessage)
}

func sendErrorInChannel(err error, session *discordgo.Session, originalMessage *discordgo.Message) *discordgo.Message {
	return sendReplyInChannel(fmt.Sprintf("Eroare: %v", err), session, originalMessage)
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

	return "Eroare: S-a stricat ceva", nil
}

func startsCommand(content string, command string) bool {
	return strings.HasPrefix(content, fmt.Sprintf("%s%s", *prefix, command))
}

// ParseTodoMessage parses a Todo
func ParseTodoMessage(content string) []Category {
	lines := strings.Split(content, "\n")
	data := []Category{}
	currentCategory := Category{}
	for i, line := range lines {
		if i > 0 {
			if len(line) < 2 {
				continue
			} else if line[0] == ' ' { /// it is a new todo
				content := ""
				line = strings.TrimPrefix(line, " ")
				lineData := strings.SplitN(line, ".", 2)
				content = lineData[1][1:]
				completed := strings.Contains(content, "~~")
				currentCategory.todos = append(currentCategory.todos, Todo{content, completed})
			} else { /// it is a new category
				if len(currentCategory.name) > 0 {
					data = append(data, currentCategory)
				}
				currentCategory = Category{}
				lineData := strings.SplitN(line, ".", 2)
				if len(lineData[1]) > 1 {
					currentCategory.name = lineData[1][1:]
				}
			}
		}
	}
	if len(currentCategory.name) > 0 {
		data = append(data, currentCategory)
	}
	fmt.Println(data)
	return data
}

// RenderTodoMessage renders a Todo in a way that humans and ParseTodoMessage can read
func RenderTodoMessage(categories []Category) string {
	content := "TODOS:\n"
	for index, category := range categories {
		content += fmt.Sprintf("%c. %s\n", (index + 'A'), category.name)
		for index, todo := range category.todos {
			content += fmt.Sprintf(" %d. %s\n", index+1, todo.content)
		}
		content += "\n"
	}
	return content
	// return "TODOS:\nA. Test\n 1. Ana are mere\n 2. Gigel are pere\n"
}

func testKey(key string) (gitlab.User, bool) {
	git := gitlab.NewClient(nil, key)
	user, _, err := git.Users.CurrentUser()
	if err != nil {
		fmt.Println(err)
		return gitlab.User{}, false
	}
	return *user, true
}

func associateUserToToken(user string, token string) error {
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("INSERT OR REPLACE INTO gitlabKeys (dtag, key) VALUES (?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(user, token)
	if err != nil {
		return err
	}
	return nil
}

func associatedKey(id string) (string, bool, error) {
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		return "", false, err
	}
	defer db.Close()
	var dtag, key string
	err = db.QueryRow("SELECT * FROM gitlabKeys WHERE dtag=?", id).Scan(&dtag, &key)
	if err != nil {
		return "", false, err
	}
	return key, true, nil
}
