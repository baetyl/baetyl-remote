package main

import (
	"github.com/baetyl/baetyl/sdk/baetyl-go"
)

func main() {
	// Running module in baetyl context
	baetyl.Run(func(ctx baetyl.Context) error {
		var cfg Config
		// load custom config
		err := ctx.LoadConfig(&cfg)
		if err != nil {
			return err
		}
		log := ctx.Log()
		remotes := make(map[string]Remote)
		for _, remote := range cfg.Remotes {
			remotes[remote.Name] = remote
		}
		rulers := make([]*ruler, 0)
		for _, rule := range cfg.Rules {
			remote, ok := remotes[rule.Remote.Name]
			if ok {
				client := newClient(remote.Address, rule, log)
				rulers = append(rulers, create(rule, ctx.Config().Hub, client))
			} else {
				log.Errorf("remote (%s) not found", rule.Remote.Name)
			}
		}
		defer func() {
			for _, ruler := range rulers {
				ruler.close()
			}
		}()
		for _, ruler := range rulers {
			err := ruler.start()
			if err != nil {
				return err
			}
		}
		ctx.Wait()
		return nil
	})
}
