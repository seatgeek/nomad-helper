package job

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/scheduler"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Move(c *cli.Context, logger *log.Logger) error {
	jobName := c.Args().First()

	// Sanity check
	if jobName == "" {
		return fmt.Errorf("must provide a job name or prefix")
	}
	if c.String("constraint") == "" {
		return fmt.Errorf("must provide new constrain name")
	}
	if c.String("operand") == "" {
		return fmt.Errorf("must provide new constrain name")
	}
	if c.String("value") == "" {
		return fmt.Errorf("must provide new constrain name")
	}

	newConstraint := api.NewConstraint(fmt.Sprintf("${%s}", c.String("constraint")), c.String("operand"), c.String("value"))

	// create Nomad API client
	nomadClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	jobsToMove := []string{jobName}

	// if we stop by prefix, then query for all the jobs
	if c.Bool("as-prefix") {
		jobsToMove = []string{}

		jobs, _, err := nomadClient.Jobs().List(&api.QueryOptions{Prefix: jobName})
		if err != nil {
			return err
		}

		for _, job := range jobs {
			if excludeFilter := c.String("exclude"); excludeFilter != "" && strings.Contains(job.Name, excludeFilter) {
				log.Infof("Excluding job %s because it's name matched the exclude filter", job.Name)
				continue
			}
			jobsToMove = append(jobsToMove, job.ID)
		}
	}

	if len(jobsToMove) == 0 {
		return fmt.Errorf("could not find any jobs")
	}

	var wg sync.WaitGroup
	for _, jobName := range jobsToMove {
		logger.Infof("Going to move job %s", jobName)

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			job, _, err := nomadClient.Jobs().Info(name, nil)
			if err != nil {
				log.Error(err)
				return
			}
			if *job.Stop {
				log.Infof("Skipping job %s because it's stopped", *job.Name)
				return
			}
			existingConstraintAppended := false
			for constraintIndex, constraint := range job.Constraints {
				if constraint.LTarget == newConstraint.LTarget {
					job.Constraints[constraintIndex] = newConstraint
					existingConstraintAppended = true
				}
			}
			if !existingConstraintAppended {
				job.Constrain(newConstraint)
			}
			planResponse, _, err := nomadClient.Jobs().Plan(job, true, nil)
			if err != nil {
				log.Error(err)
				return
			}

			// Print the diff
			colorize := colorstring.Colorize{
				Colors:  colorstring.DefaultColors,
				Disable: false,
				Reset:   true,
			}
			log.Infof(colorize.Color(fmt.Sprintf("%s\n",
				strings.TrimSpace(formatJobDiff(planResponse.Diff, false)))))

			if c.Bool("dry") {
				logger.Infof("Skipping the changes to job %s because dry flag was provided", *job.Name)
				return
			}
			_, _, err = nomadClient.Jobs().Register(job, nil)
			if err != nil {
				log.Errorf("failed to update job %s: %s", name, err)
				return
			}
			log.Infof("Job %s was successfully moved!", name)
		}(jobName)
	}

	wg.Wait()

	return nil
}

// Everything below is shamelessly borrowed from https://github.com/hashicorp/nomad/blob/master/command/job_plan.go

// formatJobDiff produces an annotated diff of the job. If verbose mode is
// set, added or deleted task groups and tasks are expanded.
func formatJobDiff(job *api.JobDiff, verbose bool) string {
	marker, _ := getDiffString(job.Type)
	out := fmt.Sprintf("%s[bold]Job: %q\n", marker, job.ID)

	// Determine the longest markers and fields so that the output can be
	// properly aligned.
	longestField, longestMarker := getLongestPrefixes(job.Fields, job.Objects)
	for _, tg := range job.TaskGroups {
		if _, l := getDiffString(tg.Type); l > longestMarker {
			longestMarker = l
		}
	}

	// Only show the job's field and object diffs if the job is edited or
	// verbose mode is set.
	if job.Type == "Edited" || verbose {
		fo := alignedFieldAndObjects(job.Fields, job.Objects, 0, longestField, longestMarker)
		out += fo
		if len(fo) > 0 {
			out += "\n"
		}
	}

	// Print the task groups
	for _, tg := range job.TaskGroups {
		_, mLength := getDiffString(tg.Type)
		kPrefix := longestMarker - mLength
		out += fmt.Sprintf("%s\n", formatTaskGroupDiff(tg, kPrefix, verbose))
	}

	return out
}

// formatTaskGroupDiff produces an annotated diff of a task group. If the
// verbose field is set, the task groups fields and objects are expanded even if
// the full object is an addition or removal. tgPrefix is the number of spaces to prefix
// the output of the task group.
func formatTaskGroupDiff(tg *api.TaskGroupDiff, tgPrefix int, verbose bool) string {
	marker, _ := getDiffString(tg.Type)
	out := fmt.Sprintf("%s%s[bold]Task Group: %q[reset]", marker, strings.Repeat(" ", tgPrefix), tg.Name)

	// Append the updates and colorize them
	if l := len(tg.Updates); l > 0 {
		order := make([]string, 0, l)
		for updateType := range tg.Updates {
			order = append(order, updateType)
		}

		sort.Strings(order)
		updates := make([]string, 0, l)
		for _, updateType := range order {
			count := tg.Updates[updateType]
			var color string
			switch updateType {
			case scheduler.UpdateTypeIgnore:
			case scheduler.UpdateTypeCreate:
				color = "[green]"
			case scheduler.UpdateTypeDestroy:
				color = "[red]"
			case scheduler.UpdateTypeMigrate:
				color = "[blue]"
			case scheduler.UpdateTypeInplaceUpdate:
				color = "[cyan]"
			case scheduler.UpdateTypeDestructiveUpdate:
				color = "[yellow]"
			case scheduler.UpdateTypeCanary:
				color = "[light_yellow]"
			}
			updates = append(updates, fmt.Sprintf("[reset]%s%d %s", color, count, updateType))
		}
		out += fmt.Sprintf(" (%s[reset])\n", strings.Join(updates, ", "))
	} else {
		out += "[reset]\n"
	}

	// Determine the longest field and markers so the output is properly
	// aligned
	longestField, longestMarker := getLongestPrefixes(tg.Fields, tg.Objects)
	for _, task := range tg.Tasks {
		if _, l := getDiffString(task.Type); l > longestMarker {
			longestMarker = l
		}
	}

	// Only show the task group's field and object diffs if the group is edited or
	// verbose mode is set.
	subStartPrefix := tgPrefix + 2
	if tg.Type == "Edited" || verbose {
		fo := alignedFieldAndObjects(tg.Fields, tg.Objects, subStartPrefix, longestField, longestMarker)
		out += fo
		if len(fo) > 0 {
			out += "\n"
		}
	}

	// Output the tasks
	for _, task := range tg.Tasks {
		_, mLength := getDiffString(task.Type)
		prefix := longestMarker - mLength
		out += fmt.Sprintf("%s\n", formatTaskDiff(task, subStartPrefix, prefix, verbose))
	}

	return out
}

// formatTaskDiff produces an annotated diff of a task. If the verbose field is
// set, the tasks fields and objects are expanded even if the full object is an
// addition or removal. startPrefix is the number of spaces to prefix the output of
// the task and taskPrefix is the number of spaces to put between the marker and
// task name output.
func formatTaskDiff(task *api.TaskDiff, startPrefix, taskPrefix int, verbose bool) string {
	marker, _ := getDiffString(task.Type)
	out := fmt.Sprintf("%s%s%s[bold]Task: %q",
		strings.Repeat(" ", startPrefix), marker, strings.Repeat(" ", taskPrefix), task.Name)
	if len(task.Annotations) != 0 {
		out += fmt.Sprintf(" [reset](%s)", colorAnnotations(task.Annotations))
	}

	if task.Type == "None" {
		return out
	} else if (task.Type == "Deleted" || task.Type == "Added") && !verbose {
		// Exit early if the job was not edited and it isn't verbose output
		return out
	} else {
		out += "\n"
	}

	subStartPrefix := startPrefix + 2
	longestField, longestMarker := getLongestPrefixes(task.Fields, task.Objects)
	out += alignedFieldAndObjects(task.Fields, task.Objects, subStartPrefix, longestField, longestMarker)
	return out
}

// getDiffString returns a colored diff marker and the length of the string
// without color annotations.
func getDiffString(diffType string) (string, int) {
	switch diffType {
	case "Added":
		return "[green]+[reset] ", 2
	case "Deleted":
		return "[red]-[reset] ", 2
	case "Edited":
		return "[light_yellow]+/-[reset] ", 4
	default:
		return "", 0
	}
}

// getLongestPrefixes takes a list  of fields and objects and determines the
// longest field name and the longest marker.
func getLongestPrefixes(fields []*api.FieldDiff, objects []*api.ObjectDiff) (longestField, longestMarker int) {
	for _, field := range fields {
		if l := len(field.Name); l > longestField {
			longestField = l
		}
		if _, l := getDiffString(field.Type); l > longestMarker {
			longestMarker = l
		}
	}
	for _, obj := range objects {
		if _, l := getDiffString(obj.Type); l > longestMarker {
			longestMarker = l
		}
	}
	return longestField, longestMarker
}

// alignedFieldAndObjects is a helper method that prints fields and objects
// properly aligned.
func alignedFieldAndObjects(fields []*api.FieldDiff, objects []*api.ObjectDiff,
	startPrefix, longestField, longestMarker int) string {

	var out string
	numFields := len(fields)
	numObjects := len(objects)
	haveObjects := numObjects != 0
	for i, field := range fields {
		_, mLength := getDiffString(field.Type)
		kPrefix := longestMarker - mLength
		vPrefix := longestField - len(field.Name)
		out += formatFieldDiff(field, startPrefix, kPrefix, vPrefix)

		// Avoid a dangling new line
		if i+1 != numFields || haveObjects {
			out += "\n"
		}
	}

	for i, object := range objects {
		_, mLength := getDiffString(object.Type)
		kPrefix := longestMarker - mLength
		out += formatObjectDiff(object, startPrefix, kPrefix)

		// Avoid a dangling new line
		if i+1 != numObjects {
			out += "\n"
		}
	}

	return out
}

// formatObjectDiff produces an annotated diff of an object. startPrefix is the
// number of spaces to prefix the output of the object and keyPrefix is the number
// of spaces to put between the marker and object name output.
func formatObjectDiff(diff *api.ObjectDiff, startPrefix, keyPrefix int) string {
	start := strings.Repeat(" ", startPrefix)
	marker, markerLen := getDiffString(diff.Type)
	out := fmt.Sprintf("%s%s%s%s {\n", start, marker, strings.Repeat(" ", keyPrefix), diff.Name)

	// Determine the length of the longest name and longest diff marker to
	// properly align names and values
	longestField, longestMarker := getLongestPrefixes(diff.Fields, diff.Objects)
	subStartPrefix := startPrefix + keyPrefix + 2
	out += alignedFieldAndObjects(diff.Fields, diff.Objects, subStartPrefix, longestField, longestMarker)

	endprefix := strings.Repeat(" ", startPrefix+markerLen+keyPrefix)
	return fmt.Sprintf("%s\n%s}", out, endprefix)
}

// formatFieldDiff produces an annotated diff of a field. startPrefix is the
// number of spaces to prefix the output of the field, keyPrefix is the number
// of spaces to put between the marker and field name output and valuePrefix is
// the number of spaces to put infront of the value for aligning values.
func formatFieldDiff(diff *api.FieldDiff, startPrefix, keyPrefix, valuePrefix int) string {
	marker, _ := getDiffString(diff.Type)
	out := fmt.Sprintf("%s%s%s%s: %s",
		strings.Repeat(" ", startPrefix),
		marker, strings.Repeat(" ", keyPrefix),
		diff.Name,
		strings.Repeat(" ", valuePrefix))

	switch diff.Type {
	case "Added":
		out += fmt.Sprintf("%q", diff.New)
	case "Deleted":
		out += fmt.Sprintf("%q", diff.Old)
	case "Edited":
		out += fmt.Sprintf("%q => %q", diff.Old, diff.New)
	default:
		out += fmt.Sprintf("%q", diff.New)
	}

	// Color the annotations where possible
	if l := len(diff.Annotations); l != 0 {
		out += fmt.Sprintf(" (%s)", colorAnnotations(diff.Annotations))
	}

	return out
}

// colorAnnotations returns a comma concatenated list of the annotations where
// the annotations are colored where possible.
func colorAnnotations(annotations []string) string {
	l := len(annotations)
	if l == 0 {
		return ""
	}

	colored := make([]string, l)
	for i, annotation := range annotations {
		switch annotation {
		case "forces create":
			colored[i] = fmt.Sprintf("[green]%s[reset]", annotation)
		case "forces destroy":
			colored[i] = fmt.Sprintf("[red]%s[reset]", annotation)
		case "forces in-place update":
			colored[i] = fmt.Sprintf("[cyan]%s[reset]", annotation)
		case "forces create/destroy update":
			colored[i] = fmt.Sprintf("[yellow]%s[reset]", annotation)
		default:
			colored[i] = annotation
		}
	}

	return strings.Join(colored, ", ")
}
