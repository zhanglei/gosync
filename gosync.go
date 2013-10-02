// Copyright 2013 Unknown
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// gosync is a tool for syncing files and directories in different hosts.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"
	"github.com/howeyc/fsnotify"
)

var (
	cfg         *goconfig.ConfigFile
	srcPath     string
	watchPath   string
	receivePath string
	hosts       []string
)

func checkConfig() bool {
	var err error
	srcPath, err = com.GetSrcPath("github.com/Unknwon/gosync")
	if err != nil {
		com.ColorLog("[ERRO] Cannot find source path\n")
		return false
	}

	cfg, err = goconfig.LoadConfigFile(srcPath + "conf/app.ini")
	if err != nil {
		com.ColorLog("[ERRO] Fail to load ( conf/app.ini )[ %s ]\n", err)
		return false
	}

	watchPath = cfg.MustValue("setting", "watch_path")
	if len(watchPath) == 0 {
		com.ColorLog("[ERRO] Invalid path in 'app.ini' key 'watch_path'\n")
		return false
	}

	receivePath = cfg.MustValue("setting", "receive_path")
	if len(receivePath) == 0 {
		com.ColorLog("[ERRO] Invalid path in 'app.ini' key 'hosts'\n")
		return false
	}

	hosts = strings.Split(cfg.MustValue("setting", "hosts"), "|")
	if len(hosts) == 0 {
		com.ColorLog("[ERRO] No valid host in 'app.ini' key 'hosts'\n")
		return false
	}

	return true
}

func sendFile(host string) {

}

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		com.ColorLog("[ERRO] Fail to create new Watcher[ %s ]\n", err)
		os.Exit(2)
	}

	err = watcher.Watch(watchPath)
	if err != nil {
		com.ColorLog("[ERRO] Fail to watch directory( %s )[ %s ]\n", watchPath, err)
		os.Exit(2)
	}

	com.ColorLog("[INFO] Start watching...\n")
	for {
		select {
		case e := <-watcher.Event:
			if e.IsCreate() {
				if com.IsDir(e.Name) {
					com.ColorLog("[WARN] Hasn't support directory yet\n")
					continue
				}

				com.ColorLog("[INFO] Found new file( %s )\n", e.Name)
				for {
					fmt.Println("Please choose one of following hosts:")
					for i, v := range hosts {
						fmt.Println(i+1, v)
					}

					index := 0
					fmt.Scanln(&index)
					if index <= 0 || len(hosts) < index {
						com.ColorLog("[ERRO] Invalid index %d\n", index)
						continue
					}

					com.ColorLog("[INFO] File will be sent to %s\n", hosts[index-1])
					sendFile(hosts[index-1])
					break
				}
			}
		case err := <-watcher.Error:
			com.ColorLog("[ERRO] Watcher error[ %s ]\n", err)
			os.Exit(2)
		}
	}
}

func main() {
	if !checkConfig() {
		os.Exit(2)
	}

	watch()
}
