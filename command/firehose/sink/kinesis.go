package sink

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// KinesisSink ...
type KinesisSink struct {
	session *session.Session
	kinesis *kinesis.Kinesis
	stopCh  chan interface{}
	putCh   chan []byte
}

// NewKinesis ...
func NewKinesis() *KinesisSink {
	sess := session.Must(session.NewSession())
	svc := kinesis.New(sess)

	return &KinesisSink{
		session: sess,
		kinesis: svc,
		stopCh:  make(chan interface{}),
		putCh:   make(chan []byte, 1000),
	}
}

// Start ...
func (s *KinesisSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	// have 3 writers to kinesis
	go s.write(1)
	go s.write(2)
	go s.write(3)

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
func (s *KinesisSink) Stop() {
	log.Infof("[kinesis] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[kinesis] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
}

// Put ..
func (s *KinesisSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *KinesisSink) write(id int) {
	log.Infof("[kinesis/%d] Starting kinesis writer", id)

	for {
		select {
		case data := <-s.putCh:
			putOutput, err := s.kinesis.PutRecord(&kinesis.PutRecordInput{
				Data:         data,
				StreamName:   aws.String("nomad-allocation-stream"),
				PartitionKey: aws.String("key1"),
			})

			if err != nil {
				log.Errorf("[kinesis/%d] %s", id, err)
			} else {
				log.Infof("[kinesis/%d] %v", id, putOutput)
			}
		}
	}
}
