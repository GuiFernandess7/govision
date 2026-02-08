package file

import (
	"bytes"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"govision/services/rabbitmq"
	storage "govision/services/storage"

	"github.com/oklog/ulid/v2"

	"github.com/labstack/echo/v4"
)

const HOST_IMAGE_URL = "https://api.imgbb.com/1/upload"

type Handler struct {
	publisher rabbitmq.JobPublisher
}

func NewHandler(p rabbitmq.JobPublisher) *Handler {
	return &Handler{publisher: p}
}

func (h *Handler) UploadFileImage(c echo.Context) error {
	log.Println("[STARTING] - calling route /image/upload...")
	var request UploadRequest

	if err := c.Bind(&request); err != nil {
		log.Printf("[ERROR] - Invalid payload: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Invalid payload",
		})
	}

	log.Println("[RUNNING] - Getting file.")
	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("[ERROR] - error getting file data: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error getting file data",
		})
	}

	log.Println("[RUNNING] - Validating file size.")
	if err := ValidateFileSize(file); err != nil {
		log.Printf("[ERROR] - Error validating file size: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	fileObject, err := file.Open()
	if err != nil {
		log.Printf("[ERROR] - error reading file: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error reading file data",
		})
	}

	defer fileObject.Close()
	log.Println("[RUNNING] - Validating file content...")
	if err := ValidateFileContent(fileObject); err != nil {
		log.Printf("[ERROR] - Error validating file content: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "unexpected error has occurred",
		})
	}

	log.Println("[RUNNING] - Processing file data...")
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, fileObject)
	if err != nil {
		log.Printf("[ERROR] - Unexpected error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "error processing file image",
		})
	}

	log.Println("[RUNNING] - Sending image to storage service")
	service := storage.StorageService[ImgBBResponse]{
		URL: HOST_IMAGE_URL,
	}

	var storage_service_key = os.Getenv("STORAGE_API_KEY")
	if storage_service_key == "" {
		log.Printf("[ERROR] - API Key not found")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "unexpected error has occurred",
		})
	}

	responseObj, err := service.GetImageUrl(buf, storage_service_key)
	if err != nil {
		log.Printf("[ERROR] - Error sending image to %v: %v", service.URL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "storage service error",
		})
	}

	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	jobID := ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
	ctx := c.Request().Context()

	if err := h.publisher.Publish(ctx, jobID, responseObj.Data.URL); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "failed to enqueue job",
		})
	}

	log.Printf("[SUCCESS] - File image processed successfully!")
	return c.JSON(http.StatusAccepted, map[string]string{
		"job_id": jobID,
		"status": "queued",
	})
}
