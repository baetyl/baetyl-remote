package main

import (
	"github.com/baetyl/baetyl/protocol/mqtt"
	"time"
)

// Config custom configuration of the timer module
type Config struct {
	// slave list
	Remotes []Remote `yaml:"remotes" json:"remotes"`
	// parse item list
	Rules []Rule `yaml:"rules" json:"rules"`
}

// Slave kafka slave device configuration
type Remote struct {
	Name string `yaml:"name" json:"name"`
	Address []string `yaml:"address" json:"address"`
}

// ParseItem parse Item configuration
type Rule struct {
	Type string `yaml:"type" json:"type" validate:"regexp=^(to|from)?$"`
	Hub struct {
		ClientID      string           `yaml:"clientid" json:"clientid"`
		Subscriptions []mqtt.TopicInfo `yaml:"subscriptions" json:"subscriptions" default:"[]"`
	} `yaml:"hub" json:"hub"`
	Remote struct {
		Name string `yaml:"name" json:"name"`
		Topic string `yaml:"topic" json:"topic"`
		GroupID string `yaml:"group_id" json:"group_id"`
		MinBytes int `yaml:"min_bytes" json:"min_bytes" default:"10e3"` // 10kB
		MaxBytes int `yaml:"max_bytes" json:"max_bytes" default:"10e6"` // 10MB
		MaxWait time.Duration `yaml:"max_wait" json:"max_wait" default:"1s"`
	} `yaml:"remote" json:"remote"`
}
