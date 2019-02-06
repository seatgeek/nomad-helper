package attach

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/nomad/api"
	"github.com/mitchellh/colorstring"
	"github.com/seatgeek/nomad-helper/helpers"
	cli "gopkg.in/urfave/cli.v1"
)

func Run(c *cli.Context) error {
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	alloc, err := helpers.FindAllocation(c, nomadClient)
	if err != nil {
		return err
	}

	node, _, err := nomadClient.Nodes().Info(alloc.NodeID, nil)
	if err != nil {
		return err
	}
	ip := node.Attributes["unique.network.ip-address"]

	if c.Bool("host") {
		colorstring.Println("[green]* Connecting to docker host...")
		return connect([]string{"-t", ip, "sudo su root"})
	}

	taskName, err := helpers.FindTask(alloc, c.String("task"))
	if err != nil {
		return err
	}

	colorstring.Printf("[green]* Going to attach to task '%s' on '%s' with command '%s'\n", taskName, node.Name, c.String("command"))
	return connect([]string{"-t", ip, fmt.Sprintf("sudo docker exec -it %s-%s %s", taskName, alloc.ID, c.String("command"))})
}

func connect(args []string) error {
	fmt.Println()

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println()
	return nil
}
