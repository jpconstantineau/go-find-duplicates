package service

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"os"

	"github.com/jpconstantineau/go-find-duplicates/bytesutil"
	"github.com/jpconstantineau/go-find-duplicates/entity"
	"github.com/jpconstantineau/go-find-duplicates/utils"
)

const (
	thresholdFileSize = 16 * bytesutil.KIBI
)

// GetDigest generates entity.FileDigest of the file provided
func GetDigest(path string, isThorough bool) (entity.FileDigest, error) {
	info, statErr := os.Lstat(path)
	if statErr != nil {
		return entity.FileDigest{}, statErr
	}
	h, hashErr := fileHash(path, isThorough)
	if hashErr != nil {
		return entity.FileDigest{}, hashErr
	}
	return entity.FileDigest{
		FileExtension: utils.GetFileExt(path),
		FileSize:      info.Size(),
		FileHash:      h,
	}, nil
}

// fileHash calculates the hash of the file provided.
// If isThorough is true, then it uses SHA512 of the entire file.
// Otherwise, it uses CRC32 of "crucial bytes" of the file.
func fileHash(path string, isThorough bool) (string, error) {
	fileInfo, statErr := os.Lstat(path)
	if statErr != nil {
		return "", fmt.Errorf("couldn't stat: %+v", statErr)
	}
	if !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("can't compute hash of non-regular file")
	}
	var prefix string
	var bytes []byte
	var fileReadErr error
	if isThorough {
		bytes, fileReadErr = os.ReadFile(path)
	} else if fileInfo.Size() <= thresholdFileSize {
		prefix = "f"
		bytes, fileReadErr = os.ReadFile(path)
	} else {
		prefix = "s"
		bytes, fileReadErr = readCrucialBytes(path, fileInfo.Size())
	}
	if fileReadErr != nil {
		return "", fmt.Errorf("couldn't calculate hash: %+v", fileReadErr)
	}
	var h hash.Hash
	if isThorough {
		h = sha512.New()
	} else {
		h = crc32.NewIEEE()
	}
	_, hashErr := h.Write(bytes)
	if hashErr != nil {
		return "", fmt.Errorf("error while computing hash: %+v", hashErr)
	}
	hashBytes := h.Sum(nil)
	return prefix + hex.EncodeToString(hashBytes), nil
}

// readCrucialBytes reads the first few bytes, middle bytes and last few bytes of the file
func readCrucialBytes(filePath string, fileSize int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	firstBytes := make([]byte, thresholdFileSize/2)
	_, fErr := file.ReadAt(firstBytes, 0)
	if fErr != nil {
		return nil, fmt.Errorf("couldn't read first few bytes (maybe file is corrupted?): %+v", fErr)
	}
	middleBytes := make([]byte, thresholdFileSize/4)
	_, mErr := file.ReadAt(middleBytes, fileSize/2)
	if mErr != nil {
		return nil, fmt.Errorf("couldn't read middle bytes (maybe file is corrupted?): %+v", mErr)
	}
	lastBytes := make([]byte, thresholdFileSize/4)
	_, lErr := file.ReadAt(lastBytes, fileSize-thresholdFileSize/4)
	if lErr != nil {
		return nil, fmt.Errorf("couldn't read end bytes (maybe file is corrupted?): %+v", lErr)
	}
	bytes := append(append(firstBytes, middleBytes...), lastBytes...)
	return bytes, nil
}
