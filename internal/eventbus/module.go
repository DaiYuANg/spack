package eventbus

import (
	"log/slog"

	"github.com/daiyuang/spack/internal/constant"
	"github.com/stanipetrosyan/go-eventbus"
	"go.uber.org/fx"
)

var Module = fx.Module("event_bus",
	fx.Provide(goeventbus.NewEventBus),
	fx.Invoke(listenCompressEvent),
)

func listenCompressEvent(eventbus goeventbus.EventBus, logger *slog.Logger) {
	eventbus.Channel(constant.CompressEvent).Subscriber().Listen(func(context goeventbus.Context) {
		logger.Debug("Message ", slog.Any("value", context.Result().Extract()))
	})
}
