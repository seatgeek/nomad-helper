package sink

import (
	"time"

	"os"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// KinesisSink ...
type KinesisSink struct {
	session      *session.Session
	kinesis      *kinesis.Kinesis
	streamName   string
	partitionKey string
	stopCh       chan interface{}
	putCh        chan []byte
}

// NewKinesis ...
func NewKinesis() (*KinesisSink, error) {
	streamName := os.Getenv("SINK_KINESIS_STREAM_NAME")
	if streamName == "" {
		return nil, fmt.Errorf("[sink/kinesis] Missing SINK_KINESIS_STREAM_NAME")
	}

	partitionKey := os.Getenv("SINK_KINESIS_PARTITION_KEY")
	if partitionKey == "" {
		return nil, fmt.Errorf("[sink/kinesis] Missing SINK_KINESIS_PARTITION_KEY")
	}

	sess := session.Must(session.NewSession())
	svc := kinesis.New(sess)

	return &KinesisSink{
		session:      sess,
		kinesis:      svc,
		streamName:   streamName,
		partitionKey: partitionKey,
		stopCh:       make(chan interface{}),
		putCh:        make(chan []byte, 1000),
	}, nil
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
	log.Infof("[sink/kinesis] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/kinesis] Waiting for queue to drain - (%d messages left)", len(s.putCh))
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
	log.Infof("[sink/kinesis/%d] Starting writer", id)

	streamName := aws.String(s.streamName)
	partitionKey := aws.String(s.partitionKey)

	for {
		select {
		case data := <-s.putCh:
			putOutput, err := s.kinesis.PutRecord(&kinesis.PutRecordInput{
				Data:         data,
				StreamName:   streamName,
				PartitionKey: partitionKey,
			})

			if err != nil {
				log.Errorf("[sink/kinesis/%d] %s", id, err)
			} else {
				log.Infof("[sink/kinesis/%d] %v", id, putOutput)
			}
		}
	}
}
