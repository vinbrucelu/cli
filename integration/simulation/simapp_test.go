//go:build !relayer
// +build !relayer

package simulation_test

import (
	"testing"

	"github.com/ignite/cli/ignite/pkg/cmdrunner/step"
	envtest "github.com/ignite/cli/integration"
)

func TestGenerateAnAppAndSimulate(t *testing.T) {
	var (
		env  = envtest.New(t)
		path = env.Scaffold("github.com/test/blog")
	)

	env.Must(env.Exec("create a list",
		step.NewSteps(step.New(
			step.Exec(envtest.IgniteApp, "s", "list", "--yes", "foo", "foobar"),
			step.Workdir(path),
		)),
	))

	env.Must(env.Exec("create an singleton type",
		step.NewSteps(step.New(
			step.Exec(envtest.IgniteApp, "s", "single", "--yes", "baz", "foobar"),
			step.Workdir(path),
		)),
	))

	env.Must(env.Exec("create an singleton type",
		step.NewSteps(step.New(
			step.Exec(envtest.IgniteApp, "s", "list", "--yes", "noSimapp", "foobar", "--no-simulation"),
			step.Workdir(path),
		)),
	))

	env.Must(env.Exec("create a message",
		step.NewSteps(step.New(
			step.Exec(envtest.IgniteApp, "s", "message", "--yes", "msgFoo", "foobar"),
			step.Workdir(path),
		)),
	))

	env.Must(env.Exec("scaffold a new module",
		step.NewSteps(step.New(
			step.Exec(envtest.IgniteApp, "s", "module", "--yes", "new_module"),
			step.Workdir(path),
		)),
	))

	env.Must(env.Exec("create a map",
		step.NewSteps(step.New(
			step.Exec(
				envtest.IgniteApp,
				"s",
				"map",
				"--yes",
				"bar",
				"foobar",
				"--module",
				"new_module",
			),
			step.Workdir(path),
		)),
	))

	env.Simulate(path, 100, 50)
}
