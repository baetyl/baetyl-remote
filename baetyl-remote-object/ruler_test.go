package main

import (
	"testing"

	"github.com/baetyl/baetyl-go/mqtt"
	"github.com/stretchr/testify/assert"
)

var ru = &Rule{
	Hub: struct {
		ClientID      string          `yaml:"clientid" json:"clientid"`
		Subscriptions []mqtt.QOSTopic `yaml:"subscriptions" json:"subscriptions" default:"[]"`
	}{
		ClientID:      "",
		Subscriptions: []mqtt.QOSTopic(nil),
	},
	Client: struct {
		Name string `yaml:"name" json:"name" validate:"nonzero"`
	}{
		Name: "example",
	},
}

func TestDefaults(t *testing.T) {
	// round 1: mqtt client ID is empty
	hub := new(mqtt.ClientConfig)
	defaults(ru, hub)
	assert.Equal(t, "example", hub.ClientID)

	// round 2: hub client ID is not empty
	ru.Hub.ClientID = "hub-test-1"
	ru.Hub.Subscriptions = []mqtt.QOSTopic{mqtt.QOSTopic{Topic: "t"}}
	defaults(ru, hub)
	assert.Equal(t, "hub-test-1", hub.ClientID)
}

func TestNewMQTTClient(t *testing.T) {
	// mqtt endpoint is not configured
	obs := new(mockMqttObserver)
	cc := mqtt.ClientConfig{}
	topics := make([]mqtt.QOSTopic, 0)
	client, err := NewMQTTClient(cc, obs, topics)
	assert.Nil(t, client)
	assert.Equal(t, "mqtt endpoint not configured", err.Error())

	// mqtt endpoint is configured
	cc.Address = "tcp://127.0.0.1:2333"
	client, err = NewMQTTClient(cc, obs, topics)
	assert.NotNil(t, client)
	assert.NoError(t, err)
	client.Close()
}

func TestProcessEvent(t *testing.T) {
	pkt := &mqtt.Publish{
		ID: mqtt.ID(1),
		Message: mqtt.Message{
			Topic:   "t",
			QOS:     0,
			Payload: []byte("for test"),
		},
	}
	hub := new(mqtt.ClientConfig)
	cfg.Name = "test"
	cfg.Kind = Kind("BOS")
	cli, err := NewClient(*cfg)
	assert.NoError(t, err)
	ruler, err := NewRuler(*ru, cli)
	assert.NoError(t, err)
	assert.NotNil(t, ruler)
	ruler.Start(*hub)
	event, err := ruler.processEvent(pkt)
	assert.Nil(t, event)
	assert.NotNil(t, err)
	assert.Equal(t, "event invalid: event type unexpected", err.Error())

	pkt.Message.Payload = []byte(
		`{
			"type": "UPLOAD",
			"content": {
				"remotePath": "image/test.png",
				"zip": false,
				"localPath": "var/lib/baetyl/image/test.png"
		}}`)
	event, err = ruler.processEvent(pkt)
	assert.Nil(t, err)
	assert.Equal(t, EventType("UPLOAD"), event.Type)
	assert.Equal(t, &UploadEvent{RemotePath: "image/test.png", LocalPath: "var/lib/baetyl/image/test.png", Zip: false}, event.Content)
}

type mockMqttObserver struct{}

func (*mockMqttObserver) OnPublish(*mqtt.Publish) error {
	return nil
}

func (*mockMqttObserver) OnPuback(*mqtt.Puback) error {
	return nil
}

func (*mockMqttObserver) OnError(err error) {}
