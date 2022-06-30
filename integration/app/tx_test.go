//go:build !relayer
// +build !relayer

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ignite/cli/ignite/pkg/cmdrunner"
	"github.com/ignite/cli/ignite/pkg/cmdrunner/step"
	"github.com/ignite/cli/ignite/pkg/randstr"
	"github.com/ignite/cli/ignite/pkg/xurl"
	envtest "github.com/ignite/cli/integration"
)

func TestGetTxViaGRPCGateway(t *testing.T) {
	var (
		env         = envtest.New(t)
		appname     = randstr.Runes(10)
		path        = env.Scaffold(fmt.Sprintf("github.com/test/%s", appname))
		host        = env.RandomizeServerPorts(path, "")
		ctx, cancel = context.WithCancel(env.Ctx())
	)

	var (
		output            = &bytes.Buffer{}
		isTxBodyRetrieved bool
		txBody            = struct {
			Tx struct {
				Body struct {
					Messages []struct {
						Amount []struct {
							Denom  string `json:"denom"`
							Amount string `json:"amount"`
						} `json:"amount"`
					} `json:"messages"`
				} `json:"body"`
			} `json:"tx"`
		}{}
	)

	// 1- list accounts
	// 2- send tokens from one to other.
	// 3- verify tx by using gRPC Gateway API.
	steps := step.NewSteps(
		step.New(
			step.Exec(
				appname+"d",
				"config",
				"output", "json",
			),
			step.PreExec(func() error {
				return env.IsAppServed(ctx, host)
			}),
		),
		step.New(
			step.Exec(
				appname+"d",
				"keys",
				"list",
				"--keyring-backend", "test",
			),
			step.PostExec(func(execErr error) error {
				if execErr != nil {
					return execErr
				}

				addresses := []string{}

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

				nodeAddr, err := xurl.TCP(host.RPC)
				if err != nil {
					return err
				}

				// send some tokens from alice to bob and confirm the corresponding tx via gRPC gateway
				// endpoint by asserting denom and amount.
				return cmdrunner.New().Run(ctx, step.New(
					step.Exec(
						appname+"d",
						"tx",
						"bank",
						"send",
						addresses[0],
						addresses[1],
						"10token",
						"--keyring-backend", "test",
						"--chain-id", appname,
						"--node", nodeAddr,
						"--yes",
					),
					step.PreExec(func() error {
						output.Reset()
						return nil
					}),
					step.PostExec(func(execErr error) error {
						if execErr != nil {
							return execErr
						}

						tx := struct {
							Hash string `json:"txHash"`
						}{}
						if err := json.NewDecoder(output).Decode(&tx); err != nil {
							return err
						}

						apiAddr, err := xurl.HTTP(host.API)
						if err != nil {
							return err
						}

						addr := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", apiAddr, tx.Hash)
						req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
						if err != nil {
							return errors.Wrap(err, "call to get tx via gRPC gateway")
						}
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							return err
						}
						defer resp.Body.Close()

						// Send error if the request failed
						if resp.StatusCode != http.StatusOK {
							return errors.New(resp.Status)
						}

						if err := json.NewDecoder(resp.Body).Decode(&txBody); err != nil {
							return err
						}
						return nil
					}),
					step.Stdout(output),
				))
			}),
			step.Stdout(output),
		))

	go func() {
		defer cancel()

		isTxBodyRetrieved = env.Exec("retrieve account addresses", steps, envtest.ExecRetry())
	}()

	env.Must(env.Serve("should serve", path, "", "", envtest.ExecCtx(ctx)))

	if !isTxBodyRetrieved {
		t.FailNow()
	}

	require.Len(t, txBody.Tx.Body.Messages, 1)
	require.Len(t, txBody.Tx.Body.Messages[0].Amount, 1)
	require.Equal(t, "token", txBody.Tx.Body.Messages[0].Amount[0].Denom)
	require.Equal(t, "10", txBody.Tx.Body.Messages[0].Amount[0].Amount)
}
