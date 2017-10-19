/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/metadataBackend/bolt"
	"github.com/root-gg/plik/server/metadataBackend/file"
)

// This script migrate upload metadata from file backend to Bolt backend
//
// go run file2bolt.go --directory ../files --db ../plik.db
// [02/01/2016 22:00:48][INFO][bolt.go:164 Create] Upload metadata successfully saved
// [02/01/2016 22:00:48][INFO][bolt.go:164 Create] Upload metadata successfully saved
// 2 upload imported
//
// Some .config "no such file or directory" errors are normal if you already switched to Bolt metadata backend
// while using the file data backend as it will create upload directories but not .config files.

func main() {
	// Parse command line arguments
	var directoryPath = flag.String("directory", "../files", "File metadatabackend base path")
	var dbPath = flag.String("db", "../plik.db", "Bold db path")
	flag.Parse()

	if *directoryPath == "" || *dbPath == "" {
		fmt.Println("usage : file2bolt --directory path --db path")
		os.Exit(1)
	}

	// Initialize File metadata backend
	fileConfig := map[string]interface{}{"Directory": *directoryPath}
	fmb := file.NewFileMetadataBackend(fileConfig)

	// Initialize Bolt metadata backend
	boltConfig := map[string]interface{}{"Path": *dbPath}
	bmb := bolt.NewBoltMetadataBackend(boltConfig)

	counter := 0

	// upload ids are the name of the second level of directories of the file metadata backend
	dirs1, err := ioutil.ReadDir(*directoryPath)
	if err != nil {
		fmt.Printf("Unable to open directory %s : %s\n", *directoryPath, err)
		os.Exit(1)
	}
	for _, dir1 := range dirs1 {
		if dir1.IsDir() {
			path := *directoryPath + "/" + dir1.Name()
			dirs2, err := ioutil.ReadDir(path)
			if err != nil {
				fmt.Printf("Unable to open directory %s : %s\n", path, err)
				os.Exit(1)
			}
			for _, dir2 := range dirs2 {
				if dir2.IsDir() {
					uploadID := dir2.Name()

					// Load upload from file metadata backend
					upload, err := fmb.Get(juliet.NewContext(), uploadID)
					if err != nil {
						fmt.Printf("Unable to load upload %s : %s\n", uploadID, err)
						continue
					}

					// Save upload to bolt metadata backend
					err = bmb.Create(juliet.NewContext(), upload)
					if err != nil {
						fmt.Printf("Unable to save upload %s : %s\n", uploadID, err)
						continue
					}

					counter++
				}
			}
		}
	}

	fmt.Printf("%d upload imported\n", counter)
}
