/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TAGSIZE standard size for a pod tab
const TAGSIZE = 16

// InitialPod is the prefix for the initial-pod tag line
const InitialPod = "initial-pod"

var (
	rootDir      = "/"
	enableDoExit bool
	blockFiles   map[string]*os.File
)

func main() {
	var err error
	fmt.Printf("hello world\n")
	flag.BoolVar(&enableDoExit, "doexit", false, "enables exit if I/O error")
	flag.Parse()
	blockFiles = make(map[string]*os.File)
	podTag := make([]byte, TAGSIZE)
	_, err = rand.Read(podTag)
	if err != nil {
		fmt.Printf("Couldn't generate podTag: %s", err.Error())
	}
	rootDir := os.Getenv("ROOT_DIR")
	initialPod := readExistingEntries(rootDir)
	fmt.Printf("initialPod: %t\n", initialPod)
	for i := 0; ; i++ {
		makeEntry(string(podTag), rootDir, i, initialPod)
	}
}

// Returns true if initial pod instance
func readExistingEntries(rootDir string) bool {
	var timeSamples int
	var prevTime time.Time
	var computeTimeDelta bool
	var key string
	printed := make(map[string]bool)
	reportedOtherKeys := make(map[string]bool)
	initialPod := true

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		fmt.Printf("Couldn't read %s\n", rootDir)
		return true
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "data") {
			f, err := os.OpenFile(filepath.Clean(rootDir+"/"+entry.Name()+"/log"), os.O_RDONLY, 0600)
			if err != nil {
				fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				continue
			}
			initialPod := false
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				// fmt.Printf("line: %s\n", line)
				if line == "" {
					key = ""
					computeTimeDelta = true
					continue
				}
				if strings.HasPrefix(line, InitialPod) {
					fmt.Printf("%s\n", line)
					continue
				}
				parts := strings.SplitN(line, " ", 2)
				if key == "" {
					key = parts[0]
				}
				if key != parts[0] && !reportedOtherKeys[parts[0]] {
					fmt.Printf("mixed keys (could be due to replicas on same node): %s and %s\n", key, parts[0])
					reportedOtherKeys[parts[0]] = true
				}
				if !printed[key] {
					fmt.Printf("%s\n", line)
					printed[key] = true
				}
				if len(parts) < 2 {
					// Should have a pod id and a time as separate parts
					continue
				}
				time, err := time.Parse(time.Stamp, parts[1])
				if err != nil {
					fmt.Printf("ERROR: could not parse time %s\n", parts[1])
					continue
				}
				if computeTimeDelta && !prevTime.IsZero() && len(parts) > 0 {
					timeSamples = timeSamples + 1
					delta := time.Sub(prevTime)
					fmt.Printf("%s: delta time seconds %s\t%d\t%.0f\n", parts[1], key, timeSamples, delta.Seconds())
					computeTimeDelta = false
					prevTime = time
				}
				prevTime = time
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("ERROR scannning %s\n", entry.Name())
			}
			err = f.Close()
			if err != nil {
				fmt.Printf("closing file %s: %v", f.Name(), err)
			}
			return initialPod
		}
	}
	return initialPod
}

var counter int

func makeEntry(podTag, rootDir string, index int, initialPod bool) {
	tag := fmt.Sprintf("%x %s\n", podTag, time.Now().Format(time.Stamp))
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		fmt.Printf("Couldn't read %s\n", rootDir)
		return
	}
	logged := false
	doExit := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "data") {
			f, err := os.OpenFile(filepath.Clean(rootDir+"/"+entry.Name()+"/log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				doExit = true
				continue
			}
			if index == 0 {
				if initialPod {
					_, err := f.WriteString(InitialPod + " " + tag)
					if err != nil {
						fmt.Printf("writing to %s: %v", f.Name(), err)
					}
					fmt.Printf("%s %s\n", InitialPod, tag)
				}
				_, err := f.WriteString("\n")
				if err != nil {
					fmt.Printf("writing to %s :%v", f.Name(), err)
				}
			}
			_, err = f.WriteString(tag)
			if err != nil {
				doExit = true
				fmt.Printf("Couldn't write %s %s", entry.Name(), err.Error())
			}
			err = f.Sync()
			if err != nil {
				doExit = true
				fmt.Printf("Couldn't sync %s %s", entry.Name(), err.Error())
			}
			err = f.Close()
			if err != nil {
				fmt.Printf("closing file %s: %v", f.Name(), err)
			}
			if !logged {
				if (counter % 10) == 0 {
					fmt.Print(tag)
					logged = true
				}
			}
		}
		if strings.HasPrefix(entry.Name(), "blockdata") {
			var f *os.File
			if index == 0 {
				f, err = os.OpenFile(filepath.Clean(rootDir+"/"+entry.Name()), os.O_WRONLY, 0600)
				if err != nil {
					fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				}
				blockFiles[entry.Name()] = f
			} else {
				f = blockFiles[entry.Name()]
			}
			_, err := f.WriteString(tag)
			if err != nil {
				fmt.Printf("couldn't write %s: %v\n", tag, err)
			}
			err = f.Sync()
			if err != nil {
				doExit = true
				fmt.Printf("Couldn't sync %s %s\n", entry.Name(), err.Error())
			}
		}
	}
	if enableDoExit && doExit {
		fmt.Printf("Exiting due to I/O error\n")
		os.Exit(2)
	}
	counter++
}
