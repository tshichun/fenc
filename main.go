package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		usage()
		os.Exit(1)
	}
	var err error
	do := os.Args[1]
	path := os.Args[2]

	if path != "" {
		path = strings.TrimSuffix(filepath.ToSlash(path), "/")
	}
	if do == "enc" {
		var key string
		for {
			key = passwd("\nenter key")
			if key == "" {
				continue
			}
			if key != passwd("\nretype key") {
				fmt.Println("\nkeys don't match")
				continue
			}
			break
		}
		var encTo string
		encTo, err = enc(path, key)
		fmt.Printf("\nencrypted to: %s", encTo)
	} else if do == "dec" {
		key := passwd("\nenter key")
		var decTo string
		decTo, err = dec(path, key)
		fmt.Printf("\ndecrypted to: %s", decTo)
	} else {
		usage()
		os.Exit(1)
	}

	if err == nil {
		fmt.Printf("\nsuccessful\n")
	} else {
		fmt.Println(err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf("usage: %s <enc|dec> <file or directory path>\n", os.Args[0])
}

func enc(path, key string) (string, error) {
	gz, err := gzcompress(path)
	if err != nil {
		if gz != "" {
			os.Remove(gz)
		}
		return "", err
	}
	enc, err := encFile(gz, key)
	os.Remove(gz)
	return enc, err
}

func dec(path, key string) (string, error) {
	gz, err := decFile(path, key)
	if err != nil {
		if gz != "" {
			os.Remove(gz)
		}
		return "", err
	}
	dst := filepath.Dir(gz) + "/" + FENC + "_dec/"
	err = gzuncompress(gz, dst)
	os.Remove(gz)
	return dst, err
}
