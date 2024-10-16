package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"signal0ne/internal/db"
	"signal0ne/internal/models"
	"signal0ne/internal/tools"
	"signal0ne/pkg/integrations/helpers"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

type OpenAIIntegrationInventory struct {
	AlertsCollection *mongo.Collection `json:"-" bson:"-"`
}

func NewOpenAIIntegrationInventory(
	alertsCollection *mongo.Collection) OpenAIIntegrationInventory {
	return OpenAIIntegrationInventory{
		AlertsCollection: alertsCollection,
	}
}

type OpenaiIntegration struct {
	models.Integration `json:",inline" bson:",inline"`
	Config             `json:"config" bson:"config"`
	Inventory          OpenAIIntegrationInventory
}

var functions = map[string]models.WorkflowFunctionDefinition{
	"summarize_context": models.WorkflowFunctionDefinition{
		Function:   summarizeContext,
		Input:      SummarizeContextInput{},
		OutputTags: []string{"copilot"},
	},
}

func (integration OpenaiIntegration) Execute(
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

	results = tools.ExecutionResultWrapper(intermediateResults, output, function.OutputTags)

	return results, nil
}

func (integration OpenaiIntegration) Initialize() map[string]string {
	// Implement your config initialization here
	return nil
}

func (integration OpenaiIntegration) Validate() error {
	// Implement your config validation here
	return nil
}

func (integration OpenaiIntegration) ValidateStep(
	input any,
	functionName string,
) error {
	function, exists := functions[functionName]
	if !exists {
		return fmt.Errorf("cannot find selected function")
	}

	//Validate input parameters for the chosen function
	err := helpers.ValidateInputParameters(input, function.Input, functionName)
	if err != nil {
		return err
	}

	return nil
}

type SummarizeContextInput struct {
	Context string `json:"context"`
}

func summarizeContext(input any, integration any) ([]any, error) {
	var parsedInput SummarizeContextInput
	var output []any
	var alertContext models.EnrichedAlert
	var partialSummaries = make([]string, 0)

	var tagContextGroups = map[string][]map[string]any{
		"code":     make([]map[string]any, 0),
		"logs":     make([]map[string]any, 0),
		"alerts":   make([]map[string]any, 0),
		"metadata": make([]map[string]any, 0),
	}

	err := helpers.ValidateInputParameters(input, &parsedInput, "summarize_context")
	if err != nil {
		return output, err
	}

	assertedIntegration := integration.(OpenaiIntegration)

	fmt.Printf("###\nExecuting OpenAi integration function...\n")

	contextBytes := []byte(parsedInput.Context)

	err = json.Unmarshal(contextBytes, &alertContext)
	if err != nil {
		return output, fmt.Errorf("error parsing input alert context: %v", err)
	}

	for _, contextValue := range alertContext.AdditionalContext {
		for _, partialContext := range contextValue.([]any) {
			tags, ok := partialContext.(map[string]any)["tags"].([]any)
			if !ok {
				continue
			}
			mainTag := tags[0].(string)
			switch mainTag {
			case "metadata":
				fmt.Printf("Metadata found %v", partialContext)
				tagContextGroups["metadata"] = append(tagContextGroups["metadata"], partialContext.(map[string]any))
			case "code":
				fmt.Printf("Code found %v", partialContext)
				tagContextGroups["code"] = append(tagContextGroups["code"], partialContext.(map[string]any))
			case "logs":
				tagContextGroups["logs"] = append(tagContextGroups["logs"], partialContext.(map[string]any))
			case "alerts":
				fmt.Printf("Alert found %v", partialContext)
				id, ok := partialContext.(map[string]any)["alertId"].(string)
				if !ok {
					fmt.Printf("Error parsing alert id: %v", err)
					continue
				}
				alert, err := db.GetEnrichedAlertById(id, context.Background(), assertedIntegration.Inventory.AlertsCollection)
				if err != nil {
					fmt.Printf("Error getting alert by id: %v", err)
					continue
				}
				partialContext.(map[string]any)["additional_context"] = alert.AdditionalContext
				tagContextGroups["alerts"] = append(tagContextGroups["alerts"], partialContext.(map[string]any))
			}
		}
	}

	model := assertedIntegration.Model
	apiKey := assertedIntegration.ApiKey

	if len(tagContextGroups["code"]) == 0 {
		delete(tagContextGroups, "code")
	}
	if len(tagContextGroups["logs"]) == 0 {
		delete(tagContextGroups, "logs")
	}
	if len(tagContextGroups["alerts"]) == 0 {
		delete(tagContextGroups, "alerts")
	}
	if len(tagContextGroups["metadata"]) == 0 {
		delete(tagContextGroups, "metadata")
	}

	if len(tagContextGroups) == 0 {
		return output, fmt.Errorf("no context found")
	}

	for contextKey, contextGroup := range tagContextGroups {

		jsonifiedContextGroup, err := json.Marshal(contextGroup)
		if err != nil {
			return output, fmt.Errorf("error parsing context group: %v", err)
		}

		prompt := fmt.Sprintf(`You are principal on-call engineer. 
		Based on these %s, draw some initial conclusions about the system and it's state and summarize the context for other engineers.
		It must fit in one paragraph.
		If you cannot draw any tangible conclusions, you must state that by saying "Not enough context" and just this.
		%s: %s`, contextKey, contextKey, jsonifiedContextGroup)

		localSum, err := callOpenAiApi(prompt, model, apiKey)
		if err != nil {
			return output, err
		}

		if strings.Contains(localSum, "Not enough context") {
			continue
		}

		partialSummaries = append(partialSummaries, localSum)

	}

	prompt := fmt.Sprintf(`You are a principal on-call engineer Based on the full context from the investigation summarize investigation context for other engineers.
		Response must contain one short paragraph of explanation of the probable root causes in full sentences. Try to correlate different contexts for an holistic overview.
		If you cannot draw any tangible conclusions, you must state that and give a reason.
		The full issue context and some initial reasoning: %s`, strings.Join(partialSummaries, "\n---"))

	summary, err := callOpenAiApi(prompt, model, apiKey)
	if err != nil {
		return output, err
	}
	fmt.Printf("Prompt %s ||||||||||||||||||||||||||||| Summary %s", prompt, summary)

	output = append(output, map[string]any{
		"summary": summary,
	})

	return output, nil
}

func callOpenAiApi(prompt string, model string, apiKey string) (string, error) {
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature"`
	}

	type Response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var response Response
	var apiURL = "https://api.openai.com/v1/chat/completions"

	client := &http.Client{}

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	reqBody, err := json.Marshal(Request{
		Model:       model,
		Messages:    messages,
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return response.Choices[0].Message.Content, nil
}
