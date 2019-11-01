package syndicate

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const createStmt = `
CREATE TABLE IF NOT EXISTS users(
  id INTEGER PRIMARY KEY,
  name TEXT,
  untappdid TEXT,
  seedfund INTEGER
);
CREATE TABLE IF NOT EXISTS beers(
  id INTEGER PRIMARY KEY,
  brewery TEXT,
  name TEXT,
  untappdid INTEGER,
  untappdrating INTEGER,
  breweryid INTEGER,
  labelURL TEXT
);
CREATE TABLE IF NOT EXISTS contributions(
  id INTEGER PRIMARY KEY,
  user INTEGER,
  beer INTEGER,
  quantity INTEGER,
  date INTEGER,
  unitprice INTEGER,
  comment TEXT
);
CREATE TABLE IF NOT EXISTS checkouts(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user INTEGER,
  contribution INTEGER,
  quantity INTEGER,
  date INTEGER
);
CREATE TABLE IF NOT EXISTS subscriptions(
  id INTEGER PRIMARY KEY,
  endpoint TEXT,
  key TEXT,
  auth TEXT
);
`

type database struct {
	db *sql.DB

	addUser           *sql.Stmt
	listUsers         *sql.Stmt
	addBeer           *sql.Stmt
	listBeers         *sql.Stmt
	addContribution   *sql.Stmt
	editContribution  *sql.Stmt
	delContribution   *sql.Stmt
	listContributions *sql.Stmt
	listCheckouts     *sql.Stmt
	addCheckout       *sql.Stmt
	delCheckout       *sql.Stmt

	listSubscriptions *sql.Stmt
	addSubscription   *sql.Stmt
	delSubscription   *sql.Stmt
}

var _ BeerDatabase = &database{}

func (d *database) Open(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	d.db = db
	if _, err := d.db.Exec(createStmt); err != nil {
		return fmt.Errorf("error creating database: %v", err)
	}
	if d.listUsers, err = db.Prepare(listUsersStmt); err != nil {
		return fmt.Errorf("sql: prepare listUsers: %v", err)
	}
	if d.addUser, err = db.Prepare(addUserStmt); err != nil {
		return fmt.Errorf("sql: prepare addUser: %v", err)
	}
	if d.listBeers, err = db.Prepare(listBeersStmt); err != nil {
		return fmt.Errorf("sql: prepare listBeers: %v", err)
	}
	if d.addBeer, err = db.Prepare(addBeerStmt); err != nil {
		return fmt.Errorf("sql: prepare addBeer: %v", err)
	}
	if d.listContributions, err = db.Prepare(listContributionsStmt); err != nil {
		return fmt.Errorf("sql: prepare listContributions: %v", err)
	}
	if d.addContribution, err = db.Prepare(addContributionStmt); err != nil {
		return fmt.Errorf("sql: prepare addContribution: %v", err)
	}
	if d.editContribution, err = db.Prepare(editContributionStmt); err != nil {
		return fmt.Errorf("sql: prepare editContribution: %v", err)
	}
	if d.delContribution, err = db.Prepare(delContributionStmt); err != nil {
		return fmt.Errorf("sql: prepare delContribution: %v", err)
	}
	if d.listCheckouts, err = db.Prepare(listCheckoutsStmt); err != nil {
		return fmt.Errorf("sql: prepare listCheckouts: %v", err)
	}
	if d.addCheckout, err = db.Prepare(addCheckoutStmt); err != nil {
		return fmt.Errorf("sql: prepare addCheckout: %v", err)
	}
	if d.delCheckout, err = db.Prepare(delCheckoutStmt); err != nil {
		return fmt.Errorf("sql: prepare delCheckout: %v", err)
	}

	if d.listSubscriptions, err = db.Prepare(listSubscriptionsStmt); err != nil {
		return fmt.Errorf("sql: prepare listSubscription: %v", err)
	}
	if d.addSubscription, err = db.Prepare(addSubscriptionStmt); err != nil {
		return fmt.Errorf("sql: prepare addSubscription: %v", err)
	}
	if d.delSubscription, err = db.Prepare(delSubscriptionStmt); err != nil {
		return fmt.Errorf("sql: prepare delSubscription: %v", err)
	}
	return nil
}

func (d *database) Close() error {
	return d.db.Close()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

const listUsersStmt = `SELECT * FROM users ORDER BY name`

func scanUsers(s rowScanner) (*User, error) {
	var (
		id        int64
		name      sql.NullString
		untappdid sql.NullString
		seedfund  sql.NullInt64
	)
	if err := s.Scan(&id, &name, &untappdid, &seedfund); err != nil {
		return nil, err
	}
	user := &User{
		ID:        id,
		Name:      name.String,
		UntappdID: untappdid.String,
		SeedFund:  float64(seedfund.Int64) / 100,
	}
	return user, nil
}

// ListUsers returns all syndicate users.
func (d *database) ListUsers() ([]*User, error) {
	rows, err := d.listUsers.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := scanUsers(rows)
		if err != nil {
			return nil, fmt.Errorf("sql: could not read row: %v", err)
		}
		users = append(users, user)
	}
	return users, nil
}

// GetUser returns the given user.
func (d *database) GetUser(id int64) (*User, error) {
	users, err := d.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

const addUserStmt = `INSERT INTO users(name, untappdid) VALUES (?,?)`

// AddUser adds a new user.
func (d *database) AddUser(u *User) (int64, error) {
	r, err := execAffectingOneRow(d.addUser, u.Name, u.UntappdID)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sql: could not get last insert id: %v", err)
	}
	return lastInsertID, nil
}

const listBeersStmt = `SELECT * FROM beers ORDER BY id desc`

func scanBeers(s rowScanner) (*Beer, error) {
	var (
		id            int64
		brewery       sql.NullString
		name          sql.NullString
		untappdid     sql.NullInt64
		untappdrating sql.NullInt64
		breweryid     sql.NullInt64
		labelURL      sql.NullString
	)
	if err := s.Scan(&id, &brewery, &name, &untappdid, &untappdrating, &breweryid, &labelURL); err != nil {
		return nil, err
	}
	beer := &Beer{
		ID:            id,
		Brewery:       brewery.String,
		Name:          name.String,
		UntappdID:     untappdid.Int64,
		UntappdRating: float64(untappdrating.Int64) / 100,
		BreweryID:     breweryid.Int64,
		LabelURL:      labelURL.String,
	}
	return beer, nil
}

// ListBeers returns all beers.
func (d *database) ListBeers() ([]*Beer, error) {
	rows, err := d.listBeers.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var beers []*Beer
	for rows.Next() {
		beer, err := scanBeers(rows)
		if err != nil {
			return nil, fmt.Errorf("sql: could not read row: %v", err)
		}
		beers = append(beers, beer)
	}
	return beers, nil
}

const addBeerStmt = `
INSERT INTO beers(
	brewery, name, untappdid, untappdrating, breweryid, labelurl
) VALUES (?,?,?,?,?,?)`

// AddBeer adds a new beer.
func (d *database) AddBeer(b *Beer) (int64, error) {
	rating := int64(b.UntappdRating * 100)
	r, err := execAffectingOneRow(d.addBeer, b.Brewery, b.Name, b.UntappdID, rating, b.BreweryID, b.LabelURL)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sql: could not get last insert id: %v", err)
	}
	return lastInsertID, nil
}

const listContributionsStmt = `SELECT * FROM contributions ORDER BY date`

func scanContributions(s rowScanner) (*Contribution, error) {
	var (
		id        int64
		user      sql.NullInt64
		beer      sql.NullInt64
		quantity  sql.NullInt64
		date      sql.NullInt64
		unitPrice sql.NullInt64
		comment   sql.NullString
	)
	if err := s.Scan(&id, &user, &beer, &quantity, &date, &unitPrice, &comment); err != nil {
		return nil, err
	}
	cont := &Contribution{
		ID:        id,
		User:      user.Int64,
		Beer:      beer.Int64,
		Quantity:  quantity.Int64,
		Date:      time.Unix(date.Int64, 0),
		UnitPrice: float64(unitPrice.Int64) / 100,
		Comment:   comment.String,
	}
	return cont, nil
}

// ListContributions returns all contributions.
func (d *database) ListContributions() ([]*Contribution, error) {
	rows, err := d.listContributions.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conts []*Contribution
	for rows.Next() {
		cont, err := scanContributions(rows)
		if err != nil {
			return nil, fmt.Errorf("sql: could not read row: %v", err)
		}
		conts = append(conts, cont)
	}
	return conts, nil
}

const addContributionStmt = `
INSERT INTO contributions(
  user, beer, quantity, date, unitprice, comment
  ) VALUES (?, ?, ?, ?, ?, ?)`

// AddContribution adds a new contribution.
func (d *database) AddContribution(c *Contribution) (int64, error) {
	r, err := execAffectingOneRow(d.addContribution, c.User, c.Beer, c.Quantity, c.Date.Unix(), int64(c.UnitPrice*100), c.Comment)
	if err != nil {
		return 0, err
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sql: could not get last insert id: %v", err)
	}
	return lastInsertID, nil
}

const editContributionStmt = `
UPDATE contributions SET quantity=?, unitprice=?, comment=?
WHERE id=?`

// EditContribution edits a contribution.
func (d *database) EditContribution(c *Contribution) error {
	_, err := execAffectingOneRow(d.editContribution, c.Quantity, int64(c.UnitPrice*100), c.Comment, c.ID)
	if err != nil {
		return err
	}
	return nil
}

const delContributionStmt = `
DELETE FROM contributions WHERE id = ?`

// DeleteContribution deletes a contribution.
func (d *database) DeleteContribution(id int64) error {
	_, err := execAffectingOneRow(d.delContribution, id)
	if err != nil {
		return err
	}
	return nil
}

const listCheckoutsStmt = `SELECT * FROM checkouts ORDER BY date`

func scanCheckouts(s rowScanner) (*Checkout, error) {
	var (
		id           int64
		user         sql.NullInt64
		contribution sql.NullInt64
		quantity     sql.NullInt64
		date         sql.NullInt64
	)
	if err := s.Scan(&id, &user, &contribution, &quantity, &date); err != nil {
		return nil, err
	}
	with := &Checkout{
		ID:           id,
		User:         user.Int64,
		Contribution: contribution.Int64,
		Quantity:     quantity.Int64,
		Date:         time.Unix(date.Int64, 0),
	}
	return with, nil
}

// ListCheckouts returns all checkouts.
func (d *database) ListCheckouts() ([]*Checkout, error) {
	rows, err := d.listCheckouts.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withs []*Checkout
	for rows.Next() {
		with, err := scanCheckouts(rows)
		if err != nil {
			return nil, fmt.Errorf("sql: could not read row: %v", err)
		}
		withs = append(withs, with)
	}
	return withs, nil
}

const addCheckoutStmt = `
INSERT INTO checkouts (
  user, contribution, quantity, date
  ) VALUES (?, ?, ?, ?)`

// AddCheckout adds a new checkout.
func (d *database) AddCheckout(c *Checkout) (int64, error) {
	r, err := execAffectingOneRow(d.addCheckout, c.User, c.Contribution, c.Quantity, c.Date.Unix())
	if err != nil {
		return 0, err
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sql: could not get last insert id: %v", err)
	}
	return lastInsertID, nil
}

const delCheckoutStmt = `
DELETE FROM checkouts WHERE id = ?`

// DeleteCheckout removes a checkout.
func (d *database) DeleteCheckout(id int64) error {
	_, err := execAffectingOneRow(d.delCheckout, id)
	if err != nil {
		return err
	}
	return nil
}

const listSubscriptionsStmt = `SELECT * FROM subscriptions`

func scanSubs(s rowScanner) (*Subscription, error) {
	var (
		id       int64
		endpoint sql.NullString
		key      sql.NullString
		auth     sql.NullString
	)
	if err := s.Scan(&id, &endpoint, &key, &auth); err != nil {
		return nil, err
	}
	sub := &Subscription{
		ID:       id,
		Endpoint: endpoint.String,
		Key:      key.String,
		Auth:     auth.String,
	}
	return sub, nil
}

func (d *database) ListSubscriptions() ([]*Subscription, error) {
	rows, err := d.listSubscriptions.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		sub, err := scanSubs(rows)
		if err != nil {
			return nil, fmt.Errorf("sql: could not read row: %v", err)
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

const addSubscriptionStmt = `
INSERT INTO subscriptions (
endpoint, key, auth) VALUES (?, ?, ?)`

func (d *database) AddSubscription(s *Subscription) (int64, error) {
	r, err := execAffectingOneRow(d.addSubscription, s.Endpoint, s.Key, s.Auth)
	if err != nil {
		return 0, nil
	}
	lastInsertID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sql: could not get last insert id: %v", err)
	}
	return lastInsertID, nil
}

const delSubscriptionStmt = `
DELETE FROM subscriptions WHERE id = ?`

func (d *database) DeleteSubscription(id int64) error {
	_, err := execAffectingOneRow(d.delSubscription, id)
	return err
}

// execAffectingOneRow executes a given statement, expecting one row to be affected.
func execAffectingOneRow(stmt *sql.Stmt, args ...interface{}) (sql.Result, error) {
	r, err := stmt.Exec(args...)
	if err != nil {
		return r, fmt.Errorf("mysql: could not execute statement: %v", err)
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return r, fmt.Errorf("mysql: could not get rows affected: %v", err)
	} else if rowsAffected != 1 {
		return r, fmt.Errorf("mysql: expected 1 row affected, got %d", rowsAffected)
	}
	return r, nil
}
