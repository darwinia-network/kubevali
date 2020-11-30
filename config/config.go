package config

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
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
	baseTemplate, err := baseTemplate.Parse(raw.CommonTemplate)
	if err != nil {
		logrus.Fatalf("Unable to parse common template: %s", err)
	}

	node := Node{}
	{ // Index
		s := renderValueOrDie(baseTemplate, raw.NodeTemplate.Index, node)
		if idx, err := strconv.Atoi(s); err != nil {
			log.Fatalf("Unable to convert .nodeTemplate.index to int: %s", err)
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

	watchlog := Watchlog{
		Enabled:       raw.Watchlog.Enabled,
		Keyword:       raw.Watchlog.Keyword,
		LastThreshold: raw.Watchlog.LastThreshold,
	}
	if raw.Watchlog.Enabled {
		// Watchlog.HealthcheckID
		if n := len(raw.Watchlog.HealthcheckIDs); node.Index >= n {
			log.Fatalf("No enough healthcheck IDs, expect %d, got %d", node.Index+1, n)
		}
		watchlog.HealthcheckID = raw.Watchlog.HealthcheckIDs[node.Index]
	}

	conf := &Config{
		Node:     node,
		Watchlog: watchlog,
	}
	return conf
}

func renderValueOrDie(baseTemplate *template.Template, text string, data interface{}) string {
	t, err := baseTemplate.Clone()
	if err != nil {
		log.Fatalf("Unable to clone template: %s", err)
	}

	t, err = t.New("").Parse(text)
	if err != nil {
		log.Fatalf("Unable to parse template: %s", err)
	}

	var buf strings.Builder
	err = t.Execute(&buf, data)
	if err != nil {
		log.Fatalf("Unable to render template: %s ", err)
	}

	return buf.String()
}
