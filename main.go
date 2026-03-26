package main

import (
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/joho/godotenv"
	"github.com/jthughes/zoom2onedrive/internal/zoom"
)

type Config struct {
	ZoomApiKey string
}

func main() {
	const port = "8080"

	godotenv.Load(".env")

	cfg := Config{
		ZoomApiKey: os.Getenv("ZOOM_API_TOKEN"),
	}

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.HandleFunc("POST /webhook", cfg.handlerZoomWebhook)

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}

func ManageRecording(payload zoom.PayloadRecordingCompleted, downloadToken string) {
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
		// req, err := http.NewRequest("GET", file.DownloadURL, nil)
		// if err != nil {

		// }
		// req.Header.Set("Content-Type", "appliccation/json")
		// req.Header.Set("authorization", fmt.Sprintf("Bearer %s", downloadToken))

		// client := &http.Client{}
		// res, err := client.Do(req)
		// if err != nil {

		// }
		// defer res.Body.Close()

	}

}
