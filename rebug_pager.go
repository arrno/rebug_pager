package rebugpager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("RebugPager", ServeHTTP)
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var (
		data map[string]string
		db   *Database
		err  error
	)

	// Closure for http response.
	handleResponse := func(statusCode int, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
		logData := map[string]any{
			"payload":    data,
			"statusCode": statusCode,
			"time":       time.Now(),
		}
		if db != nil {
			if err := db.AddDoc("inbound/pager/textLogs", logData); err != nil {
				log.Print(err.Error())
			}
		}
		message := "Success!"
		if statusCode != 200 && statusCode != 201 {
			message = "Oh no! Something went wrong..."
		}
		if statusCode == 401 {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.Header().Set("realm", "Pager Rebug")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"Response": message})
	}

	db, err = NewFirestore()
	if err != nil {
		handleResponse(http.StatusInternalServerError, err)
		return
	}
	defer db.client.Close()
	// tc := NewTwilio()

	// Parse input.
	data = map[string]string{}
	if b, err := io.ReadAll(r.Body); err != nil {
		handleResponse(http.StatusBadRequest, err)
		return
	} else if params, err := url.ParseQuery(string(b)); err != nil {
		handleResponse(http.StatusBadRequest, err)
		return
	} else {
		for key := range params {
			data[key] = params.Get(key)
		}
	}

	// Validate auth.
	if err := BasicAuth(r); err != nil {
		handleResponse(http.StatusUnauthorized, err)
		return
	}

	// Fetch existing doc.
	docPath, docData, initialSync, err := HandleUserDoc(db, data["Body"], data["From"])
	if err != nil && err.Error() == "No doc found." {
		handleResponse(http.StatusBadRequest, err)
		return
	} else if err != nil {
		handleResponse(http.StatusInternalServerError, err)
		return
	}

	// Update and write back to db.
	newDocData := MergeAutoDoc(*docData, data["Body"], data, initialSync)
	if err := db.WriteDoc(docPath, newDocData); err != nil {
		handleResponse(http.StatusInternalServerError, err)
		return
	}

	// Respond.
	handleResponse(http.StatusCreated, nil)
}

func BasicAuth(r *http.Request) error {
	credentials := strings.Split(os.Getenv("TWILIO_WEBHOOK_AUTH"), " ")
	if len(credentials) != 2 {
		return errors.New("Invalid local env callback credentials.")
	}
	if un, pw, ok := r.BasicAuth(); !ok {
		return errors.New("No credentials found on request.")
	} else if un != credentials[0] || pw != credentials[1] {
		return errors.New("Auth failed.")
	}
	return nil
}
