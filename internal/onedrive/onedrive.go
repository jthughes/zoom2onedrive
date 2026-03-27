package onedrive

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	graph "github.com/microsoftgraph/msgraph-sdk-go"
)

type OneDriveConfig struct {
	Client   *graph.GraphServiceClient
	UserID   string
	DriveID  string
	FolderID string
	BaseURL  string
}

func InitOneDrive() OneDriveConfig {
	tenantID := os.Getenv("GRAPH_TENANT_ID")
	clientID := os.Getenv("GRAPH_CLIENT_ID")
	clientSecret := os.Getenv("GRAPH_CLIENT_SECRET")
	userID := os.Getenv("GRAPH_USER_ID")
	driveID := os.Getenv("GRAPH_DRIVE_ID")
	folderID := os.Getenv("GRAPH_FOLDER_ID")
	if tenantID == "" || clientID == "" || clientSecret == "" || userID == "" || driveID == "" || folderID == "" {
		log.Fatalln("Environment Not Set")
	}
	log.Println("Load Environment")

	cred, err := azidentity.NewClientSecretCredential(
		tenantID,
		clientID,
		clientSecret,
		nil,
	)
	if err != nil {
		log.Fatalln("Faile to build credentials")
	}
	log.Println("Created Credential")

	graphClient, err := graph.NewGraphServiceClientWithCredentials(
		cred, []string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		log.Fatalln("Failed to initialise Graph client")
	}
	log.Println("Started Graph Client")

	return OneDriveConfig{
		Client:   graphClient,
		UserID:   userID,
		DriveID:  driveID,
		FolderID: folderID,
		BaseURL:  "https://graph.microsoft.com/v1.0",
	}

}

// https://learn.microsoft.com/en-us/onedrive/developer/rest-api/concepts/long-running-actions?view=odsp-graph-online
func (graph OneDriveConfig) UploadFileToFolder(downloadPath, filename string) {

	file, err := os.Open(downloadPath)
	if err != nil {
		log.Fatalln("Error opening file")
	}
	defer file.Close()
	log.Println("Opened File")

	// POST request to upload file to target
	uploadURL, err := graph.requestUpload(filename)
	if err != nil {
		log.Println(err)
		return
	}

	// https://learn.microsoft.com/en-us/onedrive/developer/rest-api/api/driveitem_createuploadsession?view=odsp-graph-online
	// New Request (PUT) - don't include auth header, only in POST
	err = graph.uploadFileToURL(file, uploadURL)
	if err != nil {
		log.Println(err)
		return
	}

	// Check for file confliect response (409), or resume disconnected upload

}

func Request(request string, endpoint string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		if err != nil {
			return nil, fmt.Errorf("failed to do request: %w", err)
		}
	}

	client := http.DefaultClient
	req, err := http.NewRequest(request, endpoint, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))
	// /

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	return resp, err
}

func (graph OneDriveConfig) requestUpload(filename string) (string, error) {
	type RequestUploadBodyItem struct {
		ConflictBehaviour string `json:"@microsoft.graph.conflictBehavior"`
		Name              string `json:"name"`
	}
	type RequestUploadBody struct {
		Item RequestUploadBodyItem `json:"item"`
	}
	uploadBody := RequestUploadBody{
		Item: RequestUploadBodyItem{
			Name:              filename,
			ConflictBehaviour: "rename",
		},
	}

	endpoint := fmt.Sprintf("%s/drives/%s/items/%s:/%s:/createUploadSession", graph.BaseURL, graph.DriveID, graph.FolderID, filename)

	res, err := Request("POST", endpoint, uploadBody)

	// Should be 200 OK.
	type ResponseCreateUploadSession struct {
		UploadUrl          string `json:"uploadUrl"`
		ExpirationDateTime string `json:"expirationDateTime"`
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to request upload: %d %s", res.StatusCode, res.Status)
	}

	var respBody ResponseCreateUploadSession
	err = json.NewDecoder(res.Body).Decode(&respBody)
	if err != nil {
		return "", fmt.Errorf("failed to decode request body: %w\n", err)
	}
	return respBody.UploadUrl, nil
}

func (graph OneDriveConfig) uploadFileToURL(file *os.File, uploadURL string) error {
	const byteRangeSizeUnit = 327680
	const byteRangeSize = byteRangeSizeUnit * 32

	nextByte := int64(0)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	totalBytes := fileInfo.Size()

	client := http.DefaultClient

	fileReader := bufio.NewReader(file)

	for nextByte < totalBytes {
		fragmentSize := min(byteRangeSize, totalBytes-nextByte)

		data := make([]byte, fragmentSize)
		fileReader.Read(data)

		buffer := bytes.NewBuffer(data)

		req, err := http.NewRequest("PUT", uploadURL, buffer)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Add("Content-Length", fmt.Sprintf("%d", fragmentSize))
		req.Header.Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", nextByte, fragmentSize-1, totalBytes))

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to do request: %w", err)
		}

		if resp.StatusCode == http.StatusAccepted {
			// Continue upload
			// 202 Accepted
			type ResponseePutUpload struct {
				ExpirationDateTime string   `json:"expirationDateTime"`
				NextExpectedRanges []string `json:"nextExpectedRanges"`
			}
			nextByte += fragmentSize

		} else if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			// Upload completed
			type ResponseUploaded struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Size int    `json:"size"`
				File any    `json:"file"`
			}
			break
		} else {
			// error
			return fmt.Errorf("received error: %d %s", resp.StatusCode, resp.Status)
		}
	}
	return nil
}
