package myutils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)


type Conf struct {
	Origins      []string
	Caches       []string
	SingleCache  string
	SingleOrigin string

	TestServer 	 string
	TestTime	 int64

	ClientConcurrency int64
}


func LoadConfig(confPath string) (conf Conf) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	file, err := os.Open(confPath)
	if err != nil {
		log.Fatal("error opening config file ", err)
		fmt.Println(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&conf); err != nil {
		log.Fatal("error during decoding config file:", err)
		fmt.Println(err)
		return
	}
	return conf
}

