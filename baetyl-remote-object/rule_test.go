package main

import (
	"testing"

	"github.com/256dpi/gomqtt/packet"
	"github.com/baetyl/baetyl/protocol/mqtt"
	"github.com/stretchr/testify/assert"
)

var ru = &Rule{
	Hub: struct {
		ClientID      string           `yaml:"clientid" json:"clientid"`
		Subscriptions []mqtt.TopicInfo `yaml:"subscriptions" json:"subscriptions" default:"[]"`
	}{
		ClientID:      "",
		Subscriptions: []mqtt.TopicInfo(nil),
	},
	Client: struct {
		Name string `yaml:"name" json:"name" validate:"nonzero"`
	}{
		Name: "example",
	},
}

func TestDefaults(t *testing.T) {
	// round 1: hub client ID is empty
	hub := new(mqtt.ClientInfo)
	defaults(ru, hub)
	assert.Equal(t, []mqtt.TopicInfo(nil), hub.Subscriptions)
	assert.Equal(t, "example", hub.ClientID)

	// round 2: hub client ID is not empty
	ru.Hub.ClientID = "hub-test-1"
	ru.Hub.Subscriptions = []mqtt.TopicInfo{mqtt.TopicInfo{Topic: "t"}}
	defaults(ru, hub)
	assert.Equal(t, ru.Hub.Subscriptions, hub.Subscriptions)
	assert.Equal(t, "hub-test-1", hub.ClientID)
}

func TestProcessEvent(t *testing.T) {
	pkt := &packet.Publish{
		ID: packet.ID(1),
		Message: packet.Message{
			Topic:   "t",
			QOS:     0,
			Payload: []byte("for test"),
		},
	}
	hub := new(mqtt.ClientInfo)
	cfg.Kind = Kind("BOS")
	cli, err := NewClient(*cfg, r)
	assert.Nil(t, err)
	ruler := NewRuler(*ru, *hub, cli)
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
