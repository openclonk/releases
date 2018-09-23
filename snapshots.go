package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
)

var snapshotPathRegexp = regexp.MustCompile(`^/snapshots/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z-[^/]+-[a-f0-9]+/[^/]+$`)

func handleSnapshots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "HEAD":
		// Let nginx handle the download.
		w.Header().Set("X-Accel-Redirect", "/files/"+r.URL.Path)
		w.Write(nil)
	case "POST":
		// Handle upload.
		// Check authorization: only a single password is allowed.
		if r.Header.Get("Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(cfgUploadPassword)) {
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
		generateSnapshotPage()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
