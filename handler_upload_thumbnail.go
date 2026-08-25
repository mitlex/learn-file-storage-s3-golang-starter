package main

import (
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
	if mediaType == "" { // ensures the client specified a content type for the file, we want this for our thumbnail struct
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading file", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized Request", nil)
		return
	}

	tn := thumbnail{
		data:      data,
		mediaType: mediaType,
	}
	videoThumbnails[video.ID] = tn

	// Update video metadata with a new thumbnail URL.
	newThumbnailURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoID)
	video.ThumbnailURL = &newThumbnailURL // not every video has a thumbnail, therefore can be NULL in db, strings nil value is "" but that could count as a URL, so we use *string which can definitely be nil.

	// Update the video record in the database.
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		delete(videoThumbnails, videoID) // we want to remove the thumbnail from our map if the database doesn't update
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
