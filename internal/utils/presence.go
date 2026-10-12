package utils

import (
	"strings"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

func GetPresenceOpts(presence config.Presence) []gateway.PresenceOpt {
	var opts []gateway.PresenceOpt

	switch strings.ToLower(presence.Status) {
	case "dnd", "do_not_disturb":
		opts = append(opts, gateway.WithOnlineStatus(discord.OnlineStatusDND))
	case "idle":
		opts = append(opts, gateway.WithOnlineStatus(discord.OnlineStatusIdle))
	case "invisible":
		opts = append(opts, gateway.WithOnlineStatus(discord.OnlineStatusInvisible))
	case "offline":
		opts = append(opts, gateway.WithOnlineStatus(discord.OnlineStatusOffline))
	default:
		opts = append(opts, gateway.WithOnlineStatus(discord.OnlineStatusOnline))
	}

	if presence.Activity != "" {
		switch strings.ToLower(presence.ActivityType) {
		case "listening":
			opts = append(opts, gateway.WithListeningActivity(presence.Activity))
		case "watching":
			opts = append(opts, gateway.WithWatchingActivity(presence.Activity))
		case "competing":
			opts = append(opts, gateway.WithCompetingActivity(presence.Activity))
		case "streaming":
			opts = append(opts, gateway.WithStreamingActivity(presence.Activity, ""))
		case "custom":
			opts = append(opts, gateway.WithCustomActivity(presence.Activity))
		case "playing":
			fallthrough
		default:
			opts = append(opts, gateway.WithPlayingActivity(presence.Activity))
		}
	}

	return opts
}
