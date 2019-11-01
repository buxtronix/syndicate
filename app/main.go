package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/buxtronix/syndicate"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	listTmpl       = parseTemplate("beers.html")
	usersTmpl      = parseTemplate("users.html")
	contributeTmpl = parseTemplate("contributions.html")
	contDetailTmpl = parseTemplate("contDetail.html")
	activityTmpl   = parseTemplate("activity.html")
)

var (
	listenAddress = flag.String("listen", ":8080", "Address to listen on")
	dbFile        = flag.String("dbfile", "beer.db", "SQLite database file")
	untappdID     = flag.String("untappd_id", "", "Client ID for Untappd API")
	untappdSecret = flag.String("untappd_secret", "", "Secret for Untappd API")

	vapidPublic  = flag.String("vapid_public", "", "VAPID public key")
	vapidPrivate = flag.String("vapid_private", "", "VAPID private key")
)

func main() {
	flag.Parse()
	switch {
	case *untappdID == "":
		log.Printf("Warning: Missing -untapped_id which breaks untappd functionality")
	case *untappdSecret == "":
		log.Printf("Warning: Missing -untapped_secret which breaks untappd functionality")
	}
	if err := syndicate.NewUntappdClient(*untappdID, *untappdSecret); err != nil {
		log.Fatal(err)
	}
	registerHandlers()
	if err := syndicate.OpenDatabase(*dbFile); err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func registerHandlers() {
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/checkout", http.StatusFound))

	r.Methods("GET").Path("/beers").
		Handler(appHandler(beersHandler))
	r.Methods("POST").Path("/beers/add").
		Handler(appHandler(addBeerHandler))

	r.Methods("GET").Path("/checkout").
		Handler(appHandler(getCheckoutHandler))
	r.Methods("POST").Path("/checkout/delete").
		Handler(appHandler(getCheckoutDeleteHandler))
	r.Methods("GET").Path("/checkout/{which:.+}").
		Handler(appHandler(getCheckoutHandler))
	r.Methods("POST").Path("/checkout").
		Handler(appHandler(addCheckoutHandler))

	r.Methods("GET").Path("/contribute/detail/{id:.+}").
		Handler(appHandler(getContributeDetailHandler))
	r.Methods("POST").Path("/contribute/delete/{id:.+}").
		Handler(appHandler(deleteContributeHandler))
	r.Methods("POST").Path("/contribute/edit/{id:.+}").
		Handler(appHandler(editContributeHandler))
	r.Methods("POST").Path("/contribute").
		Handler(appHandler(addContributeHandler))

	r.Methods("GET").Path("/users").
		Handler(appHandler(usersHandler))
	r.Methods("POST").Path("/users/add").
		Handler(appHandler(userAddHandler))

	r.Methods("GET").Path("/activity").
		Handler(appHandler(activityHandler))

	r.Methods("POST").Path("/untappd/beer").Handler(appHandler(untappdBeerHandler))

	r.Methods("POST").Path("/subscribe").
		Handler(appHandler(addSubHandler))
	r.Methods("POST").Path("/unsubscribe").
		Handler(appHandler(delSubHandler))

	r.Methods("GET").Path("/static/{path:.+}").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
}

type beerForm struct {
	Beers []*syndicate.Beer
	Users []*syndicate.User
}

// beersHandler handles display of contributed beers.
func beersHandler(w http.ResponseWriter, r *http.Request) *appError {
	beers, err := syndicate.DB.ListBeers()
	if err != nil {
		return appErrorf(err, "could not fetch beer list: %v", err)
	}
	users, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "could not fetch user list: %v", err)
	}
	bf := &beerForm{
		Beers: beers,
		Users: users,
	}
	return listTmpl.Execute(w, r, bf)
}

var untappdRE = regexp.MustCompile("([0-9]+)$")

// addBeerHandler handles adding beer.
func addBeerHandler(w http.ResponseWriter, r *http.Request) *appError {
	var uti int64
	var err error
	if fv := r.FormValue("untappdid"); fv != "" {
		uti, err = strconv.ParseInt(untappdRE.FindString(fv), 10, 64)
		if err != nil {
			return appErrorf(err, "UntappdID must be a number: %v", err)
		}
		beers, err := syndicate.DB.ListBeers()
		if err != nil {
			return appErrorf(err, "error querying existing db: %v", err)
		}
		for _, b := range beers {
			if uti > 0 && b.UntappdID == uti {
				return appErrorf(err, "Already have a beer with that untappd id")
			}
		}
	}
	beer := &syndicate.Beer{
		Brewery:   r.FormValue("brewery"),
		Name:      r.FormValue("name"),
		UntappdID: uti,
	}
	bInfo, _, err := syndicate.Untappd.GetBeerInfo(uti)
	if err != nil {
		return appErrorf(err, "error querying untappd: %v", err)
	}
	if bInfo != nil {
		beer.Brewery = bInfo.Brewery.Name
		beer.Name = bInfo.Name
		beer.UntappdRating = bInfo.OverallRating
		beer.BreweryID = int64(bInfo.Brewery.ID)
		beer.LabelURL = bInfo.Label.String()
	}
	_, err = syndicate.DB.AddBeer(beer)
	if err != nil {
		return appErrorf(err, "error inserting into db: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/beers"), http.StatusFound)
	return nil
}

func untappdBeerHandler(w http.ResponseWriter, r *http.Request) *appError {
	var uti int64
	var err error
	if fv := r.FormValue("id"); fv != "" {
		uti, err = strconv.ParseInt(untappdRE.FindString(fv), 10, 64)
		if err != nil {
			w.Write([]byte("not a valid beer id"))
			return nil
		}
	}
	bInfo, _, err := syndicate.Untappd.GetBeerInfo(uti)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("error querying untappd: %v", err)))
		return nil
	}
	if bInfo == nil {
		w.Write([]byte("Beer ID not found"))
		return nil
	}
	w.Write([]byte(fmt.Sprintf(`<div class="alert alert-info" role="alert">%s <small>(%s)</small></div>`, bInfo.Name, bInfo.Brewery.Name)))
	return nil
}

func editContributeHandler(w http.ResponseWriter, r *http.Request) *appError {
	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		return appErrorf(err, "error parsing quantity id: %v", err)
	}
	unitPrice, err := strconv.ParseFloat(r.FormValue("unitprice"), 64)
	if err != nil {
		return appErrorf(err, "error parsing unit price: %v", err)
	}
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse contribution id: %v", err)
	}
	cont, err := syndicate.GetContribution(id)
	if err != nil {
		return appErrorf(err, "could not get contribution: %v", err)
	}
	cont.Quantity = int64(quantity)
	cont.UnitPrice = unitPrice
	cont.Comment = r.FormValue("comment")
	if err := syndicate.DB.EditContribution(cont); err != nil {
		return appErrorf(err, "could not edit contribution: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/contribute/detail/%d", id), http.StatusFound)
	return nil
}

// deleteContributeHandler deletes a checkout.
func deleteContributeHandler(w http.ResponseWriter, r *http.Request) *appError {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse contribution id: %v", err)
	}
	if magic := r.FormValue("magic"); magic != "Netops!" {
		return appErrorf(err, "missing required magic value")
	}
	if err := syndicate.DB.DeleteContribution(id); err != nil {
		return appErrorf(err, "error removing contribution: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/checkout"), http.StatusFound)
	return nil
}

// addContributeHandler handles the contribution of beer.
func addContributeHandler(w http.ResponseWriter, r *http.Request) *appError {
	beerID, err := strconv.Atoi(r.FormValue("beerid"))
	if err != nil {
		return appErrorf(err, "error parsing beer id: %v", err)
	}
	userID, err := strconv.Atoi(r.FormValue("userid"))
	if err != nil {
		return appErrorf(err, "error parsing user id: %v", err)
	}
	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		return appErrorf(err, "error parsing quantity id: %v", err)
	}
	up := r.FormValue("unitprice")
	unitPrice, upErr := strconv.ParseFloat(r.FormValue("unitprice"), 64)
	tp := r.FormValue("totalprice")
	totalPrice, tpErr := strconv.ParseFloat(r.FormValue("totalprice"), 64)
	switch {
	case upErr != nil && tpErr != nil:
		return appErrorf(err, "must provide only one of unit price or total price")
	case up != "" && tp != "":
		return appErrorf(err, "must provide only one of unit price or total price")
	case tpErr == nil && upErr != nil:
		unitPrice = totalPrice / float64(quantity)
	case upErr == nil && tpErr != nil:
		break
	}
	cont := &syndicate.Contribution{
		User:      int64(userID),
		Beer:      int64(beerID),
		Quantity:  int64(quantity),
		UnitPrice: unitPrice,
		Date:      time.Now(),
		Comment:   r.FormValue("comment"),
	}
	_, err = syndicate.DB.AddContribution(cont)
	if err != nil {
		return appErrorf(err, "error adding contribution: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/checkout"), http.StatusFound)
	go sendAllSubscribers("Someone added beer!")
	return nil
}

// getContributeDetailHandler shows contribution detail.
func getContributeDetailHandler(w http.ResponseWriter, r *http.Request) *appError {
	users, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "could not fetch user list: %v", err)
	}
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse checkout list: %v", err)
	}
	cont, err := syndicate.GetContribution(id)
	if err != nil {
		return appErrorf(err, "could not get contribution: %v", err)
	}
	data := struct {
		Contribution *syndicate.Contribution
		Users        []*syndicate.User
	}{
		Contribution: cont,
		Users:        users,
	}
	return contDetailTmpl.Execute(w, r, data)
}

// getCheckoutDeleteHandler deletes a checkout.
func getCheckoutDeleteHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(r.FormValue("coid"), 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse checkout id: %v", err)
	}
	contid, err := strconv.ParseInt(r.FormValue("contid"), 10, 64)
	if err != nil {
		return appErrorf(err, "could not parse contribution id: %v", err)
	}
	if magic := r.FormValue("magic"); magic != "Netops!" {
		return appErrorf(err, "missing required magic value")
	}
	if err := syndicate.DB.DeleteCheckout(id); err != nil {
		return appErrorf(err, "error removing checkout: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/contribute/detail/%d", contid), http.StatusFound)
	return nil
}

// getCheckoutHandler handles form for checkout.
func getCheckoutHandler(w http.ResponseWriter, r *http.Request) *appError {
	vars := mux.Vars(r)
	all := vars["which"] == "all"
	conts, err := syndicate.DB.ListContributions()
	if err != nil {
		return appErrorf(err, "could not fetch contribution list: %v", err)
	}
	users, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "could not fetch user list: %v", err)
	}
	form := struct {
		All           bool
		Contributions []*syndicate.Contribution
		Users         []*syndicate.User
	}{
		All:           all,
		Contributions: []*syndicate.Contribution{},
		Users:         users,
	}
	for _, c := range conts {
		remaining, err := c.Remaining()
		if err != nil {
			return appErrorf(err, "could not fetch remaining count: %v", err)
		}
		if all || remaining > 0 {
			form.Contributions = append(form.Contributions, c)
		}
	}
	return contributeTmpl.Execute(w, r, form)
}

// addCheckoutHandler handles the checkout of beer.
func addCheckoutHandler(w http.ResponseWriter, r *http.Request) *appError {
	contID, err := strconv.ParseInt(r.FormValue("contid"), 10, 64)
	if err != nil {
		return appErrorf(err, "error parsing contribution id: %v", err)
	}
	contr, err := syndicate.GetContribution(contID)
	if err != nil {
		return appErrorf(err, "error fetching contribution id %d: %v", contID, err)
	}

	userID, err := strconv.ParseInt(r.FormValue("userid"), 10, 64)
	if err != nil {
		return appErrorf(err, "error parsing user id: %v", err)
	}
	users, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "error fetching user list: %v", err)
	}
	foundUser := false
	for _, u := range users {
		if u.ID == userID {
			foundUser = true
		}
	}
	if !foundUser {
		return appErrorf(err, "Unknown user id: %d: %v", userID, err)
	}

	quantity, err := strconv.ParseInt(r.FormValue("quantity"), 10, 64)
	if err != nil {
		return appErrorf(err, "error parsing quantity id: %v", err)
	}
	remaining, err := contr.Remaining()
	if err != nil {
		return appErrorf(err, "error finding remaining amount: %v", err)
	}

	if quantity > remaining {
		return appErrorf(err, "attempt to checkout %d with only %d available", quantity, remaining)
	}
	with := &syndicate.Checkout{
		User:         int64(userID),
		Contribution: int64(contID),
		Quantity:     int64(quantity),
		Date:         time.Now(),
	}
	_, err = syndicate.DB.AddCheckout(with)
	if err != nil {
		return appErrorf(err, "error adding checkout: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/checkout"), http.StatusFound)
	return nil
}

// activityHandler handles display of all activity.
func activityHandler(w http.ResponseWriter, r *http.Request) *appError {
	activity, err := getActivity()
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return activityTmpl.Execute(w, r, activity)
}

type oneActivity struct {
	User     int64
	Contrib  *syndicate.Contribution
	Checkout *syndicate.Checkout
	Time     time.Time
}

func getActivity() ([]*oneActivity, error) {
	contribs, err := syndicate.DB.ListContributions()
	if err != nil {
		return nil, fmt.Errorf("could not fetch contribution list: %v", err)
	}
	checkouts, err := syndicate.DB.ListCheckouts()
	if err != nil {
		return nil, fmt.Errorf("could not fetch checkout list: %v", err)
	}
	activity := []*oneActivity{}
	for _, c := range contribs {
		activity = append(activity, &oneActivity{
			User:    c.User,
			Contrib: c,
			Time:    c.Date,
		})
	}
	for _, c := range checkouts {
		activity = append(activity, &oneActivity{
			User:     c.User,
			Checkout: c,
			Time:     c.Date,
		})
	}
	sort.Slice(activity, func(i, j int) bool {
		return activity[i].Time.After(activity[j].Time)
	})
	return activity, nil
}

// usersHandler handles display of user stats.
func usersHandler(w http.ResponseWriter, r *http.Request) *appError {
	users, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "could not fetch user list: %v", err)
	}
	activity, err := getActivity()
	if err != nil {
		return appErrorf(err, "could not fetch activity list: %v", err)
	}
	ud := struct {
		Users    []*syndicate.User
		Activity []*oneActivity
	}{
		Users:    users,
		Activity: activity,
	}
	return usersTmpl.Execute(w, r, ud)
}

// userAddHandler handles adding of a new user.
func userAddHandler(w http.ResponseWriter, r *http.Request) *appError {
	newUser := r.FormValue("username")
	if newUser == "" {
		return &appError{Error: nil, Message: "missing user name"}
	}
	existingUsers, err := syndicate.DB.ListUsers()
	if err != nil {
		return appErrorf(err, "error querying existing users: %v", err)
	}
	for _, u := range existingUsers {
		if u.Name == newUser {
			return appErrorf(err, "user %s already exists", newUser)
		}
	}
	if _, err := syndicate.DB.AddUser(&syndicate.User{
		Name:      newUser,
		UntappdID: r.FormValue("untappd"),
	}); err != nil {
		return appErrorf(err, "error adding new user %s: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/users"), http.StatusFound)
	return nil
}

func addSubHandler(w http.ResponseWriter, r *http.Request) *appError {
	endPoint := r.FormValue("endpoint")
	key := r.FormValue("key")
	auth := r.FormValue("auth")

	switch {
	case endPoint == "":
		return &appError{Error: nil, Message: "missing endpoint"}
	case key == "":
		return &appError{Error: nil, Message: "missing key"}
	case auth == "":
		return &appError{Error: nil, Message: "missing auth"}
	}

	sub := &syndicate.Subscription{
		Endpoint: endPoint,
		Key:      key,
		Auth:     auth,
	}
	if _, err := syndicate.DB.AddSubscription(sub); err != nil {
		return appErrorf(err, "error adding subscription: %v", err)
	}
	return nil
}

func delSubHandler(w http.ResponseWriter, r *http.Request) *appError {
	endPoint := r.FormValue("endpoint")
	key := r.FormValue("key")
	auth := r.FormValue("auth")

	switch {
	case endPoint == "":
		return &appError{Error: nil, Message: "missing endpoint"}
	case key == "":
		return &appError{Error: nil, Message: "missing key"}
	case auth == "":
		return &appError{Error: nil, Message: "missing auth"}
	}

	subs, err := syndicate.DB.ListSubscriptions()
	if err != nil {
		return appErrorf(err, "Error listing subscriptions: %v", err)
	}

	for _, sub := range subs {
		if sub.Endpoint == endPoint {
			if err := syndicate.DB.DeleteSubscription(sub.ID); err != nil {
				return appErrorf(err, "Error removing subscription: %v", err)
			}
			return nil
		}
	}

	return &appError{Error: nil, Message: "subscription not found"}
}

func sendAllSubscribers(message string) error {
	subs, err := syndicate.DB.ListSubscriptions()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				Auth:   sub.Auth,
				P256dh: sub.Key,
			},
		}
		if _, err := webpush.SendNotification([]byte(message), s, &webpush.Options{
			TTL:             20 * 60 * 60,
			VAPIDPublicKey:  *vapidPublic,
			VAPIDPrivateKey: *vapidPrivate,
		}); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}
	return nil
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error.
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)

		http.Error(w, e.Message, e.Code)
	}
}

func appErrorf(err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
	}
}
