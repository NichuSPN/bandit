package model

import (
	"encoding/json"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Message struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func NewSystemMessage(content string) Message {
	return Message{
		Role:    RoleSystem,
		Content: content,
	}
}

func NewUserMessage(content string) Message {
	return Message{
		Role:    RoleUser,
		Content: content,
	}
}

func NewAssistantMessage(content string) Message {
	return Message{
		Role:    RoleAssistant,
		Content: content,
	}
}

func NewAssistantMessageWithTools(content string, toolCalls []ToolCall) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
}

func NewToolMessage(name string, content string) Message {
	return Message{
		Role:    RoleTool,
		Name:    name,
		Content: content,
	}
}

type EventType string

const (
	EventThinking     EventType = "thinking"
	EventToolCall     EventType = "tool_call"
	EventToolResult   EventType = "tool_result"
	EventContentChunk EventType = "content_chunk"
	EventFinished     EventType = "finished"
	EventError        EventType = "error"
)

type AgentProgressEvent struct {
	Type     EventType
	Name     string
	Args     string
	Result   string
	Chunk    string
	ErrorMsg string
}
