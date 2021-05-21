package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Node        Node
	NodeService NodeService
	Watchlog    Watchlog
	Logger      *zap.SugaredLogger
}

type Watchlog struct {
	Enabled       bool
	Keyword       string
	LastThreshold time.Duration
	HealthcheckID string
}

type Node struct {
	Stdout *os.File
	Stderr *os.File

	Index   int
	Command []string
}

type NodeService struct {
	Enabled     bool
	NodePort    int
	ForceUpdate bool
}

func initializeLogger(loggingConfig interface{}) *zap.SugaredLogger {
	zapConfig := zap.NewProductionConfig()
	if bytes, err := json.Marshal(loggingConfig); err != nil {
		log.Fatalf("Failed to read logging config: %s", err)
	} else if err := json.Unmarshal(bytes, &zapConfig); err != nil {
		log.Fatalf("Failed to read logging config: %s", err)
	}
	logger, err := zapConfig.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %s", err)
	}
	return logger.Sugar()
}

func renderOrDie(raw *RawConfig) *Config {
	// Logger
	logger := initializeLogger(raw.Logging)

	baseTemplate := template.New("")
	initTemplateFuncMap(baseTemplate)
	baseTemplate, err := baseTemplate.Parse(raw.CommonTemplate)
	if err != nil {
		logger.Fatalf("Unable to parse common template: %s", err)
	}

	node := Node{}
	{ // Stdout & Stderr
		f := func(s string) *os.File {
			switch s {
			case "stdout":
				return os.Stdout
			case "stderr":
				return os.Stderr
			case "":
				return os.Stdout
			}
			logger.Fatalf("Invalid node log output: %s", s)
			return nil
		}
		node.Stdout = f(raw.NodeStdout)
		node.Stderr = f(raw.NodeStderr)
	}

	{ // Index
		idxTemplate := raw.NodeTemplate.Index
		if idxTemplate == "" {
			idxTemplate = `{{ env "HOSTNAME" | splitList "-" | mustLast }}`
		}
		s := renderValueOrDie(logger, baseTemplate, idxTemplate, node)
		if idx, err := strconv.Atoi(s); err != nil {
			logger.Fatalf("Unable to convert .nodeTemplate.index to int: %s", err)
		} else {
			node.Index = idx
		}
	}

	{ // Command
		var cmd []string
		for _, value := range raw.NodeTemplate.Command {
			v := renderValueOrDie(logger, baseTemplate, value, node)
			cmd = append(cmd, v)
		}
		for key, value := range raw.NodeTemplate.Args {
			a := fmt.Sprintf("--%s", key)
			v := renderValueOrDie(logger, baseTemplate, value, node)
			cmd = append(cmd, a, v)
		}
		node.Command = cmd
	}

	nodeService := NodeService{
		Enabled:     raw.NodeService.Enabled,
		ForceUpdate: raw.NodeService.ForceUpdate,
	}
	if nodeService.Enabled {
		// NodeService.NodePort
		s := renderValueOrDie(logger, baseTemplate, raw.NodeService.NodePortTemplate, node)
		if port, err := strconv.Atoi(s); err != nil {
			logger.Fatalf("Unable to convert .nodeService.nodePortTemplate to int: %s", err)
		} else {
			nodeService.NodePort = port
		}
	}

	watchlog := Watchlog{
		Enabled:       raw.Watchlog.Enabled,
		Keyword:       raw.Watchlog.Keyword,
		LastThreshold: raw.Watchlog.LastThreshold,
	}
	if raw.Watchlog.Enabled {
		// Watchlog.HealthcheckID
		if n := len(raw.Watchlog.HealthcheckIDs); node.Index >= n {
			logger.Fatalf("No enough healthcheck IDs, expect %d, got %d", node.Index+1, n)
		}
		watchlog.HealthcheckID = raw.Watchlog.HealthcheckIDs[node.Index]
	}

	conf := &Config{
		Node:        node,
		NodeService: nodeService,
		Watchlog:    watchlog,
		Logger:      logger,
	}
	return conf
}

func renderValue(baseTemplate *template.Template, text string, data interface{}) (string, error) {
	t, err := baseTemplate.Clone()
	if err != nil {
		return "", fmt.Errorf("Unable to clone template: %w", err)
	}

	t, err = t.New("").Parse(text)
	if err != nil {
		return "", fmt.Errorf("Unable to parse template: %w", err)
	}

	var buf strings.Builder
	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("Unable to render template: %w", err)
	}

	return buf.String(), nil
}

func renderValueOrDie(logger *zap.SugaredLogger, baseTemplate *template.Template, text string, data interface{}) string {
	v, err := renderValue(baseTemplate, text, data)
	if err != nil {
		logger.Fatal(err)
	}
	return v
}
