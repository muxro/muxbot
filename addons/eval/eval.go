package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gitlab.com/muxro/muxbot/addons"
	"gitlab.com/muxro/muxbot/bot"
)

const evalPath = "/home/george/tmp/bot-eval"

func init() {
	addons.Register("eval", Eval{})
}

type lang struct {
	Name    string
	Aliases []string
}

func (l *lang) is(name string) bool {
	if l.Name == name {
		return true
	}

	for _, alias := range l.Aliases {
		if alias == name {
			return true
		}
	}

	return false
}

var langs []lang

type Eval struct{}

func (_ Eval) Add(b *bot.Bot) error {
	out, err := exec.Command("nix-instantiate", "--strict", "--json", "--eval", evalPath+"/eval.nix", "-A", "list").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}

	err = json.Unmarshal(out, &langs)
	if err != nil {
		return err
	}

	b.AddCommand(&cli.Command{
		Name:    "eval",
		Aliases: []string{"e"},
		Usage:   "evaluate code",
		Action:  bot.CommandHandler(eval),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "paste",
				Aliases: []string{"pb"},
				Usage:   "enable pastebin",
			},
		},
	})

	return nil
}

func eval(c *bot.CommandContext) bot.Content {
	opts, err := parse(c)
	if err != nil {
		return bot.Text{Content: err.Error()}
	}

	return bot.Delayed(c.Ctx(), &bot.DelayedConfig{Name: "executing"}, func() bot.Content {
		out, err := doEval(c.Ctx(), opts)
		if err != nil {
			return bot.Text{Content: err.Error()}
		}

		if len(out) == 0 {
			return bot.Text{
				Content: "command has no output",
			}
		}

		return bot.Text{
			Content:    out,
			Quoted:     true,
			Pagination: true,
			Pastebin:   c.Bool("paste"),
		}

	})
}

type EvalOptions struct {
	Lang  string
	Code  string
	Stdin string
}

func parse(c *bot.CommandContext) (*EvalOptions, error) {
	if c.NArg() == 0 {
		return nil, errors.New("missing arguments")
	}

	langArg := strings.TrimSpace(strings.ToLower(c.Args().Get(0)))
	hasQuote := strings.HasPrefix(langArg, "```")
	if hasQuote {
		langArg = langArg[3:]
	}

	var l *lang
	for _, ln := range langs {
		if ln.is(langArg) {
			l = &ln
			break
		}
	}

	var input string
	if hasQuote {
		input = c.RawArg(0)
	} else {
		if c.NArg() == 1 {
			return nil, errors.New("missing code")
		}
		input = c.RawArg(1)
	}

	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "`") {
		return &EvalOptions{
			Lang: l.Name,
			Code: input,
		}, nil
	}

	rawInput := []byte(input)
	tr := text.NewReader(rawInput)
	root := goldmark.DefaultParser().Parse(tr)

	var quotes []string
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindCodeSpan:
			quotes = append(quotes, string(n.Text(rawInput)))

		case ast.KindFencedCodeBlock:
			var quote string
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				quote += string(line.Value(rawInput))
			}
			quotes = append(quotes, quote)

		default:
			return ast.WalkContinue, nil
		}

		return ast.WalkSkipChildren, nil
	})

	if len(quotes) == 0 {
		return nil, errors.New("something went wrong, couldn't find code")
	}

	if len(quotes) > 2 {
		return nil, errors.New("too many quotes and code blocks found")
	}

	code, stdin := quotes[0], ""
	if len(quotes) == 2 {
		stdin = strings.TrimSpace(quotes[1])
	}

	return &EvalOptions{
		Lang:  l.Name,
		Code:  code,
		Stdin: stdin,
	}, nil
}

func doEval(ctx context.Context, opts *EvalOptions) (string, error) {
	dir, err := ioutil.TempDir("", "bot-eval")
	if err != nil {
		return "", err
	}

	log.Println(dir)

	//defer os.RemoveAll(dir)

	codeFile := dir + "/code"
	err = ioutil.WriteFile(codeFile, []byte(opts.Code), os.ModePerm)
	if err != nil {
		return "", err
	}

	var inputFile string
	if len(opts.Stdin) > 0 {
		inputFile = dir + "/stdin"
		err = ioutil.WriteFile(inputFile, []byte(opts.Code), os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	out, err := exec.Command(evalPath+"/eval.sh", opts.Lang, codeFile, inputFile).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}

	out, err = ioutil.ReadFile(dir + "/result/out")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
