package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/raphael-foliveira/htmbot/domain"
	"github.com/raphael-foliveira/htmbot/platform/slicesx"
)

type OpenAI struct {
	client openai.Client
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

func (o *OpenAI) StreamResponse(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.LLMTool,
	callback func(delta string),
) ([]domain.ChatMessage, error) {
	var (
		err              error
		hasFunctionCalls bool
	)

	openaiMessages := slicesx.Map(messages, o.chatMessageToOpenAIMessage)
	openaiMessages = o.removeOrphanToolReturns(openaiMessages)
	initialMessagesLength := len(openaiMessages)

	for {
		hasFunctionCalls = false
		stream := o.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: openaiMessages,
			},
			Model: "gpt-4o-mini",
			Tools: slicesx.Map(tools, func(tool domain.LLMTool) responses.ToolUnionParam {
				return responses.ToolUnionParam{
					OfFunction: &responses.FunctionToolParam{
						Name: tool.Name(),
						Description: param.Opt[string]{
							Value: tool.Description(),
						},
						Parameters: tool.Parameters(),
					},
				}
			}),
		})
		for stream.Next() {
			currentEvent := stream.Current()
			eventType := currentEvent.Type
			switch eventType {
			case "response.output_text.delta":
				delta := currentEvent.AsResponseOutputTextDelta()
				callback(delta.Delta)
			case "response.completed":
				completedEvent := currentEvent.AsResponseCompleted()
				response := &completedEvent.Response
				openaiMessages = append(openaiMessages, o.responsesResponseToInputItems(response)...)
				openaiMessages, hasFunctionCalls, err = o.handleResponse(ctx, tools, response, openaiMessages)
				if err != nil {
					return nil, fmt.Errorf("error handling response: %w", err)
				}
				if !hasFunctionCalls {
					return slicesx.Map(openaiMessages[initialMessagesLength:], o.openAIMessageToChatMessage), nil
				}
			}
		}
	}
}

func (o *OpenAI) removeOrphanToolReturns(messages []responses.ResponseInputItemUnionParam) []responses.ResponseInputItemUnionParam {
	for messages[0].OfFunctionCallOutput != nil {
		messages = messages[1:]
	}
	return messages
}

func (o *OpenAI) GenerateResponse(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.LLMTool,
) ([]domain.ChatMessage, error) {
	initialMessagesLength := len(messages)
	openaiMessages := slicesx.Map(messages, o.chatMessageToOpenAIMessage)

	for range 15 {
		hasFunctionCalls := false
		response, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: openaiMessages,
			},
			Model: "gpt-4o-mini",
			Tools: slicesx.Map(tools, func(tool domain.LLMTool) responses.ToolUnionParam {
				return responses.ToolUnionParam{
					OfFunction: &responses.FunctionToolParam{
						Name: tool.Name(),
						Description: param.Opt[string]{
							Value: tool.Description(),
						},
						Parameters: tool.Parameters(),
					},
				}
			}),
		})
		if err != nil {
			log.Println("error creating response:", err)
			return nil, fmt.Errorf("error creating response: %w", err)
		}
		openaiMessages = append(openaiMessages, o.responsesResponseToInputItems(response)...)

		openaiMessages, hasFunctionCalls, err = o.handleResponse(ctx, tools, response, openaiMessages)
		if err != nil {
			return nil, fmt.Errorf("error handling response: %w", err)
		}

		if !hasFunctionCalls {
			return slicesx.Map(openaiMessages[initialMessagesLength:], o.openAIMessageToChatMessage), nil
		}
	}

	return nil, fmt.Errorf("max number of iterations reached")
}

func (o *OpenAI) handleResponse(ctx context.Context, tools []domain.LLMTool, response *responses.Response, currentMessages []responses.ResponseInputItemUnionParam) ([]responses.ResponseInputItemUnionParam, bool, error) {
	hasFunctionCalls := false
	for _, op := range response.Output {
		for _, content := range op.Content {
			if content.Type == "refusal" {
				return nil, false, fmt.Errorf("message refused by the model: %s", content.Text)
			}
		}

		if op.Type == "function_call" {
			hasFunctionCalls = true
			toolCall := op.AsFunctionCall()
			chatMessage, err := o.processToolCall(ctx, tools, toolCall)
			if err != nil {
				return nil, false, fmt.Errorf("error processing tool calls: %w", err)
			}

			currentMessages = append(
				currentMessages,
				responses.ResponseInputItemParamOfFunctionCallOutput(
					toolCall.CallID,
					*chatMessage.Result,
				),
			)
		}
	}

	return currentMessages, hasFunctionCalls, nil
}

func (o *OpenAI) processToolCall(
	ctx context.Context,
	tools []domain.LLMTool,
	toolCall responses.ResponseFunctionToolCall,
) (domain.ChatMessage, error) {
	for _, tool := range tools {
		if tool.Name() == toolCall.Name {
			result, err := tool.Execute(ctx, toolCall.Arguments)
			if err != nil {
				return domain.ChatMessage{}, fmt.Errorf("error executing tool: %w", err)
			}
			toolName := tool.Name()
			return domain.ChatMessage{
				Name:   &toolName,
				Result: &result,
				CallID: &toolCall.ID,
			}, nil
		}
	}

	message, err := json.Marshal(map[string]any{
		"error": fmt.Sprintf("tool does not exist: %s", toolCall.Name),
	})
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("failed to marshal error message: %w", err)
	}

	messageStr := string(message)

	return domain.ChatMessage{
		Name:   &toolCall.Name,
		CallID: &toolCall.ID,
		Result: &messageStr,
	}, nil
}

func (o *OpenAI) chatMessageToOpenAIMessage(message domain.ChatMessage) responses.ResponseInputItemUnionParam {
	switch {
	case message.Role == "user":
		return responses.ResponseInputItemParamOfMessage(
			message.Content,
			responses.EasyInputMessageRoleUser,
		)
	case message.Role == "assistant":
		return responses.ResponseInputItemParamOfMessage(
			message.Content,
			responses.EasyInputMessageRoleAssistant,
		)

	case message.Args != nil:
		return responses.ResponseInputItemParamOfFunctionCall(
			*message.Args,
			*message.CallID,
			*message.Name,
		)

	case message.Result != nil:
		return responses.ResponseInputItemParamOfFunctionCallOutput(
			*message.CallID,
			*message.Result,
		)
	default:
		return responses.ResponseInputItemUnionParam{}
	}
}

func (o *OpenAI) openAIMessageToChatMessage(message responses.ResponseInputItemUnionParam) domain.ChatMessage {
	switch {
	case message.OfMessage != nil:
		content := ""
		if message.OfMessage.Content.OfString.Value != "" {
			content = message.OfMessage.Content.OfString.Value
		}

		switch message.OfMessage.Role {
		case "user":
			return domain.ChatMessage{
				Role:    "user",
				Content: content,
			}
		case "assistant":
			return domain.ChatMessage{
				Role:    "assistant",
				Content: content,
			}
		default:
			return domain.ChatMessage{}
		}

	case message.OfFunctionCall != nil:
		return domain.ChatMessage{
			Name:   &message.OfFunctionCall.Name,
			Args:   &message.OfFunctionCall.Arguments,
			CallID: &message.OfFunctionCall.CallID,
		}

	case message.OfFunctionCallOutput != nil:
		result := ""
		if message.OfFunctionCallOutput.Output.OfString.Value != "" {
			result = message.OfFunctionCallOutput.Output.OfString.Value
		}
		return domain.ChatMessage{
			CallID: &message.OfFunctionCallOutput.CallID,
			Result: &result,
		}

	default:
		return domain.ChatMessage{}
	}
}

func (o *OpenAI) responsesResponseToInputItems(response *responses.Response) []responses.ResponseInputItemUnionParam {
	inputItems := []responses.ResponseInputItemUnionParam{}
	for _, output := range response.Output {
		inputItem := o.responseOutputToInputItem(output)
		inputItems = append(inputItems, inputItem)
	}
	return inputItems
}

func (o *OpenAI) responseOutputToInputItem(output responses.ResponseOutputItemUnion) responses.ResponseInputItemUnionParam {
	switch output.Type {
	case "message":
		outputMessage := output.AsMessage()
		return responses.ResponseInputItemParamOfMessage(
			joinContents(outputMessage.Content),
			responses.EasyInputMessageRoleAssistant,
		)
	case "function_call":
		outputFunctionCall := output.AsFunctionCall()
		return responses.ResponseInputItemParamOfFunctionCall(
			outputFunctionCall.Arguments,
			outputFunctionCall.CallID,
			outputFunctionCall.Name,
		)
	default:
		return responses.ResponseInputItemUnionParam{}
	}
}

func joinContents(contents []responses.ResponseOutputMessageContentUnion) string {
	contentsText := []string{}
	for _, content := range contents {
		if content.Type != "refusal" {
			contentsText = append(contentsText, content.Text)
		}
	}
	return strings.Join(contentsText, "\n")
}
