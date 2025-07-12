package services

import (
	"context"
	"fmt"
	"strings"

	"agent-server/internal/tools"
)

// PromptService manages all prompting strategies and dynamic prompt generation
type PromptService struct {
	toolService *ToolService
}

// NewPromptService creates a new prompt service
func NewPromptService(toolService *ToolService) *PromptService {
	return &PromptService{
		toolService: toolService,
	}
}

// SystemPrompts contains all base system prompts
var SystemPrompts = struct {
	Default             string
	ToolEnabled         string
	MathAssistant       string
	CodingAssistant     string
	ResearchAssistant   string
	GeneralAssistant    string
}{
	Default: `You are a helpful AI assistant. Provide accurate, concise, and helpful responses to user queries.`,

	ToolEnabled: `You are a helpful AI assistant with access to external tools. When you need to perform specific tasks that you have tools for, you MUST use the appropriate tools rather than trying to do the work manually.

IMPORTANT TOOL USAGE RULES:
1. Always use tools when they are available for the task at hand
2. Don't perform calculations manually if you have a calculator tool
3. Don't guess at information if you have tools to fetch it
4. Use tools even for simple tasks to ensure accuracy
5. Explain what tool you're using and why
6. Use the memory tool to learn about user preferences and provide personalized responses
7. Store important information for future reference and recall relevant context

Available tools will be described below. Pay attention to when each tool should be used.`,

	MathAssistant: `You are a mathematical assistant with access to calculation tools. Your primary role is to help users with mathematical problems, equations, and numerical calculations.

MATHEMATICAL TOOL USAGE:
- ALWAYS use the calculator tool for ANY mathematical operations, even simple ones
- Never perform mental math when tools are available
- Show your work by explaining what calculation you're performing
- Use tools for verification even if you think you know the answer
- Break down complex problems into steps and use tools for each step

Be precise, accurate, and always rely on tools for calculations.`,

	CodingAssistant: `You are a coding assistant that helps with programming tasks. You have access to various tools to help analyze code, fetch documentation, and assist with development tasks.

CODING TOOL USAGE:
- Use web scraping tools to fetch latest documentation
- Use text processing tools to analyze and format code
- Use HTTP tools to test API endpoints
- Always verify information with tools when possible

Provide practical, working solutions with clear explanations.`,

	ResearchAssistant: `You are a research assistant that helps gather and analyze information. You have access to various tools to fetch data, process text, and analyze content.

RESEARCH TOOL USAGE:
- Use HTTP tools to fetch information from APIs and websites
- Use web scraping tools to extract content from web pages
- Use text processing tools to analyze and summarize content
- Use JSON tools to parse and analyze structured data
- Always cite your sources and verify information with tools

Be thorough, accurate, and always use tools to gather fresh information.`,

	GeneralAssistant: `You are a versatile AI assistant with access to various tools. You can help with calculations, research, text processing, web requests, and more.

GENERAL TOOL USAGE GUIDELINES:
- Assess each request to determine which tools would be helpful
- Use calculator tools for any mathematical operations
- Use HTTP/web tools for fetching external information
- Use text processing tools for analyzing or manipulating text
- Use JSON tools for working with structured data
- Always prefer tool-based solutions over manual work

Be helpful, accurate, and make full use of your available tools.`,
}

// ToolUsagePrompts contains specific instructions for each tool
var ToolUsagePrompts = map[string]string{
	"calculator": `CALCULATOR TOOL USAGE:
- Use for ANY mathematical calculation, even simple addition
- Supported operations: addition (+), subtraction (-), multiplication (*), division (/), power (^), square root (sqrt(n)), absolute value (abs(n))
- Format: Use expressions like "5 + 3", "2 * 7", "sqrt(16)", "abs(-5)", or "2^3"
- ALWAYS use this tool instead of mental math
- Example: To calculate 15 * 23, call calculator with "15 * 23"`,

	"http_get": `HTTP GET TOOL USAGE:
- Use to fetch data from web APIs and URLs
- Can retrieve JSON data, HTML content, or plain text
- Include timeout parameter (recommended: 10 seconds)
- Add custom headers if needed for API authentication
- Example: Fetch data from "https://api.example.com/data"`,

	"http_post": `HTTP POST TOOL USAGE:
- Use to send data to web APIs via POST requests
- Can send JSON data, form data, or custom content
- Include proper Content-Type headers
- Use for API interactions that require data submission
- Example: Send user data to create new records`,

	"web_scraper": `WEB SCRAPER TOOL USAGE:
- Use to extract text content from web pages
- Automatically removes HTML tags and formats content
- Set max_length parameter to limit content size
- Best for getting readable text from websites
- Example: Extract article content from news websites`,

	"text_processor": `TEXT PROCESSOR TOOL USAGE:
- Use for text manipulation and analysis operations
- Operations: uppercase, lowercase, word_count, char_count, reverse, extract_emails, extract_urls
- Always specify both "text" and "operation" parameters
- Example: Count words in text using operation "word_count"`,

	"json_processor": `JSON PROCESSOR TOOL USAGE:
- Use for JSON data manipulation and analysis
- Operations: validate, pretty_print, minify, extract_keys, get_value
- For get_value operation, specify "path" parameter (e.g., "user.name")
- Example: Extract keys from JSON object or format JSON data`,

	"mcp_proxy": `MCP PROXY TOOL USAGE:
- Use to connect to Model Context Protocol servers
- Actions: initialize, list_resources, get_resource, list_tools, call_tool
- Always specify server_url and action parameters
- Example: Initialize connection to MCP server before other operations`,

	"openmcp_proxy": `OPENMCP PROXY TOOL USAGE:
- Use to connect to OpenMCP REST API servers
- Actions: discovery, execute_tool, get_schema
- Specify server_url, action, and tool-specific parameters
- Example: Discover available tools or execute specific OpenMCP tools`,

	"memory": `MEMORY TOOL USAGE:
- Use to store and recall information for adaptive behavior and personalized interactions
- Actions: store, recall, search, update, delete, stats
- Memory types: preference (user preferences), fact (factual information), conversation (conversation context), behavior (learned behaviors)
- Store: action="store", topic="topic_name", content="memory content", memory_type="preference", importance=1-10
- Recall: action="recall", topic="topic_name", limit=10
- Search: action="search", query="search terms", memory_type="preference", limit=5
- Update: action="update", memory_id="id", content="new content"
- Delete: action="delete", memory_id="id"
- Stats: action="stats" (get memory statistics)
- ALWAYS use memory to learn user preferences and adapt your behavior
- Store important facts and preferences to provide personalized responses
- Example: Store user communication style preferences, recall conversation context`,
}

// BuildSystemPrompt creates a comprehensive system prompt with dynamic tool descriptions
func (ps *PromptService) BuildSystemPrompt(ctx context.Context, basePrompt string, availableTools []string) string {
	var prompt strings.Builder
	
	// Start with base prompt
	if basePrompt == "" {
		basePrompt = SystemPrompts.ToolEnabled
	}
	prompt.WriteString(basePrompt)
	prompt.WriteString("\n\n")

	// Add tool descriptions if tools are available
	if len(availableTools) > 0 {
		prompt.WriteString("=== AVAILABLE TOOLS ===\n")
		prompt.WriteString("You have access to the following tools. Use them whenever appropriate:\n\n")

		for _, toolName := range availableTools {
			// Get tool schema from tool service
			if tool, exists := ps.toolService.GetRegistry().Get(toolName); exists {
				schema := tool.Schema()
				
				prompt.WriteString(fmt.Sprintf("🔧 **%s**: %s\n", schema.Name, schema.Description))
				
				// Add usage instructions if available
				if usage, exists := ToolUsagePrompts[toolName]; exists {
					prompt.WriteString(usage)
					prompt.WriteString("\n\n")
				} else {
					// Generate basic usage from schema
					prompt.WriteString(ps.generateBasicUsage(schema))
					prompt.WriteString("\n\n")
				}
			}
		}

		prompt.WriteString("=== TOOL USAGE REMINDER ===\n")
		prompt.WriteString("- ALWAYS use tools when they match the task requirements\n")
		prompt.WriteString("- Don't perform manual work that tools can do\n")
		prompt.WriteString("- Explain which tool you're using and why\n")
		prompt.WriteString("- Use multiple tools if needed to complete complex tasks\n\n")
	}

	return prompt.String()
}

// generateBasicUsage creates basic usage instructions from tool schema
func (ps *PromptService) generateBasicUsage(schema tools.Schema) string {
	var usage strings.Builder
	usage.WriteString("Parameters:\n")
	
	for _, param := range schema.Parameters {
		required := "optional"
		if param.Required {
			required = "required"
		}
		
		usage.WriteString(fmt.Sprintf("  - %s (%s, %s): %s\n", 
			param.Name, param.Type, required, param.Description))
	}
	
	if len(schema.Examples) > 0 {
		usage.WriteString("Examples:\n")
		for _, example := range schema.Examples {
			usage.WriteString(fmt.Sprintf("  - %s\n", example.Description))
		}
	}
	
	return usage.String()
}

// BuildEnhancedSystemPrompt creates a system prompt optimized for specific agent types
func (ps *PromptService) BuildEnhancedSystemPrompt(ctx context.Context, agentType string, availableTools []string, customPrompt string) string {
	var basePrompt string
	
	// Select appropriate base prompt based on agent type
	switch agentType {
	case "math", "calculator", "mathematical":
		basePrompt = SystemPrompts.MathAssistant
	case "coding", "programming", "development":
		basePrompt = SystemPrompts.CodingAssistant
	case "research", "analysis", "investigation":
		basePrompt = SystemPrompts.ResearchAssistant
	default:
		basePrompt = SystemPrompts.GeneralAssistant
	}
	
	// Override with custom prompt if provided
	if customPrompt != "" {
		basePrompt = customPrompt + "\n\n" + "You have access to tools that can help you complete tasks more accurately and efficiently."
	}
	
	return ps.BuildSystemPrompt(ctx, basePrompt, availableTools)
}

// GetToolChoicePrompt returns additional prompt text to encourage tool usage
func (ps *PromptService) GetToolChoicePrompt(availableTools []string) string {
	if len(availableTools) == 0 {
		return ""
	}
	
	toolList := strings.Join(availableTools, ", ")
	return fmt.Sprintf(`

TOOL CHOICE GUIDANCE:
Available tools: %s

Consider each user request carefully:
1. Can any of your tools help with this task?
2. Would using a tool provide more accurate results?
3. Is this a task that should be done with tools rather than manually?

If yes to any of these, USE THE APPROPRIATE TOOLS.`, toolList)
}

// ValidateToolUsage analyzes if tools should have been used for a given request
func (ps *PromptService) ValidateToolUsage(request string, availableTools []string) []string {
	var suggestions []string
	requestLower := strings.ToLower(request)
	
	// Check for mathematical operations
	if containsAny(requestLower, []string{"calculate", "add", "subtract", "multiply", "divide", "square root", "sqrt", "+", "-", "*", "/", "=", "math"}) {
		if contains(availableTools, "calculator") {
			suggestions = append(suggestions, "calculator: This request involves mathematical calculations")
		}
	}
	
	// Check for web requests
	if containsAny(requestLower, []string{"fetch", "get data", "api", "url", "website", "http", "download"}) {
		if contains(availableTools, "http_get") {
			suggestions = append(suggestions, "http_get: This request involves fetching web data")
		}
		if contains(availableTools, "web_scraper") {
			suggestions = append(suggestions, "web_scraper: This request involves web content extraction")
		}
	}
	
	// Check for text processing
	if containsAny(requestLower, []string{"count words", "extract", "uppercase", "lowercase", "process text", "analyze text"}) {
		if contains(availableTools, "text_processor") {
			suggestions = append(suggestions, "text_processor: This request involves text manipulation")
		}
	}
	
	// Check for JSON operations
	if containsAny(requestLower, []string{"json", "parse", "format json", "validate json"}) {
		if contains(availableTools, "json_processor") {
			suggestions = append(suggestions, "json_processor: This request involves JSON processing")
		}
	}
	
	return suggestions
}

// ConversationPrompts contains prompts for different conversation states
var ConversationPrompts = struct {
	ToolCallIntro    string
	ToolCallSuccess  string
	ToolCallFailure  string
	MultiStepStart   string
	MultiStepNext    string
	NoToolsAvailable string
}{
	ToolCallIntro: "I'll use the %s tool to help with this task.",
	
	ToolCallSuccess: "Great! The %s tool provided the result: %s. Let me explain what this means:",
	
	ToolCallFailure: "I attempted to use the %s tool, but encountered an issue. Let me try a different approach:",
	
	MultiStepStart: "This task requires multiple steps. I'll use several tools to complete it:",
	
	MultiStepNext: "Next, I'll use the %s tool to %s:",
	
	NoToolsAvailable: "I don't have specific tools for this task, so I'll provide the best answer I can with my knowledge:",
}

// GetConversationPrompt returns appropriate conversation prompts
func (ps *PromptService) GetConversationPrompt(promptType string, args ...interface{}) string {
	switch promptType {
	case "tool_intro":
		if len(args) >= 1 {
			return fmt.Sprintf(ConversationPrompts.ToolCallIntro, args[0])
		}
	case "tool_success":
		if len(args) >= 2 {
			return fmt.Sprintf(ConversationPrompts.ToolCallSuccess, args[0], args[1])
		}
	case "tool_failure":
		if len(args) >= 1 {
			return fmt.Sprintf(ConversationPrompts.ToolCallFailure, args[0])
		}
	case "multi_step_start":
		return ConversationPrompts.MultiStepStart
	case "multi_step_next":
		if len(args) >= 2 {
			return fmt.Sprintf(ConversationPrompts.MultiStepNext, args[0], args[1])
		}
	case "no_tools":
		return ConversationPrompts.NoToolsAvailable
	}
	return ""
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}