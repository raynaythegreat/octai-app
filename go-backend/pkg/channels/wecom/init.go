package wecom

import (
	"github.com/raynaythegreat/octai-app/pkg/bus"
	"github.com/raynaythegreat/octai-app/pkg/channels"
	"github.com/raynaythegreat/octai-app/pkg/config"
)

func init() {
	channels.RegisterFactory("wecom", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
		return NewChannel(cfg.Channels.WeCom, b)
	})
}
