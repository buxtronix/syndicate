// Routines for interacting with the Untappd API.
package syndicate

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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

// GetBeerInfo returns untappd info, given an untappd beer id.
func (u *UntappdClient) GetBeerInfo(id int64) (*untappd.Beer, *http.Response, error) {
	return u.utc.Beer.Info(int(id), true)
}

// SearchBeer returns a list of beers matching the search query.
func (u *UntappdClient) SearchBeer(query string) ([]*untappd.Beer, *http.Response, error) {
	return u.utc.Beer.Search(query)
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

// Takes a shortcut URL, typically of the form https://untp.beer/0rqOe and
// queries it to fetch the beer ID.
func ResolveShortURL(uri string) (int64, error) {
	var number string
	if strings.Contains(uri, "untp.beer") {
		// Create a new http client.
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		// Make the request to the url.
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return 0, err
		}

		// Do the request.
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}

		// Check the response status code.
		if resp.StatusCode != 302 {
			return 0, fmt.Errorf("Expected a 302 redirect, got %d", resp.StatusCode)
		}

		// Get the redirect url.
		redirectUrl, err := resp.Location()
		if err != nil {
			return 0, err
		}

		// Get the number from the redirect url.
		number = redirectUrl.Path[strings.LastIndex(redirectUrl.Path, "/")+1:]
	} else {
		number = uri[strings.LastIndex(uri, "/")+1:]
	}

	// Convert the number to an int64.
	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, err
	}

	// Return the number.
	return i, nil
}
