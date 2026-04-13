package ai

import _ "embed"

//go:embed prompts/chat_system.md
var chatSystemPrompt string

//go:embed prompts/sql_system.md
var sqlSystemPrompt string
