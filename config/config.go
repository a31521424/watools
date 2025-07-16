package config

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"os"
	"path/filepath"
	"sync"
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
var (
	isDevMode bool
	devOnce   sync.Once
)

var (
	wailsCtx context.Context
	ctxOnce  sync.Once
)

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

func InitWailsContext(ctx context.Context) {
	ctxOnce.Do(func() {
		wailsCtx = ctx
	})
}

func GetWailsContext() context.Context {
	return wailsCtx
}

func InitDevMode() {
	if wailsCtx == nil {
		log.Println("wails context is nil")
		return
	}
	devOnce.Do(func() {
		info := runtime.Environment(wailsCtx)
		isDevMode = info.BuildType == "dev"
	})
}

func IsDevMode() bool {
	return isDevMode
}
