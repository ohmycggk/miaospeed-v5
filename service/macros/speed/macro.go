package speed

import (
	"time"

	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/utils"
)

type Speed struct {
	AvgSpeed  uint64
	MaxSpeed  uint64
	TotalSize uint64
	Speeds    []uint64
}

type UploadSpeed struct {
	AvgSpeed  uint64
	MaxSpeed  uint64
	TotalSize uint64
	Speeds    []uint64
}

func (m *Speed) Type() interfaces.SlaveRequestMacroType {
	return interfaces.MacroSpeed
}

func (m *Speed) Run(proxy interfaces.Vendor, r *interfaces.SlaveRequest) error {
	start := time.Now()
	Once(m, proxy, &r.Configs)
	utils.DLogf("Speed macro took %s", time.Since(start))
	return nil
}

func (m *UploadSpeed) Type() interfaces.SlaveRequestMacroType {
	return interfaces.MacroUploadSpeed
}

func (m *UploadSpeed) Run(proxy interfaces.Vendor, r *interfaces.SlaveRequest) error {
	start := time.Now()
	UploadOnceChunked(m, proxy, &r.Configs)
	utils.DLogf("UploadSpeed macro took %s", time.Since(start))
	return nil
}
