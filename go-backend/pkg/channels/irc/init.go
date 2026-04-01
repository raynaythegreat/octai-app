package irc

import (
	"github.com/raynaythegreat/octai-app/pkg/bus"
	"github.com/raynaythegreat/octai-app/pkg/channels"
	"github.com/raynaythegreat/octai-app/pkg/config"
)

func init() {
	channels.RegisterFactory("irc", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		if !cfg.Channels.IRC.Enabled {
			return nil, nil
		}
		return NewIRCChannel(cfg.Channels.IRC, b)
	})
}
