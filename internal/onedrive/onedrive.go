package onedrive

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	graph "github.com/microsoftgraph/msgraph-sdk-go"

	"github.com/microsoftgraph/msgraph-sdk-go-core/fileuploader"
	"github.com/microsoftgraph/msgraph-sdk-go/drives"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

type OneDriveConfig struct {
	Client  *graph.GraphServiceClient
	UserID  string
	DriveID string
}

func InitOneDrive() OneDriveConfig {
	tenantID := os.Getenv("GRAPH_TENANT_ID")
	clientID := os.Getenv("GRAPH_CLIENT_ID")
	clientSecret := os.Getenv("GRAPH_CLIENT_SECRET")
	userID := os.Getenv("GRAPH_USER_ID")
	driveID := os.Getenv("GRAPH_DRIVE_ID")
	if tenantID == "" || clientID == "" || clientSecret == "" || userID == "" || driveID == "" {
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
		Client:  graphClient,
		UserID:  userID,
		DriveID: driveID,
	}

}

func (graph OneDriveConfig) UploadFileToFolder(downloadPath, uploadPath string) {

	file, err := os.Open(downloadPath)
	if err != nil {
		log.Fatalln("Error opening file")
	}
	defer file.Close()
	log.Println("Opened File")

	// Use properties to specify the conflict behavior
	itemUploadProperties := models.NewDriveItemUploadableProperties()
	itemUploadProperties.SetAdditionalData(map[string]any{"@microsoft.graph.conflictBehavior": "replace"})
	uploadSessionRequestBody := drives.NewItemItemsItemCreateUploadSessionPostRequestBody()
	uploadSessionRequestBody.SetItem(itemUploadProperties)

	// Create the upload session
	// itemPath does not need to be a path to an existing item

	uploadSession, _ := graph.Client.Drives().
		ByDriveId(graph.DriveID).
		Items().
		ByDriveItemId("root:/"+uploadPath+":").
		CreateUploadSession().
		Post(context.Background(), uploadSessionRequestBody, nil)
	log.Println("Created Upload Session")

	// Max slice size must be a multiple of 320 KiB
	maxSliceSize := int64(320 * 1024)
	fileUploadTask := fileuploader.NewLargeFileUploadTask[models.DriveItemable](
		graph.Client.RequestAdapter,
		uploadSession,
		file,
		maxSliceSize,
		models.CreateDriveItemFromDiscriminatorValue,
		nil)

	// Create a callback that is invoked after each slice is uploaded
	progress := func(progress int64, total int64) {
		fmt.Printf("Uploaded %d of %d bytes\n", progress, total)
	}

	log.Println("Starting File Upload")
	// Upload the file
	uploadResult := fileUploadTask.Upload(progress)

	if uploadResult.GetUploadSucceeded() {
		fmt.Printf("Upload complete, item ID: %s\n", *uploadResult.GetItemResponse().GetId())
	} else {
		fmt.Print("Upload failed.\n")
	}
}
