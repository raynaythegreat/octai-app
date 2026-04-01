package discord

import (
	"github.com/raynaythegreat/octai-app/pkg/bus"
	"github.com/raynaythegreat/octai-app/pkg/channels"
	"github.com/raynaythegreat/octai-app/pkg/config"
)

func init() {
	channels.RegisterFactory("discord", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewDiscordChannel(cfg.Channels.Discord, b)
	})
}
