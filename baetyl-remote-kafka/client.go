package main

import (
	"context"
	"github.com/baetyl/baetyl/logger"
	"github.com/baetyl/baetyl/utils"
	"github.com/segmentio/kafka-go"
)

type readHandler func(msg kafka.Message) error

type client struct {
	writer  *kafka.Writer
	reader  *kafka.Reader
	tomb    utils.Tomb
	handler readHandler
	log     logger.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func newKafkaWriter(address []string, cfg Rule) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  address,
		Topic:    cfg.Remote.Topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func newKafkaReader(address []string, cfg Rule) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     address,
		GroupID:     cfg.Remote.GroupID,
		Topic:       cfg.Remote.Topic,
		MinBytes:    cfg.Remote.MinBytes,
		MaxBytes:    cfg.Remote.MaxBytes,
		StartOffset: kafka.LastOffset,
		MaxWait:     cfg.Remote.MaxWait,
	})
}

func newClient(address []string, cfg Rule, log logger.Logger) *client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &client{
		ctx:    ctx,
		cancel: cancel,
		log:    log,
	}
	if cfg.Type == "to" {
		c.writer = newKafkaWriter(address, cfg)
	} else if cfg.Type == "from" {
		if cfg.Remote.GroupID == "" {
			return nil
		}
		c.reader = newKafkaReader(address, cfg)
	}
	return c
}

func (c *client) WriteMessages(msg kafka.Message) error {
	if c.writer != nil {
		return c.writer.WriteMessages(c.ctx, msg)
	}
	return nil
}

func (c *client) readMessage() error {
	for {
		select {
		case <-c.dying():
			return nil
		default:
			msg, err := c.reader.ReadMessage(c.ctx)
			if err != nil {
				c.log.Errorf("failed to read kafka message: %s", err.Error())
				continue
			}
			if c.handler != nil {
				if err = c.handler(msg); err != nil {
					c.log.Errorf("failed to handle mqtt msg")
				}
			}
		}
	}
}

func (c *client) SetReadHandler(handler readHandler) {
	c.handler = handler
}

func (c *client) StartRead() error {
	go func() {
		select {
		case <-c.dying():
			c.cancel()
		}
	}()
	if c.reader != nil {
		return c.tomb.Go(c.readMessage)
	}
	return nil
}

func (c *client) dying() <-chan struct{} {
	return c.tomb.Dying()
}

func (c *client) Close() {
	c.tomb.Kill(nil)
	c.tomb.Wait()
	if c.reader != nil {
		c.reader.Close()
	}
	if c.writer != nil {
		c.writer.Close()
	}
}
