// Routines for interacting with the Untappd API.
package syndicate

import (
	"math"
	"net/http"
	"time"

	"github.com/mdlayher/untappd"
	gocache "github.com/patrickmn/go-cache"
)

var (
	Untappd *UntappdClient
)

type UntappdClient struct {
	utc          *untappd.Client
	checkinCache *gocache.Cache
}

func NewUntappdClient(untappdID, untappdSecret string) error {
	utc, err := untappd.NewClient(untappdID, untappdSecret, &http.Client{})
	if err != nil {
		return err
	}
	Untappd = &UntappdClient{
		utc: utc,
		checkinCache: gocache.New(
			10*time.Minute,
			10*time.Minute,
		),
	}
	return nil
}

// UntappdGetBeerInfo returns untappd info, given an untappd beer id.
func (u *UntappdClient) GetBeerInfo(id int64) (*untappd.Beer, *http.Response, error) {
	return u.utc.Beer.Info(int(id), true)
}

// UntappdGetUserCheckins returns the last 'count' untappd checkins for 'user'.
func (u *UntappdClient) GetUserCheckins(id string, count int) ([]*untappd.Checkin, error) {
	checkins, found := u.checkinCache.Get(id)
	if found {
		return checkins.([]*untappd.Checkin), nil
	}
	newCheckins, _, err := u.utc.User.CheckinsMinMaxIDLimit(id, 0, math.MaxInt32, count)
	if err != nil {
		return nil, err
	}
	if err := u.checkinCache.Add(id, newCheckins, gocache.DefaultExpiration); err != nil {
		return nil, err
	}
	return newCheckins, nil
}
