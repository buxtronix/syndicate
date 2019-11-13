package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/buxtronix/syndicate"
)

var (
	vapidPublic  = flag.String("vapid_public", "", "VAPID public key")
	vapidPrivate = flag.String("vapid_private", "", "VAPID private key")
)

func addSubHandler(w http.ResponseWriter, r *http.Request) *appError {
	endPoint := r.FormValue("endpoint")
	key := r.FormValue("key")
	auth := r.FormValue("auth")

	cookie, err := r.Cookie(syndicateCookie)
	if err != nil {
		log.Printf("Cookie  error: %v", err)
		return nil
	}

	switch {
	case endPoint == "":
		return &appError{Error: nil, Message: "missing endpoint"}
	case key == "":
		return &appError{Error: nil, Message: "missing key"}
	case auth == "":
		return &appError{Error: nil, Message: "missing auth"}
	}

	sub := &syndicate.Subscription{
		Endpoint:  endPoint,
		Key:       key,
		Auth:      auth,
		UserAgent: r.Header.Get("User-Agent"),
		Host:      r.Header.Get("X-Forwarded-For"),
		Cookie:    cookie.Value,
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

type subMessage struct {
	Message, URI string
}

func sendAllSubscribers(msg subMessage, uuid string) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	subs, err := syndicate.DB.ListSubscriptions()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.Cookie == uuid {
			continue
		}
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				Auth:   sub.Auth,
				P256dh: sub.Key,
			},
		}
		log.Printf("Send to endpoint: %s\n", sub.Endpoint)
		resp, err := webpush.SendNotification(data, s, &webpush.Options{
			TTL:             20 * 60 * 60,
			VAPIDPublicKey:  *vapidPublic,
			VAPIDPrivateKey: *vapidPrivate,
		})
		if err != nil {
			log.Printf("Error sending notification: %v", err)
			continue
		}
		b, _ := ioutil.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusCreated: // 201
			log.Printf(" Sent: %s", string(b))
		case http.StatusGone, http.StatusForbidden: // 410, no more endpoint.
			log.Printf(" Gone: %s", string(b))
			if err := syndicate.DB.DeleteSubscription(sub.ID); err != nil {
				log.Printf("Error removing subscription: %v", err)
			} else {
				log.Printf("  Deleted subscription.")
			}
		default:
			log.Printf(" Unknown response code: %s", resp.Status)
			log.Printf("  Message: %s", string(b))
		}
		resp.Body.Close()
	}
	return nil
}
