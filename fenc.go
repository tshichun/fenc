package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const FENC = "fenc"

var excludes = []string{
	".DS_",
	"._",
}

func gzcompress(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz := path + ".gz"
	fgz, err := os.Create(gz)
	if err != nil {
		return "", err
	}
	defer fgz.Close()
	zw := gzip.NewWriter(fgz)
	defer zw.Close()
	tw := tar.NewWriter(zw)
	defer tw.Close()
	err = compress(tw, file, "")
	return gz, err
}

func gzuncompress(path, dst string) error {
	fgz, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fgz.Close()
	zr, err := gzip.NewReader(fgz)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	for {
		th, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		path := dst + th.Name
		file, err := mkfile(path)
		if err != nil {
			return err
		}
		io.Copy(file, tr)
	}
	return nil
}

func compress(tw *tar.Writer, file *os.File, prefix string) error {
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		prefix += "/" + fi.Name()
		fis, err := file.Readdir(0)
		if err != nil {
			return err
		}
		for _, fi := range fis {
			fn := fi.Name()
			if isExclude(fn) {
				continue
			}
			f, err := os.Open(file.Name() + "/" + fn)
			if err != nil {
				return err
			}
			if err = compress(tw, f, prefix); err != nil {
				return err
			}
		}
		return nil
	}
	defer file.Close()
	h, err := tar.FileInfoHeader(fi, "")
	h.Name = prefix + "/" + h.Name
	if err != nil {
		return err
	}
	if err = tw.WriteHeader(h); err != nil {
		return err
	}
	n, err := io.Copy(tw, file)
	if err != nil {
		return err
	}
	fmt.Printf("compressed %s size %d\n", h.Name, n)
	return nil
}

func mkfile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func isExclude(path string) bool {
	for _, prefix := range excludes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func encFile(path, key string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	enc := strings.TrimSuffix(path, filepath.Ext(path)) + "." + FENC
	fenc, err := os.OpenFile(enc, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer fenc.Close()
	var sum int64
	fi, _ := file.Stat()
	size := fi.Size()
	encKey := aesKey(key)
	fencLen := len(FENC)
	reader := bufio.NewReader(file)
	for {
		buf := make([]byte, 10<<20)
		rn := 0
		for {
			n, err := reader.Read(buf[rn:])
			if err != nil && err != io.EOF {
				return "", err
			}
			if n == 0 {
				break
			}
			rn += n
		}
		if rn == 0 {
			break
		}
		sum += int64(rn)
		buf = buf[:rn]
		body, err := aesEnc(buf, encKey)
		if err != nil {
			return "", err
		}
		head := []byte(FENC)
		head = append(head, 0, 0, 0, 0)
		binary.BigEndian.PutUint32(head[fencLen:], uint32(len(body)))
		fenc.Write(head)
		fenc.Write(body)
		fmt.Printf("encrypted %d/%d\n", sum, size)
	}
	return enc, nil
}

func decFile(path, key string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz := strings.TrimSuffix(path, filepath.Ext(path)) + FENC + ".gz"
	fgz, err := os.OpenFile(gz, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer fgz.Close()
	var sum int64
	fi, _ := file.Stat()
	size := fi.Size()
	decKey := aesKey(key)
	fencLen := len(FENC)
	reader := bufio.NewReader(file)
	for {
		buf := make([]byte, fencLen+4)
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return gz, err
		}
		if n == 0 {
			break
		}
		if string(buf[0:fencLen]) != FENC {
			return gz, fmt.Errorf("bad head")
		}
		sum += int64(n)
		bodyLen := binary.BigEndian.Uint32(buf[fencLen:])
		buf = make([]byte, bodyLen)
		rn := 0
		for {
			n, err = reader.Read(buf[rn:])
			if err != nil && err != io.EOF {
				return gz, err
			}
			if n == 0 {
				break
			}
			rn += n
		}
		if rn == 0 {
			break
		}
		sum += int64(rn)
		buf = buf[:rn]
		body, err := aesDec(buf, decKey)
		if err != nil {
			return gz, err
		}
		fgz.Write(body)
		fmt.Printf("decrypted %d/%d\n", sum, size)
	}
	return gz, nil
}

func aesKey(key string) []byte {
	w := md5.New()
	w.Write([]byte(FENC + key))
	return w.Sum(nil)
}

func padding(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	padTxt := bytes.Repeat([]byte{byte(pad)}, pad)
	return append(data, padTxt...)
}

func unpadding(data []byte) []byte {
	dataLen := len(data)
	unpad := int(data[dataLen-1])
	return data[:(dataLen - unpad)]
}

func aesEnc(raw, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	raw = padding(raw, blockSize)
	ciphertext := make([]byte, blockSize+len(raw))
	iv := ciphertext[:blockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[blockSize:], raw)
	return ciphertext, nil
}

func aesDec(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	if len(data) < blockSize {
		err := fmt.Errorf("ciphertext too short")
		return nil, err
	}
	iv := data[:blockSize]
	data = data[blockSize:]
	if len(data)%blockSize != 0 {
		err := fmt.Errorf("ciphertext not full blocks")
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(data, data)
	data = unpadding(data)
	return data, nil
}
