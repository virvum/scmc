package main

import (
	"bytes"
	"io/ioutil"
	"log"

	"github.com/virvum/scmc/pkg/logger"
	"github.com/virvum/scmc/pkg/mycloud"
)

func main() {
	username := ""
	password := ""

	l := logger.New(logger.Info, true, "")

	// Authneticate with myCloud.
	mc, err := mycloud.New(username, password, l)
	if err != nil {
		log.Fatalf("mcloud.New: %v", err)
	}

	// Fetch user identity information.
	identity, err := mc.Identity()
	if err != nil {
		log.Fatalf("mc.Identity: %v", err)
	}

	log.Printf("Identity: %v", identity)

	fn := "test-file.txt"
	r := bytes.NewBuffer([]byte("this is a test"))

	// Upload the file virtual file "test-file.txt".
	if err := mc.CreateFile(fn, r); err != nil {
		log.Fatalf("mc.CreateFile: %v", err)
	}

	w := bytes.NewBuffer([]byte(""))

	// Download the file "test-file.txt".
	if err := mc.GetFile(fn, w, ""); err != nil {
		log.Fatalf("mc.GetFile: %v", err)
	}

	b, err := ioutil.ReadAll(w)
	if err != nil {
		log.Fatalf("ioutil.ReadAll: %v", err)
	}

	log.Printf("Data returned: %v", b)
}
