package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var cfg = &ClientInfo{}

func TestNewClient(t *testing.T) {
	// normal client test
	tests := []struct {
		kind Kind
	}{
		{kind: Kind("BOS")},
		{kind: Kind("CEPH")},
		{kind: Kind("S3")},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			cfg.Kind = tt.kind
			_, err := NewClient(*cfg)
			assert.NoError(t, err)
		})
	}

	// abnormal client test
	cfg.Kind = Kind("TEST")
	_, err := NewClient(*cfg)
	assert.Equal(t, "failed to create storage client (): kind type unexpected", err.Error())
}
