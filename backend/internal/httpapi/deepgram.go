package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxVoiceAudioBytes int64 = 10 << 20

var acceptedVoiceMediaTypes = map[string]bool{
	"audio/mp4":  true,
	"audio/mpeg": true,
	"audio/ogg":  true,
	"audio/wav":  true,
	"audio/webm": true,
}

type DeepgramConfig struct {
	APIKey   string
	Model    string
	Language string
	BaseURL  string
}

type DeepgramTranscriber struct {
	config DeepgramConfig
	client *http.Client
}

func NewDeepgramTranscriber(config DeepgramConfig) *DeepgramTranscriber {
	return &DeepgramTranscriber{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *DeepgramTranscriber) Handle(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(d.config.APIKey) == "" {
		writeError(w, http.StatusServiceUnavailable, "Voice transcription is not configured")
		return
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]))
	if !acceptedVoiceMediaTypes[mediaType] {
		writeError(w, http.StatusUnsupportedMediaType, "Unsupported audio type")
		return
	}
	if r.ContentLength > maxVoiceAudioBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Audio exceeds 10 MB limit")
		return
	}
	audio, err := io.ReadAll(io.LimitReader(r.Body, maxVoiceAudioBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Audio could not be read")
		return
	}
	if len(audio) == 0 {
		writeError(w, http.StatusBadRequest, "Audio is required")
		return
	}
	if int64(len(audio)) > maxVoiceAudioBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Audio exceeds 10 MB limit")
		return
	}
	text, err := d.transcribe(r.Context(), mediaType, audio)
	if err != nil {
		var timeoutErr interface{ Timeout() bool }
		if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
			writeError(w, http.StatusGatewayTimeout, "Transcription provider timed out")
			return
		}
		var providerErr deepgramProviderError
		if errors.As(err, &providerErr) && providerErr.status >= 400 && providerErr.status < 500 {
			writeError(w, http.StatusBadGateway, "Transcription provider rejected the audio")
			return
		}
		writeError(w, http.StatusBadGateway, "Transcription provider unavailable")
		return
	}
	if strings.TrimSpace(text) == "" {
		writeError(w, http.StatusBadGateway, "Invalid transcription provider response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"text": text})
}

func (d *DeepgramTranscriber) transcribe(ctx context.Context, mediaType string, audio []byte) (string, error) {
	endpoint, err := url.Parse(strings.TrimRight(d.config.BaseURL, "/") + "/listen")
	if err != nil {
		return "", err
	}
	query := endpoint.Query()
	query.Set("model", d.config.Model)
	query.Set("language", d.config.Language)
	query.Set("smart_format", "true")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(audio))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Token "+d.config.APIKey)
	req.Header.Set("Content-Type", mediaType)
	response, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", deepgramProviderError{status: response.StatusCode}
	}
	var result struct {
		Results struct {
			Channels []struct {
				Alternatives []struct {
					Transcript string `json:"transcript"`
				} `json:"alternatives"`
			} `json:"channels"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Results.Channels) == 0 || len(result.Results.Channels[0].Alternatives) == 0 {
		return "", errors.New("missing transcript")
	}
	return strings.TrimSpace(result.Results.Channels[0].Alternatives[0].Transcript), nil
}

type deepgramProviderError struct{ status int }

func (e deepgramProviderError) Error() string { return "Deepgram request failed" }
