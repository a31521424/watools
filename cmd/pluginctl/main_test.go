package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOfficialUIPackagesContainSharedDesignSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	theme, err := os.ReadFile("frontend/public/watools-ui.css")
	if err != nil {
		t.Fatal(err)
	}
	plugins, err := discoverOfficialPlugins()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, plugin := range plugins {
		path := filepath.Join(t.TempDir(), "plugin.wt")
		if err := packagePlugin(plugin.PluginDir, path); err != nil {
			t.Fatal(err)
		}
		archive, err := zip.OpenReader(path)
		if err != nil {
			t.Fatal(err)
		}
		var html, packaged []byte
		for _, entry := range archive.File {
			if entry.Name != "index.html" && entry.Name != "watools-ui.css" {
				continue
			}
			reader, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
			if entry.Name == "index.html" {
				html = content
			} else {
				packaged = content
			}
		}
		archive.Close()
		if html != nil {
			count++
			if !bytes.Contains(html, []byte(`href="watools-ui.css"`)) || !bytes.Equal(packaged, theme) {
				t.Fatalf("%s is not styled by the shared packaged source", plugin.PackageID)
			}
		} else if packaged != nil {
			t.Fatal("executable plugin contains unused UI assets")
		}
	}
	if count != 5 {
		t.Fatalf("expected five official UI plugins, got %d", count)
	}
}
