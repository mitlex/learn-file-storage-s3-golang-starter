package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

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

	mediaType := headers.Header.Get("Content-Type")
	if mediaType == "" { // Ensures the client specified a content type for the file, we need this for our imageDataURL
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading file", err)
		return
	}

	imageAsBase64String := base64.StdEncoding.EncodeToString(data) // Encoding the image as a base64 string allows us to store it in the database
	imageDataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, imageAsBase64String)
	video.ThumbnailURL = &imageDataURL // Not every video has a thumbnail, therefore can be NULL in db, strings nil value is "" but that could count as a URL, so we use *string which can definitely be nil.

	// Update the video record in the database.
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	// The browser decodes the Base64 image data in the data URL and renders it directly.
	// This replaces the separate GET request that previously fetched the thumbnail.
	respondWithJSON(w, http.StatusOK, video)
}
