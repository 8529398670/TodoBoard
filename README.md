# TodoBoard

A self-hosted, multi-user Trello-style board. Go (Fiber v3 + bolt) on the
server, plain JavaScript in the browser, no build step. Ships as a Docker
image or as one self-contained binary.

## What it does

- **Boards, lists, cards** with drag and drop (mouse and touch), labels,
  start and due dates, assignees, and a *done* flag.
- **Three views of the same board**: Kanban, Todo (each list as a checklist),
  and Timeline (Gantt-style bars you drag to move or resize; unscheduled
  cards sit in a tray you drag onto a day).
- **Checklists inside cards**, reorderable.
- **Markdown with math**: `$H_2O$` inline, and
  ```
  $$
  \ce{2H2 + O2 -> 2H2O}
  $$
  ```
  as a block (KaTeX + mhchem, so `\ce{}` and `\pu{}` work). `$5 and $10`
  stays plain text; math inside code stays code.
- **Sharing**
  - add existing accounts to a board as *editor* or *viewer*;
  - share a single card with an account (*view* or *edit*);
  - share links for a whole board or a single card, *view* or *edit*,
    optional expiry, revocable -- no account needed. Guests with an edit link
    are asked for a name, and their changes show as "Name (guest)".
- **Export**: a card as PNG (copy to clipboard or download) or Markdown (copy
  or download); a board as Markdown or as a print layout you save as PDF.
- **Pushover notifications**: due-date reminders (your lead times, your
  timezone), overdue, assigned to you, shared with you, and changes to cards
  you follow. Everyone sets defaults on the account page and can override
  them per board and per card.
- **Live updates**: open boards poll a revision stamp every few seconds, so
  other people's changes appear without a reload. Two people editing the same
  description get a conflict prompt instead of a silent overwrite.

Accounts, login links, invite links and API keys come from the
`go-webserver` template this was built on -- there is no signup form and no
passwords. See the comments in `server/security` and `server/routes/auth.go`.

## Running it

```bash
go run .                 # reads ./config.yaml; prints a one-time login link on first run
./run.sh                 # rebuild, (re)start, and open it in the browser
go test ./...
./build.sh darwin/arm64  # single-file binary in dist/
./dockerRun.sh           # hardened container
```

For plain `http://localhost`, set `server.secure_cookies: false` in
`config.yaml`.

### Push notifications

1. Create an application at <https://pushover.net/apps/build> and put its
   token in `config.yaml` under `pushover.app_token`.
2. Set `server.public_url` (e.g. `https://boards.example.com`) so
   notifications link straight to the card.
3. Each person pastes their Pushover *user key* on the Account page and
   presses "Send a test".

## Where things are

| | |
|---|---|
| `server/models/board.go`, `card.go` | boards, lists, labels, cards, checklists |
| `server/models/share_link.go` | share links (sha256-hashed `<id>.<secret>` credentials) |
| `server/models/prefs.go` | notification preferences and per-board/per-card subscriptions |
| `server/access/` | the one place that decides who may view, edit or manage what |
| `server/notify/` | Pushover client, event worker, due-date scheduler |
| `server/routes/boards.go`, `cards.go`, `sharing.go`, `notify_prefs.go` | the board API, mounted for members at `/api/boards` and for share-link guests at `/api/shared` |
| `static/js/board.js` | board page controller (members and guests) |
| `static/js/view_*.js` | Kanban, Todo, Timeline |
| `static/js/card_modal.js` | card editor |
| `static/js/markdown.js` | Markdown + KaTeX pipeline |
| `static/js/card_export.js`, `print.js` | exports |
| `language.yaml` | every string the UI or a notification shows |

## Known limits

- Markdown images must come from this server or be `data:` URLs; the
  Content-Security-Policy blocks other origins.
- Copying a card image to the clipboard needs https (or localhost); Download
  PNG works everywhere.
- bolt is one process, one file. Fine for a team; not for many servers.
