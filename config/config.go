package config

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Config struct {
	Node     Node
	Watchlog Watchlog
}

type Watchlog struct {
	Enabled       bool
	Keyword       string
	LastThreshold time.Duration
	HealthcheckID string
}

type Node struct {
	Index   int
	Command []string
}

func renderOrDie(raw *RawConfig) *Config {
	baseTemplate := template.New("")
	initTemplateFuncMap(baseTemplate)
	baseTemplate = template.Must(baseTemplate.Parse(raw.CommonTemplate))

	node := Node{}
	{ // Index
		s := renderValueOrDie(baseTemplate, raw.NodeTemplate.Index, node)
		if idx, err := strconv.Atoi(s); err != nil {
			panic(err)
		} else {
			node.Index = idx
		}
	}

	{ // Command
		var cmd []string
		for _, value := range raw.NodeTemplate.Command {
			v := renderValueOrDie(baseTemplate, value, node)
			cmd = append(cmd, v)
		}
		for key, value := range raw.NodeTemplate.Args {
			a := fmt.Sprintf("--%s", key)
			v := renderValueOrDie(baseTemplate, value, node)
			cmd = append(cmd, a, v)
		}
		node.Command = cmd
	}

	conf := &Config{
		Node: node,
		Watchlog: Watchlog{
			Enabled:       raw.Watchlog.Enabled,
			Keyword:       raw.Watchlog.Keyword,
			LastThreshold: raw.Watchlog.LastThreshold,
			HealthcheckID: raw.Watchlog.HealthcheckIDs[node.Index],
		},
	}
	return conf
}

func renderValueOrDie(baseTemplate *template.Template, text string, data interface{}) string {
	t := template.Must(baseTemplate.Clone())
	t = template.Must(t.New("").Parse(text))

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}
