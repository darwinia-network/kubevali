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
	"github.com/spf13/viper"
)

var opts struct {
	Config      string `long:"config" short:"c" description:"Path to the config file" value-name:"<PATH>" default:"kubevali.yaml"`
	WatchConfig bool   `long:"watch-config" short:"w" description:"Watch config file changes and restart node with new config"`
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

	viper.SetConfigFile(opts.Config)

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Unable to load config file: %s", err)
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
		defer conf.Logger.Sync()

		conf.Logger.Infof("Kubevali %v-%v (built %v)", buildVersion, buildCommit, buildDate)

		status := kubevali(conf, ctx)
		if !configChanged || status != 0 {
			os.Exit(status)
		}
	}
}

func kubevali(conf *config.Config, ctx context.Context) int {
	node := node.NewNode(conf)

	if conf.Watchlog.Enabled {
		logWatcher := watchlog.NewWatcher(conf)
		go logWatcher.Watch(io.TeeReader(node.Stdout, conf.Node.Stdout), "stdout")
		go logWatcher.Watch(io.TeeReader(node.Stderr, conf.Node.Stderr), "stderr")
	} else {
		go io.Copy(conf.Node.Stdout, node.Stdout)
		go io.Copy(conf.Node.Stderr, node.Stderr)
	}

	conf.Logger.Infof("Starting node: %s", node.ShellCommand())

	if opts.DryRun {
		conf.Logger.Debugf("Exit because --dry-run is specified")
		os.Exit(0)
	}

	err := node.Run(ctx)

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		conf.Logger.Debugf("Node exits: %s", exitErr)
		return exitErr.ExitCode()
	}

	if err != nil {
		conf.Logger.Fatalf("Node exits: %s", err)
	}

	conf.Logger.Debug("Node exits: OK")
	return 0
}
