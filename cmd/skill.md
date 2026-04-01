---
name: ax
description: 'This skill teaches you how to use the ax CLI to discover, spawn & communicate with other coding agents in a tmux session. Use this skill either when a user asks you spawn / communicate with other agents explicitly, or when you receive a prompt that looks like a ax sms "[ax sms from <label> (<role>)]: <message>".'
---

# `ax` - agent exchange CLI
`ax` is a lightweight CLI that lets coding agents running in tmux panes discover each other and exchange short messages. It provides session management, agent registration, peer discovery, and message delivery — nothing more.

Use cases:
- Ask a peer agent to review your implementation plan or code changes
- Coordinate work across multiple agents (e.g., frontend + backend, implementer + reviewer)
- Spawn a specialist agent on-demand to help with a subtask

# `ax` Cheatsheet

| Command | Syntax | Description |
|---------|--------|-------------|
| `ax whoami` | `ax whoami` | Show your label, role, session ID, and pane ID |
| `ax who` | `ax who [--role <role>]` | List all agents in the session (optionally filtered by role) |
| `ax join` | `ax join --role <role> --label <label>` | Register the current pane into the session |
| `ax spawn` | `ax spawn --role <role> --label <label> [--layout hsplit\|vsplit\|window] "<command>"` | Create a new tmux pane and launch an agent in it |
| `ax send` | `ax send <label> "message"` | Send a short message to a specific agent |
| `ax send` | `ax send --role <role> "message"` | Broadcast a short message to all agents with that role |

Note: `ax init` and `ax skill` are human-only commands. Never run them yourself.

## prereq: joining a ax session
Before you can use the `ax` CLI, you need to ensure that you are part of an ax session.
Verify this by running `ax whoami`.
- If the output says 'Error: not inside a tmux session — run `ax init` first'
  - Inform user. The user must respawn you inside a tmux session. You can not solve this on your own.
- If the output says 'Error: not registered as an agent in this session (run `ax join` first)'
  - Join the ax session automatically by running `ax join --role <role> --label <label>`
    - If you unsure about the role and label, ask the user for it.

## discovering other agents in your ax session
You can discover other agents in your ax session by running `ax who`.
- This will list all agents in your ax session, along with their label, role, pane ID, and status.
- Use `ax who --json` to get machine-readable JSON output (preferred over the table format).

## spawning a peer coding agent to work with you
When user asks you to spawn another coding agent, use `ax spawn` command.
This will create a new tmux pane, and launch the agent inside it. You may then use `ax send` to communicate with the new agent.
- `ax spawn` requires role & label. If you unsure about the role and label, ask the user for it.
- `ax spawn` also requires a command. Example:
  `ax spawn --role reviewer --label codex-1 "claude"`
- It is useful to quickly do `ax who` to check the peers before spawning a new one.
- Use the `--layout` flag to control where the pane is rendered on user's screen.
  - This flag supports basic options, but if user asks for advanced options, first do `ax spawn` to spawn,
    and then use regular `tmux` CLI commands to adjust the pane layout (resize, move, etc.) as desired.
    You can get the pane ID from the output of `ax spawn` command.

## sending ax sms to other agents
You can send a message to other agents by running `ax send <label> "message"`.
- You need to know the label of the agent from a previous `ax who` or `ax spawn` command.
- The message is sent via tmux send-keys internally, and long messages can break or jitter, so
  keep the messages short (hence "sms"), and communicate the meaty details via file references
- All files that need to be communicated should be stored in `$AX_SESSION_DIR/comms/<label>/`
  directory, where <label> is your label (`ax whoami`).
  - Project file references can be sent as-is, and need not be copied to this directory.
- You can broadcast a message to all agents with a specific role by running `ax send --role <role> "message"`.
- If `ax send` fails with a "pane is dead" error, the target agent has crashed. Inform the user rather than retrying.

Examples:
> "can you /review my code changes? Implementation plan is at $AX_SESSION_DIR/comms/claude-1/plan.md" that captures user's intent.

> "I've written a implementation plan at $AX_SESSION_DIR/comms/claude-1/plan.md for user's request. The crux of the user's request is captured at $AX_SESSION_DIR/comms/claude-1/request.txt" which you can also refer to evaluate the plan against. Can you /review my plan? Feel free to ask any clarifying questions on the user's request to me.

> "Here's my honest take on your written plan: $AX_SESSION_DIR/comms/codex-1/review.md, I think this needs serious rework."

## receiving ax sms from other agents
When another agent sends you an ax sms, you will see it in your terminal as a message in the format:
`[ax sms from <label> (<role>)]: <message>`.
- When you receive a message, first read any referenced files, then act on the request.
- You can reply to the message by running `ax send <label> "message"`.
  - Follow all the rules & conventions as described in the previous section.

If you are able to receive ax messages, it means you are already part of a ax session. Running `ax whoami` and `ax who` will allow you introspect and discover others.

Examples:
> "[ax sms from claude-1 (implementer)]: can you /review my code changes? Implementation plan is at $AX_SESSION_DIR/comms/claude-1/plan.md"

> "[ax sms from codex-1 (reviewer)]: Here's my honest take on your written plan: $AX_SESSION_DIR/comms/codex-1/review.md, I think this needs serious rework."
