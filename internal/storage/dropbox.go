package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

const (
	dropboxDownloadURL = "https://content.dropboxapi.com/2/files/download"
	dropboxUploadURL   = "https://content.dropboxapi.com/2/files/upload"
)

// Dropbox is an alternate Backend that stores files in a user's Dropbox
// account via the Dropbox API v2, using plain net/http (no SDK dependency).
type Dropbox struct {
	token  string
	root   string // e.g. "/Anchor"
	client *http.Client
}

// NewDropbox returns a Dropbox backend rooted at root (e.g. "/Anchor") in
// the account owned by token.
func NewDropbox(token, root string) *Dropbox {
	return &Dropbox{
		token:  token,
		root:   strings.TrimSuffix(root, "/"),
		client: http.DefaultClient,
	}
}

func (d *Dropbox) fullPath(p string) string {
	return d.root + "/" + strings.TrimPrefix(path.Clean("/"+p), "/")
}

func (d *Dropbox) ReadFile(p string) ([]byte, error) {
	arg, err := json.Marshal(map[string]string{"path": d.fullPath(p)})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, dropboxDownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Dropbox-API-Arg", string(arg))

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// 409 means the file/path doesn't exist.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dropbox download %s: %s: %s", p, resp.Status, body)
	}

	return io.ReadAll(resp.Body)
}

func (d *Dropbox) WriteFile(p string, data []byte) error {
	arg, err := json.Marshal(map[string]any{
		"path": d.fullPath(p),
		"mode": "overwrite",
		"mute": true,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, dropboxUploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Dropbox-API-Arg", string(arg))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dropbox upload %s: %s: %s", p, resp.Status, body)
	}
	return nil
}
