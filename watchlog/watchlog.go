package watchlog

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/darwinia-network/kubevali/config"
	"github.com/sirupsen/logrus"
)

type Watcher struct {
	Stdout io.Reader
	Stderr io.Reader

	HealthchecksID string
	Keyword        string
	LastThreshold  time.Duration
	LastAt         time.Time
}

func validateConfig(conf config.Config) error {
	if len(conf.Watchlog.HealthcheckID) == 0 {
		return fmt.Errorf("Config watchlog.healthcheckID should not be empty")
	}

	if len(conf.Watchlog.Keyword) == 0 {
		return fmt.Errorf("Config watchlog.keyword should not be empty")
	}

	if conf.Watchlog.LastThreshold < 1*time.Second {
		return fmt.Errorf("Config watchlog.lastThreshold should be at least 1 second, got %s", conf.Watchlog.LastThreshold)
	}

	return nil
}

func NewWatcher(conf config.Config) *Watcher {
	if err := validateConfig(conf); err != nil {
		panic(err)
	}

	return &Watcher{
		HealthchecksID: conf.Watchlog.HealthcheckID,
		Keyword:        conf.Watchlog.Keyword,
		LastThreshold:  conf.Watchlog.LastThreshold,

		LastAt: time.Now(),
	}
}

func (w *Watcher) StartWatch() {
	go func() {
		scanner := bufio.NewScanner(w.Stdout)

		for scanner.Scan() {
			t := scanner.Text()
			if strings.Contains(t, w.Keyword) {
				w.LastAt = time.Now()
				logrus.Infof("Detected keyword \"%s\"", w.Keyword)
			}
		}
	}()

	go func() {
		for _ = range time.Tick(1 * time.Minute) {
			go w.CallHealthchecksIo()
		}
	}()
}

func (w *Watcher) CallHealthchecksIo() {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	var uri string
	if since := time.Since(w.LastAt); since < w.LastThreshold {
		logrus.Debugf("Watchlog: it's been %s since last detected keyword, below threshold %s", since, w.LastThreshold)
		uri = fmt.Sprintf("https://hc-ping.com/%s", w.HealthchecksID)
	} else {
		logrus.Debugf("Watchlog: it's been %s since last detected keyword, above threshold %s", since, w.LastThreshold)
		uri = fmt.Sprintf("https://hc-ping.com/%s/fail", w.HealthchecksID)
	}

	_, err := c.Get(uri)
	if err != nil {
		logrus.Warnf("Client.Get: %s", err)
	}
}
