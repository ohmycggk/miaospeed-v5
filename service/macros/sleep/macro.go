package sleep

import (
	"github.com/miaokobot/miaospeed/interfaces"
	"time"
)

type Sleep struct {
	duration time.Duration
	Value    string
}

func (m *Sleep) Type() interfaces.SlaveRequestMacroType {
	return interfaces.MacroSleep
}

func (m *Sleep) Run(proxy interfaces.Vendor, r *interfaces.SlaveRequest) error {
	m.duration = time.Second * 10
	time.Sleep(m.duration)
	m.Value = "slept for " + m.duration.String() + " seconds"
	return nil
}
