package logging

import (
	"log"
	"os"
)

const (
	DEFLOGFLAG = log.LstdFlags | log.Lmicroseconds
)

type Handler interface {
	Log(format string, v ...interface{})
	Off()
	IsOff() bool
	Close() error
	Run()
	Name() string
}

type StdHandler struct {
	name string
	off  bool
	log  *log.Logger
}

func (h StdHandler) Log(format string, v ...interface{}) {
	if h.log != nil {
		h.log.Printf(format, v...)
	}
}

func (h *StdHandler) Off() {
	h.off = true
}

func (h StdHandler) IsOff() bool {
	return h.off
}

func (h *StdHandler) Run() {
	return
}

func (h StdHandler) Name() string {
	return h.name
}

func (h *StdHandler) Close() error {
	h.name = ""
	h.off = true
	h.log = nil
	return nil
}

func NewStdHandler() (*StdHandler, error) {
	return &StdHandler{
		name: "StdHandler",
		off:  false,
		log:  log.New(os.Stdout, "", DEFLOGFLAG),
	}, nil
}
