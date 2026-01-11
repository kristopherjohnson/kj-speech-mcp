# KJ Speech MCP Server

A Model Context Protocol (MCP) server that provides text-to-speech functionality using macOS's built-in speech synthesis.

## Features

- **speak** tool: Converts text to audible speech using the macOS `/usr/bin/say` command
- **list_voices** tool: Lists all available text-to-speech voices with their locales and descriptions
- Optional voice selection from available system voices
- Optional speech rate control (words per minute)
- Simple, lightweight implementation
- Proper error handling for permissions and invalid inputs

## Requirements

- macOS (uses `/usr/bin/say` command)
- Go 1.21 or later (for building from source)
- Audio output device available

## Installation

### Using go install

The easiest way to install is using `go install`:

```bash
go install github.com/kristopherjohnson/kj-speech-mcp@latest
```

This will install the `kj-speech-mcp` binary to your `$GOPATH/bin` directory (typically `~/go/bin`). Make sure this directory is in your PATH.

### From Source

Alternatively, you can build from source:

```bash
git clone https://github.com/kristopherjohnson/kj-speech-mcp.git
cd kj-speech-mcp
go build -o kj-speech-mcp
```

The compiled binary `kj-speech-mcp` can be placed anywhere in your PATH or referenced by absolute path.

## Usage with Claude Desktop

Add this server to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

If installed via `go install`:
```json
{
  "mcpServers": {
    "kj-speech": {
      "command": "kj-speech-mcp"
    }
  }
}
```

If built from source or installed in a custom location:
```json
{
  "mcpServers": {
    "kj-speech": {
      "command": "/path/to/kj-speech-mcp"
    }
  }
}
```

Replace `/path/to/kj-speech-mcp` with the actual path to the compiled binary.

After updating the configuration:
1. Restart Claude Desktop
2. The `speak` and `list_voices` tools will be available in your conversations

## Usage with Claude Code

Claude Code can automatically configure this MCP server using the `claude mcp add` command.

If installed via `go install` (binary in PATH):
```bash
claude mcp add kj-speech kj-speech-mcp
```

If built from source or installed in a custom location:
```bash
claude mcp add kj-speech /path/to/kj-speech-mcp
```

This will:
1. Prompt you to confirm adding the server
2. Update your Claude Code configuration
3. Make the `speak` and `list_voices` tools available in all Claude Code sessions

By default `claude mcp add` will add the server to the local (current directory) configuration. Use the `--scope user` option to make it available for all of the current user's sessions, or `--scope project` for project-level configuration.

You can verify the server is configured by running:
```bash
claude mcp list
```

To remove the server:
```bash
claude mcp remove kj-speech
```

## MCP Tool Reference

### list_voices

Lists all available text-to-speech voices on the system.

**Parameters:** None

**Returns:** JSON object containing an array of voice objects, each with:
- `name` (string): The voice name (e.g., "Samantha", "Alex", "Eddy (English (US))")
- `locale` (string): The locale code (e.g., "en_US", "fr_FR", "ja_JP")
- `description` (string): A sample phrase spoken by the voice

**Example:**
```
Use the list_voices tool to list available voices
```

**Sample Output:**
```json
{
  "voices": [
    {
      "name": "Albert",
      "locale": "en_US",
      "description": "Hello! My name is Albert."
    },
    {
      "name": "Samantha",
      "locale": "en_US",
      "description": "Hello! My name is Samantha."
    },
    {
      "name": "Amélie",
      "locale": "fr_CA",
      "description": "Bonjour! Je m'appelle Amélie."
    }
  ]
}
```

### speak

Converts text to audible speech.

**Parameters:**
- `text` (string, required): The text to be spoken aloud
- `voice` (string, optional): Voice to use for speech synthesis. If not specified, uses the system default voice.
- `rate` (number, optional): Speech rate in words per minute (valid range: 90-500). If not specified, uses the system default rate.

**Examples:**

Basic usage with default voice and rate:
```
Use the speak tool to say "Hello, world!"
```

Using a specific voice:
```
Use the speak tool to say "Hello, world!" with voice "Samantha"
```

Using a specific rate:
```
Use the speak tool to say "This is slower" with rate 120
```

Using both voice and rate:
```
Use the speak tool to say "Fast speech" with voice "Alex" and rate 300
```

**Available Voices:**

Use the `list_voices` tool to get a complete list of available voices with their locales and descriptions. Alternatively, you can run:
```bash
/usr/bin/say -v '?'
```

Common voices include: Alex, Samantha, Victoria, Daniel, Karen, and many others in various languages.

**Error Handling:**
- Returns error if `text` parameter is missing or empty
- Returns error if speech synthesis fails
- Provides helpful messages for permission issues
- Invalid voice names will cause the say command to fail with an error message

## Permissions

The `say` command typically works without special permissions in a normal user session. However, you may need to grant accessibility permissions if running in certain contexts:

1. Open **System Preferences** → **Security & Privacy** → **Privacy** → **Accessibility**
2. Add Claude Desktop (or the terminal application) if prompted

## Troubleshooting

**No audio output:**
- Verify you have an audio output device connected
- Check system volume settings
- Ensure you're not in an SSH session without audio forwarding

**Permission errors:**
- Check System Preferences accessibility permissions
- Ensure running in a user session (not as a background daemon)

**Command not found:**
- Verify `/usr/bin/say` exists on your system (standard on macOS)
- Try running `/usr/bin/say "test"` directly in Terminal

## Development

```bash
# Install dependencies
go mod download

# Build
go build -o kj-speech-mcp

# Test the say command directly
/usr/bin/say "Testing speech"

# Test with different voices
/usr/bin/say -v Samantha "Testing Samantha voice"
/usr/bin/say -v Alex "Testing Alex voice"

# Test with different rates
/usr/bin/say -r 120 "Slower speech at 120 words per minute"
/usr/bin/say -r 300 "Faster speech at 300 words per minute"

# List available voices
/usr/bin/say -v ?
```

## License

MIT License

## Author

Kristopher Johnson
