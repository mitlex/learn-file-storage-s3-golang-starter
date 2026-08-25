package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// Bit shifting to get memory in bytes.
	// We want 10 MB which is 10,485,760 bytes.
	// Operation is equivalent to 10 * 2^20.
	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form", err)
		return
	}

	file, headers, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	cTypeVal := headers.Header.Get("Content-Type")
	if cTypeVal == "" { // Ensures the client specified a content type for the file, we need this for our new file and thumbnail URL
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}

	// Split content type value into parts so we can grab the file extension i.e. image/png -> png
	cTypeValParts := strings.Split(cTypeVal, "/")
	if len(cTypeValParts) != 2 {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}
	fileExtension := cTypeValParts[1]

	fileName := fmt.Sprintf("%s.%s", video.ID, fileExtension)
	fileOnDiskPath := filepath.Join(cfg.assetsRoot, fileName)
	fileOnDisk, err := os.Create(fileOnDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	defer fileOnDisk.Close()

	_, err = io.Copy(fileOnDisk, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	tnURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
	video.ThumbnailURL = &tnURL // Not every video has a thumbnail, therefore can be NULL in db, strings nil value is "" but that could count as a URL, so we use *string which can definitely be nil.

	// Update the video record in the database.
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	// 1. The API responds with the updated video JSON containing the thumbnail URL.
	// 2. Frontend JavaScript receives the response and sets the URL as the src attribute of an <img> element.
	// 3. The browser detects the image source and automatically sends an HTTP GET request to that URL.
	// 4. The file server in main.go reads and streams the file from disk in chunks; the browser renders and caches the image as a result of the cacheMiddleware wrapper (see main.go).
	respondWithJSON(w, http.StatusOK, video)
}
