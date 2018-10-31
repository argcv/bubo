package utils

import (
	"github.com/argcv/webeh/log"
	"runtime"
)

const (
	kMinMaxProcs    = 32
	kMaxProcsFactor = 2
)

func SetMaxProcs() {
	nMaxProcs := runtime.NumCPU() * kMaxProcsFactor
	if nMaxProcs < kMinMaxProcs {
		nMaxProcs = kMinMaxProcs
	}
	runtime.GOMAXPROCS(nMaxProcs)
	log.Debugf("nMaxProcs=%v", nMaxProcs)
}
