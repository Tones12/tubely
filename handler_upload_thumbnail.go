package main

import (
	"fmt"
	"io"
	"os"
	"net/http"
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

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Coudln't parse body request", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse thumbnail request", err)
		return
	}
	fileType := header.Header.Get("Content-Type")
	splitFileType := strings.Split(fileType, "/")
	if len(splitFileType) != 2 {
		respondWithError(w, http.StatusBadRequest, "Couldn't determine content type", err)
		return
	}
	ext := "." + splitFileType[1]
	filePath := filepath.Join(cfg.assetsRoot, videoIDString + ext)
	path, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error creating file", err)
		return
	}

	_, err = io.Copy(path, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy file", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find video", err)
		return
	}
	if videoMetadata.CreateVideoParams.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid userID", nil)
		return
	}

	thumbnailURL := fmt.Sprintf("https://cuddly-goldfish-wrwr7w5qg5vh9vpp-%s.app.github.dev/assets/%s%s", cfg.port, videoIDString, ext)

	videoMetadata.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
