package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRule(t *testing.T) {
	clients := map[string]*Client{
		"cli1": &Client{},
	}

	ruleInfo := RuleInfo{
		Name: "",
		Source: struct {
			QOS   uint32 `yaml:"qos" json:"qos" validate:"min=0, max=1"`
			Topic string `yaml:"topic" json:"topic" validate:"nonzero"`
		}{
			QOS:   1,
			Topic: "t1",
		},
		Target: struct {
			Client string `yaml:"client" json:"client" default:"baetyl-broker"`
		}{
			Client: "cli1",
		},
	}

	ruler, err := NewRuler(ruleInfo, clients, "ruletest")
	assert.NoError(t, err)
	time.Sleep(time.Second)
	ruler.Close()
}
