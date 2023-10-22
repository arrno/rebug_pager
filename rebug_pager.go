package rebugpager

import (
	"encoding/json"
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
}

func NewRebugPager() (*RebugPager, error) {
	p := new(RebugPager)
	var err error
	if p.db, err = NewFirestore(); err != nil {
		return p, err
	}
	return p, nil
}

func (p *RebugPager) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Closure for failures.
	handleResponse := func(statusCode int) {
		message := "Success!"
		if statusCode != 200 && statusCode != 201 {
			message = "Oh no! Something went wrong..."
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"Response": message})
	}

	// Validate auth.
	if err := DoAuth(); err != nil {
		handleResponse(http.StatusForbidden)
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

	message, err := ParseMessage(data["Body"])
	if err != nil {
		handleResponse(http.StatusBadRequest)
		return
	}

	// fetch existing doc
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

	// respond
	handleResponse(http.StatusCreated)
}

func DoAuth() error {
	// https://www.twilio.com/docs/usage/webhooks/webhooks-security
	return nil
}
