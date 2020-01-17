package main

import (
	"io/ioutil"
	"time"

	"github.com/baetyl/baetyl/protocol/mqtt"
	"github.com/docker/go-units"
	yaml "gopkg.in/yaml.v2"
)

// Kind the type of event from cloud
type Kind string

// The type of event from cloud
const (
	Bos  Kind = "BOS"
	Ceph Kind = "CEPH"
	S3   Kind = "S3"
)

// Config config of module
type Config struct {
	Clients []ClientInfo `yaml:"clients" json:"clients" default:"[]"`
	Rules   []Rule       `yaml:"rules" json:"rules" default:"[]"`
}

// Backoff policy
type Backoff struct {
	Max   int           `yaml:"max" json:"max"`                   // retry max
	Delay time.Duration `yaml:"delay" json:"delay" default:"20s"` // delay time
	Base  time.Duration `yaml:"base" json:"base" default:"0.3s"`  // base time & *2
}

// Pool go pool
type Pool struct {
	Worker   int           `yaml:"worker" json:"worker" default:"1000"`    // max worker size
	Idletime time.Duration `yaml:"idletime" json:"idletime" default:"30s"` // delay time
}

// ClientInfo client config
type ClientInfo struct {
	Name      string        `yaml:"name" json:"name" validate:"nonzero"`
	Address   string        `yaml:"address" json:"address"`
	Region    string        `yaml:"region" json:"region" default:"us-east-1"`
	Ak        string        `yaml:"ak" json:"ak" validate:"nonzero"`
	Sk        string        `yaml:"sk" json:"sk" validate:"nonzero"`
	Kind      Kind          `yaml:"kind" json:"kind" validate:"nonzero"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout" default:"30s"`
	Backoff   Backoff       `yaml:"backoff" json:"backoff"`
	Pool      Pool          `yaml:"pool" json:"pool"`
	Bucket    string        `yaml:"bucket" json:"bucket" validate:"nonzero"`
	TempPath  string        `yaml:"temppath" json:"temppath" default:"var/lib/baetyl/tmp"`
	MultiPart MultiPart     `yaml:"multipart" json:"multipart"`
	Limit     Limit         `yaml:"limit" json:"limit"`
	Report    struct {
		Interval time.Duration `yaml:"interval" json:"interval" default:"1m"`
	} `yaml:"report" json:"report"`
}

// Rule function rule config
type Rule struct {
	Hub struct {
		ClientID      string           `yaml:"clientid" json:"clientid"`
		Subscriptions []mqtt.TopicInfo `yaml:"subscriptions" json:"subscriptions" default:"[]"`
	} `yaml:"hub" json:"hub"`
	Client struct {
		Name string `yaml:"name" json:"name" validate:"nonzero"`
	} `yaml:"client" json:"client"`
}

// MultiPart config
type MultiPart struct {
	PartSize    int64 `yaml:"partsize" json:"partsize" default:"1048576000"`
	Concurrency int   `yaml:"concurrency" json:"concurrency" default:"10"`
}

type multipart struct {
	PartSize    string `yaml:"partsize" json:"partsize"`
	Concurrency int    `yaml:"concurrency" json:"concurrency"`
}

// Limit limit config
type Limit struct {
	Enable bool   `yaml:"enable" json:"enable" default:"false"`
	Data   int64  `yaml:"data" json:"data" default:"1073741824"`
	Path   string `yaml:"path" json:"path" default:"var/lib/baetyl/data/stats.yml"`
}

type limit struct {
	Enable bool   `yaml:"enable" json:"enable"`
	Data   string `yaml:"data" json:"data"`
	Path   string `yaml:"path" json:"path"`
}

// UnmarshalYAML customizes unmarshal
func (l *Limit) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ls limit
	err := unmarshal(&ls)
	if err != nil {
		return err
	}
	if ls.Enable {
		l.Enable = ls.Enable
	}
	if ls.Data != "" {
		l.Data, err = units.RAMInBytes(ls.Data)
		if err != nil {
			return err
		}
	}
	if ls.Path != "" {
		l.Path = ls.Path
	}
	return nil
}

// UnmarshalYAML customizes unmarshal
func (m *MultiPart) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ms multipart
	err := unmarshal(&ms)
	if err != nil {
		return err
	}
	if ms.PartSize != "" {
		m.PartSize, err = units.RAMInBytes(ms.PartSize)
		if err != nil {
			return err
		}
	}
	if ms.Concurrency != 0 {
		m.Concurrency = ms.Concurrency
	}
	return nil
}

// Item data count
type Item struct {
	Bytes int64 `yaml:"bytes" json:"bytes"`
	Count int64 `yaml:"count" json:"count"`
}

// Stats month stats
type Stats struct {
	Total  Item             `yaml:"total" json:"total" default:"{}"`
	Months map[string]*Item `yaml:"months" json:"months" default:"{}"`
}

// DumpYAML in interface save in config file
func DumpYAML(path string, in interface{}) error {
	bytes, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, bytes, 0755)
	if err != nil {
		return err
	}
	return nil
}
