# AGENTS.md - Code Patterns & Architecture Guide

This document provides comprehensive guidance for AI agents and developers working on the htmbot codebase. It describes the architectural patterns, conventions, and best practices used throughout the project.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Directory Structure](#directory-structure)
4. [Core Patterns](#core-patterns)
5. [Code Conventions](#code-conventions)
6. [Domain Layer](#domain-layer)
7. [Platform Layer](#platform-layer)
8. [Module Layer](#module-layer)
9. [Frontend Patterns](#frontend-patterns)
10. [Database Patterns](#database-patterns)
11. [Testing Strategy](#testing-strategy)
12. [Common Tasks](#common-tasks)

---

## Project Overview

htmbot is a Go web application that implements an AI-powered chat interface using:

- **Backend**: Go with Echo framework
- **Frontend**: HTMX + Alpine.js + Tailwind CSS + DaisyUI
- **Templating**: Templ (type-safe Go templates)
- **Database**: PostgreSQL with pgx driver
- **AI**: OpenAI API (gpt-4o-mini)
- **Architecture**: Event-driven with pub/sub messaging

The application demonstrates clean architecture with clear separation between domain logic, platform utilities, and feature modules.

---

## Architecture

### Layered Architecture

```
┌─────────────────────────────────────────┐
│         cmd/ (Entry Points)             │
│  - Dependency injection & assembly      │
└─────────────────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
┌───────▼────────┐  ┌────────▼────────┐
│  modules/      │  │  platform/      │
│  (Features)    │  │  (Utilities)    │
│                │  │                 │
│  - chat        │  │  - agents       │
│  - handlers    │  │  - httpx        │
│  - views       │  │  - pubsub       │
│  - repository  │  │  - slicesx      │
└───────┬────────┘  └────────┬────────┘
        │                    │
        └─────────┬──────────┘
                  │
        ┌─────────▼──────────┐
        │     domain/        │
        │   (Interfaces)     │
        │                    │
        │  - ChatRepository  │
        │  - LLMAgent        │
        │  - PubSub          │
        └────────────────────┘
```

### Key Principles

1. **Dependency Inversion**: Domain defines interfaces, implementations live elsewhere
2. **Dependency Injection**: All dependencies passed via constructors
3. **Interface Segregation**: Small, focused interfaces
4. **Event-Driven**: Async processing via pub/sub channels
5. **Type Safety**: Generics for reusable components

---

## Directory Structure

```
htmbot/
├── cmd/
│   ├── app/main.go              # Web server entry point
│   └── cli/main.go              # CLI testing utility
├── domain/
│   ├── chat.go                  # Core domain types & interfaces
│   ├── llm.go                   # LLM interfaces & tools
│   └── pubsub.go                # Pub/sub interface
├── modules/
│   └── chat/
│       ├── handler.go           # HTTP handlers (Echo)
│       ├── repository.go        # Data access implementations
│       ├── message_processor.go # Async message processing
│       ├── jobs.go              # Message enqueuer
│       ├── tools.go             # LLM tools
│       └── views/
│           ├── *.templ          # Templ templates
│           └── *_templ.go       # Generated Go code (don't edit)
├── platform/
│   ├── agents/
│   │   ├── openai.go            # OpenAI API integration
│   │   └── tools.go             # Generic LLM tool framework
│   ├── httpx/
│   │   ├── redirect.go          # HTMX redirect helper
│   │   ├── sse.go               # Server-Sent Events utilities
│   │   └── templ.go             # Templ rendering helper
│   ├── pubsub/
│   │   └── channel.go           # Generic pub/sub implementation
│   ├── slicesx/
│   │   └── slicesx.go           # Generic slice utilities
│   └── components/
│       └── page.templ           # Shared UI components
├── assets/
│   └── styles.css               # Tailwind CSS
├── migrations/
│   └── *.sql                    # Database migrations (goose)
├── go.mod
└── docker-compose.yml           # PostgreSQL setup
```

### File Organization Rules

- **One logical concept per file**: `handler.go`, `repository.go`, etc.
- **Generated files separate**: `*_templ.go` files are auto-generated
- **Package names**: Lowercase, single word (`chat`, `httpx`, `slicesx`)
- **No circular dependencies**: Domain never imports platform/modules

---

## Core Patterns

### 1. Dependency Injection Pattern

All dependencies are injected via constructor functions following the `New<Type>` convention.

**Example: Handler Construction**

```go
// Definition (modules/chat/handler.go)
type Handler struct {
    enqueuer   domain.MessageEnqueuer
    pubsub     domain.PubSub[domain.ChatEvent]
    repository domain.ChatRepository
}

func NewHandler(
    enqueuer domain.MessageEnqueuer,
    pubsub domain.PubSub[domain.ChatEvent],
    repository domain.ChatRepository,
) *Handler {
    return &Handler{
        enqueuer:   enqueuer,
        pubsub:     pubsub,
        repository: repository,
    }
}

// Assembly (cmd/app/main.go)
chatRepository := chat.NewPGXRepository(dbConn)
publisher := pubsub.NewChannel(map[string][]chan domain.ChatEvent{})
enqueuer := chat.NewMessageEnqueuer(messagesChannel)
chatHandler := chat.NewHandler(enqueuer, publisher, chatRepository)
```

**Rules:**

- Use unexported struct fields (lowercase)
- Accept interfaces, return concrete types
- No initialization logic in constructors (just assignment)

### 2. Repository Pattern

The repository pattern abstracts data access behind interfaces defined in the domain layer.

**Example: Chat Repository**

```go
// Interface (domain/chat.go)
type ChatRepository interface {
    GetMessages(ctx context.Context, params GetMessagesParams) ([]ChatMessage, error)
    SaveMessage(ctx context.Context, sessionId string, messages ...ChatMessage) error
    CreateChat(ctx context.Context, sessionId string) (ChatSession, error)
    ListSessions(ctx context.Context) ([]ChatSession, error)
    GetSessionName(ctx context.Context, chatId string) (string, error)
    DeleteSession(ctx context.Context, chatId string) error
}

// Implementation (modules/chat/repository.go)
type PGXRepository struct {
    pool *pgxpool.Pool
}

func NewPGXRepository(pool *pgxpool.Pool) *PGXRepository {
    return &PGXRepository{pool: pool}
}

// Verify implementation satisfies interface
var _ domain.ChatRepository = &PGXRepository{}
```

**Best Practices:**

- Define interfaces in `domain/` package
- Implement in module packages
- Use `var _ Interface = &Implementation{}` to verify interface compliance
- Create `InMemory` implementations for testing

### 3. Generic Pub/Sub Pattern

Type-safe, topic-based pub/sub using Go generics.

**Example: Publisher Setup**

```go
// Interface (domain/pubsub.go)
type PubSub[T any] interface {
    Subscribe(topic string) (chan T, func(), error)
    Publish(topic string, message T) error
}

// Implementation (platform/pubsub/channel.go)
type Channel[T any] struct {
    subscriptions map[string][]chan T
    mu            sync.RWMutex
}

func NewChannel[T any](subscriptions map[string][]chan T) *Channel[T] {
    return &Channel[T]{subscriptions: subscriptions}
}

// Usage (modules/chat/handler.go)
messagesChannel, unsub, err := h.pubsub.Subscribe(chatName)
if err != nil {
    return fmt.Errorf("failed to subscribe to chat: %w", err)
}
defer unsub() // Cleanup function returned by Subscribe

// In another goroutine
h.pubsub.Publish(chatName, domain.ChatEvent{
    Type:      "message",
    OfMessage: newMessage,
})
```

**Key Features:**

- Generic type `T` for type safety
- Topic-based routing
- Unsubscribe callback for cleanup
- Thread-safe with `sync.RWMutex`
- Buffered channels (1000 cap) prevent blocking

### 4. Handler Pattern (Echo Framework)

HTTP handlers follow Echo's conventions with route registration.

**Example: Handler Registration**

```go
func (h *Handler) Register(e *echo.Echo) {
    g := e.Group("/chat")
    g.GET("", h.Index)
    g.POST("", h.Create)
    g.GET("/:chat-id", h.ChatPage)
    g.POST("/:chat-id/send-message", h.SendMessage)
    g.GET("/:chat-id/sse", h.ListenForMessages)
    g.DELETE("/:chat-id", h.DeleteChat)
}

func (h *Handler) SendMessage(c echo.Context) error {
    chatName := c.Param("chat-name")
    text := c.FormValue("chat-input")

    if text == "" {
        return c.NoContent(http.StatusOK)
    }

    // Business logic...

    return httpx.Render(c, chatviews.ChatForm(chatName))
}
```

**Conventions:**

- Group related routes with `e.Group(prefix)`
- Handler methods: `func (h *Handler) Name(c echo.Context) error`
- Use `c.Param()` for path parameters
- Use `c.FormValue()` for form data
- Return errors (Echo handles rendering)
- Use `httpx.Render()` for Templ templates

### 5. Event-Driven Processing

Asynchronous message processing using channels and goroutines.

**Example: Message Processor**

```go
// Enqueuer (modules/chat/jobs.go)
type MessageEnqueuer struct {
    ch chan domain.ChatEvent
}

func (m *MessageEnqueuer) EnqueueUserMessage(ctx context.Context, chatName, message string) error {
    m.ch <- domain.ChatEvent{
        ChatSessionID: chatName,
        Type:          "message",
        OfMessage:     domain.ChatMessage{Role: "user", Content: message},
    }
    return nil
}

// Processor (modules/chat/message_processor.go)
func (p *MessageProcessor) ProcessUserMessages(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case newMessage := <-p.ch:
            // Process message asynchronously
            response, err := p.agent.StreamResponse(ctx, messages, tools, callback)
            if err != nil {
                log.Errorf("failed: %w", err)
                continue // Don't crash on single message failure
            }
            // Save response...
        }
    }
}

// Startup (cmd/app/main.go)
messagesChannel := make(chan domain.ChatEvent, 1000)
messagesProcessor := chat.NewMessageProcessor(messagesChannel, publisher, agent, chatRepository)
go messagesProcessor.ProcessUserMessages(context.Background())
```

**Best Practices:**

- Buffer channels appropriately (1000 for high throughput)
- Always handle `ctx.Done()` for graceful shutdown
- Log errors in async operations, don't crash
- Use `go` statement at startup in main.go

### 6. Generic LLM Tool Pattern

Type-safe tool definition for LLM function calling.

**Example: Tool Definition**

```go
// domain/llm.go
type LLMTool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Execute(ctx context.Context, args string) (string, error)
}

// platform/agents/tools.go
func NewLLMTool[T any, K any](
    name, description string,
    execute func(context.Context, T) (K, error),
) *LLMTool {
    // Implementation uses JSON marshaling for type conversion
}

// Usage (modules/chat/tools.go)
type TestToolArgs struct {
    Message string `json:"message" jsonschema:"description=A test message"`
}

func NewTestTool() domain.LLMTool {
    return agents.NewLLMTool(
        "test_tool",
        "A test tool for demonstration",
        func(ctx context.Context, args TestToolArgs) (string, error) {
            return fmt.Sprintf("Received: %s", args.Message), nil
        },
    )
}
```

**Features:**

- Generic input/output types
- Automatic JSON schema generation
- Type-safe argument parsing
- Error handling built-in

---

## Code Conventions

### Naming Conventions

| Type          | Convention             | Example                                |
| ------------- | ---------------------- | -------------------------------------- |
| Constructors  | `New<TypeName>`        | `NewHandler`, `NewPGXRepository`       |
| Interfaces    | Descriptive, no prefix | `ChatRepository`, `LLMAgent`, `PubSub` |
| Struct fields | Unexported (lowercase) | `handler.repository`                   |
| Packages      | Lowercase, single word | `chat`, `httpx`, `slicesx`             |
| Methods       | Exported if public     | `Register`, `ProcessUserMessages`      |
| Constants     | `camelCase` queries    | `createChatQuery`, `getMessagesQuery`  |

### Error Handling

**In HTTP Handlers:**

```go
func (h *Handler) ChatPage(c echo.Context) error {
    chatMessages, err := h.repository.GetMessages(c.Request().Context(), params)
    if err != nil {
        return fmt.Errorf("failed to get messages: %w", err)
    }
    // Echo framework handles error rendering
}
```

**In Async Operations:**

```go
func (p *MessageProcessor) ProcessUserMessages(ctx context.Context) error {
    for {
        case newMessage := <-p.ch:
            if err := p.processMessage(newMessage); err != nil {
                log.Errorf("failed to process message: %w", err)
                continue // Don't crash, continue processing
            }
    }
}
```

**At Startup:**

```go
func main() {
    dbConn, err := pgxpool.New(context.Background(), mustEnv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err) // Fatal errors at startup
    }
}
```

**Rules:**

- Wrap errors with `fmt.Errorf("...: %w", err)` for context
- Use `log.Fatal` only at startup for unrecoverable errors
- Use `log.Errorf` in async operations (log and continue)
- Return errors in handlers (let Echo handle them)

### Parameter Objects

For functions with multiple related parameters, use parameter structs with defaults.

```go
// domain/chat.go
type GetMessagesParams struct {
    ChatSessionId string
    Before        time.Time
    Limit         int
}

func (g *GetMessagesParams) ApplyDefaults() {
    if g.Before.IsZero() {
        g.Before = time.Now()
    }
    if g.Limit == 0 {
        g.Limit = 20
    }
}

// Usage
messages, err := repo.GetMessages(ctx, domain.GetMessagesParams{
    ChatSessionId: chatId,
    Limit:         100,
})
```

### Variadic Arguments

Use variadic arguments for batch operations:

```go
func (p *PGXRepository) SaveMessage(
    ctx context.Context,
    chatSessionId string,
    messages ...domain.ChatMessage,
) error {
    // Batch insert using pgx.CopyFrom
}

// Usage
repo.SaveMessage(ctx, sessionId, msg1, msg2, msg3)
```

---

## Domain Layer

The `domain/` package defines core business entities and interfaces. It has NO dependencies on other packages.

### Domain Types

**Example: Chat Domain (domain/chat.go)**

```go
// Entity with database tags
type ChatSession struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Entity with optional fields (pointers)
type ChatMessage struct {
    ID               string    `json:"id" db:"id"`
    Role             string    `json:"role" db:"role"`
    Content          string    `json:"content" db:"content"`
    ChatSessionID    string    `json:"chat_session_id" db:"chat_session_id"`
    ReasoningSummary *string   `json:"reasoning_summary" db:"reasoning_summary"`
    Name             *string   `json:"name" db:"name"` // Tool name
    Args             *string   `json:"args" db:"args"` // Tool arguments
    CallID           *string   `json:"call_id" db:"call_id"` // LLM call ID
    Result           *string   `json:"result" db:"result"` // Tool result
    CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
```

**Tagging Conventions:**

- `json:"field_name"` for JSON serialization
- `db:"column_name"` for database mapping
- Use pointers for optional/nullable fields

### Domain Events

Events use a union type pattern with getter methods:

```go
type ChatEvent struct {
    ChatSessionID string
    Type          string // "message", "delta", "delta_start"
    OfMessage     ChatMessage
    OfDelta       ChatDelta
}

// Getter methods for type-safe access
func (c *ChatEvent) Delta() ChatDelta {
    return c.OfDelta
}

func (c *ChatEvent) Message() ChatMessage {
    return c.OfMessage
}
```

**Usage in templates:**

```go
templ GetMessageTemplate(event domain.ChatEvent) {
    switch event.Type {
        case "delta":
            @MessageDelta(event.Delta().ID, event.Delta().Text)
        case "delta_start":
            @MessageDeltaStart(event.Delta().ID)
        default:
            @Message(event.OfMessage)
    }
}
```

---

## Platform Layer

The `platform/` package provides cross-cutting utilities that implement domain interfaces.

### HTTP Utilities (platform/httpx/)

**Templ Rendering:**

```go
func Render(c echo.Context, component templ.Component) error {
    return component.Render(c.Request().Context(), c.Response())
}

// Usage
return httpx.Render(c, chatviews.ChatPage(chatName, messages))
```

**HTMX Redirect:**

```go
func HxRedirect(c echo.Context, url string) error {
    c.Response().Header().Set("HX-Redirect", url)
    return c.NoContent(http.StatusNoContent)
}

// Usage
return httpx.HxRedirect(c, "/chat")
```

**Server-Sent Events:**

```go
func SetupSSE(c echo.Context) {
    w := c.Response()
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
}

func WriteEventStreamTemplate(c echo.Context, event string, template templ.Component) error {
    var buf bytes.Buffer
    if err := template.Render(c.Request().Context(), &buf); err != nil {
        return err
    }
    return WriteEventStream(c.Response(), event, buf.String())
}

// Usage
httpx.SetupSSE(c)
for message := range messagesChannel {
    httpx.WriteEventStreamTemplate(c, "chat-messages", chatviews.GetMessageTemplate(message))
    c.Response().Flush()
}
```

### Slice Utilities (platform/slicesx/)

Generic utilities for slice operations:

```go
// Map transforms a slice
func Map[T any, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// Usage
openaiMessages := slicesx.Map(messages, o.chatMessageToOpenAIMessage)
```

**When to add utilities:**

- Function is reusable across multiple modules
- Generic implementation provides type safety
- No business logic, pure transformation

---

## Module Layer

Modules implement complete features. The `modules/chat/` module demonstrates all patterns.

### Handler Layer

**Responsibilities:**

- Route registration
- HTTP request/response handling
- Input validation
- Calling business logic
- Returning templates

**Example Pattern:**

```go
func (h *Handler) SendMessage(c echo.Context) error {
    // 1. Extract input
    chatName := c.Param("chat-id")
    text := c.FormValue("chat-input")

    // 2. Validate
    if text == "" {
        return c.NoContent(http.StatusOK)
    }

    // 3. Create domain object
    newMessage := domain.ChatMessage{Role: "user", Content: text}

    // 4. Call repository
    if err := h.repository.SaveMessage(c.Request().Context(), chatName, newMessage); err != nil {
        return fmt.Errorf("failed to save user message: %w", err)
    }

    // 5. Publish event
    if err := h.pubsub.Publish(chatName, domain.ChatEvent{
        Type:          "message",
        ChatSessionID: chatName,
        OfMessage:     newMessage,
    }); err != nil {
        return fmt.Errorf("failed to publish user message: %w", err)
    }

    // 6. Enqueue for processing
    if err := h.enqueuer.EnqueueUserMessage(c.Request().Context(), chatName, text); err != nil {
        return fmt.Errorf("failed to enqueue user message: %w", err)
    }

    // 7. Return template
    return httpx.Render(c, chatviews.ChatForm(chatName))
}
```

### Repository Layer

**Best Practices:**

**1. Define queries as constants:**

```go
const getMessagesQuery = `
SELECT id, role, content, name, args, call_id, result, chat_session_id
FROM chat_messages
WHERE chat_session_id = $1
AND created_at < $2
ORDER BY created_at DESC
LIMIT $3;
`
```

**2. Use pgx's type-safe collection methods:**

```go
func (p *PGXRepository) CreateChat(ctx context.Context, chatName string) (domain.ChatSession, error) {
    rows, err := p.pool.Query(ctx, createChatQuery, chatName)
    if err != nil {
        return domain.ChatSession{}, fmt.Errorf("failed to create chat: %w", err)
    }
    return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[domain.ChatSession])
}

func (p *PGXRepository) ListSessions(ctx context.Context) ([]domain.ChatSession, error) {
    rows, err := p.pool.Query(ctx, listSessionsQuery)
    if err != nil {
        return nil, err
    }
    return pgx.CollectRows(rows, pgx.RowToStructByName[domain.ChatSession])
}
```

**3. Use CopyFrom for batch inserts:**

```go
func (p *PGXRepository) SaveMessage(ctx context.Context, chatSessionId string, messages ...domain.ChatMessage) error {
    rows := [][]any{}
    for _, message := range messages {
        rows = append(rows, []any{
            message.Role,
            message.Content,
            message.Name,
            message.Args,
            message.CallID,
            message.Result,
            chatSessionId,
        })
    }

    _, err := p.pool.CopyFrom(
        ctx,
        pgx.Identifier([]string{"chat_messages"}),
        []string{"role", "content", "name", "args", "call_id", "result", "chat_session_id"},
        pgx.CopyFromRows(rows),
    )
    if err != nil {
        return fmt.Errorf("failed to save chat messages: %w", err)
    }
    return nil
}
```

**4. Create InMemory implementations for testing:**

```go
type InMemoryRepository struct {
    storage map[string][]domain.ChatMessage
}

func NewInMemoryRepository() *InMemoryRepository {
    return &InMemoryRepository{
        storage: make(map[string][]domain.ChatMessage),
    }
}
```

---

## Frontend Patterns

### Templ Templates

Templ provides type-safe Go templates that compile to Go code.

**Basic Component:**

```go
package chatviews

import (
    "github.com/raphael-foliveira/htmbot/domain"
    "github.com/raphael-foliveira/htmbot/platform/components"
)

templ ChatPage(chatName string, chatMessages []domain.ChatMessage) {
    @components.Page(chatName) {
        @ChatContainer(chatName, chatMessages)
    }
}

templ ChatContainer(chatName string, chatMessages []domain.ChatMessage) {
    <div id="chat-container" class="w-full max-w-200 mx-auto h-screen flex flex-col">
        <div class="flex flex-col gap-1 p-4 chat flex-1 overflow-auto">
            for _, msg := range chatMessages {
                @Message(msg)
            }
        </div>
    </div>
}
```

**Templ Syntax:**

- `templ ComponentName(args) { ... }` defines a component
- `@OtherComponent(args)` calls another component
- `{ goExpression }` evaluates Go code
- Regular HTML tags work as-is
- `for` loops and `if` statements use Go syntax

### HTMX Patterns

**Form Submission:**

```go
templ ChatForm(chatName string) {
    <form
        hx-post={ fmt.Sprintf("/chat/%s/send-message", chatName) }
        hx-target="this"
        hx-swap="outerHTML"
        x-data="{isSubmitting: false}"
        @submit="isSubmitting = true"
    >
        <textarea name="chat-input" class="textarea w-full"></textarea>
        <button type="submit" class="btn btn-neutral">Send</button>
    </form>
}
```

**Server-Sent Events:**

```go
<div
    hx-ext="sse"
    sse-connect={ fmt.Sprintf("/chat/%s/sse", chatName) }
    sse-swap="chat-messages"
    hx-swap="beforeend"
>
    <!-- Messages appear here -->
</div>
```

**Out-of-Band Swaps (OOB):**

```go
templ MessageDelta(eventId, content string) {
    <div
        class="chat chat-start"
        id={ eventId }
        hx-swap-oob="true"
    >
        <div class="chat-bubble">
            <span>{ content }</span>
        </div>
    </div>
}
```

### Alpine.js Patterns

**Reactive State:**

```go
<form
    x-data="{isSubmitting: false}"
    @submit="isSubmitting = true"
>
    <button
        type="submit"
        x-bind:disabled="isSubmitting"
    >
        <span x-show="!isSubmitting">Send</span>
        <span x-show="isSubmitting">Loading...</span>
    </button>
</form>
```

**Lifecycle Hooks:**

```go
<div
    x-init="$nextTick(() => $el.scrollTop = $el.scrollHeight)"
    @htmx:after-swap="$el.scrollTop = $el.scrollHeight"
>
    <!-- Auto-scroll to bottom on load and after HTMX swap -->
</div>
```

### Helper Functions in Templates

Templ allows regular Go functions within template files:

```go
func resolveMessageClass(role string) string {
    switch role {
    case "user":
        return "chat-end ml-auto"
    default:
        return "chat-start mr-auto"
    }
}

templ Message(msg domain.ChatMessage) {
    <div class={ fmt.Sprintf("chat %s", resolveMessageClass(msg.Role)) }>
        <div class="chat-bubble">{ msg.Content }</div>
    </div>
}
```

---

## Database Patterns

### Migrations

Use goose format for migrations in `migrations/` directory:

```sql
-- +goose Up
CREATE TABLE chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    chat_session_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    name VARCHAR(255),
    args TEXT,
    call_id VARCHAR(255),
    result TEXT,
    reasoning_summary TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_chat_messages_session_id ON chat_messages(chat_session_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at);

-- +goose Down
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chats;
```

**Conventions:**

- Use UUID for primary keys with `gen_random_uuid()`
- Always include `created_at` timestamps
- Add indexes for foreign keys and query fields
- Use `ON DELETE CASCADE` for dependent data
- Include both `Up` and `Down` migrations

### Struct Tags

Align struct tags for readability:

```go
type ChatMessage struct {
    ID               string    `json:"id" db:"id"`
    Role             string    `json:"role" db:"role"`
    Content          string    `json:"content" db:"content"`
    ChatSessionID    string    `json:"chat_session_id" db:"chat_session_id"`
    ReasoningSummary *string   `json:"reasoning_summary" db:"reasoning_summary"`
    CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
```

---

## Testing Strategy

### In-Memory Implementations

Create in-memory versions of repositories for testing:

```go
type InMemoryRepository struct {
    storage map[string][]domain.ChatMessage
}

func NewInMemoryRepository() *InMemoryRepository {
    return &InMemoryRepository{
        storage: make(map[string][]domain.ChatMessage),
    }
}

func (r *InMemoryRepository) SaveMessage(
    ctx context.Context,
    chatName string,
    messages ...domain.ChatMessage,
) error {
    r.storage[chatName] = append(r.storage[chatName], messages...)
    return nil
}

func (r *InMemoryRepository) GetMessages(
    ctx context.Context,
    params domain.GetMessagesParams,
) ([]domain.ChatMessage, error) {
    return r.storage[params.ChatSessionId], nil
}
```

### Interface Compliance Checks

Verify implementations satisfy interfaces at compile time:

```go
var _ domain.ChatRepository = &PGXRepository{}
var _ domain.ChatRepository = &InMemoryRepository{}
```

If the implementation doesn't satisfy the interface, compilation fails.

---

## Common Tasks

### Adding a New Feature Module

1. **Create module directory structure:**

```
modules/
└── newfeature/
    ├── handler.go
    ├── repository.go
    └── views/
        └── index.templ
```

2. **Define domain interfaces in `domain/newfeature.go`:**

```go
package domain

type NewFeatureRepository interface {
    GetData(ctx context.Context, id string) (Data, error)
    SaveData(ctx context.Context, data Data) error
}
```

3. **Implement repository:**

```go
package newfeature

type PGXRepository struct {
    pool *pgxpool.Pool
}

func NewPGXRepository(pool *pgxpool.Pool) *PGXRepository {
    return &PGXRepository{pool: pool}
}

var _ domain.NewFeatureRepository = &PGXRepository{}
```

4. **Create handler:**

```go
package newfeature

type Handler struct {
    repository domain.NewFeatureRepository
}

func NewHandler(repository domain.NewFeatureRepository) *Handler {
    return &Handler{repository: repository}
}

func (h *Handler) Register(e *echo.Echo) {
    g := e.Group("/newfeature")
    g.GET("", h.Index)
}

func (h *Handler) Index(c echo.Context) error {
    // Handler logic
    return httpx.Render(c, newfeatureviews.Index())
}
```

5. **Wire up in main.go:**

```go
newFeatureRepo := newfeature.NewPGXRepository(dbConn)
newFeatureHandler := newfeature.NewHandler(newFeatureRepo)
newFeatureHandler.Register(e)
```

### Adding an LLM Tool

1. **Define argument and return types:**

```go
// modules/chat/tools.go
type WeatherArgs struct {
    Location string `json:"location" jsonschema:"description=City name"`
    Units    string `json:"units" jsonschema:"enum=celsius,enum=fahrenheit"`
}

type WeatherResult struct {
    Temperature int    `json:"temperature"`
    Conditions  string `json:"conditions"`
}
```

2. **Implement tool function:**

```go
func NewWeatherTool() domain.LLMTool {
    return agents.NewLLMTool(
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, args WeatherArgs) (WeatherResult, error) {
            // Implementation
            return WeatherResult{
                Temperature: 72,
                Conditions:  "Sunny",
            }, nil
        },
    )
}
```

3. **Register in message processor:**

```go
response, err := p.agent.StreamResponse(
    ctx,
    messages,
    []domain.LLMTool{
        NewTestTool(),
        NewWeatherTool(), // Add here
    },
    callback,
)
```

### Adding a New Event Type

1. **Add to domain event:**

```go
// domain/chat.go
type ChatEvent struct {
    ChatSessionID string
    Type          string // Add new type: "error", "status", etc.
    OfMessage     ChatMessage
    OfDelta       ChatDelta
    OfError       ChatError // New field
}

func (c *ChatEvent) Error() ChatError {
    return c.OfError
}
```

2. **Create template handler:**

```go
// modules/chat/views/chat.templ
templ GetMessageTemplate(event domain.ChatEvent) {
    switch event.Type {
        case "delta":
            @MessageDelta(event.Delta().ID, event.Delta().Text)
        case "error":
            @ErrorMessage(event.Error())
        default:
            @Message(event.OfMessage)
    }
}

templ ErrorMessage(err domain.ChatError) {
    <div class="alert alert-error">{ err.Message }</div>
}
```

3. **Publish event:**

```go
if err := p.publisher.Publish(sessionId, domain.ChatEvent{
    Type:          "error",
    ChatSessionID: sessionId,
    OfError:       domain.ChatError{Message: err.Error()},
}); err != nil {
    log.Errorf("failed to publish error: %w", err)
}
```

### Adding HTTP Middleware

1. **Create middleware function:**

```go
// platform/httpx/middleware.go
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if token == "" {
            return echo.NewHTTPError(http.StatusUnauthorized)
        }
        // Validate token...
        return next(c)
    }
}
```

2. **Apply to routes:**

```go
func (h *Handler) Register(e *echo.Echo) {
    g := e.Group("/chat")
    g.Use(httpx.RequireAuth) // Apply to all routes in group
    g.GET("", h.Index)
}
```

### Environment Variables

Use the `mustEnv` pattern for required configuration:

```go
func mustEnv(key string) string {
    value := os.Getenv(key)
    if value == "" {
        log.Fatalf("environment variable %s is required", key)
    }
    return value
}

func main() {
    apiKey := mustEnv("OPENAI_API_KEY")
    dbURL := mustEnv("DATABASE_URL")
}
```

For optional config, use `os.Getenv` with defaults:

```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
```

---

## Summary Checklist

When writing new code, ensure:

- [ ] Interfaces defined in `domain/` package
- [ ] Dependencies injected via `New<Type>()` constructors
- [ ] Struct fields are unexported
- [ ] Errors wrapped with context: `fmt.Errorf("context: %w", err)`
- [ ] Repository implementation verified: `var _ domain.Interface = &Implementation{}`
- [ ] Handlers follow Echo conventions: `func (h) Name(c echo.Context) error`
- [ ] Route groups used for prefixes: `e.Group("/prefix")`
- [ ] Database queries as constants at package level
- [ ] Pub/sub topics match business concepts
- [ ] Context passed through all calls
- [ ] Channels buffered appropriately (typically 1000)
- [ ] Goroutines handle `ctx.Done()` for shutdown
- [ ] HTMX attributes for dynamic behavior
- [ ] Alpine.js for client-side reactivity
- [ ] Templ for type-safe templates
- [ ] Tailwind + DaisyUI for styling

---

_This guide reflects the patterns established in htmbot. Maintain consistency when extending the codebase._
