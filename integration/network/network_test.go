//go:build !relayer
// +build !relayer

package network_test

import (
	"testing"

	"github.com/ignite-hq/cli/ignite/pkg/cmdrunner/step"
	envtest "github.com/ignite-hq/cli/integration"
)

func TestGenerateAnAppWithStargateWithListAndVerify(t *testing.T) {
	var (
		env  = envtest.New(t)
		path = env.Scaffold("blog")
	)

	env.Must(env.Exec("create a list",
		step.NewSteps(step.New(
			step.Exec("starport",
				"network",
				"--local",
				"chain",
				"publish",
				"https://github.com/Pantani/mars",
				"--from",
				"alice",
			),
			step.Workdir(path),
		)),
	))
}

func TestNetwork(t *testing.T) {
	//var (
	//	stakeDenom = "stake"
	//	env        = envtest.New(t)
	//	appname    = randstr.Runes(10)
	//	//host        = env.RandomizeServerPorts(path, "")
	//	ctx, cancel = context.WithCancel(env.Ctx())
	//)
	//
	//var (
	//	output = &bytes.Buffer{}
	//)
	//
	//// 1- list accounts
	//// 2- send tokens from one to other.
	//// 3- verify tx by using gRPC Gateway API.
	//steps := step.NewSteps(
	//	step.New(
	//		step.Exec(
	//			appname+"d",
	//			"config",
	//			"output", "json",
	//		),
	//		step.PreExec(func() error {
	//			return env.IsAppServed(ctx, host)
	//		}),
	//	),
	//	step.New(
	//		step.Exec(
	//			appname+"d",
	//			"keys",
	//			"list",
	//			"--keyring-backend", "test",
	//		),
	//		step.PostExec(func(execErr error) error {
	//			if execErr != nil {
	//				return execErr
	//			}
	//
	//			addresses := make([]string, 0)
	//
	//			// collect addresses of alice and bob.
	//			var accounts []struct {
	//				Name    string `json:"name"`
	//				Address string `json:"address"`
	//			}
	//			if err := json.NewDecoder(output).Decode(&accounts); err != nil {
	//				return err
	//			}
	//			for _, account := range accounts {
	//				if account.Name == "alice" || account.Name == "bob" {
	//					addresses = append(addresses, account.Address)
	//				}
	//			}
	//			if len(addresses) != 2 {
	//				return errors.New("expected alice and bob accounts to be created")
	//			}
	//
	//			// send some tokens from alice to bob and confirm the corresponding tx via gRPC gateway
	//			// endpoint by asserting denom and amount.
	//			return cmdrunner.New().Run(ctx, step.New(
	//				step.Exec(
	//					appname+"d",
	//					"tx",
	//					"bank",
	//					"send",
	//					addresses[0],
	//					addresses[1],
	//					"10"+stakeDenom,
	//					"--keyring-backend", "test",
	//					"--chain-id", appname,
	//					"--node", xurl.TCP(host.RPC),
	//					"--yes",
	//				),
	//				step.PreExec(func() error {
	//					output.Reset()
	//					return nil
	//				}),
	//				step.PostExec(func(execErr error) error {
	//					if execErr != nil {
	//						return execErr
	//					}
	//
	//					tx := struct {
	//						Hash string `json:"txHash"`
	//					}{}
	//					if err := json.NewDecoder(output).Decode(&tx); err != nil {
	//						return err
	//					}
	//
	//					addr := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", xurl.HTTP(host.API), tx.Hash)
	//					req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	//					if err != nil {
	//						return errors.Wrap(err, "call to get tx via gRPC gateway")
	//					}
	//					resp, err := http.DefaultClient.Do(req)
	//					if err != nil {
	//						return err
	//					}
	//					defer resp.Body.Close()
	//
	//					// Send error if the request failed
	//					if resp.StatusCode != http.StatusOK {
	//						return errors.New(resp.Status)
	//					}
	//
	//					if err := json.NewDecoder(resp.Body).Decode(&txBody); err != nil {
	//						return err
	//					}
	//					return nil
	//				}),
	//				step.Stdout(output),
	//			))
	//		}),
	//		step.Stdout(output),
	//	))
	//
	//go func() {
	//	defer cancel()
	//
	//	isTxBodyRetrieved = env.Exec("retrieve account addresses", steps, envtest.ExecRetry())
	//}()
	//
	//env.Must(env.Serve("should serve", path, "", "", envtest.ExecCtx(ctx)))
	//
	//if !isTxBodyRetrieved {
	//	t.FailNow()
	//}
	//
	//require.Len(t, txBody.Tx.Body.Messages, 1)
	//require.Len(t, txBody.Tx.Body.Messages[0].Amount, 1)
	//require.Equal(t, "token", txBody.Tx.Body.Messages[0].Amount[0].Denom)
	//require.Equal(t, "10", txBody.Tx.Body.Messages[0].Amount[0].Amount)
}
