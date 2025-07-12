package config

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Project struct {
	Schema  string `json:"$schema"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

var ProjectInfo Project

func ParseProject(jsonFile []byte) {
	err := json.Unmarshal(jsonFile, &ProjectInfo)
	if err != nil {
		panic(err)
	}
	if ProjectInfo.Name == "" {
		panic("project name is empty")
	}
}

func ProjectName() string {
	return ProjectInfo.Name
}

func ProjectAuthor() string {
	return ProjectInfo.Author.Name
}

func ProjectVersion() string {
	return ProjectInfo.Version
}

func ProjectCacheDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Printf("Failed to get user cache dir: %v", err)
		panic(err)
	}
	cacheDir := filepath.Join(userCacheDir, ProjectName())
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Printf("Failed to get user cache dir: %v", err)
		panic(err)
	}
	return cacheDir
}
