package bot

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// downloadImage fetches the file identified by fileID from Telegram server and returns the file's data as a byte slice.
func (b *Bot) downloadImage(fileID string) ([]byte, error) {
	file, err := b.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("getting file: %w", err)
	}

	resp, err := http.Get(file.Link(b.bot.Token))
	if err != nil {
		return nil, fmt.Errorf("getting http response: %w", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return data, nil
}

// getImagePath checks if an image file already exists locally; if not, it downloads the image from imageURL and saves it locally.
func (b *Bot) getImagePath(imageURL string) (string, error) {
	filename := filepath.Base(imageURL)
	localImagePath := fmt.Sprintf("images/products/%s", filename)

	if _, err := os.Stat(localImagePath); os.IsNotExist(err) {
		// Create the directory if it does not exist
		if err := createDirIfNotExist("images/products"); err != nil {
			return "", fmt.Errorf("creating directory: %w", err)
		}

		// Download the image and save it to the local folder
		resp, err := http.Get(imageURL)
		if err != nil {
			return "", fmt.Errorf("getting http response: %w", err)
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("reading response body: %w", err)
		}

		if err := ioutil.WriteFile(localImagePath, data, 0644); err != nil {
			return "", fmt.Errorf("writing file: %w", err)
		}
	}

	return localImagePath, nil
}

// createDirIfNotExist checks if a directory exists at the provided path; if not, it creates the directory.
func createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}
	return nil
}

// sendProductImage gets a local image path and sends the image to the chat identified by chatID.
func (b *Bot) sendProductImage(chatID int64, imageURL string) {
	localImagePath, err := b.getImagePath(imageURL)
	if err != nil {
		log.Printf("Error getting local image path: %v", err)
		return
	}

	// Send product image
	if _, err := b.bot.Send(tgbotapi.NewPhotoUpload(chatID, localImagePath)); err != nil {
		log.Printf("Error sending image: %v", err)
	}
}
