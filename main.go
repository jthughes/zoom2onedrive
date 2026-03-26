package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/joho/godotenv"
	"github.com/jthughes/zoom2onedrive/internal/onedrive"
	"github.com/jthughes/zoom2onedrive/internal/zoom"
)

type Config struct {
	Zoom struct {
		ApiKey string
	}
	OneDrive onedrive.OneDriveConfig
}

func main() {
	const port = "8080"

	godotenv.Load(".env")

	cfg := Config{
		Zoom: struct{ ApiKey string }{
			ApiKey: os.Getenv("ZOOM_API_TOKEN"),
		},
		OneDrive: onedrive.InitOneDrive(),
	}

	cfg.OneDrive.UploadFileToFolder("", "")

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.HandleFunc("POST /webhook", cfg.handlerZoomWebhook)

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}

func (cfg Config) ManageRecording(payload zoom.PayloadRecordingCompleted, downloadToken string) {
	log.Println("Start syncing Recording")

	// log.Printf("%+v\n", payload)

	toDownload := []string{"shared_screen_with_speaker_view", "audio_only"}

	for _, file := range payload.Object.RecordingFiles {
		if slices.Contains(toDownload, file.RecordingType) == false {
			log.Printf("%s recording type available - Skipped\n", file.RecordingType)
			continue
		}
		log.Printf("%s recording type availabvle - Downloading!", file.RecordingType)
		log.Printf("%+v\n", file)

		// Download recording
		req, err := http.NewRequest("GET", file.DownloadURL, nil)
		if err != nil {

		}
		req.Header.Set("Content-Type", "appliccation/json")
		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", downloadToken))

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			// err
		}
		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			// err
		}

		// Save to temp file
		downloadPath := filepath.Join(os.TempDir(), file.ID)
		err = os.WriteFile(downloadPath, data, 0644)
		if err != nil {
			// err
		}
		defer os.Remove(downloadPath)

		cfg.OneDrive.UploadFileToFolder(downloadPath, "")

	}

}
