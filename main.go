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

// main initializes and starts the MCP server with the speak tool.
func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"kj-speech-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Define the speak tool
	speakTool := mcp.NewTool("speak",
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
	)

	// Add speak tool handler
	s.AddTool(speakTool, handleSpeak)

	// Define the list_voices tool
	listVoicesTool := mcp.NewTool("list_voices",
		mcp.WithDescription("List all available text-to-speech voices on the system with their locales and descriptions"),
	)

	// Add list_voices tool handler
	s.AddTool(listVoicesTool, handleVoices)

	// Serve via stdio
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// handleSpeak processes text-to-speech requests via the macOS say command.
// It supports optional voice selection and speech rate control.
func handleSpeak(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract text parameter using type-safe helper
	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameter 'text': %v", err)), nil
	}

	// Validate text is not empty
	text = strings.TrimSpace(text)
	if text == "" {
		return mcp.NewToolResultError("Parameter 'text' cannot be empty"), nil
	}

	// Build command arguments
	var args []string

	// Get all arguments for optional parameters
	allArgs := request.GetArguments()

	// Add optional voice parameter
	if voice, ok := allArgs["voice"].(string); ok && voice != "" {
		args = append(args, "-v", voice)
	}

	// Add optional rate parameter
	if rate, ok := allArgs["rate"].(float64); ok && rate > 0 {
		// say command typically accepts 90-500 words per minute
		if rate < 90 || rate > 500 {
			return mcp.NewToolResultError(
				fmt.Sprintf("Rate %.0f is outside acceptable range (90-500 words per minute)", rate),
			), nil
		}
		args = append(args, "-r", fmt.Sprintf("%.0f", rate))
	}

	// Add the text to speak
	args = append(args, text)

	// Execute say command
	cmd := exec.CommandContext(ctx, sayCommand, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if context was cancelled
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Speech synthesis cancelled: %v", ctx.Err()),
			), nil
		}

		// Handle permission and execution errors
		errMsg := fmt.Sprintf("Failed to execute speech synthesis: %v", err)
		if len(output) > 0 {
			errMsg = fmt.Sprintf("%s\nOutput: %s", errMsg, string(output))
		}

		// Check for common permission issues
		if strings.Contains(err.Error(), "permission denied") {
			errMsg += permissionGuidance
		}

		return mcp.NewToolResultError(errMsg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully spoke: %s", text)), nil
}

// Voice represents a single text-to-speech voice with its metadata.
type Voice struct {
	Name        string `json:"name"`
	Locale      string `json:"locale"`
	Description string `json:"description"`
}

// VoicesResponse contains the list of available voices.
type VoicesResponse struct {
	Voices []Voice `json:"voices"`
}

// handleVoices retrieves and returns a list of all available text-to-speech voices.
func handleVoices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Execute say -v '?' to get list of voices
	cmd := exec.CommandContext(ctx, sayCommand, "-v", "?")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if context was cancelled
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Voice listing cancelled: %v", ctx.Err()),
			), nil
		}

		errMsg := fmt.Sprintf("Failed to retrieve voice list: %v", err)
		if len(output) > 0 {
			errMsg = fmt.Sprintf("%s\nOutput: %s", errMsg, string(output))
		}
		return mcp.NewToolResultError(errMsg), nil
	}

	// Parse the output
	// Format: "VoiceName    locale    # description"
	// Example: "Albert              en_US    # Hello! My name is Albert."
	lines := strings.Split(string(output), "\n")
	voices := make([]Voice, 0, len(lines))

	// Regex to parse voice lines: name, locale, and description
	// Pattern: voice name (any chars), whitespace, locale, whitespace, #, description
	voicePattern := regexp.MustCompile(`^(.+?)\s+(\S+)\s+#\s*(.*)$`)

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

	// Create response
	response := VoicesResponse{
		Voices: voices,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format voice list: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
