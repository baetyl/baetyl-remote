package main

import (
	"fmt"
	"sync"

	"github.com/256dpi/gomqtt/packet"
	"github.com/baetyl/baetyl-go/v2/context"
	"github.com/baetyl/baetyl-go/v2/errors"
	"github.com/baetyl/baetyl-go/v2/log"
	"github.com/baetyl/baetyl-go/v2/mqtt"
)

// Ruler struct
type Ruler struct {
	sourceCli *mqtt.Client
	targetCli *Client
	log       *log.Logger
	tm        sync.Map
}

// NewRuler can create a ruler
func NewRuler(ctx context.Context, rule RuleInfo, targets map[string]*Client) (*Ruler, error) {
	targetCli, ok := targets[rule.Target.Client]
	if !ok {
		return nil, errors.Errorf("client (%s) not found", rule.Target.Client)
	}

	mqttCfg := ctx.SystemConfig().Broker
	mqttCfg.ClientID = fmt.Sprintf("%s-rule-%s", ctx.ServiceName(), rule.Name)
	mqttCfg.CleanSession = false
	mqttCfg.Subscriptions = []mqtt.QOSTopic{
		{
			QOS:   rule.Source.QOS,
			Topic: rule.Source.Topic,
		},
	}
	option, err := mqttCfg.ToClientOptions()
	if err != nil {
		return nil, errors.Trace(err)
	}
	mqttCli := mqtt.NewClient(option)
	ruler := &Ruler{
		sourceCli: mqttCli,
		targetCli: targetCli,
		log:       log.With(log.Any("rule", rule.Name)),
	}
	err = mqttCli.Start(mqtt.NewObserverWrapper(func(pkt *packet.Publish) error {
		event, err := ruler.processEvent(pkt)
		if err != nil {
			ruler.log.Error("error occurred in ruler.processEvent", log.Error(err))
			return nil
		}
		msg := &EventMessage{
			ID:    uint64(pkt.ID),
			QOS:   uint32(pkt.Message.QOS),
			Topic: pkt.Message.Topic,
			Event: event,
		}
		err = ruler.RuleHandler(msg)
		if err != nil {
			ruler.log.Error("error occurred in ruler.RuleHandler", log.Error(err))
		}
		return nil
	}, func(*packet.Puback) error {
		return nil
	}, func(err error) {
		ruler.log.Error("error occurs in source", log.Error(err))
	}))
	if err != nil {
		ruler.log.Error("error occurred when mqtt client start", log.Error(err))
	}
	return ruler, nil
}

// Close can create a ruler
func (r *Ruler) Close() {
	r.log.Info("rule starts to close")
	defer r.log.Info("rule closed")

	// sourceCli is internal client
	if r.sourceCli != nil {
		r.sourceCli.Close()
	}
}

func (r *Ruler) processEvent(pkt *packet.Publish) (*Event, error) {
	r.log.Debug("ruler received a event: ", log.Any("payload", string(pkt.Message.Payload)))
	e, err := NewEvent(pkt.Message.Payload)
	if err != nil {
		return nil, errors.Errorf("event invalid: %s", err.Error())
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
	return r.targetCli.CallAsync(msg, r.callback)
}

func (r *Ruler) callback(msg *EventMessage, err error) {
	if msg.QOS == 1 {
		if err == nil {
			puback := packet.NewPuback()
			puback.ID = packet.ID(msg.ID)
			err := r.sourceCli.Send(puback)
			if err != nil {
				r.log.Error("failed to send mqtt msg", log.Error(err))
			}
		}
		r.tm.Delete(msg.ID)
	}
	if err != nil {
		r.log.Error("failed to invoke object client", log.Error(err))
	}
}
