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
	"go.uber.org/zap"
)

type Watcher struct {
	Config config.Watchlog
	Logger *zap.SugaredLogger

	lastAt      time.Time
	lastAtMutex sync.Mutex
	lastLogLine string
}

func NewWatcher(conf *config.Config) *Watcher {
	conf.Logger.Infof("Watchlog enabled, healthcheck ID: %s", conf.Watchlog.HealthcheckID)

	return &Watcher{
		Config: conf.Watchlog,
		Logger: conf.Logger,
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

		w.Logger.Infof("Watchlog: found keyword \"%s\" in %s", w.Config.Keyword, streamName)

		w.lastAtMutex.Lock()
		lastAt := w.lastAt
		w.lastAt = time.Now()
		w.lastLogLine = t
		w.lastAtMutex.Unlock()

		// Start notifying healthchecks.io once first time found keyword
		if lastAt.IsZero() {
			timerDone = w.Timer()
		}
	}

	if err := scanner.Err(); err != nil {
		w.Logger.Errorf("Watchlog: %s", err)
	} else {
		w.Logger.Debugf("Watchlog: scanner hit EOF")
	}

	timerDone <- true
}

func (w *Watcher) Timer() chan bool {
	go w.notifyHealthchecksIo()

	w.Logger.Debugf("Watchlog: timer starting")

	ticker := time.NewTicker(1 * time.Minute)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				w.Logger.Debugf("Watchlog: timer stopped")
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

	var (
		uri string
		log string
	)

	if since := time.Since(w.lastAt); since < w.Config.LastThreshold {
		log = fmt.Sprintf("Watchlog: %s since last detected keyword, below threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s", w.Config.HealthcheckID)
	} else {
		log = fmt.Sprintf("Watchlog: %s since last detected keyword, above threshold %s", since, w.Config.LastThreshold)
		uri = fmt.Sprintf("http://hc-ping.com/%s/fail", w.Config.HealthcheckID)
	}

	w.Logger.Debugf(log)

	body := fmt.Sprintf("%s\n\n%s", log, w.lastLogLine)
	_, err := c.Post(uri, "text/plain", strings.NewReader(body))
	if err != nil {
		w.Logger.Warnf("Client.Post: %s", err)
	}
}
