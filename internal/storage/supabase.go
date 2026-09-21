package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type SupabaseStorage struct {
	BaseURL string
	APIKey  string
	Bucket  string
	Client  *http.Client
}

func NewSupabaseStorage(baseURL, apiKey, bucket string) *SupabaseStorage {
	return &SupabaseStorage{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Bucket:  bucket,
		Client:  &http.Client{},
	}
}

func (s *SupabaseStorage) Upload(
	ctx context.Context,
	key string,
	body io.Reader,
	contentType string,
) error {
	url := fmt.Sprintf(
		"%s/storage/v1/object/%s/%s",
		s.BaseURL,
		s.Bucket,
		key,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		body,
	)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("apikey", s.APIKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"supabase upload failed: status=%d body=%s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	return nil
}

func (s *SupabaseStorage) Download(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	url := fmt.Sprintf(
		"%s/storage/v1/object/%s/%s",
		s.BaseURL,
		s.Bucket,
		key,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("apikey", s.APIKey)

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		responseBody, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf(
			"supabase download failed: status=%d body=%s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	return resp.Body, nil
}

func (s *SupabaseStorage) Delete(
	ctx context.Context,
	key string,
) error {
	url := fmt.Sprintf(
		"%s/storage/v1/object/%s/%s",
		s.BaseURL,
		s.Bucket,
		key,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		url,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("apikey", s.APIKey)

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"supabase delete failed: status=%d body=%s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	return nil
}
