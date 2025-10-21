package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
)

// ✅ Embed everything in *this same directory and its subfolders*
//go:embed **/*
var files embed.FS

func main() {
	fmt.Println("Listing embedded asset files:")
	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(" -", path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

