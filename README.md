# Pager lambda function for SMS to HTML
This repository hosts a GCP lambda meant to serve as a Twilio inbound SMS callback.

## Prerequisits
- The database service expects a local `serviceAccount.json` file for authentication into a firebase project.
- The SMS service expects environment variables `TWILIO_AUTH_TOKEN` and `TWILIO_CALLBACK_URL` for the active Twilio account/config.