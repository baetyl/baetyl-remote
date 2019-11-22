package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const CREATECLIENTERROR = "failed to create storage client (test): kind type unexpected"

func example(map[string]interface{}) error {
	return nil
}

var r = report(example)

var cfg = &ClientInfo{
	Name: "test",
	Report: struct {
		Interval time.Duration `yaml:"interval" json:"interval" default:"1m"`
	}{
		Interval: time.Duration(1000000000),
	},
}

func TestNewClient(t *testing.T) {
	// round 1: test BOS client
	cfg.Kind = Kind("BOS")
	_, err := NewClient(*cfg, r)
	assert.Nil(t, err)

	// round 2: test CEPH client
	cfg.Kind = Kind("CEPH")
	_, err = NewClient(*cfg, r)
	assert.Nil(t, err)

	// round 3: test AWS S3 client
	cfg.Kind = Kind("S3")
	cfg.Region = "us-east-1"
	_, err = NewClient(*cfg, r)
	assert.Nil(t, err)

	// round 4: test default
	cfg.Kind = Kind("TEST")
	_, err = NewClient(*cfg, r)
	assert.NotNil(t, err)
	assert.Equal(t, CREATECLIENTERROR, err.Error())
}
