// Routines for interacting with the Untappd API.
package syndicate

import (
	"math"
	"net/http"

	"github.com/mdlayher/untappd"
)

var (
	utc           *untappd.Client
	UntappdID     string
	UntappdSecret string
)

func makeUTC() {
	var err error
	if utc != nil {
		return
	}
	utc, err = untappd.NewClient(UntappdID, UntappdSecret, &http.Client{})
	if err != nil {
		panic(err)
	}
}

// UntappdGetBeerInfo returns untappd info, given an untappd beer id.
func UntappdGetBeerInfo(id int64) (*untappd.Beer, *http.Response, error) {
	makeUTC()
	return utc.Beer.Info(int(id), true)
}

// UntappdGetUserCheckins returns the last 'count' untappd checkins for 'user'.
func UntappdGetUserCheckins(id string, count int) ([]*untappd.Checkin, *http.Response, error) {
	makeUTC()
	return utc.User.CheckinsMinMaxIDLimit(id, 0, math.MaxInt32, count)
}
