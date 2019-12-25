package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/xanzy/go-gitlab"

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

// IssueMsgOptions stores data about the rendering of the list
type IssueMsgOptions struct {
	ShowGroup    bool
	ShowRepo     bool
	ShowTags     bool
	ShowAuthor   bool
	ShowAssignee bool
}

// IssuesListOptions stores data about the issues
type IssuesListOptions struct {
	Group      string
	Repo       string
	Author     string
	Assignee   string
	Tags       []string
	Title      string
	InternalID int
	URL        string
}

// IssuesSearchOptions stores data about the searching of issues
type IssuesSearchOptions struct {
	Group    string
	Repo     string
	Author   string
	Assignee string
	Tags     []string
	Self     bool
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

	err = initDB()
	if err != nil {
		log.Fatal(err)
	}

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

	sendMessage := func(sentMsg string) *discordgo.Message {
		resulting, _ := session.ChannelMessageSend(message.ChannelID, sentMsg)
		return resulting
	}

	sendReply := func(sentMsg string) *discordgo.Message {
		return sendMessage(fmt.Sprintf("(%s) %s", strings.Split(message.Author.String(), "#")[0], sentMsg))
	}

	sendError := func(err error) *discordgo.Message {
		return sendReply(fmt.Sprintf("Eroare: %v", err))
	}

	sendEmbed := func(embed *discordgo.MessageEmbed) *discordgo.Message {
		msg, err := session.ChannelMessageSendEmbed(message.ChannelID, embed)
		if err != nil {
			sendError(err)
			return nil
		}
		return msg
	}

	// It is a command
	if strings.HasPrefix(message.Content, *prefix) {
		commandMessage := strings.Join(strings.Split(message.Content[1:], " ")[1:], " ")

		if startsCommand(message.Content, "help") {
			sendReply(strings.ReplaceAll(`
^help - shows this
^ping - pong
^echo - echoes back whatever you send it
^eval - compute simple expression
^g - searches something on google and returns the first result
^gis - searches something on google image search and returns the first result
^yt - searches something on youtube and returns the first result
^issues <list,add> - gitlab issue query and addition
^glkey - associates a gitlab personal access key with your account
			`, "^", *prefix))
		} else if startsCommand(message.Content, "g ") {
			res, err := scrapeFirstWebRes(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(fmt.Sprintf("%s -- %s", res["url"], res["desc"]))
		} else if startsCommand(message.Content, "gis ") {
			res, err := scrapeFirstImgRes(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(res)
		} else if startsCommand(message.Content, "yt") {
			res, err := getFirstYTResult(commandMessage)
			if err != nil {
				sendError(err)
				return
			}
			sendReply(res)
		} else if startsCommand(message.Content, "ping") {
			sendReply("pong")
		} else if startsCommand(message.Content, "echo") {
			if commandMessage != "" {
				sendReply(commandMessage)
			}
		} else if startsCommand(message.Content, "eval") {
			expr, err := govaluate.NewEvaluableExpression(commandMessage)
			if err != nil {

				sendError(err)
				return
			}
			result, err := expr.Evaluate(nil)
			if err != nil {

				sendError(err)
				return
			}
			sendReply(fmt.Sprintf("%v", result))
		} else if startsCommand(message.Content, "encode") {
			params := strings.SplitN(commandMessage, " ", 3)
			sendReply("TODO")
			if len(params) != 3 {
				sendReply("Error: Trebuie specificata baza, tipul (int/string) si ce trebuie encodat")
				return
			}
			if strings.Contains(params[0], "64") {
				sendReply(base64.StdEncoding.EncodeToString([]byte(params[2])))
			}
		} else if startsCommand(message.Content, "decode") {
			sendReply("TODO")
		} else if startsCommand(message.Content, "todo") {
			pinnedMessages, err := session.ChannelMessagesPinned(message.ChannelID)
			if err != nil {
				sendError(err)
				return
			}
			todoPin := &discordgo.Message{}
			if len(pinnedMessages) < 1 || pinnedMessages[0].Author.ID != session.State.User.ID {
				sendReply("Primul mesaj pinned nu e al botului, rezolvam asta...")
				result := sendMessage("TODOS:")
				err := session.ChannelMessagePin(message.ChannelID, result.ID)
				todoPin = result
				if err != nil {
					sendError(err)
					return
				}
			} else {
				todoPin = pinnedMessages[0]
			}

			params := strings.Split(commandMessage, " ")
			if len(params) < 1 {
				sendReply("Usage: " + *prefix + "todo <add/remove/clean/move/rename/done>")
				return
			}
			contents := ParseTodoMessage(todoPin.Content)
			switch params[0] {
			case "add":
				if len(params) < 3 {
					sendReply("Usage: " + *prefix + "todo add <category letter> <text>")
					return
				}
				categoryIndex := 0
				if len(params[1]) == 1 || params[1][0] >= 'A' || params[1][0] <= 'Z' {
					categoryIndex = int(params[1][0] - 'A')
				}
				if categoryIndex >= len(contents) {
					sendReply("You can't add to a todo non-existent category")
					return
				}
				todoText := strings.Join(params[2:], " ")
				contents[categoryIndex].todos = append(contents[categoryIndex].todos, Todo{content: todoText, completed: false})
			case "create":
				if len(params) < 2 {
					sendReply("Usage: " + *prefix + "todo create <category name>")
					return
				}
				categoryName := strings.Join(params[1:], " ")
				contents = append(contents, Category{name: categoryName})
			case "done":
				if len(params) < 3 {
					sendReply("Usage: " + *prefix + "todo done <category> <todo index>")
					return
				}
				categoryIndex := 0
				if len(params[1]) == 1 && params[1][0] >= 'A' && params[1][0] <= 'Z' {
					categoryIndex = int(params[1][0] - 'A')
				}
				todoIndex, err := strconv.Atoi(params[2])
				if err != nil {
					sendReply("Error: Invalid todo index")
				}
				todoIndex--
				contents[categoryIndex].todos[todoIndex].content = "~~" + contents[categoryIndex].todos[todoIndex].content + "~~"
			case "clean":
				if len(params) > 1 && params[1] == "sure" {
					sendReply("Deleting todos")
					contents = []Category{}
				} else {
					sendReply("This might be dangerous, if you are sure you want to do this type `" + *prefix + "todo clean sure`")
				}
			}
			_, err = session.ChannelMessageEdit(message.ChannelID, todoPin.ID, RenderTodoMessage(contents))
			if err != nil {
				sendError(err)
			}

		} else if startsCommand(message.Content, "issues") {
			git := gitlab.NewClient(nil, *gitlabToken)
			opt := &gitlab.ListProjectsOptions{Membership: gitlab.Bool(true)}
			projects, _, err := git.Projects.ListProjects(opt)
			if err != nil {
				sendError(err)
				return
			}
			params := strings.Split(commandMessage, " ")
			if len(params) < 1 {
				sendReply("Usage: " + *prefix + "issues <list|create|modify>")
				return
			}
			switch params[0] {
			case "list":
				searchOpts, msgOpts := parseIssueOpts(params[1:], projects)
				issueList := []IssuesListOptions{}
				if searchOpts.Self == true {
					selfUname, err := getGitlabUnameFromUser(message.Author.ID)
					if err != nil {
						sendError(err)
						return
					}
					if selfUname == "" {
						sendReply("You mentioned $self, but you don't have a gitlab key associated")
						return
					}
					searchOpts.Assignee = selfUname
				}
				for _, project := range projects {
					if searchOpts.Group == "" || project.Namespace.Path == searchOpts.Group {
						if searchOpts.Repo == "" || project.Name == searchOpts.Repo {
							issues, _, err := git.Issues.ListProjectIssues(project.ID, &gitlab.ListProjectIssuesOptions{Sort: gitlab.String("asc"), Labels: searchOpts.Tags})
							if err != nil {
								sendError(err)
								return
							}
							for _, issue := range issues {
								if searchOpts.Author == "" || issue.Author.Name == searchOpts.Author {
									if (searchOpts.Assignee == "") ||
										(issue.Assignee != nil && issue.Assignee.Name == searchOpts.Assignee) {

										assignee := ""
										if issue.Assignee != nil {
											assignee = issue.Assignee.Name
										}
										issueList = append(issueList, IssuesListOptions{Group: project.Namespace.Path,
											Repo:       project.Name,
											Author:     issue.Author.Name,
											Assignee:   assignee,
											Tags:       issue.Labels,
											Title:      issue.Title,
											InternalID: issue.IID,
											URL:        issue.WebURL})
									}
								}
							}
						}
					}
				}
				if len(issueList) == 0 {
					sendReply("No issue found")
				} else {
					issues := []string{}
					for _, issue := range issueList {
						issueText := "["
						if msgOpts.ShowGroup {
							issueText += issue.Group
							if msgOpts.ShowRepo {
								issueText += "/"
							}
						}
						if msgOpts.ShowRepo {
							issueText += issue.Repo
						}
						issueText += "#" + strconv.Itoa(issue.InternalID) + " "
						if len(issue.Tags) > 0 && msgOpts.ShowTags {
							issueText += "["
							for i, tag := range issue.Tags {
								issueText += tag
								if i != len(issue.Tags)-1 {
									issueText += ", "
								}
							}
							issueText += "] "
						}
						issueText += issue.Title
						if issue.Assignee != "" && msgOpts.ShowAssignee {
							issueText += " - assigned to " + issue.Assignee
						} else if msgOpts.ShowAuthor {
							issueText += " - created by " + issue.Author
						}
						issueText += "](" + issue.URL + ")"
						issues = append(issues, issueText)
					}
					sendEmbed(&discordgo.MessageEmbed{Description: strings.Join(issues, "\n")})
				}
			case "add":
				asTok, exists := associatedKey(message.Author.ID)
				if exists == false {
					sendReply("Eroare: Nu ai asociat un Personal Access Token gitlab cu contul tau")
					return
				}
				if len(params) < 3 {
					sendReply("Utilizare: " + *token + "issues add <project name> <issue title>")
					return
				}
				ok := false
				for _, project := range projects {
					if project.Name == params[1] || project.PathWithNamespace == params[1] {
						ok = true
						issueName := strings.Join(params[2:], " ")
						userGit := gitlab.NewClient(nil, asTok)
						_, _, err := userGit.Issues.CreateIssue(project.ID, &gitlab.CreateIssueOptions{Title: gitlab.String(issueName)})
						if err != nil {
							sendError(err)
							return
						}
						sendReply("Issue adaugat")
						break
					}
				}
				if ok == false {
					sendReply("Error: No project found")
				}
			case "activeRepo":
				if len(params) < 2 {
					sendReply("Usage: " + *prefix + "issues activeRepo <set/get/erase>")
					return
				}
				switch params[1] {
				case "set":
					if len(params) != 3 {
						sendReply("Usage: " + *prefix + "isseus activeRepo set <repo>")
						return
					}
					err := setActiveRepo(message.Author.ID, params[2])
					if err != nil {
						sendError(err)
					}
					sendReply("Set active repo " + params[2])
				case "get":

				}
			}
		} else if startsCommand(message.Content, "glkey") {
			key := commandMessage
			err := session.ChannelMessageDelete(message.ChannelID, message.ID)
			if err != nil {
				sendError(err)
				return
			}
			result, ok := testKey(key)
			if ok == true {
				err := associateUserToToken(message.Author.ID, key)

				if err != nil {
					sendError(err)
					return
				}
				sendReply("Associated user with gitlab user " + result.Name)
			} else {
				sendReply("Invalid key")
			}
		}
	}

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
}

func getIssueProject(issue *gitlab.Issue) (*gitlab.Project, error) {
	git := gitlab.NewClient(nil, *gitlabToken)
	project, _, err := git.Projects.GetProject(issue.ProjectID, nil)
	return project, err
}

func parseIssueOpts(params []string, projects []*gitlab.Project) (IssuesSearchOptions, IssueMsgOptions) {
	ret := IssuesSearchOptions{}
	msgOptions := IssueMsgOptions{ShowGroup: true, ShowAuthor: true, ShowRepo: true, ShowTags: true, ShowAssignee: true}
	if len(params) < 1 { // It's empty
		return ret, msgOptions
	}
	for _, param := range params {
		if param[0] == '^' { // Author
			msgOptions.ShowAuthor = false
			ret.Author = param[1:]
		} else if param[0] == '$' { // assignee
			if param == "$any" {
				msgOptions.ShowAuthor = false
			} else if param == "$self" {
				ret.Self = true
			} else {
				ret.Assignee = param[1:]
			}
			msgOptions.ShowAssignee = false
		} else if param[0] == '+' {
			ret.Tags = append(ret.Tags, param[1:])
			msgOptions.ShowTags = false
		} else {
			msgOptions.ShowGroup = false
			if strings.Contains(param, "/") {
				msgOptions.ShowRepo = false
				repoName := strings.Split(param, "/")
				ret.Group = repoName[0]
				ret.Repo = repoName[1]
			} else {

				if isRepo(param, projects) == true {
					msgOptions.ShowRepo = false
					ret.Repo = param
				} else {
					ret.Group = param
				}
			}
		}
	}
	return ret, msgOptions
}

func isRepo(name string, projects []*gitlab.Project) bool {
	for _, project := range projects {
		if project.Name == name {
			return true
		}
	}
	return false
}
