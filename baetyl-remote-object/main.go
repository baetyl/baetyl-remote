package main

import (
	"fmt"

	"github.com/baetyl/baetyl-go/context"
)

// mo bridge module of mqtt servers
type mo struct {
	cfg Config
	rrs []*Ruler
}

func main() {
	context.Run(func(ctx context.Context) error {
		var cfg Config
		err := ctx.LoadConfig(&cfg)
		if err != nil {
			return err
		}
		// http clients
		clients := make(map[string]Client)
		for _, c := range cfg.Clients {
			clients[c.Name], err = NewClient(c)
			defer clients[c.Name].Close()
			if err != nil {
				return err
			}
		}
		// rulers
		rulers := make([]*Ruler, 0)
		for _, rule := range cfg.Rules {
			cli, ok := clients[rule.Client.Name]
			if !ok {
				return fmt.Errorf("client (%s) not found", rule.Client.Name)
			}
			ruler, err := NewRuler(rule, cli)
			if err != nil {
				return fmt.Errorf("failed to create ruler")
			}
			rulers = append(rulers, ruler)
		}
		defer func() {
			for _, ruler := range rulers {
				ruler.Close()
			}
		}()
		for _, cli := range clients {
			err := cli.Start()
			if err != nil {
				return err
			}
		}
		for _, ruler := range rulers {
			err := ruler.Start(ctx.Config().Mqtt)
			if err != nil {
				return err
			}
		}
		ctx.Wait()
		return nil
	})
}
