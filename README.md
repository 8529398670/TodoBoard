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
- **New boards start as three columns** (To do / Doing / Done), and each card
  there has *Doing* and *Done* tick boxes that move it between them.
- **Staged lists** (a layout in Board settings): any number of lists, each
  with its own To do / Doing / Done. A column shows one stage at a time,
  starting on To do; swipe it or use its tabs to flip it, or use the bar in
  the navbar or the Left and Right arrow keys to flip every column. The bar's
  *All* tab merges every list into one centred list, which flips through
  To do, Doing and Done the same way, sorted by a board-wide rank: set a
  card's rank in its details or drag it in All, and the others shift to make
  room.
- **Calendar boards** (a layout in Board settings): every card is filed
  under a day, and each day has its own To do / Doing / Done. Look at a
  month of day cells, a week of day columns, or one day; drag cards between
  days, or onto one from the tray of undated cards. Cards still in To do or
  Doing on a past day carry over to today when the board is opened (a board
  setting, on by default). Switching a board of date-titled columns
  (`06NOV2026`, `2026-11-06`) to Calendar files their cards under those days.
- **Card numbers**: each column numbered 1..n; type a new number and press
  Enter to move the card there. On or off per board, or left to each
  person's account default.
- **Label colors**: the palette, or any color of your own. A custom color
  you give a label is saved to your account and offered on every board; name
  or remove them on the Account page.
- **Card links**: every card has its own address,
  `/b/<board>/<column>/<ULID>`, from *Copy link* in its right-click menu or
  details. The ULID finds the card, so a link still opens it after the card
  moves column; the address bar then shows its current column. Older
  `?card=<id>` links still work.
- **Copy a column** as a Markdown bullet or numbered list of its card titles.
- **Notes** on every board, from the journal icon in the navbar or Board
  settings: markdown for links, documents and an index.
- **Purpose**, a section each board can turn on in Board settings (off by
  default), which adds its own navbar icon. It says what the board is for and
  how to work on it, and agents are told to read it first. Turned off, it is
  hidden from everyone, agents included, and the text is kept.
- Members edit both, whole-board share links read them, and links showing only
  part of a board do not get them.
- **Undo / redo** every board change from the tab (Ctrl+Z / Ctrl+Y, ⌘Z /
  ⇧⌘Z, or the arrows in the header) -- deleted cards, lists and labels
  included.
- **Home page folders and search**: your own folders, nested up to three levels (other members of a
  shared board file it however they like), drag boards between them, and a
  search box over board and folder names.
- **Checklists inside cards**, reorderable.
- **Markdown with math**: `$H_2O$` inline, and
  ```
  $$
  \ce{2H2 + O2 -> 2H2O}
  $$
  ```
  as a block (KaTeX + mhchem, so `\ce{}` and `\pu{}` work). `$5 and $10`
  stays plain text; math inside code stays code.
- **Comments** on every card, in Markdown: board editors and the owner
  comment, and so does anyone holding a *comment* link or a card shared with
  them to comment on. Authors edit and remove their own; the owner removes
  any. A board can let link holders comment without giving a name (off by
  default; they show as *Anonymous*). People following a card hear about new
  comments with its other changes.
- **Sharing**
  - add existing accounts to a board as *editor* or *viewer*;
  - share a single card with an account (*view*, *comment* or *edit*);
  - share links for a whole board, a single column, or a single card, *view*,
    *comment* or *edit*,
    optional expiry, revocable -- no account needed. Guests with an edit link
    are asked for a name, and their changes show as "Name (guest)";
  - a public link to one card, from its right-click menu, for the owner and
    editors once the owner allows it in the board's Sharing panel. Read-only,
    and it shows that card and nothing of the board. Turning the setting off
    pauses every public link until it is turned back on.
- **Export**: a card as PNG (copy to clipboard or download) or Markdown (copy
  or download); a board as Markdown, as a spreadsheet (formatted .xlsx or
  .csv, one row per card), or as a print layout you save as PDF.
- **Notifications**, in your browser or through Pushover: due-date reminders
  (your lead times, your timezone), overdue, assigned to you, shared with you,
  and changes to cards you follow. Everyone sets defaults on the account page
  and can override them per board and per card. A board can also send all of
  its activity, gathered into one notification at a time.
- **Live updates**: open boards poll a revision stamp every few seconds, so
  other people's changes appear without a reload. Two people editing the same
  description get a conflict prompt instead of a silent overwrite.

Accounts, login links, invite links and API keys come from the
`go-webserver` template this was built on -- there is no signup form and no
passwords. See the comments in `server/security` and `server/routes/auth.go`.

## Running it

```bash
scripts/run.sh               # rebuild, (re)start, print where everything is, open the browser
scripts/login.sh             # a fresh login link for the owner, from whatever is running
scripts/run.sh manage list-users   # the admin CLI, against the running server
go test ./...
scripts/build.sh darwin/arm64      # single-file binary in dist/
docker/run.sh                # hardened container; scripts/login.sh --docker for its link
```

On Windows, `scripts\run.bat` and `scripts\login.bat` do the same.

Everything is configured in `config/config.yaml` (gitignored -- it holds the
secret key; the first `scripts/run.sh` writes it). For plain
`http://localhost`, set `server.secure_cookies: false` there.

### Push notifications

Browser notifications are on by default (`notify.web_push`). They need the site
on HTTPS (or `localhost`), and each person turns them on under Notifications.
Set `notify.web_push_contact` to a `mailto:` or `https://` address so a push
service can reach you.

On an iPhone or iPad they need iOS 16.4 or later and work only from the Home
Screen, and that app does not share Safari's sign-in:

1. In Safari, create a login link on the Account page and copy it.
2. Tap Share, then Add to Home Screen, and open the new icon.
3. Paste the link into the sign-in box there.
4. Under Notifications, turn them on for this device.

If that app's session ends, notifications keep arriving, and the same paste
signs you back in.

With both set up, each person chooses under Notifications where theirs go:
Pushover, their devices, or both. A saved Pushover key is kept whichever they
choose.

For Pushover as well:

1. Create an application at <https://pushover.net/apps/build> and put its
   token in `config/config.yaml` under `pushover.app_token`.
2. Set `server.public_url` (e.g. `https://boards.example.com`) so
   notifications link straight to the card (printed login links use it too).
3. Each person pastes their Pushover *user key* on the Account page and
   presses "Send a test".

### Claude Code and other agents (MCP)

`POST /mcp` speaks the Model Context Protocol with an API key, so an agent can
read and edit your boards as you. Make a key on the Account page, then:

```bash
claude mcp add --transport http todoboard http://localhost:8080/mcp --header "Authorization: Bearer <key>"
```

Tools: `whoami`, `list_team`, `list_boards`, `get_board`, `get_board_notes`,
`update_board_notes`, `get_card`,
`find_cards`, `create_board`, `create_list`, `update_list`, `create_label`,
`create_card`, `update_card`, `add_checklist_item`, `update_checklist_item`,
plus the `admin_*` account tools for an admin-scoped key. Every board tool
goes through `server/access`, so a key sees and changes what its account can
on the web -- app admin rights grant nothing on boards.

A user-scoped key can be limited on the Account page, when it is made or
afterwards: to only the boards you pick, or to all your boards except those.
Anywhere outside the limit a board answers `board_not_found`, as if it did not
exist, and it is left out of `list_boards`, `find_cards` and `GET /api/backup`.
A limited key cannot use home-page folders or change notification settings; an
"only these" key also cannot create or import boards, so new boards stay out of
its reach until you add them. Admin-scoped keys cannot be limited. Do not
downgrade to a build without this feature while limited keys exist: an older
binary drops the limit the next time it rewrites a key's row.

Left out on purpose: deleting anything (archive a card with `update_card`
instead), share links, board membership, backup import, and anything that
mints a credential. Only keys are accepted -- a browser session gets 403.
`server.mcp: false` (or `auth.api_keys: false`) removes the endpoint.

## Where things are

| | |
|---|---|
| `server/models/board.go`, `card.go` | boards, lists, labels, cards, checklists |
| `server/models/card_identity.go` | each card's per-board number and ULID |
| `server/models/share_link.go` | share links (sha256-hashed `<id>.<secret>` credentials) |
| `server/models/folder.go` | each person's home page folders |
| `server/models/prefs.go` | notification preferences and per-board/per-card subscriptions |
| `server/access/` | the one place that decides who may view, edit or manage what |
| `server/notify/` | Pushover client, event worker, due-date scheduler |
| `server/routes/boards.go`, `cards.go`, `sharing.go`, `notify_prefs.go` | the board API, mounted for members at `/api/boards` and for share-link guests at `/api/shared` |
| `static/js/board.js` | board page controller (members and guests) |
| `static/js/view_*.js` | Kanban, Todo, Timeline |
| `static/js/stages.js` | a staged board's stage controls: bar, tabs, swipe, arrow keys |
| `static/js/card_modal.js` | card editor |
| `static/js/trash.js`, `archived.js` | the board's trash and archive panels |
| `static/js/board_notes.js` | the board's Notes and Purpose panels |
| `static/js/history.js` | undo / redo |
| `static/js/markdown.js` | Markdown + KaTeX pipeline |
| `static/js/card_export.js`, `print.js` | exports |
| `server/sheet/`, `server/routes/export.go` | the .xlsx / .csv board export |
| `server/mcp/`, `server/routes/mcp_tools*.go` | the `/mcp` protocol and its tools |
| `config/openapi.json`, `server/routes/openapi.go` | the API as OpenAPI 3.1, by hand, served at `GET /api/openapi.json` to anyone signed in; `go test ./server/routes` fails until a new route is in it |
| `config/language.yaml` | every string the UI or a notification shows |
| `config/defaults.yaml` | every setting, documented, with its default |

## Known limits

- Markdown images must come from this server or be `data:` URLs; the
  Content-Security-Policy blocks other origins.
- Copying a card image to the clipboard needs https (or localhost); Download
  PNG works everywhere.
- bolt is one process, one file. Fine for a team; not for many servers.
