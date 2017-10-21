package logging

import (
	"os"
	"fmt"
	"log"
	"path"
	"sync"
	"time"
)

type FileHandler struct {
	name     string
	off      bool
	filePath string
	fileFd   *os.File
	log      *log.Logger
}

func (fh FileHandler) Log(format string, v ...interface{}) {
	if fh.log != nil {
		fh.log.Printf(format, v...)
	}
}

func (fh *FileHandler) Off() {
	fh.off = true
}

func (fh FileHandler) IsOff() bool {
	return fh.off
}

func (fh *FileHandler) Run() {
	return
}

func (fh FileHandler) Name() string {
	return fh.name
}

func (fh *FileHandler) Close() error {
	fh.name = ""
	fh.off = true
	fh.log = nil
	fh.filePath = ""
	err := fh.fileFd.Close()
	if err != nil {
		return err
	}
	return nil
}

func (fh *FileHandler) SetFlags(flag int) {
	fh.log.SetFlags(flag)
}

func NewFileHandler(name string, filePath string) (*FileHandler, error) {
	if filePath == "" {
		return nil, fmt.Errorf("handler filePath is null.")
	}
	err := os.MkdirAll(path.Dir(filePath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	fileFd, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &FileHandler{
		name:     name,
		off:      false,
		filePath: filePath,
		fileFd:   fileFd,
		log:      log.New(fileFd, "", DEFLOGFLAG),
	}, nil
}

type SizeRotatingFileHandler struct {
	FileHandler
	fileCount  int
	fileSize   int64
	fileLock   *sync.RWMutex
	nextSuffix int
	running    bool
	rotatingLock *sync.RWMutex
	rotatingRun bool
	rotatingInterval int
}

func (srfh *SizeRotatingFileHandler) Log(format string, v ...interface{}) {
	if srfh.log != nil {
		srfh.fileLock.RLock()
		srfh.log.Printf(format, v...)
		srfh.fileLock.RUnlock()
		go srfh.rotating()
	}
}

func (srfh *SizeRotatingFileHandler) Off() {
	srfh.off = true
}

func (srfh SizeRotatingFileHandler) IsOff() bool {
	return srfh.off
}

func (srfh *SizeRotatingFileHandler) Run() {
	srfh.running = true
	srfh.rotateOnce()
}

func (srfh SizeRotatingFileHandler) Name() string {
	return srfh.name
}

func (srfh *SizeRotatingFileHandler) Close() error {
	srfh.rotateOnce()
	srfh.fileCount = 0
	srfh.fileSize = 0
	srfh.nextSuffix = 0
	srfh.fileLock.Lock()
	defer srfh.fileLock.Unlock()
	err := srfh.FileHandler.Close()
	if err != nil {
		return err
	}
	return nil
}

func (srfh *SizeRotatingFileHandler) SetFlags(flag int) {
	srfh.log.SetFlags(flag)
}

func (srfh *SizeRotatingFileHandler) isRotatingRun() bool {
	srfh.rotatingLock.RLock()
	defer srfh.rotatingLock.RUnlock()
	return srfh.rotatingRun
}

func (srfh *SizeRotatingFileHandler) setRotatingRun(bRun bool) {
	srfh.rotatingLock.Lock()
	defer srfh.rotatingLock.Unlock()
	srfh.rotatingRun = bRun
}

func (srfh *SizeRotatingFileHandler) rotateOnce() {
	srfh.fileLock.Lock()
	defer srfh.fileLock.Unlock()
	fileinfo, err := os.Stat(srfh.filePath)
	if err != nil {
		srfh.Log("ERROR cat not get %s status. quit rotating.", srfh.Name())
		return
	}
	if fileinfo.Size() >= srfh.fileSize {
		flag := srfh.log.Flags()
		srfh.fileFd.Close()
		os.Rename(srfh.filePath, fmt.Sprintf("%s.%d", srfh.filePath, srfh.nextSuffix))
		srfh.fileFd, _ = os.OpenFile(srfh.filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		srfh.log = log.New(srfh.fileFd, "", flag)
		if srfh.nextSuffix >= (srfh.fileCount - 1) {
			srfh.nextSuffix = 1
		} else {
			srfh.nextSuffix++
		}
	}
}

func (srfh *SizeRotatingFileHandler) rotating() {
	if srfh.nextSuffix == 0 || srfh.isRotatingRun() {
		return
	}
	srfh.setRotatingRun(true)
	srfh.rotateOnce()
	time.Sleep(time.Duration(srfh.rotatingInterval)*time.Millisecond)
	srfh.setRotatingRun(false)
}

func (srfh *SizeRotatingFileHandler) checkNextSuffix() {
	if srfh.nextSuffix == 0 {
		return
	}
	var minModTime int64
	for n := 1; n < srfh.fileCount; n++ {
		filepath := fmt.Sprintf("%s.%d", srfh.filePath, n)
		if fileinfo, err := os.Stat(filepath); !os.IsNotExist(err) {
			if (minModTime == 0 || minModTime > fileinfo.ModTime().Unix()) {
				minModTime = fileinfo.ModTime().Unix()
				srfh.nextSuffix = n
			}
		} else {
			srfh.nextSuffix = n
			break
		}
	}
}

func NewSizeRotatingFileHandler(
	name string, filePath string, fileCount int, fileSize int64,
) (*SizeRotatingFileHandler, error) {
	fh, err := NewFileHandler(name, filePath)
	if err != nil {
		return nil, err
	}

	//check fileCount and fileSize
	nSuffix := 1
	nCount := 1
	if fileCount < 0 || fileCount == 1 {
		nSuffix = 0
	} else {
		nCount = fileCount
	}
	nSize := MB
	if fileSize > 0 {
		nSize = fileSize
	}

	nInterval := 0
	switch {
		case nSize < 10*KB:
			nInterval = 0
		case nSize <= MB: 
			nInterval = 100
		case nSize <= 100*MB:
			nInterval = 10000
		case nSize <= GB:
			nInterval = 50000
		default:
			nInterval = 100000
	}

	srfh := &SizeRotatingFileHandler {
		FileHandler      : *fh,
		fileCount        : nCount,
		fileSize         : nSize,
		fileLock         : new(sync.RWMutex),
		nextSuffix       : nSuffix,
		running          : false,
		rotatingLock     : new(sync.RWMutex),
		rotatingRun      : false,
		rotatingInterval : nInterval,
	}
	srfh.checkNextSuffix()
	return srfh, nil
}
