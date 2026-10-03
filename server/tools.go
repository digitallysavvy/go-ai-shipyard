package main

import (
	"context"
	"fmt"

	"github.com/digitallysavvy/go-ai/pkg/ai"
	"github.com/digitallysavvy/go-ai/pkg/provider/types"
)

func objectSchema(props map[string]interface{}, required ...string) map[string]interface{} {
	if required == nil {
		required = []string{}
	}
	return map[string]interface{}{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

// tools returns the chat agent's tools. They are built per request so the
// coding-agent tool can stream progress through this request's writer.
func (s *server) tools(kind agentKind, writer ai.UIMessageStreamWriter) []types.Tool {
	return []types.Tool{
		{
			Name:        "run_tests",
			Description: "Run `go test ./...` in the workspace and return whether it passed plus the output.",
			Parameters:  objectSchema(map[string]interface{}{}),
			Execute: func(ctx context.Context, _ map[string]interface{}, _ types.ToolExecutionOptions) (interface{}, error) {
				return s.ws.RunTests(ctx), nil
			},
		},
		{
			Name:        "list_files",
			Description: "List the files in the workspace.",
			Parameters:  objectSchema(map[string]interface{}{}),
			Execute: func(ctx context.Context, _ map[string]interface{}, _ types.ToolExecutionOptions) (interface{}, error) {
				files, err := s.ws.Files()
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{"files": sortedKeys(files)}, nil
			},
		},
		{
			Name:        "read_file",
			Description: "Read a file from the workspace.",
			Parameters: objectSchema(map[string]interface{}{
				"path": map[string]interface{}{"type": "string", "description": "Path relative to the workspace root, e.g. shortlink.go"},
			}, "path"),
			Execute: func(ctx context.Context, input map[string]interface{}, _ types.ToolExecutionOptions) (interface{}, error) {
				p, _ := input["path"].(string)
				content, err := s.ws.ReadFile(p)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{"path": p, "content": content}, nil
			},
		},
		{
			Name: "delegate_to_coding_agent",
			Description: fmt.Sprintf("Hand a code change to %s, a coding agent that edits files and runs commands in a sandbox "+
				"copy of the workspace. Its changes are synced back when it finishes. Requires user approval.", kind.CoderName()),
			Parameters: objectSchema(map[string]interface{}{
				"task": map[string]interface{}{"type": "string", "description": "Precise instructions: what is broken, where, and constraints."},
			}, "task"),
			// The model can ask; only the user can say yes. useChat renders the
			// approval request and sends the decision back on the next request.
			ToolApproval: true,
			Execute: func(ctx context.Context, input map[string]interface{}, opts types.ToolExecutionOptions) (interface{}, error) {
				task, _ := input["task"].(string)
				return s.coder.Run(ctx, kind, task, opts.ToolCallID, writer)
			},
		},
	}
}
