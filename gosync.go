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
	"io"
	"net"
	"os"
	"path"
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

const confTpl = `[setting]
listen_addr=10.0.0.30:5000
watch_path=watch
receive_path=receive
hosts=10.0.0.30:5000

[move_case]`

func checkConfig() bool {
	var err error
	srcPath, err = com.GetSrcPath("github.com/Unknwon/gosync")
	if err != nil {
		com.ColorLog("[ERRO] Fail to locate source path[ %s ]\n", err)
		return false
	}

	if !com.IsExist(srcPath + "conf/app.ini") {
		com.SaveFile(srcPath+"conf/app.ini", []byte(confTpl))
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

func handler(conn net.Conn) {
	defer conn.Close()
	p := make([]byte, 1024)
	n, err := conn.Read(p)
	if err != nil {
		com.ColorLog("[ERRO] S: Cannot read header[ %s ]\n", err)
		return
	} else if n == 0 {
		com.ColorLog("[ERRO] S: Empty header\n")
		return
	}

	fileName := string(p[:n])
	f, err := os.Create(receivePath + "/" + fileName)
	if err != nil {
		com.ColorLog("[ERRO] S: Fail to create file[ %s ]\n", err)
		return
	}
	defer f.Close()

	conn.Write([]byte("ok"))

	io.Copy(f, conn)
	for {
		buffer := make([]byte, 1024*200)
		n, err := conn.Read(buffer)
		//blockSize := int64(n)
		_ = n
		if err != nil && err != io.EOF {
			fmt.Println("cannot read", err)
		} else if err == io.EOF {
			break
		}
	}
}

func serve() {
	l, err := net.Listen("tcp", cfg.MustValue("setting", "listen_addr"))
	if err != nil {
		com.ColorLog("[ERRO] Fail to start server[ %s ]\n", err)
		os.Exit(2)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
				com.ColorLog("[ERRO] Network error[ %s ]\n", err)
			}
			continue
		}
		go handler(conn)
	}
}

func sendFile(host, fileName string) {
	f, err := os.Open(fileName)
	if err != nil {
		com.ColorLog("[ERRO] Fail to open file[ %s ]\n", err)
		return
	}
	defer f.Close()

	fi, err := os.Stat(fileName)
	if err != nil {
		com.ColorLog("[ERRO] Fail to stat file[ %s ]\n", err)
		return
	}

	fileName = path.Base(strings.Replace(fileName, "\\", "/", -1))
	com.ColorLog("[INFO] File name: %s; size: %dB\n", fileName, fi.Size())

	conn, err := net.Dial("tcp", host)
	if err != nil {
		com.ColorLog("[ERRO] Fail to establish connection[ %s ]\n", err)
		return
	}
	defer conn.Close()

	com.ColorLog("[SUCC] Connection established\n")

	conn.Write([]byte(fileName))
	p := make([]byte, 2)
	_, err = conn.Read(p)
	if err != nil {
		com.ColorLog("[ERRO] Cannot get response from server[ %s ]\n", err)
		return
	} else if string(p) != "ok" {
		com.ColorLog("[ERRO] Invalid response: %s\n", string(p))
		return
	}

	com.ColorLog("[SUCC] Header sent\n")

	io.Copy(conn, f)
	com.ColorLog("[SUCC] File sent\n")
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
					sendFile(hosts[index-1], e.Name)
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

	go serve()
	watch()
}
