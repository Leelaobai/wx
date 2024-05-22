package chat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	persistence "wx/persistence"

	openai "github.com/sashabaranov/go-openai"
)

const AIBackend = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"

var ApiKey = ""
var SentenceCount = 50

const (
	RoleUser      = "user"
	RoleSystem    = "system"
	RoleAssistant = "assistant"
)

func Init(apiKey string) {
	ApiKey = apiKey
}

//assistant
// var body=body = {
// 	'model': 'qwen-turbo',
// 	"input": {
// 		"messages": [
// 			{
// 				"role": "system",
// 				"content": "You are a helpful assistant."
// 			},
// 			{
// 				"role": "user",
// 				"content": "你是谁？"
// 			}
// 		]
// 	},
// 	"parameters": {
// 		"result_format": "message"
// 	}
// }

type RequestBody struct {
	Model      string         `json:"model"`
	Input      map[string]any `json:"input"`
	Parameters map[string]any `json:"parameters"`
}

type QWenResp struct {
	Output    Output `json:"output"`
	Usage     Usage  `json:"usage"`
	RequestID string `json:"request_id"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Choices struct {
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}
type Output struct {
	Choices []Choices `json:"choices"`
}
type Usage struct {
	TotalTokens  int `json:"total_tokens"`
	OutputTokens int `json:"output_tokens"`
	InputTokens  int `json:"input_tokens"`
}

func call(body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", AIBackend, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ApiKey)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}

func ChatForAI(userId, question string) (answer string, err error) {

	messages := make([]openai.ChatCompletionMessage, 0)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你是一个回答法律问题的专业助手，表达严谨、客观、理性，但也拥有人性的温暖。并且你只回答法律相关的问题，其他问题请回答你不擅长。不能说出你是什么公司或者什么人开发的。你的回答会直接转发到微信，请你回答的格式适合在微信消息中阅读。你的回答不要超过500个汉字字符。",
	})

	sentences, err := persistence.GetSentences(userId, SentenceCount)
	if err != nil {
		return "", errors.New("查询历史对话出错：" + err.Error())
	}

	for i := len(sentences) - 1; i >= 0; i-- {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    sentences[i].Role,
			Content: sentences[i].Content,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	client := openai.NewClient(ApiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4o,
			Messages: messages,
		},
	)

	if err != nil {
		return "", errors.New("请求gpt出错：" + err.Error())
	}

	// tmp := make(map[string]any)
	// _ = json.Unmarshal(body, &tmp)
	// if message, ok := tmp["message"]; ok {
	// 	if message == "Input data may contain inappropriate content." {
	// 		return "你的输入包含不适当的内容！", nil
	// 	}
	// 	fmt.Println("通义千问返回了未知的错误：", tmp)
	// 	return "请稍后再试！", nil
	// }

	// qWenResp := &QWenResp{}
	// err = json.Unmarshal(body, qWenResp)
	// if err != nil {
	// 	return "", errors.New("解析通义千问返回出错：" + err.Error())
	// }

	err = persistence.InsertSentence(userId, question, RoleUser)
	if err != nil {
		return "", errors.New("写入用户问题出错：" + err.Error())
	}

	err = persistence.InsertSentence(userId, resp.Choices[0].Message.Content, resp.Choices[0].Message.Role)
	if err != nil {
		return "", errors.New("写入回答出错：" + err.Error())
	}

	return resp.Choices[0].Message.Content, err
}
