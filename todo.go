package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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

func todoHandler(session *discordgo.Session, message *discordgo.MessageCreate, sendReply messageSender, sendMessage messageSender, sendError errorSender) {
	params := strings.Split(message.Content, " ")[1:]

	pinnedMessages, err := session.ChannelMessagesPinned(message.ChannelID)
	if err != nil {
		sendError(err)
		return
	}
	todoPin := &discordgo.Message{}
	if len(pinnedMessages) < 1 || pinnedMessages[0].Author.ID != session.State.User.ID {
		sendReply("The first pinned message wasn't made by the bot, we're fixing that...")
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
		sendReply("Todo updated")
	case "create":
		if len(params) < 2 {
			sendReply("Usage: " + *prefix + "todo create <category name>")
			return
		}
		categoryName := strings.Join(params[1:], " ")
		contents = append(contents, Category{name: categoryName})
		sendReply("Todo updated")
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
		sendReply("Todo updated")
	case "clean":
		if len(params) > 1 && params[1] == "sure" {
			sendReply("Deleting todos")
			contents = []Category{}
		} else {
			sendReply("This might be dangerous, if you are sure you want to do this type `" + *prefix + "todo clean sure`")
		}
	default:
		sendReply("Unknown command")
	}
	_, err = session.ChannelMessageEdit(message.ChannelID, todoPin.ID, RenderTodoMessage(contents))
	if err != nil {
		sendError(err)
	}
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
