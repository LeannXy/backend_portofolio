package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudinaryUploadUsesSignedRequest(t *testing.T) {
	const secret = "test-cloudinary-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			http.Error(w, "invalid method", http.StatusMethodNotAllowed)
			return
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Errorf("expected multipart form data, got %q", r.Header.Get("Content-Type"))
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		values := map[string]string{}
		var image []byte
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("read multipart body: %v", err)
				http.Error(w, "invalid multipart request", http.StatusBadRequest)
				return
			}
			value, err := io.ReadAll(part)
			if err != nil {
				t.Errorf("read multipart field: %v", err)
				http.Error(w, "invalid multipart field", http.StatusBadRequest)
				return
			}
			if part.FormName() == "file" {
				image = value
			} else {
				values[part.FormName()] = string(value)
			}
		}

		signatureParams := map[string]string{
			"folder":    values["folder"],
			"public_id": values["public_id"],
			"timestamp": values["timestamp"],
		}
		if values["api_key"] != "test-api-key" ||
			values["signature"] != uploadSignature(signatureParams, secret) ||
			values["public_id"] != "test-image" ||
			string(image) != "webp-data" {
			t.Errorf("unexpected or invalid signed upload request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"secure_url":"https://res.cloudinary.com/demo/image/upload/portfolio/test-image.webp"}`)
	}))
	defer server.Close()

	cloudinary := &Cloudinary{
		cloudName:  "demo",
		apiKey:     "test-api-key",
		apiSecret:  secret,
		uploadURL:  server.URL,
		httpClient: server.Client(),
	}
	result, err := cloudinary.UploadWebP(context.Background(), "test-image", strings.NewReader("webp-data"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if result != "https://res.cloudinary.com/demo/image/upload/portfolio/test-image.webp" {
		t.Fatalf("unexpected secure URL: %s", result)
	}
}

func TestCloudinaryRejectsNonSecureResponseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"secure_url":"http://res.cloudinary.com/demo/image.webp"}`)
	}))
	defer server.Close()

	cloudinary := &Cloudinary{
		apiKey:     "test-api-key",
		apiSecret:  "test-cloudinary-secret",
		uploadURL:  server.URL,
		httpClient: server.Client(),
	}
	if _, err := cloudinary.UploadWebP(context.Background(), "test-image", strings.NewReader("webp-data")); err == nil {
		t.Fatal("expected insecure image URL to be rejected")
	}
}
