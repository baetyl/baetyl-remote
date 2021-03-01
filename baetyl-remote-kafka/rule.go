package main

import (
	"github.com/256dpi/gomqtt/packet"
	"github.com/baetyl/baetyl/logger"
	"github.com/baetyl/baetyl/protocol/mqtt"
	"github.com/segmentio/kafka-go"
)

type ruler struct {
	rule   *Rule
	hub    *mqtt.Dispatcher
	client *client
	log    logger.Logger
}

func create(rule Rule, hub mqtt.ClientInfo, client *client) *ruler {
	defaults(&rule, &hub)
	log := logger.WithField("rule", rule.Remote.Name)
	return &ruler{
		rule:   &rule,
		hub:    mqtt.NewDispatcher(hub, log),
		client: client,
		log:    log,
	}
}

func (rr *ruler) start() error {
	hubHandler := mqtt.NewHandlerWrapper(
		func(p *packet.Publish) error {
			msg := p.Message
			kafkaMsg := kafka.Message{
				Key:   []byte(msg.Topic),
				Value: msg.Payload,
			}
			err := rr.client.WriteMessages(kafkaMsg)
			if err != nil {
				rr.log.Errorf("failed to writer msg id=%d to kafka")
				return err
			}
			if p.Message.QOS == 1 {
				r := &packet.Puback{ID: p.ID}
				return rr.hub.Send(r)
			}
			return nil
		},
		func(p *packet.Puback) error {
			return nil
		},
		func(e error) {
			rr.log.Errorln("hub error:", e.Error())
		},
	)
	if err := rr.hub.Start(hubHandler); err != nil {
		return err
	}
	rr.client.SetReadHandler(func(msg kafka.Message) error {
		for _, subscription := range rr.rule.Hub.Subscriptions {
			pkt := packet.NewPublish()
			pkt.Message.Topic = subscription.Topic
			pkt.Message.QOS = packet.QOS(subscription.QOS)
			pkt.Message.Payload = msg.Value
			if err := rr.hub.Send(pkt); err != nil {
				return err
			}
		}
		return nil
	})
	if err := rr.client.StartRead(); err != nil {
		return err
	}
	return nil
}

func (rr *ruler) close() {
	rr.hub.Close()
	rr.client.Close()
}

func defaults(rule *Rule, hub *mqtt.ClientInfo) {
	hub.ClientID = rule.Hub.ClientID + rule.Type + rule.Remote.Name
	hub.Subscriptions = rule.Hub.Subscriptions
}
