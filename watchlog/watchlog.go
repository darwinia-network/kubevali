package watchlog

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/darwinia-network/kubevali/config"
	"github.com/sirupsen/logrus"
)

type Watcher struct {
	Config config.Watchlog

	lastAt      time.Time
	lastAtMutex sync.Mutex
}

func NewWatcher(conf config.Watchlog) *Watcher {
	logrus.Infof("Watchlog enabled, healthcheck ID: %s", conf.HealthcheckID)

	return &Watcher{
		Config: conf,
	}
}

func (w *Watcher) Watch(r io.Reader, streamName string) {
	scanner := bufio.NewScanner(r)
	var timerDone chan bool

	for scanner.Scan() {
		t := scanner.Text()
		if !strings.Contains(t, w.Config.Keyword) {
			continue
		}

		logrus.Infof("Watchlog: found keyword \"%s\" in %s", w.Config.Keyword, streamName)

		w.lastAtMutex.Lock()
		lastAt := w.lastAt
		w.lastAt = time.Now()
		w.lastAtMutex.Unlock()

		// Start notifying healthchecks.io once first time found keyword
		if lastAt.IsZero() {
			timerDone = w.Timer()
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.Errorf("Watchlog: %s", err)
	} else {
		logrus.Debugf("Watchlog: scanner hit EOF")
	}

	timerDone <- true
}

func (w *Watcher) Timer() chan bool {
	go w.notifyHealthchecksIo()

	logrus.Debugf("Watchlog: timer starting")

	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				logrus.Debugf("Watchlog: timer stopped")
				return
			case <-ticker.C:
				w.notifyHealthchecksIo()
			}
		}
	}()

	return done
}

func (w *Watcher) notifyHealthchecksIo() {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}

	var uri string
	if since := time.Since(w.lastAt); since < w.Config.LastThreshold {
		logrus.Debugf("Watchlog: %s since last detected keyword, below threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s", w.Config.HealthcheckID)
	} else {
		logrus.Debugf("Watchlog: %s since last detected keyword, above threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s/fail", w.Config.HealthcheckID)
	}

	_, err := c.Get(uri)
	if err != nil {
		logrus.Warnf("Client.Get: %s", err)
	}
}
