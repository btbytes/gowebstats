package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
)

type Config struct {
	WhitelistedDomains []string `toml:"whitelisted_domains"`
	LogDir             string   `toml:"log_dir"`
	LogQueueSize       int      `toml:"log_queue_size"`
	Port               int      `toml:"port"`
}

type RequestInfo struct {
	Time      int32  `parquet:"name=time, type=INT32, convertedtype=DATE"` // human // change type to int32
	IP        string `parquet:"name=ip, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	UserAgent string `parquet:"name=user_agent, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
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
		log.Fatalf("Error creating log directory: %v", err)
	}

	// Start HTTP server
	log.Printf("Starting gowebstats on %d", config.Port)                                                  // human
	log.Printf("Will write parquet files to %s dir after %d entries", config.LogDir, config.LogQueueSize) // human
	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil); err != nil {
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
		Time:      int32(time.Now().Unix() / 3600 / 24),
		IP:        getIP(r),
		UserAgent: r.UserAgent(),
	})
	if len(logQueue) == config.LogQueueSize {
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
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s.parquet", timestamp)
	filepath := filepath.Join(config.LogDir, filename)

	// Create Parquet file writer
	fw, err := local.NewLocalFileWriter(filepath)
	if err != nil {
		log.Fatalf("Error creating Parquet file: %v", err)
	}

	// Create Parquet file writer options
	pw, err := writer.NewParquetWriter(fw, new(RequestInfo), 4)
	if err != nil {
		log.Fatalf("Error creating Parquet writer: %v", err)
	}
	pw.CompressionType = parquet.CompressionCodec_SNAPPY // human
	// defer pw.WriteStop() // human
	for i := 0; i < len(logQueue); i++ { // human
		ri := logQueue[i]                   // human
		if err = pw.Write(ri); err != nil { // human
			log.Println("Parquet Write error", err) // human
		} // human
	}
	if err = pw.WriteStop(); err != nil { // human
		log.Println("WriteStop error -- ", err) // human
		return                                  // human
	} // human
	fw.Close() // human
	log.Printf("Wrote log file: %s", filename)
}
