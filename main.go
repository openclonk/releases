package main

import (
	"fmt"
	"net/http"
	"os"
)

var cfgBasePath string = checkEnvUnset("OC_REL_BASE_PATH")
var cfgUploadPassword string = checkEnvUnset("OC_REL_UPLOAD_PASSWORD")
var cfgPort string = checkEnvUnset("PORT")

func main() {
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
