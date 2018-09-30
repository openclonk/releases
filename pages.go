package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var snapshotPageMutex *sync.Mutex = &sync.Mutex{}

// generateSnapshotPage generates the HTML page for the snapshots.
func generateSnapshotPage() {
	snapshotPageMutex.Lock()
	defer snapshotPageMutex.Unlock()

	// Always read the tempalte to make changes easy.
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
	tmpFilename := cfgHtmlPath + "/.snapshots.html"
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
	data.Branches, err = getSnapshotBranches()
	if err != nil {
		fmt.Println("could not generate snapshot page: reading snapshots failed: ", err)
		return
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Println("could not generate snapshot page: template execution failed: ", err)
		return
	}

	err = os.Rename(tmpFilename, cfgHtmlPath+"/snapshots.html")
	if err != nil {
		fmt.Println("could not generate snapshot page: moving file failed: ", err)
		return
	}
}

type SnapshotBranchInfo struct {
	Name     string
	Date     string
	Revision string
	LinuxDL  string
}

func getSnapshotBranches() ([]SnapshotBranchInfo, error) {
	branches := make(map[string]*SnapshotBranchInfo)
	var currentBranch *SnapshotBranchInfo = nil
	firstVisit := true
	err := filepath.Walk(cfgBasePath+"/snapshots", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if m := snapshotDirRegexp.FindStringSubmatch(info.Name()); m != nil {
				branchName := m[2]
				currentBranch = &SnapshotBranchInfo{
					Name:     branchName,
					Date:     m[1],
					Revision: m[3],
				}
				branches[branchName] = currentBranch
			} else {
				if firstVisit {
					return nil
				} else {
					return filepath.SkipDir
				}
			}
		} else if currentBranch != nil {
			if r := regexp.MustCompile(`\.AppImage$`); r.MatchString(info.Name()) {
				currentBranch.LinuxDL = path[len(cfgBasePath):]
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Now sort the map into an array.
	branchArray := []SnapshotBranchInfo{}
	// master always comes first.
	if branches["master"] != nil {
		branchArray = append(branchArray, *branches["master"])
		delete(branches, "master")
	}
	// Sort the remaining branches by latest build.
	for len(branches) > 0 {
		minBranch := ""
		for b, c := range branches {
			if branches[minBranch] == nil || c.Date < branches[minBranch].Date {
				minBranch = b
			}
		}
		branchArray = append(branchArray, *branches[minBranch])
		delete(branches, minBranch)
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
