package rebugpager

import (
	"errors"
	"net/http"
	"os"

	"github.com/twilio/twilio-go/client"
)

type TwilioClient struct {
	callbackUrl string
	validator   client.RequestValidator
}

func NewTwilio() *TwilioClient {
	tc := TwilioClient{}
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	tc.callbackUrl = os.Getenv("TWILIO_CALLBACK_URL")
	tc.validator = client.NewRequestValidator(authToken)
	return &tc
}

// https://www.twilio.com/docs/usage/webhooks/webhooks-security
func (tc *TwilioClient) DoAuth(params map[string]string, r *http.Request) error {
	signature := r.Header.Get("x-twilio-signature")
	if len(signature) > 0 && tc.validator.Validate(r.RequestURI, params, signature) {
		return nil
	}
	return errors.New("Unable to validate incomming request.")
}
