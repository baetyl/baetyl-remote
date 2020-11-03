package main

import (
	"github.com/baetyl/baetyl-go/v2/context"
)

func main() {
	context.Run(func(ctx context.Context) error {
		if err := ctx.CheckSystemCert(); err != nil {
			return err
		}

		var cfg Config
		err := ctx.LoadCustomConfig(&cfg)
		if err != nil {
			return err
		}

		// clients
		clients := make(map[string]*Client)
		defer func() {
			for _, c := range clients {
				c.Close()
			}
		}()
		for _, c := range cfg.Clients {
			client, err := NewClient(c)
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
