package onebot

import (
	"github.com/raynaythegreat/octai-app/pkg/bus"
	"github.com/raynaythegreat/octai-app/pkg/channels"
	"github.com/raynaythegreat/octai-app/pkg/config"
)

func init() {
	channels.RegisterFactory("onebot", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewOneBotChannel(cfg.Channels.OneBot, b)
	})
}
