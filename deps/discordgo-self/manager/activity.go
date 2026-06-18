package manager

import (
	"time"

	"github.com/hytams/discordgo-self/gateway"
	"github.com/hytams/discordgo-self/types"
)

// ActivityManager manages user activities.
type ActivityManager struct {
	Gateway *gateway.Gateway
	stop    chan struct{}
}

// NewActivityManager creates a new activity manager.
func NewActivityManager(gw *gateway.Gateway) *ActivityManager {
	return &ActivityManager{
		Gateway: gw,
		stop:    make(chan struct{}),
	}
}

// SetActivity sets the current activity.
func (am *ActivityManager) SetActivity(activity *types.Activity) error {
	activities := []interface{}{*activity}
	return am.Gateway.UpdatePresence(string(types.StatusOnline), activities, false)
}

// ActivityBuilder helps construct activities.
type ActivityBuilder struct {
	activity *types.Activity
}

// NewActivityBuilder creates a new activity builder.
func NewActivityBuilder(name string, activityType types.ActivityType) *ActivityBuilder {
	return &ActivityBuilder{
		activity: &types.Activity{
			Name: name,
			Type: activityType,
		},
	}
}

// WithDetails sets the details.
func (ab *ActivityBuilder) WithDetails(details string) *ActivityBuilder {
	ab.activity.Details = details
	return ab
}

// WithState sets the state.
func (ab *ActivityBuilder) WithState(state string) *ActivityBuilder {
	ab.activity.State = state
	return ab
}

// WithTime sets the start and end timestamps.
func (ab *ActivityBuilder) WithTime(start, end time.Time) *ActivityBuilder {
	if ab.activity.Timestamps == nil {
		ab.activity.Timestamps = &types.Timestamps{}
	}
	if !start.IsZero() {
		ab.activity.Timestamps.Start = start.UnixMilli()
	}
	if !end.IsZero() {
		ab.activity.Timestamps.End = end.UnixMilli()
	}
	return ab
}

// WithLargeImage sets the large image asset and text.
func (ab *ActivityBuilder) WithLargeImage(assetID, text string) *ActivityBuilder {
	if ab.activity.Assets == nil {
		ab.activity.Assets = &types.Assets{}
	}
	ab.activity.Assets.LargeImage = assetID
	ab.activity.Assets.LargeText = text
	return ab
}

// WithSmallImage sets the small image asset and text.
func (ab *ActivityBuilder) WithSmallImage(assetID, text string) *ActivityBuilder {
	if ab.activity.Assets == nil {
		ab.activity.Assets = &types.Assets{}
	}
	ab.activity.Assets.SmallImage = assetID
	ab.activity.Assets.SmallText = text
	return ab
}

// WithButton adds a custom button.
func (ab *ActivityBuilder) WithButton(label, url string) *ActivityBuilder {
	return ab
}

// WithParty sets party information.
func (ab *ActivityBuilder) WithParty(id string, current, max int) *ActivityBuilder {
	ab.activity.Party = &types.Party{
		ID:   id,
		Size: []int{current, max},
	}
	return ab
}

// Build returns the constructed activity.
func (ab *ActivityBuilder) Build() *types.Activity {
	return ab.activity
}

// SpoofSpotify creates a fake Spotify activity.
func SpoofSpotify(song, artist, album, coverID string) *types.Activity {
	start := time.Now().UnixMilli()
	end := time.Now().Add(3 * time.Minute).UnixMilli()

	return &types.Activity{
		Name:    "Spotify",
		Type:    types.ActivityTypeListening,
		Details: song,
		State:   artist + "; " + album,
		Timestamps: &types.Timestamps{
			Start: start,
			End:   end,
		},
		Assets: &types.Assets{
			LargeImage: "spotify:" + coverID,
			LargeText:  album,
		},
		ApplicationID: 770205856722026526,
		Flags:         48,
	}
}

// StartRotatingActivities cycles through a list of activities.
func (am *ActivityManager) StartRotatingActivities(interval time.Duration, activities []*types.Activity) {
	ticker := time.NewTicker(interval)
	go func() {
		i := 0
		for {
			select {
			case <-ticker.C:
				if len(activities) == 0 {
					continue
				}
				am.SetActivity(activities[i])
				i = (i + 1) % len(activities)
			case <-am.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the activity manager.
func (am *ActivityManager) Stop() {
	close(am.stop)
}
