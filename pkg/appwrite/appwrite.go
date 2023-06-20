package appwrite

import (
	"fmt"
	"io"
	"os"
	
	"net/http"
)

type Appwrite struct {
	ProjectId		string
	BucketId		string
	ApiSecret		string
	BaseURL         string

	client 		    *http.Client
}


func New(projectId, backetId, apiSecret, baseURL string) (*Appwrite, error) {
	appwrite := &Appwrite{
		ProjectId:       projectId,
		BucketId:        backetId,
		ApiSecret:       apiSecret,
		BaseURL:         baseURL,

		client:           http.DefaultClient,
	}
	return appwrite, nil
}

func (a *Appwrite) GetFile(fileId, filePath string) error {
	url := a.BaseURL + "/v1/storage/buckets/" + a.BucketId + "/files/" + fileId + "/download"

	// Create a new HTTP POST request with the JSON payload
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	// Set the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Appwrite-Response-Format", "1.0.0")
	req.Header.Set("X-Appwrite-Project", a.ProjectId)
    req.Header.Set("X-Appwrite-Key", a.ApiSecret)

	// Send HTTP GET request to retrieve the file
	response, err := a.client.Do(req)
	if err != nil {
		return err
	}
	
	defer response.Body.Close()

	// Check if the request was successful (status code 200)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file. Status code: %d", response.StatusCode)
	}

	// Create a new file to save the downloaded content
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the file content to the local file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}