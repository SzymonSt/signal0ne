package alertmanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"signal0ne/internal/models"
	"signal0ne/internal/tools"
	"signal0ne/pkg/integrations/helpers"
	"strings"
)

var functions = map[string]models.WorkflowFunctionDefinition{
	"get_relevant_alerts": models.WorkflowFunctionDefinition{
		Function: getRelevantAlerts,
		Input:    GetRelevantAlertsInput{},
	},
}

type AlertmanagerIntegration struct {
	models.Integration `json:",inline" bson:",inline"`
	Config             `json:",inline" bson:",inline"`
}

func (integration AlertmanagerIntegration) Execute(
	input any,
	output map[string]string,
	functionName string) ([]map[string]any, error) {

	var results []map[string]any

	function, ok := functions[functionName]
	if !ok {
		return results, fmt.Errorf("%s.%s: cannot find requested function", integration.Name, functionName)
	}

	intermediateResults, err := function.Function(input, integration)
	if err != nil {
		return results, fmt.Errorf("%s.%s:%v", integration.Name, functionName, err)
	}

	results = tools.ExecutionResultWrapper(intermediateResults, output)

	return results, nil
}

func (integration AlertmanagerIntegration) Validate() error {
	if integration.Config.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if integration.Config.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}
	return nil
}

func (integration AlertmanagerIntegration) ValidateStep(
	input any,
	functionName string,
) error {
	function, exists := functions[functionName]
	if !exists {
		return fmt.Errorf("cannot find selected function")
	}

	err := helpers.ValidateInputParameters(input, function.Input, functionName)
	if err != nil {
		return err
	}

	return nil
}

type GetRelevantAlertsInput struct {
	Filter string `json:"filter"`
}

func getRelevantAlerts(input any, integration any) ([]any, error) {
	var parsedInput GetRelevantAlertsInput
	var output []any

	err := helpers.ValidateInputParameters(input, &parsedInput, "compare_traces")
	if err != nil {
		return output, err
	}

	fmt.Printf("Executing Jaeger integration function...")

	assertedIntegration := integration.(AlertmanagerIntegration)

	host := assertedIntegration.Host
	port := assertedIntegration.Port
	apiPath := "/api/v2/alerts?"
	url := fmt.Sprintf("http://%s:%s%s", host, port, apiPath)

	filters := strings.Split(parsedInput.Filter, ",")

	alerts, err := getAlerts(url, filters)
	if err != nil {
		return output, err
	}

	for _, alert := range alerts {
		source, exists := alert.(map[string]any)["labels"].(map[string]any)["name"].(string)
		if !exists {
			source = ""
		}
		output = append(output, map[string]any{
			"alert":         alert,
			"output_source": source,
		})
	}

	return output, nil
}

func getAlerts(url string, filters []string) ([]any, error) {
	client := &http.Client{}

	if !(len(filters) > 0) {
		return nil, nil
	} else {
		for fi, filter := range filters {
			if filter != "" {
				if fi != 0 {
					url += "&"
				}
				url += ("filter=" + filter)
			}
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var bodyHandler []any

	if resp.StatusCode != 200 {
		err = fmt.Errorf("%s", resp.Status)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &bodyHandler)
	if err != nil {
		err = fmt.Errorf("cannot parse response body, error %v", err)
		return nil, err
	}

	return bodyHandler, nil
}