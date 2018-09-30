package main

import (
	"fmt"
	"net/http"
	"os"
)

// Path where the snapshot files go.
var cfgBasePath string = checkEnvUnset("OC_REL_BASE_PATH")

// Path where the HTML pages go.
var cfgHtmlPath string = checkEnvUnset("OC_REL_HTML_PATH")

// Authentication data in the format <username>:<password> (basic auth)
var cfgUploadPassword string = checkEnvUnset("OC_REL_UPLOAD_PASSWORD")

// Port to listen on.
var cfgPort string = checkEnvUnset("PORT")

func main() {
	updateStaticFiles()

	http.HandleFunc("/snapshots/", handleSnapshots)

	fmt.Println("listening on", cfgPort)
	err := http.ListenAndServe(cfgPort, nil)
	if err != nil {
		fmt.Println("listen failed:", err)
		os.Exit(2)
	}
}

func checkEnvUnset(env string) string {
	s := os.Getenv(env)
	if s == "" {
		fmt.Println("error:", env, "required but not set")
		os.Exit(1)
	}
	return s
}

func updateStaticFiles() {
	generateSnapshotPage()
	createLatestLinks()
}
