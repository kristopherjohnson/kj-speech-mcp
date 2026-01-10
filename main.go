// Package main implements an MCP server that provides text-to-speech functionality
// using macOS's built-in speech synthesis via /usr/bin/say.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	sayCommand = "/usr/bin/say"

	permissionGuidance = `

Permission denied. Please ensure:
1. The application has accessibility permissions in System Preferences
2. You are running in a user session with audio output available
3. You are not running in an SSH session without audio forwarding`
)

// voicePattern parses "say -v ?" output: "VoiceName  locale  # description"
var voicePattern = regexp.MustCompile(`^(.+?)\s+(\S+)\s+#\s*(.*)$`)

func main() {
	s := server.NewMCPServer(
		"kj-speech-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(mcp.NewTool("speak",
		mcp.WithDescription("Converts text to audible speech using macOS text-to-speech"),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The text to be spoken aloud"),
		),
		mcp.WithString("voice",
			mcp.Description("Voice to use for speech synthesis (optional, uses system default if not specified)"),
		),
		mcp.WithNumber("rate",
			mcp.Description("Speech rate in words per minute (optional, uses system default if not specified)"),
		),
	), handleSpeak)

	s.AddTool(mcp.NewTool("list_voices",
		mcp.WithDescription("List all available text-to-speech voices on the system with their locales and descriptions"),
	), handleVoices)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// formatCommandError builds an error message for say command failures.
func formatCommandError(ctx context.Context, action string, err error, output []byte) string {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("%s cancelled: %v", action, ctx.Err())
	}

	errMsg := fmt.Sprintf("Failed to %s: %v", strings.ToLower(action), err)
	if len(output) > 0 {
		errMsg = fmt.Sprintf("%s\nOutput: %s", errMsg, string(output))
	}
	if strings.Contains(err.Error(), "permission denied") {
		errMsg += permissionGuidance
	}
	return errMsg
}

// handleSpeak processes text-to-speech requests via the macOS say command.
func handleSpeak(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameter 'text': %v", err)), nil
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return mcp.NewToolResultError("Parameter 'text' cannot be empty"), nil
	}

	var args []string
	allArgs := request.GetArguments()

	if voice, ok := allArgs["voice"].(string); ok && voice != "" {
		args = append(args, "-v", voice)
	}

	if rate, ok := allArgs["rate"].(float64); ok && rate > 0 {
		if rate < 90 || rate > 500 {
			return mcp.NewToolResultError(
				fmt.Sprintf("Rate %.0f is outside acceptable range (90-500 words per minute)", rate),
			), nil
		}
		args = append(args, "-r", fmt.Sprintf("%.0f", rate))
	}

	args = append(args, text)

	cmd := exec.CommandContext(ctx, sayCommand, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(formatCommandError(ctx, "Execute speech synthesis", err, output)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully spoke: %s", text)), nil
}

// Voice represents a single text-to-speech voice with its metadata.
type Voice struct {
	Name        string `json:"name"`
	Locale      string `json:"locale"`
	Description string `json:"description"`
}

// handleVoices retrieves and returns a list of all available text-to-speech voices.
func handleVoices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd := exec.CommandContext(ctx, sayCommand, "-v", "?")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(formatCommandError(ctx, "Retrieve voice list", err, output)), nil
	}

	lines := strings.Split(string(output), "\n")
	voices := make([]Voice, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := voicePattern.FindStringSubmatch(line)
		if len(matches) == 4 {
			voices = append(voices, Voice{
				Name:        strings.TrimSpace(matches[1]),
				Locale:      strings.TrimSpace(matches[2]),
				Description: strings.TrimSpace(matches[3]),
			})
		}
	}

	jsonData, err := json.MarshalIndent(struct {
		Voices []Voice `json:"voices"`
	}{Voices: voices}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format voice list: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
