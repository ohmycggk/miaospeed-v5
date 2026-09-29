package matrices

import (
	"testing"

	"github.com/miaokobot/miaospeed/interfaces"
)

func TestRegisteredMatricesAreValid(t *testing.T) {
	types := []interfaces.SlaveRequestMatrixType{
		interfaces.MatrixAverageSpeed,
		interfaces.MatrixMaxSpeed,
		interfaces.MatrixPerSecondSpeed,
		interfaces.MatrixAverageUploadSpeed,
		interfaces.MatrixMaxUploadSpeed,
		interfaces.MatrixPerSecondUploadSpeed,
		interfaces.MatrixUDPType,
		interfaces.MatrixInboundGeoIP,
		interfaces.MatrixOutboundGeoIP,
		interfaces.MatrixScriptTest,
		interfaces.MatrixHTTPPing,
		interfaces.MatrixRTTPing,
		interfaces.MatrixMAXRTTPing,
		interfaces.MatrixTotalHTTPPing,
		interfaces.MatrixTotalRTTPing,
		interfaces.MatrixSDRTT,
		interfaces.MatrixSDHTTP,
		interfaces.MatrixHTTPCode,
		interfaces.MatrixPacketLoss,
		interfaces.MatrixHijack,
		interfaces.MatrixSleep,
	}
	for _, matrixType := range types {
		if !matrixType.Valid() {
			t.Fatalf("%s should be valid", matrixType)
		}
		got := Find(matrixType)
		if got == nil || got.Type() == interfaces.MatrixInvalid || got.Type() != matrixType {
			t.Fatalf("Find(%s) = %v", matrixType, got)
		}
	}
}
