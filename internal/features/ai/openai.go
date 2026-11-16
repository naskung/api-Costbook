package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AI แบบ OpenAI = เจ้าไหนก็ได้ที่ใช้รูปแบบ API เดียวกับ OpenAI (POST {base}/chat/completions + GET {base}/models)
// เช่น OpenAI, DeepSeek, Groq, OpenRouter, Mistral, xAI, Together, Typhoon, Gemini, Ollama ในเครื่อง
// ร้านใส่ที่อยู่ API (base URL) + key + ชื่อรุ่น เองในหน้าตั้งค่า

// ErrNotOpenAIAPI ที่อยู่ที่ใส่มาตอบกลับมา แต่ไม่ใช่ API แบบ OpenAI (เช่น เป็นหน้าเว็บ)
var ErrNotOpenAIAPI = errors.New("not an OpenAI-compatible API")

// OpenAI ส่งคำขอไปหา AI แบบ OpenAI (แยกเป็น interface เพื่อให้ test ใช้ตัวปลอมได้)
type OpenAI interface {
	Complete(ctx context.Context, req openAIRequest) (openAIResponse, error)
	Models(ctx context.Context) ([]string, error)
}

// openAIMessage ข้อความ 1 อัน role = system / user / assistant / tool
// assistant ที่เรียกเครื่องมือมี tool_calls, ผลของเครื่องมือส่งกลับเป็น role tool + tool_call_id
type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// openAIRequest model ใส่ให้เองตอนส่ง (รุ่นที่ร้านเลือก)
type openAIRequest struct {
	Model    string           `json:"model"`
	Messages []openAIMessage  `json:"messages"`
	Tools    []map[string]any `json:"tools,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
}

// OpenAIError error ที่ AI แบบ OpenAI ตอบกลับ (status = HTTP status)
type OpenAIError struct {
	Status  int
	Message string
}

func (e *OpenAIError) Error() string {
	return fmt.Sprintf("openai-compatible %d: %s", e.Status, e.Message)
}

type openAIClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewOpenAI สร้างตัวเรียก AI แบบ OpenAI — apiKey ว่างได้ (เช่น Ollama ในเครื่องไม่ต้องใช้ key)
func NewOpenAI(baseURL, apiKey, model string) OpenAI {
	return &openAIClient{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, http: &http.Client{}}
}

// Complete POST {base}/chat/completions
func (c *openAIClient) Complete(ctx context.Context, req openAIRequest) (openAIResponse, error) {
	req.Model = c.model
	body, err := json.Marshal(req)
	if err != nil {
		return openAIResponse{}, err
	}
	data, err := c.send(ctx, http.MethodPost, "/chat/completions", body)
	if err != nil {
		return openAIResponse{}, err
	}
	var resp openAIResponse
	if json.Unmarshal(data, &resp) != nil {
		return openAIResponse{}, ErrNotOpenAIAPI
	}
	return resp, nil
}

// Models GET {base}/models → ชื่อรุ่นทั้งหมดที่ key นี้ใช้ได้ (ไม่เสียเงิน ใช้ลอง key ด้วย)
func (c *openAIClient) Models(ctx context.Context) ([]string, error) {
	data, err := c.send(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &list) != nil {
		return nil, ErrNotOpenAIAPI
	}
	models := []string{}
	for _, item := range list.Data {
		if item.ID != "" {
			models = append(models, item.ID)
		}
	}
	return models, nil
}

// send ส่งคำขอ (key ทาง header Authorization: Bearer) ตอบไม่ใช่ 2xx → OpenAIError
func (c *openAIClient) send(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, &OpenAIError{Status: res.StatusCode, Message: shortText(openAIErrorMessage(data), 500)}
	}
	return data, nil
}

// openAIErrorMessage อ่านข้อความ error จาก {"error": {"message": "..."}} (บางเจ้าส่ง {"error": "..."})
func openAIErrorMessage(data []byte) string {
	var nested struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &nested) == nil && nested.Error.Message != "" {
		return nested.Error.Message
	}
	var flat struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &flat) == nil && flat.Error != "" {
		return flat.Error
	}
	return strings.TrimSpace(string(data))
}

// openAITools แปลงเครื่องมือของแชทเป็นรูปแบบ function ของ OpenAI
func openAITools(specs []toolSpec) []map[string]any {
	tools := []map[string]any{}
	for _, spec := range specs {
		parameters := map[string]any{"type": "object", "properties": spec.properties}
		if len(spec.properties) == 0 {
			parameters["properties"] = map[string]any{}
		}
		if len(spec.required) > 0 {
			parameters["required"] = spec.required
		}
		tools = append(tools, map[string]any{
			"type":     "function",
			"function": map[string]any{"name": spec.name, "description": spec.description, "parameters": parameters},
		})
	}
	return tools
}

// openAIAnswer คำตอบแรก — ไม่มีคำตอบ = ErrNoReport, ถูกกรองเนื้อหา = ErrRefused
func openAIAnswer(resp openAIResponse) (openAIMessage, error) {
	if len(resp.Choices) == 0 {
		return openAIMessage{}, ErrNoReport
	}
	if resp.Choices[0].FinishReason == "content_filter" {
		return openAIMessage{}, ErrRefused
	}
	answer := resp.Choices[0].Message
	answer.Role = "assistant"
	return answer, nil
}

// withoutThinking ตัดส่วนที่ AI คิดในใจออก (บางรุ่นส่งมาใน <think>...</think> หน้าคำตอบ)
func withoutThinking(text string) string {
	if end := strings.LastIndex(text, "</think>"); end >= 0 {
		text = text[end+len("</think>"):]
	}
	return strings.TrimSpace(text)
}

// openAIChat ตอบคำถามล่าสุดในแชท: วนให้ใช้เครื่องมือ (อ่านข้อมูล / เสนองาน) จนได้คำตอบ — กติกาเดียวกับ chat() ของ Claude
func openAIChat(ctx context.Context, client OpenAI, tools toolbox, history []ChatMessage) (ChatReply, error) {
	messages := []openAIMessage{{
		Role:    "system",
		Content: chatSystemPrompt + "\n\nวันนี้คือวันที่ " + tools.today.Format("2006-01-02"),
	}}
	for _, m := range history {
		messages = append(messages, openAIMessage{Role: m.Role, Content: strings.TrimSpace(m.Content)})
	}
	reply := ChatReply{Actions: []Action{}}

	for round := 0; round < maxChatRounds; round++ {
		resp, err := client.Complete(ctx, openAIRequest{Messages: messages, Tools: openAITools(chatTools)})
		if err != nil {
			return ChatReply{}, err
		}
		answer, err := openAIAnswer(resp)
		if err != nil {
			return ChatReply{}, err
		}
		messages = append(messages, answer)
		if len(answer.ToolCalls) == 0 {
			reply.Reply = withoutThinking(answer.Content)
			if reply.Reply == "" && len(reply.Actions) == 0 {
				return ChatReply{}, ErrNoReport
			}
			return reply, nil
		}

		for _, call := range answer.ToolCalls {
			output, err := callTool(ctx, tools, call.Function.Name, json.RawMessage(call.Function.Arguments), call.ID, &reply)
			if err != nil {
				return ChatReply{}, err
			}
			data, _ := json.Marshal(output)
			messages = append(messages, openAIMessage{Role: "tool", ToolCallID: call.ID, Content: string(data)})
		}
	}
	return ChatReply{}, ErrTooManySteps
}

// openAIAsker ถามครั้งเดียวให้ตอบเป็น JSON ตาม schema (ใส่ schema ไว้ในคำสั่ง ไม่ใช้ response_format
// เพราะแต่ละเจ้ารองรับไม่เหมือนกัน) แล้วตัดเอาเฉพาะ JSON object ในคำตอบ
func openAIAsker(client OpenAI) jsonAsker {
	return func(ctx context.Context, system, input string, schema map[string]any) (string, error) {
		schemaJSON, err := json.Marshal(schema)
		if err != nil {
			return "", err
		}
		resp, err := client.Complete(ctx, openAIRequest{Messages: []openAIMessage{
			{Role: "system", Content: system + "\n\nตอบเป็น JSON object อย่างเดียว ห้ามมีข้อความอื่น ตาม JSON Schema นี้:\n" + string(schemaJSON)},
			{Role: "user", Content: input},
		}})
		if err != nil {
			return "", err
		}
		answer, err := openAIAnswer(resp)
		if err != nil {
			return "", err
		}
		return jsonObjectIn(withoutThinking(answer.Content)), nil
	}
}

// jsonObjectIn ตัดเอาเฉพาะ JSON object ในข้อความ (บางรุ่นใส่ ```json หรือมีคำนำหน้า) — ไม่เจอ = ""
func jsonObjectIn(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}
