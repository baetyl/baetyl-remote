package main

import (
	"fmt"
	"sync"

	"github.com/baetyl/baetyl-go/log"
	"github.com/baetyl/baetyl-go/mqtt"
)

// Ruler struct
type Ruler struct {
	rule *Rule
	cli  Client
	hub  *mqtt.Client
	log  *log.Logger
	tm   sync.Map
}

// NewRuler can create a ruler
func NewRuler(rule Rule, cli Client) (*Ruler, error) {
	ruler := &Ruler{
		rule: &rule,
		cli:  cli,
		log:  log.With(log.Any("rule", rule.Client.Name)),
	}
	return ruler, nil
}

// Close can create a ruler
func (r *Ruler) Close() {
	r.hub.Close()
}

// Start can create a ruler
func (r *Ruler) Start(cc mqtt.ClientConfig) error {
	defaults(r.rule, &cc)
	hub, err := NewMQTTClient(cc, r, r.rule.Hub.Subscriptions)
	if err != nil {
		return fmt.Errorf("failed to create mqtt client: %s", err.Error())
	}
	r.hub = hub
	return nil
}

// NewMQTTClient creates a mqtt client
func NewMQTTClient(cc mqtt.ClientConfig, obs mqtt.Observer, topics []mqtt.QOSTopic) (*mqtt.Client, error) {
	if cc.Address == "" {
		return nil, fmt.Errorf("mqtt endpoint not configured")
	}
	cli, err := mqtt.NewClient(cc, obs)
	if err != nil {
		return nil, err
	}
	var subs []mqtt.Subscription
	for _, topic := range topics {
		subs = append(subs, mqtt.Subscription{Topic: topic.Topic, QOS: mqtt.QOS(topic.QOS)})
	}
	if len(subs) > 0 {
		err = cli.Subscribe(subs)
		if err != nil {
			return nil, err
		}
	}
	return cli, nil
}

// OnPublish create a ruler
func (r *Ruler) OnPublish(pkt *mqtt.Publish) error {
	event, err := r.processEvent(pkt)
	if err != nil {
		r.log.Error(err.Error())
		return err
	}
	msg := &EventMessage{
		ID:    uint64(pkt.ID),
		QOS:   uint32(pkt.Message.QOS),
		Topic: pkt.Message.Topic,
		Event: event,
	}
	return r.RuleHandler(msg)
}

// OnPuback handles puback packet
func (r *Ruler) OnPuback(pkt *mqtt.Puback) error {
	return nil
}

// OnError handles error
func (r *Ruler) OnError(err error) {
	r.log.Error(err.Error())
}

// processEvent processes event
func (r *Ruler) processEvent(pkt *mqtt.Publish) (*Event, error) {
	r.log.Debug("start to process event", log.Any("payload", string(pkt.Message.Payload)))
	e, err := NewEvent(pkt.Message.Payload)
	if err != nil {
		return nil, fmt.Errorf("event invalid: %s", err.Error())
	}
	return e, nil
}

// RuleHandler filter topic & handler
func (r *Ruler) RuleHandler(msg *EventMessage) error {
	if msg.QOS == 1 {
		if _, ok := r.tm.Load(msg.ID); !ok {
			r.tm.Store(msg.ID, struct{}{})
		} else {
			return nil
		}
	}
	return r.cli.CallAsync(msg, r.callback)
}

func (r *Ruler) callback(msg *EventMessage, err error) {
	if err != nil {
		r.log.Error(err.Error())
	}
	if msg.QOS == 1 && err == nil {
		r.tm.Delete(msg.ID)
	}
}

// defaults sets clientID for mqtt client
func defaults(rule *Rule, cc *mqtt.ClientConfig) {
	if rule.Hub.ClientID != "" {
		cc.ClientID = rule.Hub.ClientID
	} else {
		cc.ClientID = rule.Client.Name
	}
}
