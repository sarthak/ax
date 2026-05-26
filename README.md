# ax — agent exchange

A lightweight CLI that lets coding agents discover each other and exchange short messages (like sms), so you can run a team of agents side-by-side instead of one at a time.

## Motivation

Running a single coding agent is easy. Running two or more that need to coordinate is not.

Today the common workaround is to nest one agent inside another — for example, asking Claude to invoke the Codex CLI to review its own work. This *technically* gets the agents to talk, but you, the human, are pushed out of the loop. You can't watch Codex's reasoning stream, you can't interrupt it mid-review, and you can't ask follow-up questions about why it landed on a particular take. You only see whatever the outer agent chooses to surface back.

`ax` proposes a different shape:

- Each agent runs in its own tmux pane, so you can watch all of them at once.
- `ax` is the thin layer they use to find each other and pass short messages (like sms).
- You stay in the loop — you can read both transcripts, jump into either pane to ask "why did you say that?", and steer either side at any moment.

A concrete example: Claude writes an implementation in one pane while Codex reviews it in another. Claude calls `ax send codex-1 "review my plan at $AX_SESSION_DIR/comms/claude-1/plan.md"`; Codex reads it, replies, and you watch the whole exchange happen live. See [Use cases](#use-cases) for more.

## Highlights

- **Agent-agnostic.** Anything that runs in a terminal works — Claude Code, Codex, Gemini CLI, Cursor's agent, your own shell scripts. No SDK integration, no plugin, no wrapper.
- **Built on tmux.** No new window manager, no custom UI. If you already know how to split a pane, you already know how to lay out your agents.
- **`ax skill` bootstraps agents in one command.** Pipe the skill file into your agent's skill directory and the agent immediately knows how to discover peers, spawn new ones, and exchange messages.

## Install

```sh
go install github.com/sarthak/ax@latest
```

Verify:

```sh
ax --help
```

## Quickstart

```sh
# 1. Start an ax-enabled tmux session (creates one if you aren't in tmux yet).
ax init

# 2. Install the skill so your agent learns how to use ax.
ax skill > ~/.claude/skills/ax.md

# 3. Register this pane as your primary agent BEFORE launching it.
ax join --role implementer --label claude-1
#
# (Agents can self-join too — the skill tells them how — but doing it yourself
# means you pick the role and label up front. It's just more controllable.)

# 4. Start your primary agent in this pane.
claude

# 5. Add a peer. Two ways:
#
#    a) Preferred — do it by hand, for the same reason as step 3:
#         tmux split-window -h
#         # in the new pane:
#         ax join --role reviewer --label codex-1
#         codex
#
#    b) Or ask your primary agent to spawn one for you:
#         "spawn a codex agent as a reviewer using ax spawn"
#       which runs something like:
#         ax spawn --role reviewer --label codex-1 "codex"

# 6. From there on, agents message each other directly:
#    ax send codex-1 "review my plan at $AX_SESSION_DIR/comms/claude-1/plan.md"
```

You watch both panes side-by-side. You can interrupt either agent at any point.

## How it works

Three pieces, that's it:

- **The tmux session is the room.** Every agent lives in a pane inside one shared session.
- **A registry file is the phonebook.** `ax join` adds an agent (label, role, pane ID); `ax who` reads it back.
- **`tmux send-keys` is the transport.** `ax send` types the message into the target agent's pane, prefixed so the receiver can recognize it as an inbound message.

There's no daemon, no socket, no network. The session directory under `$AX_SESSION_DIR` holds the registry and a `comms/<label>/` directory per agent for any files they want to share.

## Using ax with coding agents

`ax` ships with a skill file (a short markdown instruction set) that teaches an agent how to use the CLI. Install it into your agent's skill directory:

```sh
# Claude Code
ax skill > ~/.claude/skills/ax.md

# Or pipe it wherever your agent reads skills/instructions from.
```

Once installed, the agent will, on its own:

- Run `ax whoami` / `ax who` to figure out who's in the session.
- Run `ax join` if it isn't registered yet.
- Use `ax spawn` to create peers when you ask for one.
- Use `ax send` to message peers and recognize incoming `[ax sms from ...]` messages as requests it should act on.

You can preview the skill content with `ax skill` (it just prints to stdout).

## Commands

| Command                                                                                | Description                                                                                        |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `ax init`                                                                              | Initialize a new ax session (creates a tmux session if you aren't in one).                         |
| `ax skill`                                                                             | Print the skill file to stdout. Pipe into your agent's skill dir.                                  |
| `ax whoami`                                                                            | Show your own label, role, session ID, and pane ID.                                                |
| `ax who [--role <role>]`                                                               | List agents in the session, optionally filtered by role. Use `--json` for machine-readable output. |
| `ax join --role <role> --label <label>`                                                | Register the current pane as an agent.                                                             |
| `ax spawn --role <role> --label <label> [--layout hsplit\|vsplit\|window] "<command>"` | Create a new pane and launch an agent in it.                                                       |
| `ax send <label> "<message>"`                                                          | Send a short message to a specific agent.                                                          |
| `ax send --role <role> "<message>"`                                                    | Broadcast a message to every agent with that role.                                                 |

Run `ax <command> --help` for full flag details.

## Use cases

- **Implementer ↔ reviewer.** Claude writes the code, Codex reviews it. Each side runs in its own pane; you read both transcripts and steer either at will.
- **Frontend + backend pair.** One agent on the API, one on the UI. They coordinate contract changes via `ax send` instead of you copy-pasting between sessions.
- **Spawn-a-specialist.** Your main agent realizes it needs help with, say, a tricky SQL migration. It spawns a peer focused on that subtask, hands it the context, and folds the result back in.
- **Broadcast to a role.** `ax send --role reviewer "new PR up at ..."` pings every reviewer agent at once.

## License

[MIT](LICENSE)
