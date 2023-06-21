package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"strings"
	"strconv"

	"github.com/ftp27/GoHLStreamer/pkg/spaces"
	"github.com/ftp27/GoHLStreamer/pkg/cache"
	"github.com/ftp27/GoHLStreamer/pkg/appwrite"
	"github.com/joho/godotenv"
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
	cacheSize       int

	useAppwrite	    bool
	appwriteHost    string
	appwriteProject string
	appwriteSecret  string
	appwriteBucket  string

	storage 		*spaces.Spaces
	appwriteClient  *appwrite.Appwrite
	cacheStorage    *cache.LRUCache
)

func prepareConfig() {
	// Load settings from .env file
	err := godotenv.Load()

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

	cacheSizeStr := os.Getenv("CACHE_SIZE")
	cacheSize, err = strconv.Atoi(cacheSizeStr)
	if err != nil {
		log.Fatal("Invalid CACHE_SIZE value:", err)
	}

	appwriteHost = os.Getenv("APPWRITE_HOST")
	if appwriteHost == "" {
		log.Println("Appwrite integration is disabled")
		useAppwrite = false
	} else {
		useAppwrite = true
		appwriteProject = os.Getenv("APPWRITE_PROJECT")
		appwriteSecret = os.Getenv("APPWRITE_SECRET")
		appwriteBucket = os.Getenv("APPWRITE_BUCKET")
	}
}

func prepareStorage() {
	var err error
	storage, err = spaces.New(endpoint, accessKeyID, secretAccessKey, bucketName, baseURL, inputDir, outputDir)
	if err != nil {
		log.Fatalln("Failed to initialize DigitalOcean Spaces client:", err)
	}
	
	exists, err := storage.BucketExists()
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Fatalf("Bucket '%s' does not exist", bucketName)
	}
}

func prepareAppwrite() {
	if !useAppwrite {
		return
	}
	var err error
	appwriteClient, err = appwrite.New(appwriteProject, appwriteBucket, appwriteSecret, appwriteHost)
	if err != nil {
		log.Fatal(err)
	}
}

func prepareCache() {
	var err error
	cacheStorage, err = cache.New(cacheSize, tempDir)
	if err != nil {
		log.Fatal(err)
	}
}

func getObjectAndFileName(objectPath string) (string, string, error) {
	parts := strings.SplitN(objectPath, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid object path: %s", objectPath)
	}
	return parts[0], parts[1], nil
}

func writeErrorJson(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json := fmt.Sprintf(`{"error": "%s"}`, message)
	w.Write([]byte(json))
}

func cloudFilePath(objectId string, filename string) string {
	return fmt.Sprintf("%s/%s/%s", outputDir, objectId, filename)
}

func tmpFilePath(objectId string, filename string) string {
	return fmt.Sprintf("%s/%s/%s", tempDir, objectId, filename)
}

func Run() {
	prepareConfig()
	prepareStorage()
	prepareCache()
	prepareAppwrite()

	http.HandleFunc("/hls/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path[len("/hls/"):]
		objectName, filename, error := getObjectAndFileName(urlPath)
		if error != nil {
			writeErrorJson(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Check if the request is for the HLS playlist
		isPlaylistRequest := filename == "playlist.m3u8"

		// Check filename is one element
		if !isPlaylistRequest && !isValidSegmentFilename(filename) {
			writeErrorJson(w, "Invalid request", http.StatusBadRequest)
			return
		}
	
		// Construct the object path based on the request type
		objectPath := cloudFilePath(objectName, filename)
		localPath := tmpFilePath(objectName, filename)
	
		// Check if HLS files already exist.
		hlsFilesExist, err := storage.CheckObject(objectPath)
		if err != nil {
			log.Println("Failed to check if HLS files exist:", err)
			writeErrorJson(w, "Failed to check HLS files", http.StatusInternalServerError)
			return
		}

		if !hlsFilesExist {
			// Convert MP4 to HLS format.
			err := convertToHLS(objectName)
			if err != nil {
				log.Println("Failed to convert MP4 to HLS:", err)
				writeErrorJson(w, "Failed to load HLS", http.StatusInternalServerError)
				return
			}
		}

		// Retrieve the requested object from DigitalOcean Spaces
		object, err := storage.GetObject(objectPath)
		if err != nil {
			log.Println("Failed to retrieve object:", err)
			writeErrorJson(w, "Failed to retrieve object", http.StatusInternalServerError)
			return
		}
	
		// Set the appropriate content type for the response
		if isPlaylistRequest {
			w.Header().Set("Content-Type",  "application/vnd.apple.mpegurl")
		} else {
			w.Header().Set("Content-Type", "video/mp2t")
		}
	
		// Serve the object content
		http.ServeContent(w, r, localPath, time.Now(), object)
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

func convertToHLS(objectName string) error {
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
	
	if useAppwrite {
		err = appwriteClient.GetFile(objectName, mp4FilePath)
	} else {
		err = storage.FGetObject(mp4DOPath, mp4FilePath)
	}
	if err != nil {
		return err
	}
	defer os.Remove(mp4FilePath)

	// Run FFmpeg command to convert MP4 to HLS.
	cmd := exec.Command(ffmpegPath,
		"-i", mp4FilePath,
		"-c:v", "copy",
		"-c:a", "copy",
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
	err = uploadHLSFiles(objectName, hlsDir)
	if err != nil {
		return err
	}

	return nil
}

func uploadHLSFiles(objectName, hlsDir string) error {
	err := filepath.Walk(hlsDir, func(path string, info os.FileInfo, err error) error {
		defer os.Remove(path)

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
		objectPath := cloudFilePath(objectName, relativePath)
		log.Println("Uploading HLS file to DigitalOcean Spaces:", objectPath)
		_, err = storage.PutObject(objectPath, path)
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
