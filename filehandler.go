package logging

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DEFROTATINGINTERVAL = 120
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
	chQuit     chan bool
	chTimer    chan bool
}

func (srfh SizeRotatingFileHandler) Log(format string, v ...interface{}) {
	if srfh.log != nil {
		srfh.fileLock.RLock()
		srfh.log.Printf(format, v...)
		srfh.fileLock.RUnlock()
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
	srfh.CheckNextSuffix()

	//start Timer
	go func(srfh *SizeRotatingFileHandler, interval int) {
		for {
			if !srfh.running {
				//fmt.Println("SizeRotatingFileHandler %s Timer quit.", srfh.Name())
				return
			}
			srfh.chTimer <- true
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}(srfh, DEFROTATINGINTERVAL)

	//wait Timer for check file size or quit
	for {
		select {
		case <-srfh.chTimer:
			fileinfo, err := os.Stat(srfh.filePath)
			if err != nil {
				srfh.Log("ERROR cat not get %s status. skip rotating.", srfh.Name())
				continue
			}
			if fileinfo.Size() > srfh.fileSize {
				if srfh.nextSuffix > srfh.fileCount {
					srfh.nextSuffix = 1
				}
				srfh.fileLock.Lock()
				srfh.fileFd.Close()
				os.Rename(srfh.filePath, fmt.Sprintf("%s.%d", srfh.filePath, srfh.nextSuffix))
				srfh.fileFd, _ = os.OpenFile(srfh.filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
				srfh.log = log.New(srfh.fileFd, "", DEFLOGFLAG)
				srfh.nextSuffix++
				srfh.fileLock.Unlock()
			}
		case <-srfh.chQuit:
			srfh.running = false
			return
		}
	}
}

func (srfh SizeRotatingFileHandler) Name() string {
	return srfh.name
}

func (srfh *SizeRotatingFileHandler) Close() error {
	srfh.fileCount = 0
	srfh.fileSize = 0
	srfh.fileLock = nil
	srfh.nextSuffix = 0
	srfh.running = false
	srfh.chQuit <- true
	close(srfh.chQuit)
	close(srfh.chTimer)
	err := srfh.FileHandler.Close()
	if err != nil {
		return err
	}
	return nil
}

func (srfh *SizeRotatingFileHandler) CheckNextSuffix() {
	filename := path.Base(srfh.filePath)
	files, err := ioutil.ReadDir(path.Dir(srfh.filePath))
	if err != nil {
		return
	}

	var minModTime int64 = 0
	var nextSuffix int = 0
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), filename+".") {
			suffix := strings.Replace(file.Name(), filename+".", "", -1)
			isuffix, err := strconv.Atoi(suffix)
			if err == nil {
				if minModTime == 0 || minModTime > file.ModTime().Unix() {
					minModTime = file.ModTime().Unix()
					nextSuffix = isuffix
				}
			}
		}
	}
	if nextSuffix < srfh.fileCount {
		srfh.nextSuffix = nextSuffix
	}
}

func NewSizeRotatingFileHandler(
	name string, filePath string, fileCount int, fileSize int64,
) (*SizeRotatingFileHandler, error) {
	fh, err := NewFileHandler(name, filePath)
	if err != nil {
		return nil, err
	}
	return &SizeRotatingFileHandler{
		FileHandler: *fh,
		fileCount:   fileCount,
		fileSize:    fileSize,
		fileLock:    new(sync.RWMutex),
		nextSuffix:  1,
		running:     false,
		chQuit:      make(chan bool),
		chTimer:     make(chan bool),
	}, nil
}
