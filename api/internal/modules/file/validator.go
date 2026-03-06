package file

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
)

const MAX_FILE_SIZE = 15 * 1024 * 1024

func ValidateFileContent(src multipart.File) error {
	buffer := make([]byte, 512)
	if _, err := src.Read(buffer); err != nil {
		return err
	}

	contentType := http.DetectContentType(buffer)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}

	if !allowedTypes[contentType] {
		return errors.New("Invalid file image.")
	}

	if _, err := src.Seek(0, 0); err != nil {
		return err
	}

	return nil
}

func ValidateFileSize(file *multipart.FileHeader) error {
	if file.Size > MAX_FILE_SIZE {
		return fmt.Errorf("The file is too big. Limit is %v", MAX_FILE_SIZE)
	}
	return nil
}
