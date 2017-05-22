package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	tasksCmd.AddCommand(infoTaskCmd)
}

var infoTaskCmd = &cobra.Command{
	Use:   "info <DeploymentId> <TaskId>",
	Short: "Get information about a deployment task",
	Long:  `Display information about a given task specifying the deployment id and the task id.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("Expecting a deployment id and a task id (got %d parameters)", len(args))
		}
		client, err := getClient()
		if err != nil {
			errExit(err)
		}

		url := "/deployments/" + args[0] + "/tasks/" + args[1]
		request, err := client.NewRequest("GET", url, nil)
		if err != nil {
			errExit(err)
		}

		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			errExit(err)
		}
		if response.StatusCode != 200 {
			// Try to get the reason
			printErrors(response.Body)
			errExit(fmt.Errorf("Expecting HTTP Status code 200 got %d, reason %q", response.StatusCode, response.Status))
		}
		var task rest.Task
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &task)
		if err != nil {
			errExit(err)
		}
		fmt.Println("Task: ", task.ID)
		fmt.Println("Task status:", task.Status)
		fmt.Println("Task type:", task.Type)

		return nil
	},
}