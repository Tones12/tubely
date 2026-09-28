package main

import (
	"fmt"
	"io"
	"os"
	"net/http"
	"path/filepath"
	"mime"
	"crypto/rand"
	"encoding/base64"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random bytes", err)
		return
	}
	thumbnailURLString := base64.RawURLEncoding.EncodeToString(randBytes)
	
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

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse media type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "only jpeg or png media types are accepted", err)
		return
	}
	ext, err := mime.ExtensionsByType(mediaType)

	filePath := filepath.Join(cfg.assetsRoot, thumbnailURLString + ext[0])
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

	thumbnailURL := fmt.Sprintf("https://cuddly-goldfish-wrwr7w5qg5vh9vpp-%s.app.github.dev/assets/%s%s", cfg.port, thumbnailURLString, ext[0])
	//if testing is failing due to cloudspace proxy behaviour, switch to the below url generation scheme
	//thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, videoIDString, ext[0])
	videoMetadata.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
