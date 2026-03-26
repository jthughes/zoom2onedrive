package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/jthughes/zoom2onedrive/internal/zoom"
)

func (cfg Config) handlerZoomWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request")

	bodydata, err := io.ReadAll(r.Body)
	if err != nil {
		// error response
	}

	//	Test signatures
	if err := zoom.ValidateSender(r, bodydata, cfg.Zoom.ApiKey); err != nil {
		respondWithError(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), err)
		return
	}

	whBody := zoom.WebhookBody{}
	err = json.Unmarshal(bodydata, &whBody)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), err)
		return
	}

	if whBody.Event == "endpoint.url_validation" {
		log.Println("Starting Validation of Webhook")
		zoom.ValidateZoomWebhook(w, whBody, cfg.Zoom.ApiKey)
		return
	}

	if whBody.Event != "recording.completed" {
		log.Println("Acknowledged Unexpected Webhook")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Acknoweldege webhook
	log.Println("Processing Recording Completed webhook")
	w.WriteHeader(http.StatusNoContent)

	payload := zoom.PayloadRecordingCompleted{}
	err = json.Unmarshal(whBody.Payload, &payload)
	if err != nil {
		// err
		return
	}
	//	Start download go routine
	go cfg.ManageRecording(payload, whBody.DownloadToken)
}
