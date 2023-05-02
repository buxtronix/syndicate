package syndicate

import (
	"fmt"
	"time"

	"github.com/mdlayher/untappd"
)

var fractions = []string{"", "¹⁄₁₂", "⅙", "¼", "⅓", "⁵⁄₁₂", "½", "⁷⁄₁₂", "⅔", "¾", "⅚", "¹¹⁄₁₂"}

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
			total += (float64(t.Twelfths) / 12) * conts[t.Contribution].UnitPrice
		}
	}
	return total, nil
}

// TotalDebitCredit returns the total debits/credits for the user.
func (u *User) TotalDebitCredit() (float64, error) {
	dcs, err := DB.ListDebitCredits()
	if err != nil {
		return 0, err
	}
	var total float64
	for _, dc := range dcs {
		if dc.User == u.ID {
			total += dc.Amount
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
	dc, err := u.TotalDebitCredit()
	if err != nil {
		return 0, err
	}
	return added - taken + u.SeedFund + dc, nil
}

// LastCheckins returns the users last 'count' checkins on Untappd.
func (u *User) LastCheckins(count int) ([]*untappd.Checkin, error) {
	if u.UntappdID == "" {
		return nil, nil
	}
	checkins, err := Untappd.GetUserCheckins(u.UntappdID, count)
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
func (b *Beer) Available() (float64, error) {
	var available int64 // In twelfths.
	contr, err := DB.ListContributions()
	if err != nil {
		return 0, err
	}
	contributions := map[int64]*Contribution{}
	for _, c := range contr {
		contributions[c.ID] = c
		if c.Beer == b.ID {
			available += c.Quantity * int64(12)
		}
	}

	taken, err := DB.ListCheckouts()
	if err != nil {
		return 0, err
	}
	for _, w := range taken {
		if contributions[w.Contribution].Beer == b.ID {
			available -= w.Twelfths
		}
	}
	return float64(available) / 12, nil
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
	// Comment is a freeform comment for the contribution.
	Comment string
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
func (c *Contribution) Remaining() (float64, error) {
	takes, err := DB.ListCheckouts()
	if err != nil {
		return 0, err
	}
	remaining := c.Quantity * 12 // In twelfths
	for _, take := range takes {
		if take.Contribution == c.ID {
			remaining -= take.Twelfths
		}
	}
	return float64(remaining) / 12, nil
}

func (c *Contribution) RemainingStr() (string, error) {
	takes, err := DB.ListCheckouts()
	if err != nil {
		return "", err
	}
	remaining := c.Quantity * 12 // In twelfths
	for _, take := range takes {
		if take.Contribution == c.ID {
			remaining -= take.Twelfths
		}
	}
	whole := remaining/12
	remainder := remaining % 12
	if whole < 1 {
		return fmt.Sprintf("%s", fractions[remainder]), nil
	}
	return fmt.Sprintf("%d%s", whole, fractions[remainder]), nil
}

// Untouched returns true if none of the contribution has been claimed.
func (c *Contribution) Untouched() (bool, error) {
	rem, err := c.Remaining()
	if err != nil {
		return false, err
	}
	return  float64(c.Quantity) - rem < 0.05, nil // float uncertainty.
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
	Quantity float64
	// Date is the date contributed.
	Date time.Time
	// Twelfths is the quantity, in twelfths of a beer. This
	// allows for splitting by half, quarter, thirds.
	Twelfths int64
}

// QuantityStr returns the quantity checked out as a string.
func (c *Checkout) QuantityStr() string {
	whole := c.Twelfths/12
	remainder := c.Twelfths % 12
	if whole < 1 {
		return fmt.Sprintf("%s", fractions[remainder])
	}
	return fmt.Sprintf("%d%s", whole, fractions[remainder])
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

// GetContribution gets the contribution.
func (c *Checkout) GetContribution() (*Contribution, error) {
	return GetContribution(c.Contribution)
}

// Subscription is a web push subscription.
type Subscription struct {
	// ID is the ID of the subscription.
	ID int64
	// Endpoint is the web push endpoint.
	Endpoint string
	// Key is the push key field.
	Key string
	// Auth is the push auth field.
	Auth string
	// UserAgent is the subscriber's user agent.
	UserAgent string
	// Host is the subscribing user agent address.
	Host string
	// Cookie is the cookie of the browser.
	Cookie string
}

// DebitCredit represents a misc non-beer debit or credit for a user.
type DebitCredit struct {
	// ID is the ID of the DebitCredit.
	ID int64
	// User is the user to whom this applies.
	User int64
	// Amount is the amount of debit or credit, in cents.
	Amount float64
	// Date is the date the debit or credit was applied.
	Date time.Time
	// Comment is a freeform comment or description of the debit or credit.
	Comment string
}

// GetUser gets the user associated with a contribution.
func (dc *DebitCredit) GetUser() (*User, error) {
	users, err := DB.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.ID == dc.User {
			return u, nil
		}
	}
	return nil, fmt.Errorf("did not find user with id %d", dc.User)
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

	// ListSubscriptions lists all subscriptions.
	ListSubscriptions() ([]*Subscription, error)
	// AddSubscription adds a new subscription
	AddSubscription(*Subscription) (id int64, err error)
	// DeleteSubscription removes a subscription.
	DeleteSubscription(int64) error

	// ListDebitCredits lists all debits or credits.
	ListDebitCredits() ([]*DebitCredit, error)
	// AddDebitCredit adds a debit or credit.
	AddDebitCredit(*DebitCredit) (id int64, err error)
	// DeleteDebitCredit deletes a debit or credit.
	DeleteDebitCredit(int64) error
}
