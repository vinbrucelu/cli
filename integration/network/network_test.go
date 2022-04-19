//go:build !relayer
// +build !relayer

package network_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/pkg/errors"

	"github.com/ignite-hq/cli/ignite/chainconfig"

	"github.com/ignite-hq/cli/ignite/pkg/cmdrunner/step"
	envtest "github.com/ignite-hq/cli/integration"
)

var (
	chainSource    = "https://github.com/Pantani/mars"
	appName        = "marsd"
	spnCoordinator = "alice"
	spnValidator1  = "bob"
	spnValidator2  = "carol"
	spnValidator3  = "dave"
	defaultOptions = []envtest.ExecOption{
		envtest.ExecStdout(bufio.NewWriter(os.Stdout)),
		envtest.ExecStderr(bufio.NewWriter(os.Stderr)),
	}
)

func TestGenerateAnAppWithStargateWithListAndVerify(t *testing.T) {
	var (
		env         = envtest.New(t)
		path        = env.TmpDir()
		ctx, cancel = context.WithTimeout(env.Ctx(), envtest.ServeTimeout)
	)
	defer cancel()

	// check the Ignite blockchain and chains are served
	//env.Must(env.Exec("publish a chain",
	//	step.NewSteps(step.New(
	//		step.Exec(envtest.IgniteApp,
	//			appName,
	//			"config",
	//			"output", "json",
	//		),
	//		step.PreExec(func() error {
	//			return env.IsAppServed(ctx, host)
	//		}),
	//	))),
	//)

	t.Run("publish", func(t *testing.T) {
		env.Must(env.Exec("publish a chain",
			step.NewSteps(step.New(
				step.Exec(envtest.IgniteApp,
					"network",
					"--local",
					"chain",
					"publish",
					chainSource,
					"--from",
					spnCoordinator,
					"--keyring-backend", "test",
				),
				step.Workdir(path),
			)),
			append(defaultOptions, envtest.ExecCtx(ctx))...,
		))
	})
	t.Run("init", func(t *testing.T) {
		env.Must(env.Exec("init a chain",
			step.NewSteps(step.New(
				step.Exec(envtest.IgniteApp,
					"network",
					"--local",
					"chain",
					"init",
					"1",
					"--default-values",
					"--overwrite-home",
					"--from",
					spnCoordinator,
					"--keyring-backend", "test",
				),
				step.Workdir(path),
			)),
			append(defaultOptions, envtest.ExecCtx(ctx))...,
		))
	})

	var wg sync.WaitGroup
	for i, validator := range []string{spnValidator1, spnValidator1, spnValidator2, spnValidator3} {
		t.Run(fmt.Sprintf("join with %s", validator), func(t *testing.T) {
			wg.Add(1)
			go func() {
				validator := validator
				i := i
				defer wg.Done()
				env.Must(env.Exec(
					fmt.Sprintf("join as validator %d (%s)", i, validator),
					validatorStep(validator),
					append(defaultOptions, envtest.ExecCtx(ctx))...,
				))
			}()
		})
	}
	wg.Wait()

	t.Run("reject requests", func(t *testing.T) {
		env.Must(env.Exec("reject requests",
			step.NewSteps(step.New(
				step.Exec(envtest.IgniteApp,
					"network",
					"--local",
					"request",
					"reject",
					"1",
					"1-2",
					"--from",
					spnCoordinator,
					"--keyring-backend", "test",
				),
				step.Workdir(path),
			)),
			append(defaultOptions, envtest.ExecCtx(ctx))...,
		))
	})

	t.Run("approve requests", func(t *testing.T) {
		env.Must(env.Exec("approve requests",
			step.NewSteps(step.New(
				step.Exec(envtest.IgniteApp,
					"network",
					"--local",
					"request",
					"approve",
					"1",
					"3-6",
					"--from",
					spnCoordinator,
					"--keyring-backend", "test",
				),
				step.Workdir(path),
			)),
			append(defaultOptions, envtest.ExecCtx(ctx))...,
		))
	})
}

func Hosts(ctx context.Context, env envtest.Env) chainconfig.Host {
	return chainconfig.Host{
		RPC:     "",
		P2P:     "",
		Prof:    "",
		GRPC:    "",
		GRPCWeb: "",
		API:     "http://0.0.0.0:1317",
	}
	// Tendermint node: http://0.0.0.0:26657
	// Token faucet: http://0.0.0.0:4500
}
func createAccount(ctx context.Context, env envtest.Env) step.Steps {
	var output = &bytes.Buffer{}
	return step.NewSteps(
		step.New(
			step.Exec(
				envtest.BlockchainApp,
				"config",
				"output", "json",
			),
			//step.PreExec(func() error {
			//	return env.IsAppServed(ctx, nil)
			//}),
		),
		step.New(
			step.Exec(
				envtest.BlockchainApp,
				"keys",
				"list",
				"--keyring-backend", "test",
			),
			step.PostExec(func(execErr error) error {
				if execErr != nil {
					return execErr
				}

				addresses := make([]string, 0)

				// collect addresses of alice and bob.
				var accounts []struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				}
				if err := json.NewDecoder(output).Decode(&accounts); err != nil {
					return err
				}
				for _, account := range accounts {
					if account.Name == "alice" || account.Name == "bob" {
						addresses = append(addresses, account.Address)
					}
				}
				if len(addresses) != 2 {
					return errors.New("expected alice and bob accounts to be created")
				}
				return nil
			}),
			step.Stdout(output),
		))
}

func validatorStep(from string) step.Steps {
	return step.NewSteps(step.New(
		step.Exec(envtest.IgniteApp,
			"account",
			"create",
			from,
		),
		step.Exec(envtest.IgniteApp,
			"network",
			"--local",
			"chain",
			"join",
			"1",
			"--amount",
			"95000001stake",
			"--default-peer",
			"--from",
			from,
			"--keyring-backend", "test",
		),
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
