package main

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
)

var snapshotDirRegexpString = `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)-([^/]+)-([a-f0-9]+)`
var snapshotDirRegexp = regexp.MustCompile(`^` + snapshotDirRegexpString + `$`)
var snapshotPathRegexp = regexp.MustCompile(`^/snapshots/` + snapshotDirRegexpString + `/[^/]+$`)

func handleSnapshots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "HEAD":
		// Let nginx handle the download.
		w.Header().Set("X-Accel-Redirect", "/files/"+r.URL.Path)
		w.Write(nil)
	case "POST":
		// Handle upload.
		// Check authorization: only a single password is allowed.
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Basic "+base64.StdEncoding.EncodeToString([]byte(cfgUploadPassword)))) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="OC Releases", charset="UTF-8"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Validate that the path matches the following schema:
		//   POST /snapshots/<time>-<branch>-<commit>/<filename>
		fmt.Println(r.URL.EscapedPath())
		if !snapshotPathRegexp.MatchString(r.URL.EscapedPath()) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Write the file.
		// TODO: Try not to overwrite files?
		filePath := cfgBasePath + "/" + r.URL.EscapedPath()
		if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("couldn't create directories: ", err)
			return
		}
		file, err := os.Create(filePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("couldn't create file: ", err)
			return
		}
		defer file.Close()
		_, err = io.Copy(file, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("couldn't write file: ", err)
			return
		}
		// Update download page.
		updateStaticFiles()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
