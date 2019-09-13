package syndicate

import (
	"fmt"
	"time"

	"github.com/mdlayher/untappd"
)

// User represents a Syndicate user.
type User struct {
	// ID is the unique user id.
	ID int64
	// Name is the name of the user.
	Name string
	// UntappdID is their username on Untappd.
	UntappdID string
	// SeedFund is a cost to shift the user for net position, cents.
	SeedFund float64
}

// TotalAdded returns the total beer value added to the syndicate.
func (u *User) TotalAdded() (float64, error) {
	conts, err := DB.ListContributions()
	if err != nil {
		return 0, err
	}
	var total float64
	for _, c := range conts {
		if c.User == u.ID {
			total += c.UnitPrice * float64(c.Quantity)
		}
	}
	return total, nil
}

// TotalAdded returns the total beer value taken from the syndicate.
func (u *User) TotalTaken() (float64, error) {
	takes, err := DB.ListCheckouts()
	if err != nil {
		return 0, err
	}
	cs, err := DB.ListContributions()
	if err != nil {
		return 0, err
	}
	conts := map[int64]*Contribution{}
	for _, c := range cs {
		conts[c.ID] = c
	}
	var total float64
	for _, t := range takes {
		if t.User == u.ID {
			total += float64(t.Quantity) * conts[t.Contribution].UnitPrice
		}
	}
	return total, nil
}

// NetPosition returns the users's net financial position in the syndicate.
func (u *User) NetPosition() (float64, error) {
	added, err := u.TotalAdded()
	if err != nil {
		return 0, err
	}
	taken, err := u.TotalTaken()
	if err != nil {
		return 0, err
	}
	return added - taken + u.SeedFund, nil
}

// LastCheckins returns the users last 'count' checkins on Untappd.
func (u *User) LastCheckins(count int) ([]*untappd.Checkin, error) {
	if u.UntappdID == "" {
		return nil, nil
	}
	checkins, _, err := UntappdGetUserCheckins(u.UntappdID, count)
	if err != nil {
		return nil, err
	}
	return checkins, nil
}

// Beer represents a beer.
type Beer struct {
	// ID is the primary key.
	ID int64
	// Brewery is the beer's brewery.
	Brewery string
	// Name is the name of the beer.
	Name string
	// UntappdID is the Untappd beer ID.
	UntappdID int64
	// UntappdRating i the Untappd rating.
	UntappdRating float64
	// BreweryID is the brewer untappd id.
	BreweryID int64
	// LabelURL is the URL of the label.
	LabelURL string
}

// GetBeer gets the given beer.
func GetBeer(id int64) (*Beer, error) {
	beers, err := DB.ListBeers()
	if err != nil {
		return nil, err
	}
	for _, b := range beers {
		if b.ID == id {
			return b, nil
		}
	}
	return nil, fmt.Errorf("no such beer id: %d", id)
}

// RatingWidth returns the width of the beer's rating stars.
func (b *Beer) RatingWidth() int {
	return int(b.UntappdRating * 100 / 5.0)
}

// Available returns the number of units available of the beer.
func (b *Beer) Available() (int64, error) {
	var available int64
	contr, err := DB.ListContributions()
	if err != nil {
		return 0, err
	}
	contributions := map[int64]*Contribution{}
	for _, c := range contr {
		contributions[c.ID] = c
		if c.Beer == b.ID {
			available += c.Quantity
		}
	}

	taken, err := DB.ListCheckouts()
	if err != nil {
		return 0, err
	}
	for _, w := range taken {
		if contributions[w.Contribution].Beer == b.ID {
			available -= w.Quantity
		}
	}
	return available, nil
}

// GetContribution returns the given contribution.
func GetContribution(id int64) (*Contribution, error) {
	conts, err := DB.ListContributions()
	if err != nil {
		return nil, err
	}
	for _, c := range conts {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no such contribution id: %d", id)
}

// Contribution represents a contribution to the beer pool.
type Contribution struct {
	// ID is the primary key.
	ID int64
	// User is the user to contributed the beer.
	User int64
	// Beer is the ID of the beer contributed.
	Beer int64
	// Quantity is the quantity of beers contributed.
	Quantity int64
	// Date is the date contributed.
	Date time.Time
	// UnitPrice is the unit price of the beers.
	UnitPrice float64
}

// GetBeer gets the beer associated with a contribution.
func (c *Contribution) GetBeer() (*Beer, error) {
	beers, err := DB.ListBeers()
	if err != nil {
		return nil, err
	}
	for _, b := range beers {
		if b.ID == c.Beer {
			return b, nil
		}
	}
	return nil, fmt.Errorf("did not find beer with id %d", c.Beer)
}

// GetUser gets the user associated with a contribution.
func (c *Contribution) GetUser() (*User, error) {
	users, err := DB.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.ID == c.User {
			return u, nil
		}
	}
	return nil, fmt.Errorf("did not find user with id %d", c.User)
}

// Remaining is the remaining beer from that contribution.
func (c *Contribution) Remaining() (int64, error) {
	takes, err := DB.ListCheckouts()
	if err != nil {
		return 0, err
	}
	remaining := c.Quantity
	for _, take := range takes {
		if take.Contribution == c.ID {
			remaining -= take.Quantity
		}
	}
	return remaining, nil
}

// GetCheckouts returns all the checkouts of that contribution.
func (c *Contribution) GetCheckouts() ([]*Checkout, error) {
	couts, err := DB.ListCheckouts()
	if err != nil {
		return nil, err
	}
	ret := []*Checkout{}
	for _, cout := range couts {
		if cout.Contribution == c.ID {
			ret = append(ret, cout)
		}
	}
	return ret, nil
}

// Checkout represents a checkout from the Syndicate.
type Checkout struct {
	// ID is the primary key.
	ID int64
	// User is the user who took the beer.
	User int64
	// Contribution is the contribution taken from.
	Contribution int64
	// Quantity is the quantity of beers taken.
	Quantity int64
	// Date is the date contributed.
	Date time.Time
}

// GetUser gets the user associated with a checkout.
func (c *Checkout) GetUser() (*User, error) {
	users, err := DB.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.ID == c.User {
			return u, nil
		}
	}
	return nil, fmt.Errorf("did not find user with id %d", c.User)
}

// DB is the database handler.
var DB BeerDatabase

// OpenDatabase opens the database handler.
func OpenDatabase(filename string) error {
	db := &database{}
	if err := db.Open(filename); err != nil {
		return err
	}
	DB = db
	return nil
}

// BeerDatabase is the interface to the database implementation.
type BeerDatabase interface {
	// ListUsers returns all users.
	ListUsers() ([]*User, error)
	// AddUser adds the given user to the syndicate.
	AddUser(*User) (id int64, err error)

	// ListBeers returns all beers.
	ListBeers() ([]*Beer, error)
	// AddBeer adds a new beer.
	AddBeer(*Beer) (id int64, err error)

	// ListContributions returns all contributions.
	ListContributions() ([]*Contribution, error)
	// AddContribution adds a new contribution.
	AddContribution(*Contribution) (id int64, err error)
	// DeleteContribution deletes the given contribution.
	DeleteContribution(int64) error
	// EditContribution edits the given contribution.
	EditContribution(*Contribution) error

	// ListCheckouts lists all checkouts.
	ListCheckouts() ([]*Checkout, error)
	// AddCheckout adds a checkout.
	AddCheckout(*Checkout) (id int64, err error)
	// DeleteCheckout deletes a checkout.
	DeleteCheckout(int64) error
}
