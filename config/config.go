package config

import (
	_ "embed"
	"encoding/json"
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
