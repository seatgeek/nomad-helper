package helpers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/mitchellh/colorstring"
	cli "github.com/urfave/cli"
)

func FindAllocation(c *cli.Context, client *api.Client) (*api.Allocation, error) {
	if allocID := c.String("alloc"); allocID != "" {
		return FindAllocationByPrefix(allocID, client)
	}

	if jobID := c.String("job"); jobID != "" {
		return FindAllocationFromScratch(jobID, client)
	}

	return FindAllocationFromScratch("", client)
}

func FindAllocationByPrefix(allocID string, client *api.Client) (*api.Allocation, error) {
	if len(allocID) == 36 {
		return FindAllocationByID(allocID, client)
	}

	allocations, _, err := client.Allocations().List(&api.QueryOptions{Prefix: allocID})
	if err != nil {
		return nil, err
	}

	if len(allocations) == 1 {
		colorstring.Printf("[green]* Autoselected allocation '%s'\n", allocations[0].ID)
		return FindAllocationByID(allocations[0].ID, client)
	}

	if len(allocations) == 0 {
		return nil, fmt.Errorf("No allocations found with prefix: %s", allocID)
	}

	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].JobID < allocations[j].JobID
	})

	for {
		colorstring.Println("[yellow]? Please select an allocation:")
		for i, alloc := range allocations {
			colorstring.Printf(" [bold]%2d)[reset] %s - %s @ %s\n", i, alloc.ID[0:8], alloc.Name, getClientName(alloc.NodeID, client))
		}

		colorstring.Printf("[yellow]! Pick a number: ")
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("Unable to read input")
		}

		val, err := strconv.Atoi(strings.Trim(text, "\n"))
		if err != nil {
			fmt.Printf("Error, not a valid number: %s", err)
			continue
		}

		for i, alloc := range allocations {
			if i == val {
				colorstring.Printf("[green]* Selected [bold]%s - %s @ %s\n", alloc.ID[0:8], alloc.Name, getClientName(alloc.NodeID, client))
				return FindAllocationByID(alloc.ID, client)
			}
		}

		colorstring.Println("[red]Error, not a valid allocation selection!")
	}
}

func FindAllocationByID(allocID string, client *api.Client) (*api.Allocation, error) {
	alloc, _, err := client.Allocations().Info(allocID, nil)
	if err != nil {
		return nil, err
	}

	return alloc, nil
}

func FindAllocationByJob(jobID string, client *api.Client) (*api.Allocation, error) {
	if _, _, err := client.Jobs().Info(jobID, nil); err != nil {
		return nil, fmt.Errorf("Could not look up job, maybe it doesn't exist?")
	}

	allocs, _, err := client.Jobs().Allocations(jobID, false, nil)
	if err != nil {
		return nil, err
	}

	filtered := make([]*api.AllocationListStub, 0)
	for _, alloc := range allocs {
		if alloc.ClientStatus != "running" {
			continue
		}

		filtered = append(filtered, alloc)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("Error, no allocations found for this job")
	}

	if len(filtered) == 1 {
		colorstring.Printf("[green]* Autoselected allocation '%s'\n", filtered[0].ID)
		return FindAllocationByID(filtered[0].ID, client)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	for {
		colorstring.Println("[yellow]? Please select an allocation:")
		for i, alloc := range filtered {
			colorstring.Printf(" [bold]%2d)[reset] %s - %s @ %s\n", i, alloc.ID[0:8], alloc.Name, getClientName(alloc.NodeID, client))
		}

		colorstring.Printf("[yellow]! Pick a number: ")
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("Unable to read input")
		}

		val, err := strconv.Atoi(strings.Trim(text, "\n"))
		if err != nil {
			fmt.Printf("Error, not a valid number: %s", err)
			continue
		}

		for i, alloc := range filtered {
			if i == val {
				colorstring.Printf("[green]* Selected [bold]%s - %s @ %s\n", alloc.ID[0:8], alloc.Name, getClientName(alloc.NodeID, client))
				return FindAllocationByID(alloc.ID, client)
			}
		}

		colorstring.Println("[red]Error, not a valid allocation selection!")
	}
}

func FindAllocationFromScratch(prefix string, client *api.Client) (*api.Allocation, error) {
	jobs, _, err := client.Jobs().List(&api.QueryOptions{Prefix: prefix})
	if err != nil {
		return nil, err
	}

	filtered := make([]*api.JobListStub, 0)
	for _, job := range jobs {
		if job.Status != "running" {
			continue
		}

		filtered = append(filtered, job)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("Error, no jobs found with prefix: '%s'", prefix)
	}

	if len(filtered) == 1 {
		colorstring.Printf("[green]* Autoselected job '%s'\n", filtered[0].ID)
		return FindAllocationByJob(filtered[0].ID, client)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})

	for {
		colorstring.Println("[yellow]? Please select a job:")
		for i, job := range filtered {
			colorstring.Printf(" [bold]%3d[reset]) %s\n", i, job.ID)
		}

		colorstring.Printf("[yellow]! Pick a number: ")
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("Unable to read input")
		}

		val, err := strconv.Atoi(strings.Trim(text, "\n"))
		if err != nil {
			colorstring.Printf("[red]Error, not a valid number:[reset][bold] %s\n", err)
			continue
		}

		for i, job := range filtered {
			if i == val {
				colorstring.Printf("[green]* Selected [bold]%s[reset]\n", job.ID)
				return FindAllocationByJob(job.ID, client)
			}
		}

		log.Println("Error, not a valid job selection!")
	}
}

func FindTask(alloc *api.Allocation, taskName string) (string, error) {
	if len(alloc.TaskStates) == 0 {
		return "", fmt.Errorf("Error, no tasks found for this allocation")
	}

	if len(alloc.TaskStates) == 1 {
		for name := range alloc.TaskStates {
			colorstring.Printf("[green]* Autoselected task '%s'\n", name)
			return name, nil
		}
	}

	if taskName != "" {
		for name := range alloc.TaskStates {
			if name == taskName {
				return taskName, nil
			}
		}
	}

	for {
		options := make([]string, 0)

		colorstring.Println("[yellow]> Please select a task")
		for name := range alloc.TaskStates {
			colorstring.Printf(" [bold]%2d[reset]) %s\n", len(options), name)
			options = append(options, name)
		}

		colorstring.Print("[yellow]! Pick a number: ")
		reader := bufio.NewReader(os.Stdin)

		text, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("Unable to read input")
		}

		val, err := strconv.Atoi(strings.Trim(text, "\n"))
		if err != nil {
			colorstring.Printf("[red]Error, not a valid number:[reset][bold] %s\n", err)
			continue
		}

		for i, taskName := range options {
			if i == val {
				return taskName, nil
			}
		}

		colorstring.Println("[red]! error, not a valid task selection!")
	}
}

var clientCache map[string]string

func getClientName(ID string, client *api.Client) string {
	if clientCache == nil {
		clientCache = make(map[string]string)
		nodes, _, _ := client.Nodes().List(nil)
		for _, node := range nodes {
			clientCache[node.ID] = node.Name
		}
	}

	if name, ok := clientCache[ID]; ok {
		return name
	}

	return "could not look up name"
}
