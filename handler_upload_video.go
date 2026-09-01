package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// Get the video first to check authorization to save compute on parsing the uploaded data
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Video not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve video metadata", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Bit shifting to get memory in bytes.
	// We want a 1GiB video upload limit which is 1,073,741,824 bytes.
	// Operation is equivalent to 1 * 2^30.
	const maxUploadSize = 1 << 30

	// Limit the entire request body, including multipart overhead, to 1 GiB.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// FormFile parses multipart form data using Go's default memory threshold.
	// A file within that threshold may remain in RAM; a larger file is stored
	// on temporary disk rather than split between RAM and disk.
	// Later, io.Copy streams it into our own temporary file.
	file, headers, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(headers.Header.Get("Content-Type")) // Media type = MIME type, content-type val = mime type + params (like charset)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}

	// "" uses system default directory.
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}
	// We write close after remove because defer is LIFO; close will happen before remove.
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	// This allows us to read the file again from the beginning
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	fileNameBytes := make([]byte, 32)
	_, err = rand.Read(fileNameBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}
	hexFileName := hex.EncodeToString(fileNameBytes)
	fullFileName := hexFileName + ".mp4"

	aspectRatio, err := getVideoAspectRatio(tempFile.Name()) // Name returns the path used to create the temporary file.
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to process video", err)
		return
	}

	prefix := ""
	switch aspectRatio {
	case "9:16":
		prefix = "portrait"
	case "16:9":
		prefix = "landscape"
	default:
		prefix = "other"
	}

	key := prefix + "/" + fullFileName

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        tempFile, // *os.File implements Read method, thus satisfies io.Reader interface
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to store video", err)
		return
	}

	// S3 URLs are in the format: https://<bucket-name>.s3.<region>.amazonaws. com/<key>
	s3VideoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	video.VideoURL = &s3VideoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		// We delete the orphaned object because we don't want it there if we failed to update our db record metadata
		_, deleteErr := cfg.s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
			Bucket: &cfg.s3Bucket,
			Key:    &key,
		})
		// Attempt best-effort cleanup while preserving the database error.
		if deleteErr != nil {
			log.Printf("Failed to delete video in s3 after database update failure: %v", deleteErr)
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to save video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
