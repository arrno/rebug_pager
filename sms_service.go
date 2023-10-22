package rebugpager

import (
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

// signature is x-twilio-signature header
func (tc *TwilioClient) Validate(params map[string]string, signature string) bool {
	return tc.validator.Validate(tc.callbackUrl, params, signature)
}
