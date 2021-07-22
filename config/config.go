package config

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strconv"
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

	// Renderer
	render := &TemplateRenderer{
		BaseTemplate: baseTemplate,
		Logger:       logger,
		Data:         node,
	}

	{ // Index
		idxTemplate := raw.NodeTemplate.Index
		if idxTemplate == "" {
			idxTemplate = `{{ env "HOSTNAME" | splitList "-" | mustLast }}`
		}
		s := render.RenderValueOrDie(idxTemplate)
		if idx, err := strconv.Atoi(s); err != nil {
			logger.Fatalf("Unable to convert .nodeTemplate.index to int: %s", err)
		} else {
			node.Index = idx
		}
	}

	{ // Command
		var cmd []string
		for _, value := range raw.NodeTemplate.Command {
			cmd = render.RenderCommandOrDie(value, cmd, "")
		}
		// Args
		for key, values := range raw.NodeTemplate.Args {
			switch values.(type) {
			case bool:
				cmd = append(cmd, key, strconv.FormatBool(values.(bool)))
			case int:
				cmd = append(cmd, key, strconv.Itoa(values.(int)))
			case string:
				cmd = render.RenderCommandOrDie(values.(string), cmd, key)
			case []interface{}:
				for i, value := range values.([]interface{}) {
					if reflect.TypeOf(value).Kind() == reflect.String {
						cmd = render.RenderCommandOrDie(value.(string), cmd, key)
					} else {
						log.Fatalf("Invalid type %T of nodeTemplate.args[\"%s\"][%d]", value, key, i)
					}
				}
			default:
				log.Fatalf("Invalid type %T of nodeTemplate.args[\"%s\"]", values, key)
			}
		}
		node.Command = cmd
	}

	nodeService := NodeService{
		Enabled:     raw.NodeService.Enabled,
		ForceUpdate: raw.NodeService.ForceUpdate,
	}
	if nodeService.Enabled {
		// NodeService.NodePort
		s := render.RenderValueOrDie(raw.NodeService.NodePortTemplate)
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
