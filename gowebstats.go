package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	WhitelistedDomains []string `toml:"whitelisted_domains"`
	LogDir             string   `default:"logs" toml:"log_dir"`
	QueueSize          int      `toml:"queue_size"`
	Port               string   `default:":8080" toml:"port"`
}

type RequestInfo struct {
	Time      time.Time `json:"time"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

var (
	configFile string
	config     Config
	logQueue   []RequestInfo
	logMutex   sync.Mutex
)

func main() {
	flag.StringVar(&configFile, "config", "config.toml", "Path to config file")
	flag.Parse()

	// Read config file
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		log.Fatalf("Error creating log directory (%s): %v", config.LogDir, err)
	}

	// Start HTTP server
	fmt.Printf("Starting gowebstats on %s\n", config.Port)
	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(config.Port, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Check if request is whitelisted
	if !isWhitelisted(r.Host) {
		http.NotFound(w, r)
		return
	}

	// Write empty CSS file with correct headers
	w.Header().Set("Content-Type", "text/css")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "")

	// Save request info
	logMutex.Lock()
	logQueue = append(logQueue, RequestInfo{
		Time:      time.Now(),
		IP:        getIP(r),
		UserAgent: r.UserAgent(),
	})
	if len(logQueue) == config.QueueSize {
		writeLog()
		logQueue = nil
	}
	logMutex.Unlock()
}

func isWhitelisted(host string) bool {
	for _, domain := range config.WhitelistedDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func getIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
	}
	return ip
}

func writeLog() {
	// Generate filename with timestamp
	timestamp := time.Now().Format("2006-01-02T15:04:05")
	filename := fmt.Sprintf("%s.json", timestamp)
	filepath := filepath.Join(config.LogDir, filename)

	// Encode log queue to JSON
	data, err := json.Marshal(logQueue)
	if err != nil {
		log.Fatalf("Error encoding log data: %v", err)
	}

	// Write log file
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		log.Fatalf("Error writing log file: %v", err)
	}

	log.Printf("Wrote log file: %s", filename)
}
