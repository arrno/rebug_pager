package rebugpager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	rp, err := NewRebugPager()
	if err != nil {
		log.Fatal("Oh no!")
	}
	defer rp.db.client.Close()
	functions.HTTP("RebugPager", rp.ServeHTTP)
}

// it's better to store these globally in memory outside of the http handler.
type RebugPager struct {
	db *Database
	tc *TwilioClient
}

func NewRebugPager() (*RebugPager, error) {
	p := new(RebugPager)
	var err error
	if p.db, err = NewFirestore(); err != nil {
		return p, err
	}
	p.tc = NewTwilio()
	return p, nil
}

func (p *RebugPager) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Closure for http response.
	handleResponse := func(statusCode int) {
		message := "Success!"
		if statusCode != 200 && statusCode != 201 {
			message = "Oh no! Something went wrong..."
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"Response": message})
	}

	// Parse input.
	data := map[string]string{}
	if b, err := io.ReadAll(r.Body); err != nil {
		handleResponse(http.StatusBadRequest)
		return
	} else if params, err := url.ParseQuery(string(b)); err != nil {
		handleResponse(http.StatusBadRequest)
		return
	} else {
		for key := range params {
			data[key] = params.Get(key)
		}
	}

	// Validate auth.
	if err := p.DoAuth(data, r); err != nil {
		handleResponse(http.StatusForbidden)
	}

	// Parse message.
	message, err := ParseMessage(data["Body"])
	if err != nil {
		handleResponse(http.StatusBadRequest)
		return
	}

	// Fetch existing doc.
	docPath := fmt.Sprintf("inbound/pager/sessions/%s", message.DocID)
	var docData UserDoc
	if err := p.db.GetDoc(docPath, &docData); err != nil {
		handleResponse(http.StatusBadRequest)
		return
	}
	// Update and write back to db.
	docData = MergeDoc(docData, message)
	if err := p.db.WriteDoc(docPath, docData); err != nil {
		handleResponse(http.StatusInternalServerError)
		return
	}

	// Respond.
	handleResponse(http.StatusCreated)
}

// https://www.twilio.com/docs/usage/webhooks/webhooks-security
func (p *RebugPager) DoAuth(params map[string]string, r *http.Request) error {
	signature := r.Header.Get("x-twilio-signature")
	if len(signature) > 0 && p.tc.Validate(params, signature) {
		return nil
	}
	return errors.New("Unable to validate incomming request.")
}
