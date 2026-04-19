package transport

import (
	"github.com/a2aproject/a2a-go/a2a"
)

// FromA2AMessage converts an A2A v1.0 Message to an internal transport Message.
func FromA2AMessage(msg a2a.Message, to AgentID) Message {
	var content string
	for _, part := range msg.Parts {
		switch tp := part.(type) {
		case *a2a.TextPart:
			content = tp.Text
		case a2a.TextPart:
			content = tp.Text
		default:
			continue
		}
		break
	}

	return Message{
		To:      to,
		Role:    string(msg.Role),
		Content: content,
	}
}

// ToA2AMessage converts an internal transport Message to an A2A v1.0 Message.
func ToA2AMessage(msg Message) a2a.Message {
	role := a2a.MessageRoleAgent
	if msg.Role == RoleUser {
		role = a2a.MessageRoleUser
	}

	return a2a.Message{
		Role: role,
		Parts: a2a.ContentParts{
			&a2a.TextPart{Text: msg.Content},
		},
	}
}
