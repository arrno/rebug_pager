# Pager lambda function for SMS to HTML
This repository hosts a GCP lambda meant to serve as a Twilio inbound SMS callback.

## Prerequisits
- The database service expects an environment variable `FIREBASE_AUTH` containing service account json for authentication into the firebase project.
- The SMS service expects environment variables `TWILIO_AUTH_TOKEN`, `TWILIO_WEBHOOK_AUTH`, and `TWILIO_CALLBACK_URL` for the active Twilio account/config.

## Architecture
This is a very simple service. Our main HTTP handler is in `rebug_parser.go`. We have a little bit of types and business logic defined in `inbound_message.go`. Our database and sms services are very modest and are located in `database.go` and `sms_service.go` respectively. `deploy.sh` is our makeshift CICD pipeline. That's it!