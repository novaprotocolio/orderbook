package main

import (
	// "context"
	"fmt"
	"os"
	"sync"

	// "time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/manifoldco/promptui"

	demo "github.com/novaprotocolio/orderbook/common"
	"github.com/novaprotocolio/orderbook/terminal"
	"gopkg.in/urfave/cli.v1"
)

var (
	app       = cli.NewApp()
	rpcClient *rpc.Client
	prompt    *promptui.Select
	commands  []terminal.Command
)

func init() {
	// Initialize the CLI app and start tomo
	app.Commands = []cli.Command{
		cli.Command{
			Name: "rpc",
			Action: func(c *cli.Context) error {
				// must return export function
				return Start()
			},
			Flags: []cli.Flag{
				cli.IntFlag{Name: "wsPort, p", Value: demo.WSDefaultPort},
			},
		},
		cli.Command{
			Name: "savekey",
			Action: func(c *cli.Context) error {
				// must return export function
				return SaveKey(c.String("path"))
			},
			Flags: []cli.Flag{
				cli.StringFlag{Name: "path, p", Value: "../keystore"},
			},
		},
		cli.Command{
			Name: "loadkey",
			Action: func(c *cli.Context) error {
				// must return export function
				return LoadKey(c.String("path"))
			},
			Flags: []cli.Flag{
				cli.StringFlag{Name: "path, p", Value: "../keystore"},
			},
		},
	}
}

func main() {

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

func initPrompt() {
	// init prompt commands
	commands = []terminal.Command{
		{
			Name: "getOrder",
			Arguments: []terminal.Argument{
				{Name: "pair_name", Value: "TOMO/WETH"},
				{Name: "order_id", Value: "1"},
			},
			Description: "Get the order from the orderbook storgae",
		},
		{
			Name: "updateEndpoint",
			Arguments: []terminal.Argument{
				{Name: "endpoint", AllowEdit: true},
			},
			Description: "Update the websocket port to call RPC",
		},	
		{
			Name: "getBestAskList",
			Arguments: []terminal.Argument{
				{Name: "pair_name", Value: "TOMO/WETH"},
			},
			Description: "Get best ask list",
		},
		{
			Name: "getBestBidList",
			Arguments: []terminal.Argument{
				{Name: "pair_name", Value: "TOMO/WETH"},
			},
			Description: "Get best bid list",
		},
		{
			Name:        "quit",
			Description: "Quit the program",
		},
	}
	// sort.Sort(terminal.CommandsByName(commands))
	prompt = terminal.NewPrompt("Your choice:", 4, commands)
}

func SaveKey(path string) error {

	privkey, _ := crypto.GenerateKey()
	return crypto.SaveECDSA(path, privkey)

}

func LoadKey(path string) error {
	privkey, _ := crypto.LoadECDSA(path)
	demo.LogInfo("privkey", "publickey", privkey.PublicKey)
	return nil
}

func logResult(result interface{}, err error) {
	if err != nil {
		demo.LogCrit("RPC call fail", "err", err)
	} else {
		demo.LogInfo("Get response", "result", result)
	}
}
func callRPC(result interface{}, function string, params ...interface{}) {
	// assume there is no argument at all
	err := rpcClient.Call(&result, function, params...)
	demo.LogInfo("Call", "function", function, "params", params)
	logResult(result, err)
}

func Start() error {

	initPrompt()

	// we can use websockets ...

	// get a valid topic byte
	// get a valid topic byte
	// call the RPC method, will be name_methodInCammelCase
	// process command
	fmt.Println("---------------Welcome to Backend testing---------------------")
	var endWaiter sync.WaitGroup
	endWaiter.Add(1)

	// start serving
	go func() {
		// var nodeName = "node1"
		var endpoint = "http://127.0.0.1:8545"
		// var ctx context.Context
		// var cancel context.CancelFunc
		// var signerAddress = "0x28074f8d0fd78629cd59290cac185611a8d60109"
		for {
			// loop command
			commands[1].Arguments[0].Value = endpoint
			// if wsPort != "18543" {
			// 	signerAddress = "0x6e6BB166F420DDd682cAEbf55dAfBaFda74f2c9c"
			// }
			// commands[0].Arguments[1].Value = signerAddress

			selected, _, err := prompt.Run()

			// unknow error, should retry
			if err != nil {
				demo.LogInfo("Prompt failed %v\n", err)
				continue
			}

			// get selected command and run it
			command := commands[selected]
			if command.Name == "quit" {
				demo.LogInfo("Server quiting...")
				endWaiter.Done()
				demo.LogInfo("-> Goodbye\n")
				// if cancel != nil {
				// 	cancel()
				// }
				return
			}
			results := command.Run()

			if command.Name == "updateEndpoint" {
				demo.LogInfo("-> Update", "endpoint", results["endpoint"])
				endpoint = results["endpoint"]
				continue
			}

			// ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
			rpcClient, err = rpc.Dial(endpoint)
			// rightRPCClient, err := rpc.DialWebsocket(ctx, rightEndpoint, "*")
			if err != nil {
				demo.LogCrit("RPC dial failed", "endpoint", endpoint, "err", err)
				// cancel()
				continue
			}
			// process command
			var result interface{}
			switch command.Name {
			case "getOrder":
				demo.LogInfo("-> Get orders", "pair_name", results["pair_name"], "order_id", results["order_id"])
				// put message on channel
				callRPC(result, "orderbook_getOrder", results["pair_name"], results["order_id"])			
			case "getBestAskList":
				demo.LogInfo("-> Best ask list:")
				callRPC(result, "orderbook_getBestAskList", results["pair_name"])
			case "getBestBidList":
				demo.LogInfo("-> Best bid list:")
				callRPC(result, "orderbook_getBestBidList", results["pair_name"])
			default:
				demo.LogInfo(fmt.Sprintf("-> Unknown command: %s\n", command.Name))
			}

		}
	}()

	// wait for command processing
	endWaiter.Wait()

	// bring down the servicenodes
	if rpcClient != nil {
		rpcClient.Close()
	}

	return nil
}
