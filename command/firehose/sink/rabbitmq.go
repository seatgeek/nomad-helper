package sink

import (
	"time"

	"os"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

// RabbitmqSink ...
type RabbitmqSink struct {
	conn       *amqp.Connection
	exchange   string
	routingKey string
	stopCh     chan interface{}
	putCh      chan []byte
}

// NewRabbitmq ...
func NewRabbitmq() (*RabbitmqSink, error) {
	connStr := os.Getenv("SINK_AMQP_CONNECTION")
	if connStr == "" {
		return nil, fmt.Errorf("[sink/amqp] Missing SINK_AMQP_CONNECTION (example: amqp://guest:guest@127.0.0.1:5672/)")
	}

	exchange := os.Getenv("SINK_AMQP_EXCHANGE")
	if exchange == "" {
		return nil, fmt.Errorf("[sink/amqp] Missing SINK_AMQP_EXCHANGE")
	}

	routingKey := os.Getenv("SINK_AMQP_ROUTING_KEY")
	if routingKey == "" {
		return nil, fmt.Errorf("[sink/amqp] Mising SINK_AMQP_ROUTING_KEY")
	}

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, fmt.Errorf("[sink/amqp] Failed to connect to AMQP: %s", err)
	}

	return &RabbitmqSink{
		conn:       conn,
		exchange:   exchange,
		routingKey: routingKey,
		stopCh:     make(chan interface{}),
		putCh:      make(chan []byte, 1000),
	}, nil
}

// Start ...
func (s *RabbitmqSink) Start() error {
	// Stop chan for all tasks to depend on
	s.stopCh = make(chan interface{})

	// have 3 writers to rabbitmq
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
func (s *RabbitmqSink) Stop() {
	log.Infof("[sink/amqp] ensure writer queue is empty (%d messages left)", len(s.putCh))

	for len(s.putCh) > 0 {
		log.Info("[sink/amqp] Waiting for queue to drain - (%d messages left)", len(s.putCh))
		time.Sleep(1 * time.Second)
	}

	close(s.stopCh)
	defer s.conn.Close()
}

// Put ..
func (s *RabbitmqSink) Put(data []byte) error {
	s.putCh <- data

	return nil
}

func (s *RabbitmqSink) write(id int) {
	log.Infof("[sink/amqp/%d] Starting writer", id)

	ch, err := s.conn.Channel()
	if err != nil {
		log.Error(err)
		return
	}

	defer ch.Close()

	for {
		select {
		case data := <-s.putCh:
			err = ch.Publish(
				s.exchange,   // exchange
				s.routingKey, // routing key
				true,         // mandatory
				false,        // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        data,
				})

			if err != nil {
				log.Errorf("[sink/amqp/%d] %s", id, err)
			} else {
				log.Debugf("[sink/amqp/%d] publish ok", id)
			}
		}
	}
}
