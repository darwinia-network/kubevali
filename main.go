package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/darwinia-network/kubevali/config"
	"github.com/darwinia-network/kubevali/node"
	"github.com/darwinia-network/kubevali/watchlog"
	"github.com/fsnotify/fsnotify"
	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var opts struct {
	Config      string `long:"config" short:"c" description:"Path to the config file" value-name:"<PATH>" default:"kubevali.yaml"`
	WatchConfig bool   `long:"watch-config" short:"w" description:"Watch config file changes and restart node with new config"`
	LogLevel    uint32 `long:"log-level" description:"The log level (0 ~ 6), use 5 for debugging, see https://pkg.go.dev/github.com/sirupsen/logrus#Level" value-name:"N" default:"4"`
	DryRun      bool   `long:"dry-run" description:"Print the final rendered command line and exit"`
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

	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote:     true,
		DisableTimestamp: true,
	})
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.Level(opts.LogLevel))
	logrus.Infof("Kubevali %v-%v (built %v)", buildVersion, buildCommit, buildDate)

	viper.SetConfigFile(opts.Config)

	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalf("Unable to load config file: %s", err)
	}

	if opts.WatchConfig {
		viper.WatchConfig()
	}

	var configChanged bool
	for {
		configChanged = false
		ctx, cancel := context.WithCancel(context.Background())

		viper.OnConfigChange(func(e fsnotify.Event) {
			configChanged = true
			cancel()
		})

		conf := config.Unmarshal()
		status := kubevali(conf, ctx)
		if !configChanged || status != 0 {
			os.Exit(status)
		}
	}
}

func kubevali(conf *config.Config, ctx context.Context) int {
	node := node.NewNode(conf.Node)

	if conf.Watchlog.Enabled {
		logWatcher := watchlog.NewWatcher(conf.Watchlog)
		go logWatcher.Watch(io.TeeReader(node.Stdout, os.Stdout), "stdout")
		go logWatcher.Watch(io.TeeReader(node.Stderr, os.Stdout), "stderr") // Redirect to STDOUT
	} else {
		go io.Copy(os.Stdout, node.Stdout)
		go io.Copy(os.Stdout, node.Stderr)
	}

	logrus.Infof("Starting node: %s", node.ShellCommand())

	if opts.DryRun {
		logrus.Debugf("Exit because --dry-run is specified")
		os.Exit(0)
	}

	err := node.Run(ctx)

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		logrus.Debugf("Node exits: %s", exitErr)
		return exitErr.ExitCode()
	}

	if err != nil {
		log.Fatalf("Node exits: %s", err)
	}

	logrus.Debug("Node exits: OK")
	return 0
}
