package sink

import (
	"time"

	log "github.com/Sirupsen/logrus"
)

// StdoutSink ...
type StdoutSink struct {
	stopCh chan interface{}
	putCh  chan []byte
}

// NewStdout ...
func NewStdout() (*StdoutSink, error) {
	return &StdoutSink{
		stopCh: make(chan interface{}),
		putCh:  make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *StdoutSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	// wait forever for a stop signal to happen
	for {
		select {
		case <-s.stopCh:
			break
		}
		break
	}

	return nil
}

// Stop ...
func (s *StdoutSink) Stop() {
	log.Infof("[sink/stdout] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/stdout] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *StdoutSink) Put(data []byte) error {
	log.Infof("[sink/stdout] %s", string(data))
	return nil
}
