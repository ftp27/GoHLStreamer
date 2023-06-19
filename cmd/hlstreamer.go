package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"strings"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	bucketName      string
	baseURL         string
	tempDir         string
	inputDir        string
	outputDir       string
	ffmpegPath      string
)

func main() {
	// Load settings from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Read environment variables
	endpoint = os.Getenv("ENDPOINT")
	accessKeyID = os.Getenv("ACCESS_KEY_ID")
	secretAccessKey = os.Getenv("SECRET_ACCESS_KEY")
	bucketName = os.Getenv("BUCKET_NAME")
	baseURL = os.Getenv("BASE_URL")
	tempDir = os.Getenv("TEMP_DIR")
	inputDir = os.Getenv("INPUT_DIR")
	outputDir = os.Getenv("OUTPUT_DIR")
	ffmpegPath = os.Getenv("FFMPEG_PATH")

	// Initialize the DigitalOcean Spaces client.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalln("Failed to initialize DigitalOcean Spaces client:", err)
	}

	// Check if the bucket exists
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Fatalf("Bucket '%s' does not exist", bucketName)
	}

	// Create temporary directory for HLS files.
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		log.Fatalln("Failed to create temporary directory:", err)
	}

	http.HandleFunc("/hls/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path[len("/hls/"):]
		parts := strings.SplitN(urlPath, "/", 2)
		objectName := parts[0]
		log.Println("Request for object:", parts, len(parts))
		if len(parts) != 2 {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		filename := parts[1]
		log.Println("Request for object:", objectName)

		// Check if the request is for the HLS playlist
		isPlaylistRequest := filename == "playlist.m3u8"

		log.Println("Request for playlist:", isPlaylistRequest)
		// Check filename is one element
		if !isPlaylistRequest && !isValidSegmentFilename(filename) {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
	
		// Construct the object path based on the request type
		objectPath := outputDir + "/" + objectName + "/" + filename
	
		// Check if HLS files already exist.
		hlsFilesExist, err := checkHLSFilesExist(objectName)
		if err != nil {
			log.Println("Failed to check if HLS files exist:", err)
			http.Error(w, "Failed to check if HLS files exist", http.StatusInternalServerError)
			return
		}
	
		if !hlsFilesExist {
			// Convert MP4 to HLS format.
			err := convertToHLS(minioClient, bucketName, objectName)
			if err != nil {
				log.Println("Failed to convert MP4 to HLS:", err)
				http.Error(w, "Failed to load HLS", http.StatusInternalServerError)
				return
			}
		}
	
		// Retrieve the requested object from DigitalOcean Spaces
		object, err := minioClient.GetObject(context.Background(), bucketName, objectPath, minio.GetObjectOptions{})
		if err != nil {
			log.Println("Failed to retrieve object:", err)
			http.Error(w, "Failed to retrieve object", http.StatusInternalServerError)
			return
		}
	
		// Set the appropriate content type for the response
		contentType := "application/vnd.apple.mpegurl"
		if !isPlaylistRequest {
			contentType = "video/mp2t"
		}
		w.Header().Set("Content-Type", contentType)
	
		// Serve the object content
		http.ServeContent(w, r, filepath.Base(objectPath), time.Now(), object)
	})

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func isValidSegmentFilename(filename string) bool {
	validExtensions := []string{".ts"} // List of valid HLS segment extensions
	ext := filepath.Ext(filename)
	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

func checkHLSFilesExist(objectName string) (bool, error) {
	manifestPath := filepath.Join(tempDir, objectName, "playlist.m3u8")
	log.Println("Checking if HLS manifest exists:", manifestPath)
	_, err := os.Stat(manifestPath)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func convertToHLS(client *minio.Client, bucket, objectName string) error {
	// Create temporary directory for the HLS files.
	hlsDir := filepath.Join(tempDir, objectName)
	err := os.MkdirAll(hlsDir, 0755)
	if err != nil {
		return err
	}

	// Set output paths.
	manifestPath := filepath.Join(hlsDir, "playlist.m3u8")
	segmentPath := filepath.Join(hlsDir, "segment%03d.ts")

	// Download the MP4 object from DigitalOcean Spaces.
	mp4FilePath := filepath.Join(tempDir, objectName+".mp4")
	mp4DOPath := fmt.Sprintf("%s/%s.mp4", inputDir, objectName)
	log.Println("Downloading MP4 file from DigitalOcean Spaces:", mp4DOPath)
	err = client.FGetObject(context.Background(), bucket, mp4DOPath, mp4FilePath, minio.GetObjectOptions{})
	if err != nil {
		log.Println("Here!")
		return err
	}
	defer os.Remove(mp4FilePath)

	// Run FFmpeg command to convert MP4 to HLS.
	cmd := exec.Command(ffmpegPath,
		"-i", mp4FilePath,
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "128k",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-hls_segment_filename", segmentPath,
		"-hls_playlist_type", "vod",
		manifestPath,
	)
	err = cmd.Run()
	if err != nil {
		return err
	}

	// Upload HLS files to DigitalOcean Spaces.
	err = uploadHLSFiles(client, bucket, objectName, hlsDir)
	if err != nil {
		return err
	}

	return nil
}

func uploadHLSFiles(client *minio.Client, bucket, objectName, hlsDir string) error {
	err := filepath.Walk(hlsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Calculate the object name relative to the HLS directory.
		relativePath, err := filepath.Rel(hlsDir, path)
		if err != nil {
			return err
		}

		// Upload the HLS file to DigitalOcean Spaces.
		objectPath := fmt.Sprintf("%s/%s/%s", outputDir, objectName, relativePath)
		log.Println("Uploading HLS file to DigitalOcean Spaces:", objectPath)
		_, err = client.FPutObject(context.Background(), bucket, objectPath, path, minio.PutObjectOptions{})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
