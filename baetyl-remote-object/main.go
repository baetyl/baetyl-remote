package main

import (
	"time"

	"github.com/baetyl/baetyl-go/v2/context"
	"github.com/baetyl/baetyl-go/v2/errors"
	"github.com/baetyl/baetyl-go/v2/log"
	"github.com/baetyl/baetyl-go/v2/utils"
)

func main() {
	context.Run(func(ctx context.Context) error {
		var cfg Config
		err := ctx.LoadCustomConfig(&cfg)
		if err != nil {
			return err
		}
		err = AddDefaultCfg(&cfg)
		if err != nil {
			log.L().Warn("failed to init default ipc", log.Error(err))
		}

		// clients
		clients := make(map[string]*Client)
		defer func() {
			for _, c := range clients {
				c.Close()
			}
		}()
		for _, c := range cfg.Clients {
			client, err := NewClient(ctx, c)
			if err != nil {
				return err
			}
			clients[c.Name] = client
		}

		// rulers
		rulers := make([]*Ruler, 0)
		defer func() {
			for _, r := range rulers {
				r.Close()
			}
		}()
		for _, ruleInfo := range cfg.Rules {
			ruler, err := NewRuler(ctx, ruleInfo, clients)
			if err != nil {
				return err
			}
			rulers = append(rulers, ruler)
		}
		ctx.Wait()
		return nil
	})
}

func AddDefaultCfg(cfg *Config) error {
	cliInfo := ClientInfo{
		Name:         MinioStsCli,
		Kind:         S3,
		ObjectConfig: ObjectConfig{},
		DefaultPath:  "",
		StsDeadline:  time.Now(),
	}
	err := utils.SetDefaults(&cliInfo)
	if err != nil {
		return errors.Trace(err)
	}
	cfg.Clients = append(cfg.Clients, cliInfo)
	cfg.Rules = append(cfg.Rules, RuleInfo{
		Name: IpcRule,
		Source: struct {
			QOS   uint32 `yaml:"qos" json:"qos" validate:"min=0, max=1"`
			Topic string `yaml:"topic" json:"topic" validate:"nonzero"`
		}{QOS: 0, Topic: BaetylIpcTopic},
		Target: struct {
			Client string `yaml:"client" json:"client" default:"baetyl-sts"`
		}{Client: MinioStsCli},
	})
	return nil
}
