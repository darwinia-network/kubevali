package main

import (
	"fmt"
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

	fmt.Printf("Kubevali %v-%v (built %v)\n", buildVersion, buildCommit, buildDate)

	log.SetOutput(os.Stderr)
	log.SetLevel(log.Level(opts.LogLevel))

	conf := config.NewConfig()
	node := node.NewNode(*conf)

	logWatcher := watchlog.NewWatcher(*conf)
	logWatcher.Stdout = node.Stdout
	logWatcher.Stderr = node.Stderr

	logWatcher.StartWatch()

	err := node.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		log.Debugf("Node.Run(): %s", exitError.Error())
		os.Exit(exitError.ExitCode())
	} else if err != nil {
		log.Fatalf("Node.Run(): %s", err.Error())
	} else {
		log.Debug("Node.Run(): OK")
	}
}
