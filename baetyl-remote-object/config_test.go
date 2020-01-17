package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/baetyl/baetyl/utils"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var c Config

	// round 1: load normal configuration yaml file
	err := utils.LoadYAML("example/etc/baetyl/service-bos.yml", &c)
	assert.Nil(t, err)

	assert.Len(t, c.Clients, 1)
	assert.Equal(t, "baidu-bos", c.Clients[0].Name)
	assert.Equal(t, "bj.bcebos.com", c.Clients[0].Address)
	assert.Equal(t, Kind("BOS"), c.Clients[0].Kind)
	assert.Equal(t, int64(10485760), c.Clients[0].MultiPart.PartSize)
	assert.Equal(t, 10, c.Clients[0].MultiPart.Concurrency)
	assert.Equal(t, 1000, c.Clients[0].Pool.Worker)
	assert.Equal(t, time.Duration(30000000000), c.Clients[0].Pool.Idletime)
	assert.Equal(t, "bos-remote-demo", c.Clients[0].Bucket)
	assert.Equal(t, "var/lib/baetyl/tmp", c.Clients[0].TempPath)
	assert.Equal(t, true, c.Clients[0].Limit.Enable)
	assert.Equal(t, int64(9663676416), c.Clients[0].Limit.Data)
	assert.Equal(t, "var/lib/baetyl/data/stats.yml", c.Clients[0].Limit.Path)
	assert.Equal(t, time.Duration(60000000000), c.Clients[0].Report.Interval)

	assert.Len(t, c.Rules, 1)
	assert.Equal(t, "remote-write-bos", c.Rules[0].Hub.ClientID)
	assert.Equal(t, "t", c.Rules[0].Hub.Subscriptions[0].Topic)

	// round 2: load bad configuration yaml file
	err = utils.LoadYAML("example/test/baetyl/service.yml", &c)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid size: '10a'", err.Error())
}

func TestDumpYAML(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	tmpfile, err := ioutil.TempFile(dir, "test")
	assert.Nil(t, err)
	err = DumpYAML(tmpfile.Name(), cfg)
	assert.Nil(t, err)
}
