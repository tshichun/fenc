package main

import (
	"bufio"
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
	var (
		err error
		key string
	)
	do := os.Args[1]
	path := os.Args[2]

	if path != "" {
		path = strings.TrimSuffix(filepath.ToSlash(path), "/")
	}
	if do == "enc" {
		var (
			retype string
			count  int
		)
		input := bufio.NewScanner(os.Stdin)
		fmt.Printf("enter key: ")
		for input.Scan() {
			if count == 0 {
				key = input.Text()
				fmt.Printf("retype key: ")
			} else if count == 1 {
				retype = input.Text()
			}
			count++
			if count == 2 {
				break
			}
		}
		if key == "" || key != retype {
			fmt.Println("keys don't match")
			os.Exit(1)
		}
		var encTo string
		encTo, err = enc(path, key)
		fmt.Printf("\nencrypted to: %s\n", encTo)
	} else if do == "dec" {
		input := bufio.NewScanner(os.Stdin)
		fmt.Printf("enter key: ")
		for input.Scan() {
			key = input.Text()
			break
		}
		var decTo string
		decTo, err = dec(path, key)
		fmt.Printf("\ndecrypted to: %s\n", decTo)
	} else {
		usage()
		os.Exit(1)
	}

	if err == nil {
		fmt.Printf("Successful\n")
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
