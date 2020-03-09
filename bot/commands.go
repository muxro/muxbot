package bot

import (
	"bytes"
	"context"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/urfave/cli/v2"
)

func (b *Bot) AddCommand(cmd *cli.Command) {
	b.cmds = append(b.cmds, cmd)
}

type CommandContext struct {
	*cli.Context
}

func (c *CommandContext) RawArg() string {
	cmd := c.App.Metadata["cmd-str"].(string)
	pos := c.App.Metadata["cmd-pos"].([]int)

	cmdArgs := c.Args().Slice()
	rawArgPos := pos[len(pos)-len(cmdArgs)]

	return cmd[rawArgPos:]
}

func CommandHandler(handler func(c *CommandContext) Content) cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx := &CommandContext{
			Context: c,
		}

		content := handler(ctx)
		return Reply(c.Context, content)
	}
}

type commands []*cli.Command

func (b *Bot) commandHandler(ctx context.Context, msg *discordgo.Message) (bool, error) {
	content := msg.Content
	prefix := b.config.Prefix

	if !strings.HasPrefix(content, prefix) {
		return false, nil
	}

	cmd := content[len(prefix):]

	args, pos := parseCmdString(cmd)
	params := append([]string{"bot"}, args...)

	var outBuf bytes.Buffer
	app := &cli.App{
		Name:      prefix,
		Writer:    &outBuf,
		ErrWriter: &outBuf,
		Metadata: map[string]interface{}{
			"cmd-str": cmd,
			"cmd-pos": pos,
		},
		Commands:               b.cmds,
		ExitErrHandler:         func(*cli.Context, error) {},
		CustomAppHelpTemplate:  appHelpTemplate,
		UseShortOptionHandling: true,
	}

	err := app.RunContext(ctx, params)
	if err != nil {
		return true, Reply(ctx, Text{Content: err.Error()})
	}

	out := strings.TrimSpace(outBuf.String())
	if len(out) > 0 {
		return true, Reply(ctx, Text{
			Content: out,
			Quoted:  true,
		})
	}

	return true, nil
}

func parseCmdString(line string) ([]string, []int) {
	args := []string{}
	pos := []int{}
	buf := ""

	var lastPos int
	var escaped, doubleQuoted, singleQuoted bool
	for i, r := range line {
		if escaped {
			buf += string(r)
			escaped = false
			continue
		}

		if isSpace(r) {
			if singleQuoted || doubleQuoted {
				buf += string(r)
				continue
			}

			if len(buf) != 0 {
				args = append(args, buf)
				pos = append(pos, lastPos)
			}

			lastPos = i + 1
			buf = ""
			continue
		}

		switch r {
		case '\\':
			escaped = true

		case '"':
			if singleQuoted {
				buf += string(r)
			} else {
				doubleQuoted = !doubleQuoted
			}

		case '\'':
			if doubleQuoted {
				buf += string(r)
			} else {
				singleQuoted = !singleQuoted
			}

		default:
			buf += string(r)
		}
	}

	if len(buf) != 0 {
		args = append(args, buf)
		pos = append(pos, lastPos)
	}

	return args, pos
}

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

var appHelpTemplate = `
USAGE: {{ .Name}}command [command options] [arguments...]
{{- if .VisibleCommands}}
COMMANDS:
	{{- range .VisibleCategories}}
		{{- if .Name}}
			{{- "\n"}}CATEGORY {{.Name}}:
			{{- range .VisibleCommands}}
				{{- "\n\t"}}{{join .Names ", "}} -- {{.Usage}}
			{{- end}}
		{{- else}}
			{{- range .VisibleCommands}}
				{{- "\n\t"}}{{join .Names ", "}} -- {{.Usage}}
			{{- end}}
		{{- end}}
	{{- end}}
{{- end}}`

var tmpl1 = `
{{.HelpName}} - {{.Usage}}

USAGE:

{{if .Subcommands}}
{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}
{{else}}
{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}
{{end}}`

//{{if .Description}}DESCRIPTION:{{.Description}}{{end}}
//
//{{if .Category}}CATEGORY:{{.Category}}{{end}}
//
//{{- if .VisibleCommands}}
//COMMANDS:
//	{{- range .VisibleCategories}}
//		{{- if .Name}}
//			{{- "\n"}}CATEGORY {{.Name}}:
//			{{- range .VisibleCommands}}
//				{{- "\n\t"}}{{join .Names ", "}} -- {{.Usage}}
//			{{- end}}
//		{{- else}}
//			{{- range .VisibleCommands}}
//				{{- "\n\t"}}{{join .Names ", "}} -- {{.Usage}}
//			{{- end}}
//		{{- end}}
//	{{- end}}
//{{- end}}
//
//
//{{if .VisibleFlags}}
//OPTIONS:
//{{range .VisibleFlags}}{{.}}{{end}}
//{{end}}
//
//aaa
//
//	` + "```````"
