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
	Config config.Watchlog

	LastAt time.Time
}

func NewWatcher(conf config.Watchlog) *Watcher {
	logrus.Infof("Watchlog enabled, healthcheck ID: %s", conf.HealthcheckID)

	return &Watcher{
		Config: conf,
		LastAt: time.Now(),
	}
}

func (w *Watcher) Watch(r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		t := scanner.Text()
		if strings.Contains(t, w.Config.Keyword) {
			w.LastAt = time.Now()
			logrus.Infof("Watchlog: detected keyword \"%s\"", w.Config.Keyword)
		}
	}
}

func (w *Watcher) Timer() {
	for _ = range time.Tick(1 * time.Minute) {
		go w.notifyHealthchecksIo()
	}
}

func (w *Watcher) notifyHealthchecksIo() {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	var uri string
	if since := time.Since(w.LastAt); since < w.Config.LastThreshold {
		logrus.Debugf("Watchlog: it's been %s since last detected keyword, below threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s", w.Config.HealthcheckID)
	} else {
		logrus.Debugf("Watchlog: it's been %s since last detected keyword, above threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s/fail", w.Config.HealthcheckID)
	}

	_, err := c.Get(uri)
	if err != nil {
		logrus.Warnf("Client.Get: %s", err)
	}
}
