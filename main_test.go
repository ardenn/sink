package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default config when file doesn't exist
	cfg, err := loadConfig("nonexistent.yaml")
	if err != nil {
		t.Fatalf("Expected no error when config file is missing, got: %v", err)
	}
	if cfg.Port != 8080 || cfg.UploadDir != "./uploads" || cfg.MaxFileSize != 10 {
		t.Errorf("Expected default values, got: %+v", cfg)
	}

	// Test with a valid config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	yamlContent := []byte(`
port: 9090
uploadDir: "/tmp/uploads"
authToken: "mysecret"
maxFileSizeMb: 20
`)
	if _, err := tmpFile.Write(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp config file: %v", err)
	}
	tmpFile.Close()

	cfg, err = loadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load valid config: %v", err)
	}
	if cfg.Port != 9090 || cfg.UploadDir != "/tmp/uploads" || cfg.AuthToken != "mysecret" || cfg.MaxFileSize != 20 {
		t.Errorf("Config values did not match yaml content: %+v", cfg)
	}
}

func TestAuthMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test without token configured (should require empty or not check, actually our logic says if expectedToken == "" it fails unless they provide "" which isn't possible from header, wait, logic in main.go:
	// if expectedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
	// It means if expectedToken is empty, it always rejects? Let's check main.go logic.
	// In main.go:
	// if expectedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1
	// Yes, if expectedToken is "", it always returns Unauthorized. Let's test this behavior.

	tests := []struct {
		name           string
		expectedToken  string
		requestToken   string
		expectedStatus int
	}{
		{"Valid Token", "secret", "secret", http.StatusOK},
		{"Invalid Token", "secret", "wrong", http.StatusUnauthorized},
		{"Empty Expected Token", "", "anything", http.StatusUnauthorized},
		{"Empty Request Token", "secret", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := authMiddleware(tt.expectedToken, nextHandler)
			req := httptest.NewRequest("GET", "/", nil)
			if tt.requestToken != "" {
				req.Header.Set("X-Auth-Token", tt.requestToken)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestUploadHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uploads-*")
	if err != nil {
		t.Fatalf("Failed to create temp upload dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		UploadDir:   tmpDir,
		MaxFileSize: 1, // 1MB for tests
	}
	handler := uploadHandler(cfg)

	t.Run("Invalid Method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/upload", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}
	})

	t.Run("Valid Upload", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := io.Copy(fw, strings.NewReader("hello world")); err != nil {
			t.Fatalf("Failed to copy file content: %v", err)
		}
		w.Close()

		req := httptest.NewRequest("POST", "/upload", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		// Verify file was written
		content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
		if err != nil {
			t.Fatalf("Failed to read uploaded file: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("Expected file content 'hello world', got '%s'", string(content))
		}
	})

	t.Run("Directory Traversal Attempt", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("file", "../../../etc/passwd")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := io.Copy(fw, strings.NewReader("malicious content")); err != nil {
			t.Fatalf("Failed to copy file content: %v", err)
		}
		w.Close()

		req := httptest.NewRequest("POST", "/upload", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			// Should succeed but write to `passwd` in the tmpDir due to filepath.Base
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		// Verify it was written securely as just "passwd"
		_, err = os.Stat(filepath.Join(tmpDir, "passwd"))
		if os.IsNotExist(err) {
			t.Errorf("Expected file 'passwd' to be created safely, but it wasn't")
		}
	})

	t.Run("Missing File Field", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("wrongfield", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		io.Copy(fw, strings.NewReader("hello"))
		w.Close()

		req := httptest.NewRequest("POST", "/upload", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for missing file field, got %d", http.StatusBadRequest, rr.Code)
		}
	})
}
