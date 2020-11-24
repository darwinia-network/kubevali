package main

import (
	"io"
	"os"
	"os/exec"

	"github.com/darwinia-network/kubevali/config"
	"github.com/darwinia-network/kubevali/node"
	"github.com/darwinia-network/kubevali/watchlog"
	flags "github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var opts struct {
	Config   string `long:"config" short:"c" description:"Path to the config file" value-name:"<PATH>" default:"kubevali.yaml"`
	LogLevel uint32 `long:"log-level" description:"The log level (0 ~ 6), use 5 for debugging, see https://pkg.go.dev/github.com/sirupsen/logrus#Level" value-name:"N" default:"4"`
}

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(0)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableQuote:     true,
		DisableTimestamp: true,
	})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.Level(opts.LogLevel))
	log.Infof("Kubevali %v-%v (built %v)", buildVersion, buildCommit, buildDate)

	conf := config.NewConfig(opts.Config)
	node := node.NewNode(conf.Node)

	if conf.Watchlog.Enabled {
		logWatcher := watchlog.NewWatcher(conf.Watchlog)
		go logWatcher.Watch(io.TeeReader(node.Stdout, os.Stdout))
		go logWatcher.Watch(io.TeeReader(node.Stderr, os.Stdout)) // Redirect to STDOUT
		go logWatcher.Timer()
	} else {
		go io.Copy(os.Stdout, node.Stdout)
		go io.Copy(os.Stdout, node.Stderr)
	}

	err := node.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		log.Debugf("Node.Run(): %s", exitErr.Error())
		os.Exit(exitErr.ExitCode())
	} else if err != nil {
		log.Fatalf("Node.Run(): %s", err.Error())
	} else {
		log.Debug("Node.Run(): OK")
	}
}
