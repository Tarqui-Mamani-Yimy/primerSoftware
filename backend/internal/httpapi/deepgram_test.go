package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type timeoutError struct{}

func (timeoutError) Error() string   { return "provider timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestDeepgramTranscriberReturnsTranscriptWithoutLeakingProviderSecret(t *testing.T) {
	var gotAuth string
	transcriber := NewDeepgramTranscriber(DeepgramConfig{
		APIKey: "server-only-test-secret", Model: "nova-3", Language: "es", BaseURL: "https://deepgram.test/v1",
	})
	transcriber.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		if r.Header.Get("Content-Type") != "audio/webm" {
			t.Errorf("provider content type = %q", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "audio" {
			t.Errorf("provider audio = %q", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"results":{"channels":[{"alternatives":[{"transcript":"crear una clase Usuario"}]}]}}`)),
		}, nil
	})}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/voice/transcriptions", strings.NewReader("audio"))
	request.Header.Set("Content-Type", "audio/webm")
	response := httptest.NewRecorder()

	transcriber.Handle(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Body.String() != `{"text":"crear una clase Usuario"}`+"\n" {
		t.Fatalf("body = %s", response.Body.String())
	}
	if gotAuth != "Token "+transcriber.config.APIKey {
		t.Fatalf("provider authorization = %q", gotAuth)
	}
	if strings.Contains(response.Body.String(), transcriber.config.APIKey) {
		t.Fatal("provider secret leaked in response")
	}
}

func TestDeepgramTranscriberSanitizesProviderTimeout(t *testing.T) {
	const secret = "server-only-test-secret"
	transcriber := NewDeepgramTranscriber(DeepgramConfig{APIKey: secret, BaseURL: "https://deepgram.test/v1"})
	transcriber.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, timeoutError{}
	})}

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("audio"))
	request.Header.Set("Content-Type", "audio/webm")
	response := httptest.NewRecorder()

	transcriber.Handle(response, request)

	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusGatewayTimeout, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) || strings.Contains(response.Body.String(), "provider timeout") {
		t.Fatalf("provider details leaked in response: %s", response.Body.String())
	}
}

func TestDeepgramTranscriberRejectsInvalidInput(t *testing.T) {
	transcriber := NewDeepgramTranscriber(DeepgramConfig{APIKey: "configured"})
	tests := []struct {
		name        string
		contentType string
		body        string
		want        int
	}{
		{name: "unsupported media", contentType: "text/plain", body: "audio", want: http.StatusUnsupportedMediaType},
		{name: "empty audio", contentType: "audio/webm", want: http.StatusBadRequest},
		{name: "missing configuration", contentType: "audio/webm", body: "audio", want: http.StatusBadGateway},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := transcriber
			if test.name == "missing configuration" {
				current = NewDeepgramTranscriber(DeepgramConfig{})
				test.want = http.StatusServiceUnavailable
			}
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			current.Handle(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestDeepgramTranscriberRejectsOversizedAudio(t *testing.T) {
	transcriber := NewDeepgramTranscriber(DeepgramConfig{APIKey: "configured"})
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", int(maxVoiceAudioBytes)+1)))
	request.Header.Set("Content-Type", "audio/webm")
	response := httptest.NewRecorder()

	transcriber.Handle(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}
