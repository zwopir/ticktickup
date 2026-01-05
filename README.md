# TickTickUp

TickTick doesn't allow importing tasks as CSV. This wails.io app allows dragging CSV or JSON files into the app which are then uploaded to TickTick using their developer API.

When starting the app, user authentication is requested if needed (token expired). A list of available projects and lists is fetched and presented as a dropdown to select where the items should be imported.

## Features

- Drag & drop CSV or JSON files to import tasks
- OAuth2 authentication with TickTick
- Automatic token refresh
- Select target project/list from dropdown
- Create new lists directly from the app
- Refresh project list
- Support for subtasks

## Setup

1. Register an app at the [TickTick Developer Portal](https://developer.ticktick.com/manage)
2. Set the OAuth Redirect URL to `http://localhost:8765/callback`
3. Copy your Client ID and Client Secret
4. Run the app and enter your credentials when prompted

## Import File Formats

### CSV Format

CSV files must include a header row. Supported columns:

| Column | Required | Description |
|--------|----------|-------------|
| `title` | Yes | Task title (alternatives: `name`, `task`) |
| `content` | No | Task description (alternative: `description`) |
| `dueDate` | No | Due date (alternatives: `due_date`, `due`) |
| `tags` | No | Comma-separated tags |
| `subtasks` | No | Semicolon-separated subtasks (see format below) |

#### Subtask Format in CSV

Each subtask uses pipe-separated fields: `title|startDate|dueDate`

- Only `title` is required
- Dates are optional and can be empty
- Multiple subtasks are separated by semicolons

Examples:
- Simple: `Milk;Eggs;Bread`
- With dates: `Research|2024-01-10|2024-01-15;Write report||2024-01-18`
- Mixed: `Task 1|2024-01-10|2024-01-15;Task 2;Task 3||2024-01-20`

#### CSV Example

```csv
title,content,dueDate,tags,subtasks
Buy groceries,Weekly shopping,2024-01-15,"shopping,home","Milk;Eggs;Bread"
Project work,Important project,2024-01-20,work,"Research|2024-01-10|2024-01-15;Write report||2024-01-18;Review"
Simple task,Just a note,,,
```

### JSON Format

JSON files can be:
- An array of task objects
- An object with a `tasks` array
- A single task object

#### Task Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Task title |
| `content` | string | No | Task description |
| `dueDate` | string | No | Due date (ISO format) |
| `priority` | int | No | Priority (0-5) |
| `tags` | string[] | No | Array of tags |
| `subtasks` | object[] | No | Array of subtask objects |

#### Subtask Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Subtask title |
| `description` | string | No | Subtask description |
| `startDate` | string | No | Start date (ISO format) |
| `dueDate` | string | No | Due date (ISO format) |

#### JSON Example

```json
[
  {
    "title": "Buy groceries",
    "content": "Weekly shopping",
    "dueDate": "2024-01-15",
    "tags": ["shopping", "home"],
    "subtasks": [
      {"title": "Milk"},
      {"title": "Eggs"},
      {"title": "Bread"}
    ]
  },
  {
    "title": "Project tasks",
    "content": "Q1 deliverables",
    "priority": 3,
    "subtasks": [
      {
        "title": "Research phase",
        "startDate": "2024-01-10",
        "dueDate": "2024-01-15"
      },
      {
        "title": "Write report",
        "dueDate": "2024-01-20"
      },
      {"title": "Review and submit"}
    ]
  },
  {
    "title": "Simple task",
    "content": "Just a note"
  }
]
```

## Development

### Prerequisites

- Go 1.25+
- Node.js 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

### Live Development

```bash
wails dev
```

### Building

```bash
wails build
```

The built application will be in `build/bin/`.

## Configuration

Configuration is stored in `~/.ticktickup/`:

- `config.json` - Client ID and Secret
- `token.json` - OAuth tokens (auto-managed)

## License

MIT
