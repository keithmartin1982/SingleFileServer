package main

import (
	"crypto/sha256"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var (
	filename string
	port     string
	//go:embed root.html
	rootHtml string
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%7.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func hashFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", errors.New("can't open file" + err.Error())
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", errors.New("can't hash file" + err.Error())
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func fileInfo(filename string) (fileSize int64, fileHash string, err error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, "", errors.New("file not found" + err.Error())
	}
	fileSize = info.Size()
	if fileHash, err = hashFile(filename); err != nil {
		return 0, "", errors.New("can't hash file" + err.Error())
	}
	return
}

func lastIndex(in string) string {
	var fnsa []string
	if runtime.GOOS == "windows" {
		fnsa = strings.Split(in, "\\")
	} else {
		fnsa = strings.Split(in, "/")
	}
	return fnsa[len(fnsa)-1]
}

func main() {
	flag.StringVar(&port, "p", "8080", "port")
	flag.StringVar(&filename, "f", "", "filename")
	flag.Parse()
	if len(filename) < 5 {
		flag.PrintDefaults()
		os.Exit(2)
	}
	fmt.Print("hashing file...")
	fileSizeInt, fileHash, err := fileInfo(filename)
	if err != nil {
		log.Printf("%v\n", err)
		os.Exit(2)
	}
	fmt.Println("done")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("root").Parse(rootHtml)
		if err != nil {
			log.Printf("failed to parse template")
		}
		if err = t.Execute(w, struct {
			Filename string
			Size     string
			Hash     string
		}{
			Filename: lastIndex(filename),
			Size:     formatBytes(fileSizeInt),
			Hash:     fileHash,
		}); err != nil {
			panic(err)
		}
	})
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("serving file to %s\n", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", strconv.FormatInt(fileSizeInt, 10))
		http.ServeFile(w, r, filename)
	})
	fmt.Printf("serving @ http://127.0.0.1:%s/\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Printf("server error: %v\n", err)
	}
}
