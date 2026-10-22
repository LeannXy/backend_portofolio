package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var cloudNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Cloudinary struct {
	cloudName  string
	apiKey     string
	apiSecret  string
	uploadURL  string
	httpClient *http.Client
}

func NewCloudinaryFromEnv() (*Cloudinary, error) {
	cloudName := strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME"))
	apiKey := strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY"))
	apiSecret := strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET"))

	if cloudName == "" && apiKey == "" && apiSecret == "" {
		return nil, nil
	}
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, and CLOUDINARY_API_SECRET must all be set")
	}
	if !cloudNamePattern.MatchString(cloudName) {
		return nil, fmt.Errorf("CLOUDINARY_CLOUD_NAME contains unsupported characters")
	}

	return &Cloudinary{
		cloudName:  cloudName,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		uploadURL:  "https://api.cloudinary.com/v1_1/" + url.PathEscape(cloudName) + "/image/upload",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (cloudinary *Cloudinary) UploadWebP(ctx context.Context, publicID string, source io.Reader) (string, error) {
	if cloudinary == nil {
		return "", fmt.Errorf("Cloudinary is not configured")
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	parameters := map[string]string{
		"folder":    "portfolio",
		"public_id": publicID,
		"timestamp": timestamp,
	}

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	for key, value := range map[string]string{
		"api_key":   cloudinary.apiKey,
		"folder":    parameters["folder"],
		"public_id": parameters["public_id"],
		"timestamp": timestamp,
		"signature": uploadSignature(parameters, cloudinary.apiSecret),
	} {
		if err := form.WriteField(key, value); err != nil {
			return "", fmt.Errorf("write Cloudinary upload field: %w", err)
		}
	}
	imagePart, err := form.CreateFormFile("file", publicID+".webp")
	if err != nil {
		return "", fmt.Errorf("create Cloudinary image field: %w", err)
	}
	if _, err := io.Copy(imagePart, source); err != nil {
		return "", fmt.Errorf("copy image to Cloudinary request: %w", err)
	}
	if err := form.Close(); err != nil {
		return "", fmt.Errorf("finish Cloudinary upload request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, cloudinary.uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("create Cloudinary upload request: %w", err)
	}
	request.Header.Set("Content-Type", form.FormDataContentType())

	response, err := cloudinary.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("send Cloudinary upload request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read Cloudinary upload response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Cloudinary upload returned HTTP %d", response.StatusCode)
	}

	var result struct {
		SecureURL string `json:"secure_url"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode Cloudinary upload response: %w", err)
	}
	parsedURL, err := url.Parse(result.SecureURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host != "res.cloudinary.com" {
		return "", fmt.Errorf("Cloudinary returned an invalid secure image URL")
	}
	return result.SecureURL, nil
}

func uploadSignature(parameters map[string]string, secret string) string {
	keys := make([]string, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+parameters[key])
	}
	digest := sha1.Sum([]byte(strings.Join(values, "&") + secret))
	return hex.EncodeToString(digest[:])
}
