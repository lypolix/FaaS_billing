package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

func (h Handler) UploadServiceArtifact(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var s models.Service
	if err := database.DB.First(&s, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	baseDir := os.Getenv("ARTIFACTS_DIR")
	if baseDir == "" {
		baseDir = "./data/artifacts"
	}

	safeName := filepath.Base(file.Filename)
	dir := filepath.Join(baseDir, uid.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create artifacts dir"})
		return
	}

	dst := filepath.Join(dir, safeName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + err.Error()})
		return
	}

	f, err := os.Open(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open saved file"})
		return
	}
	defer f.Close()

	hsh := sha256.New()
	if _, err := io.Copy(hsh, f); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash file"})
		return
	}
	sha := hex.EncodeToString(hsh.Sum(nil))

	s.ArtifactPath = dst
	s.ArtifactName = safeName
	s.ArtifactSize = file.Size
	s.ArtifactSHA = sha

	if err := database.DB.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update service"})
		return
	}

	artifactURL := "/api/v1/artifacts/" + uid.String() + "/" + safeName
	c.JSON(http.StatusOK, gin.H{
		"service_id":   s.ID,
		"artifact_url": artifactURL,
		"size_bytes":   s.ArtifactSize,
		"sha256":       s.ArtifactSHA,
	})
}

func (h Handler) DownloadArtifact(c *gin.Context) {
	serviceID := c.Param("service_id")
	uid, err := uuid.Parse(serviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var s models.Service
	if err := database.DB.First(&s, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	if s.ArtifactPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "artifact not uploaded"})
		return
	}

	c.File(s.ArtifactPath)
}
