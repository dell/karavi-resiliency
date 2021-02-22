/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

//TAGSIZE standard size for a pod tab
const TAGSIZE = 16

var rootDir = "/"
var enableDoExit bool
var blockFiles map[string]*os.File

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
	readExistingEntries(rootDir)
	for i := 0; ; i++ {
		makeEntry(string(podTag), rootDir, i)
		time.Sleep(5 * time.Second)
	}
}

func readExistingEntries(rootDir string) {
	var timeSamples int
	var prevTime time.Time
	var computeTimeDelta bool
	var key string
	entries, err := ioutil.ReadDir(rootDir)
	if err != nil {
		fmt.Printf("Couldn't read %s\n", rootDir)
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "data") {
			f, err := os.OpenFile(rootDir+"/"+entry.Name()+"/log", os.O_RDONLY, 0644)
			if err != nil {
				fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				continue
			}
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				//fmt.Printf("line: %s\n", line)
				if line == "" {
					key = ""
					computeTimeDelta = true
					continue
				}
				parts := strings.SplitN(line, " ", 2)
				if key == "" {
					key = parts[0]
				}
				if key != parts[0] {
					fmt.Printf("ERROR: mixed keys %s and %s\n", key, parts[0])
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
			f.Close()
			return
		}
	}
}

var counter int

func makeEntry(podTag, rootDir string, index int) {
	tag := fmt.Sprintf("%x %s\n", podTag, time.Now().Format(time.Stamp))
	entries, err := ioutil.ReadDir(rootDir)
	if err != nil {
		fmt.Printf("Couldn't read %s\n", rootDir)
		return
	}
	logged := false
	doExit := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "data") {
			f, err := os.OpenFile(rootDir+"/"+entry.Name()+"/log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				doExit = true
				continue
			}
			if index == 0 {
				f.WriteString("\n")
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
			f.Close()
			if !logged {
				if (counter % 10) == 0 {
					fmt.Printf(tag)
					logged = true
				}
			}
		}
		if strings.HasPrefix(entry.Name(), "blockdata") {
			var f *os.File
			if index == 0 {
				f, err = os.OpenFile(rootDir+"/"+entry.Name(), os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("Couldn't open %s %s\n", entry.Name(), err.Error())
				}
				blockFiles[entry.Name()] = f
			} else {
				f = blockFiles[entry.Name()]
			}
			f.WriteString(tag)
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
