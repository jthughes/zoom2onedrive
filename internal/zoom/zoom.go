package zoom

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type WebhookBody struct {
	Event          string          `json:"event"`
	Payload        json.RawMessage `json:"payload"`
	EventTimestamp int64           `json:"event_ts"`
	DownloadToken  string          `json:"download_token,omitempty"`
}
type PayloadVerfication struct {
	PlainToken string `json:"plainToken"`
}

type PayloadRecordingCompleted struct {
	AccountID string `json:"account_id"`
	Object    struct {
		AccountID      string `json:"account_id"`
		Duration       int    `json:"duration"`
		ID             int64  `json:"id"`
		RecordingCount int    `json:"recording_count"`
		RecordingFiles []struct {
			DownloadURL    string `json:"download_url"`
			FileExtension  string `json:"file_extension"`
			FileSize       int    `json:"file_size"`
			FileType       string `json:"file_type"`
			ID             string `json:"id"`
			RecordingStart string `json:"recording_start"`
			RecordingEnd   string `json:"recording_end"`
			RecordingType  string `json:"recording_type"`
			Status         string `json:"status"`
			FileName       string `json:"file_name"`
		} `json:"recording_files"`
		ShareURL  string `json:"share_url"`
		StartTime string `json:"start_time"`
		TotalSize int    `json:"total_size"`
		Type      int    `json:"type"`
		UUID      string `json:"uuid"`
	} `json:"object"`
}

func ValidateSender(r *http.Request, body []byte, apiKey string) error {
	log.Println("Verifying Sender")
	requestSignature := r.Header.Get("x-zm-signature")
	if requestSignature == "" {
		// error response
		return fmt.Errorf("Signature Missing")
	}
	requestTimestamp := r.Header.Get("x-zm-request-timestamp")
	if requestTimestamp == "" {
		// error response
		return fmt.Errorf("Timestamp Missing")
	}

	data := string(body)

	message := fmt.Sprintf("v0:%s:%s", requestTimestamp, data)
	hash := hmac.New(sha256.New, []byte(apiKey))
	_, err := hash.Write([]byte(message))
	if err != nil {
		// err
		return err
	}

	caculatedSignature := fmt.Sprintf("v0=%s", hex.EncodeToString(hash.Sum(nil)))
	log.Printf("Calculated signature: %s\n", caculatedSignature)

	// Verify webhook
	if caculatedSignature != requestSignature {
		return fmt.Errorf("Sender Validation Failed")
	}
	log.Println("Sender Verified")
	return nil
}

func ValidateZoomWebhook(w http.ResponseWriter, body WebhookBody, apiKey string) {
	payload := PayloadVerfication{}
	err := json.Unmarshal(body.Payload, &payload)
	if err != nil {
		log.Printf("Failed to unmarshal: %v\n", err)
		return
	}

	hash := hmac.New(sha256.New, []byte(apiKey))
	_, err = hash.Write([]byte(payload.PlainToken))
	if err != nil {
		log.Printf("Failed to hash PlainToken: %v\n", err)
		return
	}

	encryptedToken := hex.EncodeToString(hash.Sum(nil))

	type response struct {
		PlainToken     string `json:"plainToken"`
		EncryptedToken string `json:"encryptedToken"`
	}

	resp := response{
		PlainToken:     payload.PlainToken,
		EncryptedToken: string(encryptedToken),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	log.Println("Validated")
}
