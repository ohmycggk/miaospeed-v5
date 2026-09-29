package averagespeed

import (
	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/service/macros/speed"
)

type AverageSpeed struct {
	interfaces.AverageSpeedDS
}

type AverageUploadSpeed struct {
	interfaces.AverageSpeedDS
}

func (m *AverageSpeed) Type() interfaces.SlaveRequestMatrixType {
	return interfaces.MatrixAverageSpeed
}

func (m *AverageSpeed) MacroJob() interfaces.SlaveRequestMacroType {
	return interfaces.MacroSpeed
}

func (m *AverageSpeed) Extract(entry interfaces.SlaveRequestMatrixEntry, macro interfaces.SlaveRequestMacro) {
	if mac, ok := macro.(*speed.Speed); ok {
		m.Value = mac.AvgSpeed
	}
}

func (m *AverageUploadSpeed) Type() interfaces.SlaveRequestMatrixType {
	return interfaces.MatrixAverageUploadSpeed
}

func (m *AverageUploadSpeed) MacroJob() interfaces.SlaveRequestMacroType {
	return interfaces.MacroUploadSpeed
}

func (m *AverageUploadSpeed) Extract(entry interfaces.SlaveRequestMatrixEntry, macro interfaces.SlaveRequestMacro) {
	if mac, ok := macro.(*speed.UploadSpeed); ok {
		m.Value = mac.AvgSpeed
	}
}
