package main

import (
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port        int    `yaml:"port"`
	UploadDir   string `yaml:"uploadDir"`
	AuthToken   string `yaml:"authToken"`
	MaxFileSize int64  `yaml:"maxFileSizeMb"`
}

func loadConfig(path string) (*Config, error) {
	// Sane defaults
	cfg := &Config{
		Port:        8080,
		UploadDir:   "./uploads",
		MaxFileSize: 10,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file %s not found, using defaults", path)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func authMiddleware(expectedToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Auth-Token")
		// Use subtle.ConstantTimeCompare to prevent timing attacks
		if expectedToken == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func uploadHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the multipart form, limiting body size
		maxBytes := cfg.MaxFileSize * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			http.Error(w, "File too large or malformed request", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error retrieving 'file' from form-data", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Secure the filename to prevent directory traversal attacks
		filename := filepath.Base(header.Filename)
		if filename == "." || filename == "/" {
			http.Error(w, "Invalid filename", http.StatusBadRequest)
			return
		}

		dstPath := filepath.Join(cfg.UploadDir, filename)

		// Create the destination file
		dst, err := os.Create(dstPath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy file content
		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Error writing file: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "File %s uploaded successfully\n", filename)
	}
}

func main() {
	configPath := "config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.AuthToken == "" {
		log.Println("WARNING: authToken is empty! Requests will be unauthorized unless configured.")
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", authMiddleware(cfg.AuthToken, uploadHandler(cfg)))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Server starting on %s...", addr)
	log.Printf("Upload directory: %s", cfg.UploadDir)
	log.Printf("Max file size: %d MB", cfg.MaxFileSize)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
