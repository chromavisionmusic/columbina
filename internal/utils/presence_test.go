package utils

import (
	"testing"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
)

func applyOpts(opts []gateway.PresenceOpt) gateway.MessageDataPresenceUpdate {
	var update gateway.MessageDataPresenceUpdate
	for _, opt := range opts {
		opt(&update)
	}
	return update
}

func TestGetPresenceOpts_Status(t *testing.T) {
	tests := []struct {
		status     string
		wantStatus discord.OnlineStatus
	}{
		{"dnd", discord.OnlineStatusDND},
		{"do_not_disturb", discord.OnlineStatusDND},
		{"DND", discord.OnlineStatusDND},
		{"idle", discord.OnlineStatusIdle},
		{"IDLE", discord.OnlineStatusIdle},
		{"invisible", discord.OnlineStatusInvisible},
		{"offline", discord.OnlineStatusOffline},
		{"online", discord.OnlineStatusOnline},
		{"unknown", discord.OnlineStatusOnline},
		{"", discord.OnlineStatusOnline},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			opts := GetPresenceOpts(config.Presence{
				Status: tt.status,
			})
			update := applyOpts(opts)
			if update.Status != tt.wantStatus {
				t.Errorf("Status for %q = %v, want %v", tt.status, update.Status, tt.wantStatus)
			}
			if len(update.Activities) != 0 {
				t.Errorf("Expected 0 activities for empty activity, got %d", len(update.Activities))
			}
		})
	}
}

func TestGetPresenceOpts_Activities(t *testing.T) {
	tests := []struct {
		name         string
		activityType string
		activityText string
		wantType     discord.ActivityType
		checkName    bool
	}{
		{
			name:         "listening",
			activityType: "listening",
			activityText: "music",
			wantType:     discord.ActivityTypeListening,
			checkName:    true,
		},
		{
			name:         "watching",
			activityType: "WATCHING",
			activityText: "a movie",
			wantType:     discord.ActivityTypeWatching,
			checkName:    true,
		},
		{
			name:         "competing",
			activityType: "competing",
			activityText: "in chess",
			wantType:     discord.ActivityTypeCompeting,
			checkName:    true,
		},
		{
			name:         "streaming",
			activityType: "streaming",
			activityText: "live stream",
			wantType:     discord.ActivityTypeStreaming,
			checkName:    true,
		},
		{
			name:         "custom",
			activityType: "custom",
			activityText: "just chilling",
			wantType:     discord.ActivityTypeCustom,
			checkName:    false,
		},
		{
			name:         "playing",
			activityType: "playing",
			activityText: "Genshin Impact",
			wantType:     discord.ActivityTypeGame,
			checkName:    true,
		},
		{
			name:         "default unknown activity type defaults to playing",
			activityType: "other_type",
			activityText: "a game",
			wantType:     discord.ActivityTypeGame,
			checkName:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := GetPresenceOpts(config.Presence{
				Status:       "online",
				ActivityType: tt.activityType,
				Activity:     tt.activityText,
			})
			update := applyOpts(opts)

			if len(update.Activities) != 1 {
				t.Fatalf("expected 1 activity, got %d", len(update.Activities))
			}

			act := update.Activities[0]
			if act.Type != tt.wantType {
				t.Errorf("activity type = %v, want %v", act.Type, tt.wantType)
			}
			if tt.checkName && act.Name != tt.activityText {
				t.Errorf("activity name = %q, want %q", act.Name, tt.activityText)
			}
		})
	}
}
