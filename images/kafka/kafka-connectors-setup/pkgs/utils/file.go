package utils

import (
	"io"
	"net/http"
	"os"
	"path"

	archiver "github.com/mholt/archiver/v3"
)

// DownloadFile Download files to directory
func (u *UtilsImpl) DownloadFile(downloadDirectory, url string) (string, error) {
	filename := path.Base(url)
	filepath := path.Join(downloadDirectory, filename)

	resp, err := http.Get(url)
	if err != nil {
		return filename, err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return filename, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return filename, err
}

// ExtractFile Extract archives using p7zip
func (u *UtilsImpl) ExtractFile(filepath, destination string) error {
	return archiver.Unarchive(filepath, destination)
}
