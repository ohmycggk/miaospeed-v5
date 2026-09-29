package packetloss

import (
	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/service/macros/ping"
)

type PacketLoss struct {
	interfaces.PacketLossDS
}

func (m *PacketLoss) Type() interfaces.SlaveRequestMatrixType {
	return interfaces.MatrixPacketLoss
}

func (m *PacketLoss) MacroJob() interfaces.SlaveRequestMacroType {
	return interfaces.MacroPing
}

func (m *PacketLoss) Extract(entry interfaces.SlaveRequestMatrixEntry, macro interfaces.SlaveRequestMacro) {
	if mac, ok := macro.(*ping.Ping); ok {
		m.Value = mac.PacketLoss
	}
}
