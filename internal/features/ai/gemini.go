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

// geminiBaseURL ที่อยู่ API ของ Gemini (test เปลี่ยนเป็น server ปลอมได้)
const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// DefaultGeminiModel รุ่นของ Gemini ที่ใช้เป็น AI ฟรี (เปลี่ยนได้ด้วย GEMINI_MODEL ใน .env)
const DefaultGeminiModel = "gemini-3.8-flash"

// Gemini ส่งคำขอไปหา Gemini 1 ครั้ง (แยกเป็น interface เพื่อให้ test ใช้ตัวปลอมได้)
type Gemini interface {
	Generate(ctx context.Context, req geminiRequest) (geminiResponse, error)
}

// geminiContent ข้อความ 1 ก้อนในบทสนทนา role = user / model
// parts เก็บเป็น JSON ดิบ → ส่งคำตอบของ Gemini กลับไปได้ครบทุกช่อง (มี thoughtSignature ที่ Gemini ต้องได้กลับไป)
type geminiContent struct {
	Role  string            `json:"role,omitempty"`
	Parts []json.RawMessage `json:"parts"`
}

// GenerationConfig ใช้ตอนให้ตอบเป็น JSON ตามรูปแบบ (responseMimeType + responseJsonSchema)
type geminiRequest struct {
	SystemInstruction geminiContent    `json:"systemInstruction"`
	Contents          []geminiContent  `json:"contents"`
	Tools             []map[string]any `json:"tools,omitempty"`
	GenerationConfig  map[string]any   `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content           geminiContent   `json:"content"`
		FinishReason      string          `json:"finishReason"`
		GroundingMetadata geminiGrounding `json:"groundingMetadata"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// geminiGrounding ข้อมูลการค้น Google (มีเมื่อส่งเครื่องมือ google_search ไป)
// groundingChunks = หน้าเว็บที่ใช้, searchEntryPoint.renderedContent = แถบคำค้นแนะนำของ Google (HTML) ที่ต้องแสดงคู่กับผล
type geminiGrounding struct {
	GroundingChunks []struct {
		Web struct {
			URI   string `json:"uri"`
			Title string `json:"title"`
		} `json:"web"`
	} `json:"groundingChunks"`
	SearchEntryPoint struct {
		RenderedContent string `json:"renderedContent"`
	} `json:"searchEntryPoint"`
}

// geminiPart อ่านเฉพาะช่องที่ใช้ จากแต่ละ part
type geminiPart struct {
	Text         string `json:"text"`
	Thought      bool   `json:"thought"`
	FunctionCall *struct {
		ID   string          `json:"id"`
		Name string          `json:"name"`
		Args json.RawMessage `json:"args"`
	} `json:"functionCall"`
}

// GeminiError error ที่ API ของ Gemini ตอบกลับ (status = HTTP status)
type GeminiError struct {
	Status  int
	Message string
}

func (e *GeminiError) Error() string {
	return fmt.Sprintf("gemini %d: %s", e.Status, e.Message)
}

type geminiClient struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

// NewGemini สร้างตัวเรียก Gemini — ไม่มี key = คืน nil (ไม่มี AI ฟรี)
func NewGemini(apiKey, model string) Gemini {
	if apiKey == "" {
		return nil
	}
	if model == "" {
		model = DefaultGeminiModel
	}
	return &geminiClient{apiKey: apiKey, model: model, baseURL: geminiBaseURL, http: &http.Client{}}
}

// Generate POST /models/{model}:generateContent (ส่ง key ทาง header ไม่ใส่ใน URL → ไม่หลุดไปใน log)
func (g *geminiClient) Generate(ctx context.Context, req geminiRequest) (geminiResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return geminiResponse{}, err
	}
	url := g.baseURL + "/models/" + g.model + ":generateContent"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return geminiResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	res, err := g.http.Do(httpReq)
	if err != nil {
		return geminiResponse{}, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return geminiResponse{}, err
	}
	if res.StatusCode != http.StatusOK {
		return geminiResponse{}, &GeminiError{Status: res.StatusCode, Message: geminiErrorMessage(data)}
	}
	var resp geminiResponse
	err = json.Unmarshal(data, &resp)
	return resp, err
}

// geminiErrorMessage อ่านข้อความ error จาก {"error": {"message": "..."}}
func geminiErrorMessage(data []byte) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &body) != nil || body.Error.Message == "" {
		return strings.TrimSpace(string(data))
	}
	return body.Error.Message
}

// geminiText ทำ part ข้อความ
func geminiText(text string) json.RawMessage {
	data, _ := json.Marshal(map[string]string{"text": text})
	return data
}

// geminiHistory แปลงประวัติแชทจาก Flutter เป็นรูปแบบของ Gemini (assistant → model)
func geminiHistory(history []ChatMessage) []geminiContent {
	contents := []geminiContent{}
	for _, m := range history {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{Role: role, Parts: []json.RawMessage{geminiText(strings.TrimSpace(m.Content))}})
	}
	return contents
}

// geminiBlocked Gemini ไม่ยอมตอบเพราะนโยบายความปลอดภัย
func geminiBlocked(resp geminiResponse) bool {
	if resp.PromptFeedback.BlockReason != "" || len(resp.Candidates) == 0 {
		return true
	}
	switch resp.Candidates[0].FinishReason {
	case "SAFETY", "PROHIBITED_CONTENT", "BLOCKLIST", "SPII", "RECITATION":
		return true
	}
	return false
}

// geminiChat ตอบคำถามล่าสุดในแชทด้วย Gemini: วนให้ใช้เครื่องมือ (อ่านข้อมูล / เสนองาน) จนได้คำตอบ — กติกาเดียวกับ chat() ของ Claude
func geminiChat(ctx context.Context, gemini Gemini, tools toolbox, history []ChatMessage) (ChatReply, error) {
	system := geminiContent{Parts: []json.RawMessage{
		geminiText(chatSystemPrompt),
		geminiText("วันนี้คือวันที่ " + tools.today.Format("2006-01-02")),
	}}
	contents := geminiHistory(history)
	reply := ChatReply{Actions: []Action{}}

	for round := 0; round < maxChatRounds; round++ {
		resp, err := gemini.Generate(ctx, geminiRequest{SystemInstruction: system, Contents: contents, Tools: geminiTools(chatTools)})
		if err != nil {
			return ChatReply{}, err
		}
		if geminiBlocked(resp) {
			return ChatReply{}, ErrRefused
		}
		answer := resp.Candidates[0].Content
		answer.Role = "model"
		contents = append(contents, answer)

		texts, results, err := runGeminiParts(ctx, tools, answer.Parts, &reply)
		if err != nil {
			return ChatReply{}, err
		}
		if len(results) == 0 {
			reply.Reply = strings.Join(texts, "\n\n")
			if reply.Reply == "" && len(reply.Actions) == 0 {
				return ChatReply{}, ErrNoReport
			}
			return reply, nil
		}
		contents = append(contents, geminiContent{Role: "user", Parts: results})
	}
	return ChatReply{}, ErrTooManySteps
}

// runGeminiParts อ่านคำตอบของ Gemini: เก็บข้อความ + ทำทุกเครื่องมือที่เรียก แล้วทำผลส่งกลับ (functionResponse)
func runGeminiParts(ctx context.Context, tools toolbox, parts []json.RawMessage, reply *ChatReply) ([]string, []json.RawMessage, error) {
	texts := []string{}
	results := []json.RawMessage{}
	for _, raw := range parts {
		var part geminiPart
		if json.Unmarshal(raw, &part) != nil {
			continue
		}
		if part.FunctionCall == nil {
			if !part.Thought && strings.TrimSpace(part.Text) != "" {
				texts = append(texts, strings.TrimSpace(part.Text))
			}
			continue
		}

		call := part.FunctionCall
		output := map[string]any{}
		if strings.HasPrefix(call.Name, "propose_") && len(reply.Actions) >= maxActions {
			output["error"] = "เสนอได้ไม่เกิน 5 งานต่อครั้ง"
		} else {
			args := call.Args
			if len(args) == 0 {
				args = json.RawMessage("{}")
			}
			text, action, err := tools.run(ctx, call.Name, args)
			var toolErr toolError
			switch {
			case errors.As(err, &toolErr):
				output["error"] = toolErr.Error()
			case err != nil:
				return nil, nil, err
			default:
				output["result"] = text
				if action != nil {
					action.ID = geminiActionID(call.ID, call.Name, len(reply.Actions))
					reply.Actions = append(reply.Actions, *action)
				}
			}
		}

		response := map[string]any{"name": call.Name, "response": output}
		if call.ID != "" {
			response["id"] = call.ID
		}
		data, _ := json.Marshal(map[string]any{"functionResponse": response})
		results = append(results, data)
	}
	return texts, results, nil
}

// geminiActionID id ของงานที่เสนอ (Gemini บางรุ่นไม่ส่ง id มา → ใช้ชื่อเครื่องมือ + ลำดับแทน)
func geminiActionID(callID, name string, index int) string {
	if callID != "" {
		return callID
	}
	return fmt.Sprintf("%s_%d", name, index+1)
}
