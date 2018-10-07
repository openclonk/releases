package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

var snapshotPageMutex *sync.Mutex = &sync.Mutex{}

// generateSnapshotPage generates the HTML page for the snapshots.
func generateSnapshotPage() {
	snapshotPageMutex.Lock()
	defer snapshotPageMutex.Unlock()

	// Always read the template to make changes easy.
	tmpl, err := template.ParseFiles("snapshots.tmpl")
	if err != nil {
		fmt.Println("could not generate snapshot page: reading template failed: ", err)
		return
	}

	header, botter, err := getHeader()
	if err != nil {
		fmt.Println("could not generate snapshot page: retrieving header failed: ", err)
		return
	}

	// Don't overwrite the original file until everything is done.
	tmpFilename := cfgHtmlPath + "/snapshots/.index.html"
	file, err := os.Create(tmpFilename)
	if err != nil {
		fmt.Println("could not generate snapshot page: file creation failed: ", err)
		return
	}
	defer file.Close()

	data := struct {
		Header   template.HTML
		Botter   template.HTML
		Branches []SnapshotBranchInfo
	}{
		Header: header,
		Botter: botter,
	}
	data.Branches, err = getSnapshotBranchList()
	if err != nil {
		fmt.Println("could not generate snapshot page: reading snapshots failed: ", err)
		return
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("could not generate snapshot page: template execution failed: ", err)
		return
	}

	err = os.Rename(tmpFilename, cfgHtmlPath+"/snapshots/index.html")
	if err != nil {
		fmt.Println("could not generate snapshot page: moving file failed: ", err)
		return
	}
}

var snapshotVersionInfoMutex *sync.Mutex = &sync.Mutex{}

// generateSnapshotPage generates the HTML page for the snapshots.
func generateSnapshotVersionInfo() {
	snapshotVersionInfoMutex.Lock()
	defer snapshotVersionInfoMutex.Unlock()

	funcMap := template.FuncMap{
		"unixtime": func(str string) (int64, error) {
			t, err := time.Parse(time.RFC3339, str)
			if err != nil {
				return 0, err
			}
			return t.Unix(), nil
		},
	}

	// Always read the template to make changes easy.
	tmpl, err := template.New("snapshotversion.tmpl").Funcs(funcMap).ParseFiles("snapshotversion.tmpl")
	if err != nil {
		fmt.Println("could not generate version info: reading template failed: ", err)
		return
	}

	// Don't overwrite the original file until everything is done.
	tmpFilename := cfgHtmlPath + "/snapshots/.version.txt"
	file, err := os.Create(tmpFilename)
	if err != nil {
		fmt.Println("could not generate version info: file creation failed: ", err)
		return
	}
	defer file.Close()

	data := struct {
		Branches []SnapshotBranchInfo
	}{}
	data.Branches, err = getCompleteSnapshots()
	if err != nil {
		fmt.Println("could not generate version info: reading snapshots failed: ", err)
		return
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("could not generate version info: template execution failed: ", err)
		return
	}

	err = os.Rename(tmpFilename, cfgHtmlPath+"/snapshots/version.txt")
	if err != nil {
		fmt.Println("could not generate version info: moving file failed: ", err)
		return
	}
}

var createLatestLinksMutex *sync.Mutex = &sync.Mutex{}

// createLatestLink creates symlinks for the latest build of each branch.
func createLatestLinks() {
	createLatestLinksMutex.Lock()
	defer createLatestLinksMutex.Unlock()
	snapshots, err := getCompleteSnapshots()
	if err != nil {
		fmt.Println("could not link latest build: reading snapshots failed: ", err)
	}
	// For each branch, find the latest build with all files.
	for _, branchInfo := range snapshots {
		branch := branchInfo.Name
		// Creating relative symlinks is mildly annoying with the Go API, so just call ln instead...
		cmd := exec.Command("ln", "-Tsf", branchInfo.Dir, "latest-"+branch)
		cmd.Dir = cfgBasePath + "/snapshots"
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err = cmd.Run()
		if err != nil {
			fmt.Println("could not link latest build of ", branch, ": ", out.String())
			// Still continue with the others.
		}
	}
}

type SnapshotBranchInfo struct {
	Dir       string
	Name      string
	Date      string
	Revision  string
	LinuxDL   string
	WindowsDL string
}

// IsComplete checks whether there are downloads for all platforms.
func (bi *SnapshotBranchInfo) IsComplete() bool {
	return bi.LinuxDL != "" && bi.WindowsDL != ""
}

var (
	appImageRegexp *regexp.Regexp = regexp.MustCompile(`\.AppImage$`)
	winZipRegexp   *regexp.Regexp = regexp.MustCompile(`win.*\.zip$`)
)

func getSnapshotFiles() (map[string][]SnapshotBranchInfo, error) {
	branches := make(map[string][]SnapshotBranchInfo)
	path := cfgBasePath + "/snapshots"
	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	// Iterate over directories in reverse to have the newest snapshot first.
	for i := range dirs {
		dir := dirs[len(dirs)-1-i]
		if !dir.IsDir() {
			continue
		}
		if m := snapshotDirRegexp.FindStringSubmatch(dir.Name()); m != nil {
			branchName := m[2]
			branchInfo := SnapshotBranchInfo{
				Dir:      dir.Name(),
				Name:     branchName,
				Date:     m[1],
				Revision: m[3],
			}
			files, err := ioutil.ReadDir(path + "/" + dir.Name())
			if err != nil {
				return nil, fmt.Errorf("couldn't read directory %s: %v", dir.Name(), err)
			}
			for _, file := range files {
				filePath := "/snapshots/" + dir.Name() + "/" + file.Name()
				switch {
				case appImageRegexp.MatchString(file.Name()):
					branchInfo.LinuxDL = filePath
				case winZipRegexp.MatchString(file.Name()):
					branchInfo.WindowsDL = filePath
				}
			}
			branches[branchName] = append(branches[branchName], branchInfo)
		}
	}
	return branches, nil
}

// getCompleteSnapshots returns a list of the latest complete (i.e.,
// all platforms done) snapshot infos for each branch.
func getCompleteSnapshots() ([]SnapshotBranchInfo, error) {
	branches, err := getSnapshotFiles()
	if err != nil {
		return nil, err
	}
	// For each branch, find the latest build with all files.
	var snapshots []SnapshotBranchInfo
	for _, branchInfos := range branches {
		for _, branchInfo := range branchInfos {
			if branchInfo.IsComplete() {
				snapshots = append(snapshots, branchInfo)
				break
			}
		}
	}
	return snapshots, nil
}

func getSnapshotBranchList() ([]SnapshotBranchInfo, error) {
	branches, err := getSnapshotFiles()
	if err != nil {
		return nil, err
	}
	// Now sort the map into an array.
	branchArray := []SnapshotBranchInfo{}
	// master always comes first.
	if len(branches["master"]) > 0 {
		branchArray = append(branchArray, branches["master"][0])
		delete(branches, "master")
	}
	// Sort the remaining branches by latest build.
	for len(branches) > 0 {
		maxBranch := ""
		for b, c := range branches {
			if branches[maxBranch] == nil || c[0].Date > branches[maxBranch][0].Date {
				maxBranch = b
			}
		}
		branchArray = append(branchArray, branches[maxBranch][0])
		delete(branches, maxBranch)
	}
	return branchArray, nil
}

// httpGet loads some URL via HTTP GET.
func httpGet(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	content, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// getHeader loads header and footer of the OpenClonk layout.
func getHeader() (template.HTML, template.HTML, error) {
	header, err := httpGet("https://www.openclonk.org/header/header.html")
	if err != nil {
		return template.HTML(""), template.HTML(""), err
	}

	botter, err := httpGet("https://www.openclonk.org/header/botter.html")
	if err != nil {
		return template.HTML(""), template.HTML(""), err
	}
	return template.HTML(header), template.HTML(botter), nil
}
