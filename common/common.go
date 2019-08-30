package common

import (
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/fatih/color"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	Log      = log.New("DemoLog", "v1.0.1")
	BasePath string
	verbose  = flag.Bool("v", false, "more verbose logs")
)

func init() {
	var err error
	BasePath, err = os.Getwd()
	if err != nil {
		Log.Crit("Could not determine working directory", "err", err)
	}
	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	loglevel := log.LvlInfo
	if *verbose {
		loglevel = log.LvlTrace
	}
	hf := log.LvlFilterHandler(loglevel, hs)
	h := log.CallerFileHandler(hf)
	log.Root().SetHandler(h)
}
func setInterval(someFunc func(), milliseconds int, async bool) chan bool {
	interval := time.Duration(milliseconds) * time.Millisecond
	ticker := time.NewTicker(interval)
	clear := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				if async {
					go someFunc()
				} else {
					someFunc()
				}
			case <-clear:
				ticker.Stop()
				return
			}
		}
	}()
	return clear
}
func LogInfo(msg string, ctx ...interface{}) {
	Log.Info(color.HiGreenString(msg), ctx...)
}
func LogCrit(msg string, ctx ...interface{}) {
	Log.Info(color.HiRedString(msg), ctx...)
}
func LogDebug(msg string, ctx ...interface{}) {
	Log.Info(color.HiMagentaString(msg), ctx...)
}
func LogTrace(msg string, ctx ...interface{}) {
	Log.Trace(color.HiCyanString(msg), ctx...)
}
func LogError(msg string, ctx ...interface{}) {
	Log.Info(color.HiRedString(msg), ctx...)
}
func LogWarn(msg string, ctx ...interface{}) {
	Log.Info(color.HiYellowString(msg), ctx...)
}
func ToJSON(obj interface{}) string {
	bytes, _ := json.Marshal(obj)
	return string(bytes)
}
func NewServiceNodeWithPrivateKey(privateKey *ecdsa.PrivateKey, port int, httpport int, wsport int, modules ...string) (*node.Node, error) {
	dataDir := fmt.Sprintf("%s%d", DatadirPrefix, port)
	return NewServiceNodeWithPrivateKeyAndDataDir(privateKey, dataDir, port, httpport, wsport, modules...)
}
func NewServiceNodeWithPrivateKeyAndProtocols(privateKey *ecdsa.PrivateKey, port int, httpport int, wsport int, protocols []p2p.Protocol, modules ...string) (*node.Node, error) {
	dataDir := fmt.Sprintf("%s%d", DatadirPrefix, port)
	return NewServiceNodeWithPrivateKeyAndDataDirAndProtocols(privateKey, dataDir, port, httpport, wsport, protocols, modules...)
}
func NewServiceNodeWithPrivateKeyAndDataDir(privateKey *ecdsa.PrivateKey, dataDir string, port int, httpport int, wsport int, modules ...string) (*node.Node, error) {
	return NewServiceNodeWithPrivateKeyAndDataDirAndProtocols(privateKey, dataDir, port, httpport, wsport, nil, modules...)
}
func NewServiceNodeWithPrivateKeyAndDataDirAndProtocols(privateKey *ecdsa.PrivateKey, dataDir string, port int, httpport int, wsport int, protocols []p2p.Protocol, modules ...string) (*node.Node, error) {
	if port == 0 {
		port = P2pPort
	}
	cfg := &node.DefaultConfig
	if protocols != nil {
		cfg.P2P.Protocols = append(cfg.P2P.Protocols, protocols...)
	}
	cfg.P2P.ListenAddr = fmt.Sprintf(":%d", port)
	cfg.P2P.EnableMsgEvents = true
	cfg.IPCPath = IPCName
	cfg.DataDir = dataDir
	if privateKey != nil {
		cfg.P2P.PrivateKey = privateKey
	}
	if httpport > 0 {
		cfg.HTTPHost = "0.0.0.0"
		cfg.HTTPPort = httpport
		cfg.HTTPCors = []string{"*"}
		cfg.HTTPModules = append(cfg.HTTPModules, modules...)
	}
	if wsport > 0 {
		cfg.WSHost = "0.0.0.0"
		cfg.WSPort = wsport
		cfg.WSOrigins = []string{"*"}
		cfg.WSModules = append(cfg.WSModules, modules...)
	}
	stack, err := node.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("ServiceNode create fail: %v", err)
	}
	return stack, nil
}
func NewServiceNode(port int, httpport int, wsport int, modules ...string) (*node.Node, error) {
	return NewServiceNodeWithPrivateKey(nil, port, httpport, wsport, modules...)
}
func NewServer(privkey *ecdsa.PrivateKey, name string, version string, proto *p2p.Protocol, port int) *p2p.Server {
	cfg := p2p.Config{PrivateKey: privkey, Name: common.MakeName(name, version), MaxPeers: 1, EnableMsgEvents: true}
	if proto != nil {
		cfg.Protocols = []p2p.Protocol{*proto}
	}
	if port > 0 {
		cfg.ListenAddr = fmt.Sprintf(":%d", port)
	}
	srv := &p2p.Server{Config: cfg}
	return srv
}
func getEnodeFromRPC(rawurl string) (string, error) {
	rpcclient, err := rpc.Dial(rawurl)
	if err != nil {
		return "", fmt.Errorf("cannot add remote host: %v", err)
	}
	var nodeinfo p2p.NodeInfo
	err = rpcclient.Call(&nodeinfo, "admin_nodeInfo")
	if err != nil {
		return "", fmt.Errorf("RPC nodeinfo call failed: %v", err)
	}
	return nodeinfo.Enode, nil
}
func WaitForCtrlC() {
	var endWaiter sync.WaitGroup
	endWaiter.Add(1)
	var signalChannel chan os.Signal
	signalChannel = make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	go func() {
		<-signalChannel
		endWaiter.Done()
	}()
	endWaiter.Wait()
}
func RandomID() (id enode.ID) {
	for i := range id {
		id[i] = byte(rand.Intn(255))
	}
	return id
}
