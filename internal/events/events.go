package events

import (
	"context"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/webhook"
)

var _ bot.EventListener = (*Handler)(nil)

type Handler struct {
	cfg           *config.Config
	webhookClient *webhook.Client
}

func NewHandler(cfg *config.Config, env config.Env) *Handler {
	var webhookClient *webhook.Client

	if env.EventsWebhook != "" {
		client, err := webhook.NewWithURL(env.EventsWebhook)
		if err != nil {
			log.Errorf("Failed to initialize events webhook client: %v", err)
		} else {
			webhookClient = client
			log.Info("Events webhook client initialized")
		}
	} else {
		log.Warn("BOT_EVENTS_WEBHOOK not provided; event webhook logging disabled")
	}

	return &Handler{
		cfg:           cfg,
		webhookClient: webhookClient,
	}
}

func (h *Handler) OnEvent(event bot.Event) {
	switch e := event.(type) {
	case *dgevents.GuildJoin:
		h.OnGuildJoin(e)
	}
}

func (h *Handler) Close(ctx context.Context) {
	if h.webhookClient != nil {
		h.webhookClient.Close(ctx)
	}
}
