package main

import (
	"flag"
	"os"

	"github.com/miaokobot/miaospeed/preconfigs"
	"github.com/miaokobot/miaospeed/service"
	"github.com/miaokobot/miaospeed/utils"
)

func InitConfigServer() *utils.GlobalConfig {
	gcfg := &utils.GCFG

	sflag := flag.NewFlagSet(cmdName+" server", flag.ExitOnError)
	sflag.StringVar(&gcfg.Token, "token", "", "specify the token used to sign request")
	sflag.StringVar(&gcfg.Binder, "bind", "", "bind a socket, can be format like 0.0.0.0:9966 or /tmp/unix_socket")
	sflag.UintVar(&gcfg.ConnTaskTreading, "connthread", 64, "parallel threads when processing normal connectivity tasks")
	sflag.Uint64Var(&gcfg.SpeedLimit, "speedlimit", 0, "speed ratelimit (in Bytes per Second), default with no limits")
	sflag.UintVar(&gcfg.PauseSecond, "pausesecond", 0, "pause such period after each speed job (seconds)")
	sflag.BoolVar(&gcfg.MiaoKoSignedTLS, "mtls", false, "enable miaoko certs for tls verification")
	sflag.BoolVar(&gcfg.NoSpeedFlag, "nospeed", false, "decline all speedtest requests")
	sflag.StringVar(&gcfg.MaxmindDB, "mmdb", "", "reroute all geoip query to local mmdbs. for example: test.mmdb,testcity.mmdb")

	wsPath := sflag.String("path", "", "websocket path clients must use, for example koipy or /koipy. empty accepts every path")
	whiteList := sflag.String("whitelist", "", "bot id whitelist, separated by comma or pipe, for example 1111,2222 or 1111|2222")
	pubKeyPath := sflag.String("serverpublickey", "", "PEM file that overrides the embedded TLS certificate")
	privKeyPath := sflag.String("serverprivatekey", "", "PEM file that overrides the embedded TLS private key")
	parseFlag(sflag)

	gcfg.Path = utils.NormalizeWSPath(*wsPath)
	if gcfg.Path != "" {
		utils.DWarnf("MiaoSpeed Server | Using a custom websocket path: %s", gcfg.Path)
	}
	gcfg.WhiteList = utils.SplitWhiteList(*whiteList)

	pubPEM, keyPEM, err := utils.ReadServerCertificatePair(*pubKeyPath, *privKeyPath)
	if err != nil {
		utils.DErrorf("MiaoSpeed Server | %s", err.Error())
		os.Exit(1)
	}
	if pubPEM != "" {
		utils.DWarnf("MiaoSpeed Server | Override predefined tls certificates")
		preconfigs.MIAOKO_TLS_CRT = pubPEM
		preconfigs.MIAOKO_TLS_KEY = keyPEM
	}

	return gcfg
}

func RunCliServer() {
	InitConfigServer()
	utils.DWarnf("MiaoSpeed speedtesting client %s", utils.VERSION)

	// load maxmind db
	if utils.LoadMaxMindDB(utils.GCFG.MaxmindDB) != nil {
		os.Exit(1)
	}

	// start task server
	go service.StartTaskServer()

	// start api server
	service.CleanUpServer()
	go service.InitServer()

	<-utils.MakeSysChan()

	// clean up
	service.CleanUpServer()
	utils.DLog("shutting down.")
}
