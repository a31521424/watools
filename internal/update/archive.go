package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"watools/pkg/utils"
)

// Wails release bundles use regular files and directories. Reject special files,
// duplicate entries and oversized archives before an extractor touches disk.
func extractUpdateArchive(source, dest string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) > 50000 {
		return fmt.Errorf("update archive has too many files")
	}
	var total uint64
	for _, entry := range archive.File {
		if entry.Mode()&os.ModeType != 0 && !entry.FileInfo().IsDir() {
			return fmt.Errorf("update archive contains a special file")
		}
		if entry.UncompressedSize64 > 512<<20 || total+entry.UncompressedSize64 > 2<<30 {
			return fmt.Errorf("update archive exceeds extraction limits")
		}
		total += entry.UncompressedSize64
		target, err := utils.ResolvePathWithinBase(dest, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, entry.Mode().Perm()&0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, entry.Mode().Perm()&0777)
		if err != nil {
			return err
		}
		reader, err := entry.Open()
		if err != nil {
			file.Close()
			return err
		}
		written, copyErr := io.Copy(file, io.LimitReader(reader, 512<<20+1))
		readErr := reader.Close()
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if uint64(written) != entry.UncompressedSize64 {
			return fmt.Errorf("invalid expanded update size")
		}
	}
	return nil
}
