# TodoBoard API

Every route this server answers, apart from the pages at /login/*, /join/* and /docs/*. Generated from the code by scripts/api_docs.sh: an app's own routes get their docs the same way, and a test fails until they do.

This file is generated: do not edit it. Change the docs next to the handlers and run `scripts/api_docs.sh`.

## Contents

- [Calling the API](#calling-the-api)
- [Errors](#errors)
- [Account](#account)
- [Sign-in](#sign-in)
- [API keys](#api-keys)
- [Admin](#admin)
- [Invite links](#invite-links)
- [Owner](#owner)
- [Monitoring](#monitoring)
- [Backup](#backup)
- [Boards](#boards)
- [Cards](#cards)
- [Comments](#comments)
- [Folders](#folders)
- [Google Calendar](#google-calendar)
- [Notifications](#notifications)
- [Shared](#shared)
- [Sharing](#sharing)
- [Trash](#trash)
- [Schemas](#schemas)

## Calling the API

| Way in | How |
|---|---|
| Browser session | A browser session. Its POSTs carry csrf_token in the body. The cookie's name is server.cookie_name, with the __Host- prefix while server.secure_cookies is on. |
| API key | An API key, as Authorization: Bearer <id>.<secret>. |

## Errors

An error reply is a JSON object whose `error` says what went wrong.

| Status | Meaning | Body |
|---|---|---|
| 400 | The request was refused as it stands. | [`Error`](#error) |
| 401 | Not signed in. | [`Error`](#error) |
| 403 | Not allowed, or a bad csrf_token. | [`Error`](#error) |
| 404 | No such thing, or this feature is switched off. | [`Error`](#error) |
| 409 | The record has moved on: an undo too late, or a stale edit. | [`Error`](#error) |
| 429 | Too many requests. | [`Error`](#error) |
| 503 | In maintenance mode. Retry-After says when to try again. | [`Maintenance`](#maintenance) |

## Account

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/account/display`](#post-apiaccountdisplay) | Save your display settings | browser session or API key, anyone signed in |
| POST | [`/api/account/label-colors`](#post-apiaccountlabel-colors) | Save your label colours | browser session or API key, anyone signed in |
| POST | [`/api/account/login-link`](#post-apiaccountlogin-link) | A login link for yourself | browser session or API key, anyone signed in |
| POST | [`/api/account/login-link/cancel`](#post-apiaccountlogin-linkcancel) | Stop your own login link working | browser session or API key, anyone signed in |
| POST | [`/api/account/password`](#post-apiaccountpassword) | Set or clear your password | browser session only, anyone signed in |
| POST | [`/api/account/pin`](#post-apiaccountpin) | Set or clear your PIN | browser session only, anyone signed in |
| POST | [`/api/account/rename`](#post-apiaccountrename) | Rename your account | browser session or API key, anyone signed in |
| GET | [`/api/account/sessions`](#get-apiaccountsessions) | Your signed-in devices | browser session or API key, anyone signed in |
| POST | [`/api/account/sessions/revoke-others`](#post-apiaccountsessionsrevoke-others) | Sign out every other device | browser session or API key, anyone signed in |
| POST | [`/api/account/sessions/{session_id}/label`](#post-apiaccountsessionssession_idlabel) | Name one of your devices | browser session or API key, anyone signed in |
| POST | [`/api/account/sessions/{session_id}/revoke`](#post-apiaccountsessionssession_idrevoke) | Sign out one of your devices | browser session or API key, anyone signed in |
| POST | [`/api/account/theme`](#post-apiaccounttheme) | Pick your theme | browser session or API key, anyone signed in |
| POST | [`/api/account/themes`](#post-apiaccountthemes) | Replace your own themes | browser session or API key, anyone signed in |
| POST | [`/api/account/two-factor/confirm`](#post-apiaccounttwo-factorconfirm) | Turn two-factor on | browser session only, anyone signed in |
| POST | [`/api/account/two-factor/disable`](#post-apiaccounttwo-factordisable) | Turn two-factor off | browser session only, anyone signed in |
| POST | [`/api/account/two-factor/start`](#post-apiaccounttwo-factorstart) | Begin two-factor setup | browser session only, anyone signed in |
| POST | [`/api/bulk/account/sessions/revoke`](#post-apibulkaccountsessionsrevoke) | Sign out one of your devices, for many at once | browser session or API key, anyone signed in |
| GET | [`/api/language`](#get-apilanguage) | The UI's text | anyone |
| POST | [`/api/logout`](#post-apilogout) | Sign out | browser session or API key, anyone signed in |
| POST | [`/api/logs/events`](#post-apilogsevents) | Record what happened on the page | browser session or API key, anyone signed in |
| GET | [`/api/me`](#get-apime) | Who am I | browser session or API key, anyone signed in |
| GET | [`/api/team`](#get-apiteam) | List team | browser session or API key, anyone signed in |

### POST /api/account/display

Save your display settings

How times and boards are shown to you. timezone (empty for the browser's own) and hour12 (null for the locale's) are always set from what is sent; every other field changes only when sent: number_cards, plain_icons (grey status icons), done_on_top, add_on_top, theme (clear_theme goes back to plain light), own_theme_on_boards (keep your theme on boards that have their own), swipe_columns (phones) and folder_sort. Answers your display settings.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_display`](mcp.md#set_display) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `add_on_top` | boolean or null | no |  |  |
| `clear_theme` | boolean | no |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `done_on_top` | boolean or null | no |  |  |
| `folder_sort` | string or null | no |  |  |
| `hour12` | boolean or null | no |  |  |
| `number_cards` | boolean or null | no |  |  |
| `own_theme_on_boards` | boolean or null | no |  |  |
| `plain_icons` | boolean or null | no |  |  |
| `swipe_columns` | boolean or null | no |  |  |
| `theme` | object or null | no |  |  |
| `theme.accent` | string | yes |  |  |
| `theme.background` | string | no |  |  |
| `theme.button` | object or null | no |  |  |
| `theme.button.case` | string | no | one of `normal`, `upper` |  |
| `theme.button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `theme.button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `theme.button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `theme.card` | string | no |  |  |
| `theme.column` | string | no |  |  |
| `theme.dark` | object or null | no |  |  |
| `theme.dark.accent` | string | yes |  | The main color: what you can press. |
| `theme.dark.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.light` | object or null | no |  |  |
| `theme.light.accent` | string | yes |  | The main color: what you can press. |
| `theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.mode` | string | yes |  |  |
| `theme.overrides` | map of string | no |  |  |
| `theme.second` | string | no |  |  |
| `theme.surface` | string | no |  |  |
| `timezone` | string | no |  |  |

**Reply 200**: Success

An empty object.

### POST /api/account/label-colors

Save your label colours

Replace your own saved label colours with this list, in its order. Labels already using a colour you leave out keep it. Answers your display settings.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_label_colors`](mcp.md#set_label_colors) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `colors` | array of object | yes |  | Your whole list of saved label colours, in order; it replaces the one stored. |
| `colors[].color` | string | yes |  |  |
| `colors[].name` | string | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### POST /api/account/login-link

A login link for yourself

A link that signs your own account in on a browser or another device. An API key may ask only if your account let it, from a browser session (POST /api/keys/{key_id}/sign-in); its link works once, for at most 15 minutes.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_my_login_link`](mcp.md#create_my_login_link) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_at` | string (date-time) or null | yes |  | null when it never expires. |
| `expires_in_seconds` | integer | yes |  | 0 when it never expires. |
| `login_path` | string | yes |  | Path of the login link. A credential: treat it like a password. |
| `max_uses` | integer | yes |  | 0 when unlimited. |
| `unlimited` | boolean | yes |  |  |

### POST /api/account/login-link/cancel

Stop your own login link working

Void your account's live login link without making a new one. Devices it already signed in stay signed in. Answers with removed: 0 or 1.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`cancel_my_login_link`](mcp.md#cancel_my_login_link) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `removed` | integer | yes |  | Links removed: 0 or 1. |

### POST /api/account/password

Set or clear your password

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `code` | string | no |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `password_digest` | string | yes |  |  |
| `username` | string | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `has_password` | boolean | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/account/pin

Set or clear your PIN

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `pin` | string | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `has_pin` | boolean | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/account/rename

Rename your account

Change your own display name (1-80 characters). expect, when given, is the name you think it has now; if someone changed it meanwhile the rename is refused with the current name, so nothing is overwritten unseen.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`rename_account`](mcp.md#rename_account) |
| Errors | 400, 401, 403, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `display_name` | string | yes | 1 to 80 characters |  |
| `expect` | string or null | no |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### GET /api/account/sessions

Your signed-in devices

Where your account is signed in, a page at a time: id, when it started and was last seen, the browser, and the name you gave it. Through an API key no device is current. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_my_sessions`](mcp.md#list_my_sessions) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `sessions` | array of [`Session`](#session) | yes |  |  |
| `total` | integer | yes |  |  |

### POST /api/account/sessions/revoke-others

Sign out every other device

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`revoke_my_other_sessions`](mcp.md#revoke_my_other_sessions) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `removed` | integer | yes |  |  |

### POST /api/account/sessions/{session_id}/label

Name one of your devices

Give one of your devices a name (at most 80 characters; empty clears it). expect, when given, is the name you think it has now.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`label_my_session`](mcp.md#label_my_session) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `session_id` | path | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expect` | string or null | no |  | The name you think it has. Answers 409 if it has another. |
| `label` | string | yes | at most 80 characters | At most 80 characters. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `label` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/account/sessions/{session_id}/revoke

Sign out one of your devices

End one of your signed-in sessions; that browser has to sign in again.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`revoke_my_session`](mcp.md#revoke_my_session) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `session_id` | path | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/account/theme

Pick your theme

Switch to another theme from your list (or the defaults), by its id.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_theme`](mcp.md#set_theme) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `theme_id` | string | yes |  | A theme's id. Empty for the first in the list. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `theme_id` | string | yes |  |  |

### POST /api/account/themes

Replace your own themes

Replace your whole list of color themes (1 to 40) and pick the one you use. themes: null goes back to following the app's defaults. A theme is the Theme schema in the API reference (config/api.json): id, name, mode light|dark, accent, second, and optional colors as #rrggbb.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_themes`](mcp.md#set_themes) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `builtins` | integer | no |  | Which built-in themes this list has seen: the highest since in Theme.builtIn (static/js/theme.js) when it was saved. Built-in themes added after that join the list the browser shows. Left out: 0, the first ones. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `theme_id` | string | no |  | The theme you use. |
| `themes` | array of [`Theme`](#theme) or null | no |  | Your whole list, 1 to 40 themes. null goes back to following the app's defaults. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `theme_id` | string | yes |  |  |
| `themes` | array of [`Theme`](#theme) or null | yes |  | What you now see; null while the built-in themes apply. |
| `themes_builtins` | integer | yes |  | Which built-in themes that list has seen (see builtins on POST /api/account/themes). |
| `themes_custom` | boolean | yes |  |  |

### POST /api/account/two-factor/confirm

Turn two-factor on

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `code` | string | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `recovery_codes` | array of string | yes |  |  |

### POST /api/account/two-factor/disable

Turn two-factor off

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `code` | string | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/account/two-factor/start

Begin two-factor setup

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `secret` | string | yes |  |  |

### POST /api/bulk/account/sessions/revoke

Sign out one of your devices, for many at once

Runs POST /api/account/sessions/{session_id}/revoke once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`revoke_my_session_bulk`](mcp.md#revoke_my_session_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].session_id` | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### GET /api/language

The UI's text

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 429, 503 |

**Reply 200**: Success

An empty object.

### POST /api/logout

Sign out

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: Ends a browser session; a key has none to end. |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/logs/events

Record what happened on the page

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The browser's own activity reports (track.js), not an action. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `events` | array of object | yes |  |  |
| `tab` | string | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `accepted` | integer | yes |  |  |

### GET /api/me

Who am I

The account this API key belongs to, the role it acts with (role -- never higher than the key's own scope), and the account's own role.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`whoami`](mcp.md#whoami) |
| Errors | 401, 429, 503 |

**Reply 200**: Success

[`Me`](#me).

### GET /api/team

List team

People with an active account, a page at a time: id and display name.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_team`](mcp.md#list_team) |
| Errors | 400, 401, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `members` | array of [`Member`](#member) | yes |  |  |
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `total` | integer | yes |  |  |

## Sign-in

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/login`](#post-apilogin) | Redeem a login link | anyone |
| GET | [`/api/login/options`](#get-apiloginoptions) | What sign-in offers | anyone |
| POST | [`/api/login/password`](#post-apiloginpassword) | Sign in with a password | anyone |
| POST | [`/api/login/two-factor`](#post-apilogintwo-factor) | Finish a sign-in with a code | anyone |

### POST /api/login

Redeem a login link

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `credential` | string | yes |  |  |

**Reply 200**: Success

[`SignIn`](#signin).

### GET /api/login/options

What sign-in offers

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `passwords` | boolean | yes |  |  |
| `salt` | string | no |  |  |
| `two_factor` | boolean | yes |  |  |

### POST /api/login/password

Sign in with a password

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `password_digest` | string | yes |  |  |
| `username` | string | yes |  |  |

**Reply 200**: Success

[`SignIn`](#signin).

### POST /api/login/two-factor

Finish a sign-in with a code

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `challenge` | string | yes |  |  |
| `code` | string | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

## API keys

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/admin/keys`](#get-apiadminkeys) | Every key in the app | browser session only, admins |
| POST | [`/api/admin/keys`](#post-apiadminkeys) | Create a key that acts as someone else | browser session only, admins |
| POST | [`/api/admin/keys/{key_id}/delete`](#post-apiadminkeyskey_iddelete) | Delete anyone's key | browser session only, admins |
| POST | [`/api/admin/keys/{key_id}/expiry`](#post-apiadminkeyskey_idexpiry) | Change when anyone's key expires | browser session only, admins |
| POST | [`/api/admin/keys/{key_id}/rename`](#post-apiadminkeyskey_idrename) | Rename anyone's key | browser session only, admins |
| POST | [`/api/admin/keys/{key_id}/restore`](#post-apiadminkeyskey_idrestore) | Undo deleting anyone's key | browser session only, admins |
| POST | [`/api/admin/keys/{key_id}/revoke`](#post-apiadminkeyskey_idrevoke) | Revoke anyone's key | browser session or API key, admins |
| GET | [`/api/admin/users/{user_id}/keys`](#get-apiadminusersuser_idkeys) | An account's keys | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/keys`](#post-apiadminusersuser_idkeys) | Make a key for an account | browser session or API key, admins |
| GET | [`/api/keys`](#get-apikeys) | Your API keys | browser session only, anyone signed in |
| POST | [`/api/keys`](#post-apikeys) | Create an API key | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/boards`](#post-apikeyskey_idboards) | Limit one of your keys to some boards | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/delete`](#post-apikeyskey_iddelete) | Delete one of your keys | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/expiry`](#post-apikeyskey_idexpiry) | Change when one of your keys expires | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/mint-keys`](#post-apikeyskey_idmint-keys) | Let one of your keys make keys | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/rename`](#post-apikeyskey_idrename) | Rename one of your keys | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/restore`](#post-apikeyskey_idrestore) | Undo deleting one of your keys | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/revoke`](#post-apikeyskey_idrevoke) | Revoke one of your keys | browser session only, anyone signed in |
| POST | [`/api/keys/{key_id}/sign-in`](#post-apikeyskey_idsign-in) | Let one of your keys make sign-in links | browser session only, anyone signed in |

### GET /api/admin/keys

Every key in the app

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | query | string | no |  | Keep keys whose name or owner contains this, in any case. |
| `live` | query | string | no | one of `true` | Leave out revoked and expired keys. |
| `sort` | query | string | no | one of `display_name`, `name`, `role`, `last_used_at`, `expires_at`, `created_at` | Field to sort the whole list by. Rows without it go last. |
| `dir` | query | string | no | one of `asc`, `desc` | Default asc. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `keys` | array of [`APIKey`](#apikey) | yes |  |  |
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `total` | integer | yes |  |  |

### POST /api/admin/keys

Create a key that acts as someone else

Any admin may do this for an ordinary user, and only the owner for another admin. Nobody may do it for the owner. The key records who made it and shows that to the person it acts as.

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board_access` | string | no | one of ``, `only`, `except` | Empty: every board. only/except: just, or all but, board_ids. Not for admin keys. |
| `board_ids` | array of integer | no |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_days` | integer or null | no | 0 to 3650 | Absent or null: the configured default. 0: never expires. |
| `name` | string | yes | 1 to 80 characters |  |
| `role` | string | no | one of `admin`, `user`, `viewer`; default `user` |  |
| `user_id` | integer | yes | at least 1 |  |

**Reply 201**: Success

[`NewAPIKey`](#newapikey).

### POST /api/admin/keys/{key_id}/delete

Delete anyone's key

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/admin/keys/{key_id}/expiry

Change when anyone's key expires

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_days` | integer | yes | 0 to 3650 | Counted from now. 0 means it never expires. |

**Reply 200**: Success

[`APIKey`](#apikey).

### POST /api/admin/keys/{key_id}/rename

Rename anyone's key

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expect` | string or null | no |  | The name you think it has. Answers 409 if it has another. |
| `name` | string | yes | 1 to 80 characters | 1-80 characters. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `name` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/admin/keys/{key_id}/restore

Undo deleting anyone's key

|  |  |
|---|---|
| Who may call | browser session only, admins |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

[`APIKey`](#apikey).

### POST /api/admin/keys/{key_id}/revoke

Revoke anyone's key

Stops the key at once, and every key it made, down the chain. Needs a browser session or a key with may_mint_keys (any other key gets 403 key_minting_off), which may not revoke the owner's keys.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | None, HTTP only: Manages API keys, which no MCP tool does. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/admin/users/{user_id}/keys

An account's keys

One account's keys, revoked and expired ones included, newest first. Each row says which key minted it (minted_by_key, empty when a person made it). Needs a browser session or a key with may_mint_keys; any other key gets 403 key_minting_off.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | None, HTTP only: Manages API keys, which no MCP tool does. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `keys` | array of [`APIKey`](#apikey) | yes |  |  |

### POST /api/admin/users/{user_id}/keys

Make a key for an account

Needs a browser session or a key the owner allowed to make keys (may_mint_keys); any other key gets 403 key_minting_off. The role is capped at the account's own; expires_in_days is required (1-365). Never for the owner. The new key records the key that made it, can never make keys or sign-in links itself, and is revoked when that key is revoked or deleted. Answers with the key and its credential, shown only now.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | None, HTTP only: Makes an API key, which no MCP tool does. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_days` | integer | yes | 1 to 365 | Required: 1-365. A minted key always expires. |
| `name` | string | yes | 1 to 80 characters |  |
| `role` | string | yes | one of `admin`, `user`, `viewer` | Capped at the account's own role. |

**Reply 201**: Success

[`NewAPIKey`](#newapikey).

### GET /api/keys

Your API keys

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | query | string | no |  | Keep keys whose name contains this, in any case. |
| `live` | query | string | no | one of `true` | Leave out revoked and expired keys. |
| `sort` | query | string | no | one of `name`, `role`, `last_used_at`, `expires_at`, `created_at` | Field to sort the whole list by. Rows without it go last. |
| `dir` | query | string | no | one of `asc`, `desc` | Default asc. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `keys` | array of [`APIKey`](#apikey) | yes |  |  |
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `total` | integer | yes |  |  |

### POST /api/keys

Create an API key

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board_access` | string | no | one of ``, `only`, `except` | Empty: every board. only/except: just, or all but, board_ids. Not for admin keys. |
| `board_ids` | array of integer | no |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_days` | integer or null | no | 0 to 3650 | Absent or null: the configured default. 0: never expires. |
| `name` | string | yes | 1 to 80 characters |  |
| `role` | string | no | one of `admin`, `user`, `viewer`; default `user` |  |

**Reply 201**: Success

[`NewAPIKey`](#newapikey).

### POST /api/keys/{key_id}/boards

Limit one of your keys to some boards

board_access empty reaches every board; only or except, with board_ids, just those or all but those. Admin keys cannot be limited.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board_access` | string | yes | one of ``, `only`, `except` |  |
| `board_ids` | array of integer | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/keys/{key_id}/delete

Delete one of your keys

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/keys/{key_id}/expiry

Change when one of your keys expires

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_days` | integer | yes | 0 to 3650 | Counted from now. 0 means it never expires. |

**Reply 200**: Success

[`APIKey`](#apikey).

### POST /api/keys/{key_id}/mint-keys

Let one of your keys make keys

Owner only, on one of your own admin-role keys. When on, the key may make keys for other accounts (never the owner's) through POST /api/admin/users/{user_id}/keys, list them, and revoke them. Each key it makes records it, and revoking or deleting this key revokes them all.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `may_mint_keys` | boolean | yes |  | true lets this key make, list and revoke other accounts' keys, each recording this key and revoked with it. |

**Reply 200**: Success

[`APIKey`](#apikey).

### POST /api/keys/{key_id}/rename

Rename one of your keys

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expect` | string or null | no |  | The name you think it has. Answers 409 if it has another. |
| `name` | string | yes | 1 to 80 characters | 1-80 characters. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `name` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/keys/{key_id}/restore

Undo deleting one of your keys

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

[`APIKey`](#apikey).

### POST /api/keys/{key_id}/revoke

Revoke one of your keys

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/keys/{key_id}/sign-in

Let one of your keys make sign-in links

Off by default. When on, the key may ask POST /api/account/login-link for a link that signs your account in on a browser, which then can do everything you can, including what keys may not. The link works once, for at most 15 minutes.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `key_id` | path | string | yes |  | The key's id, from GET /api/keys (GET /api/admin/keys for anyone's). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `may_sign_in` | boolean | yes |  | true lets the key make a login link for your account: one that works once, for at most 15 minutes. |

**Reply 200**: Success

[`APIKey`](#apikey).

## Admin

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/admin/theme-defaults`](#get-apiadmintheme-defaults) | Default themes | browser session or API key, admins |
| POST | [`/api/admin/theme-defaults`](#post-apiadmintheme-defaults) | Replace the default themes | browser session or API key, admins |
| GET | [`/api/admin/users`](#get-apiadminusers) | List users | browser session or API key, admins |
| POST | [`/api/admin/users`](#post-apiadminusers) | Create an account | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/cancel-login-link`](#post-apiadminusersuser_idcancel-login-link) | Stop an account's login link working | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/delete`](#post-apiadminusersuser_iddelete) | Delete an account | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/disabled`](#post-apiadminusersuser_iddisabled) | Disable or enable a user | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/expiry`](#post-apiadminusersuser_idexpiry) | Set when an account expires | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/reissue-login`](#post-apiadminusersuser_idreissue-login) | A new login link for an account | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/rename`](#post-apiadminusersuser_idrename) | Rename someone | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/reset-two-factor`](#post-apiadminusersuser_idreset-two-factor) | Turn off an account's two-factor | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/revoke-access`](#post-apiadminusersuser_idrevoke-access) | Revoke a user's access | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/role`](#post-apiadminusersuser_idrole) | Change an account's role | browser session or API key, admins |
| GET | [`/api/admin/users/{user_id}/sessions`](#get-apiadminusersuser_idsessions) | An account's signed-in devices | browser session or API key, admins |
| POST | [`/api/admin/users/{user_id}/sessions/{session_id}/revoke`](#post-apiadminusersuser_idsessionssession_idrevoke) | Sign out one of an account's devices | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/cancel-login-link`](#post-apibulkadminuserscancel-login-link) | Stop an account's login link working, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/delete`](#post-apibulkadminusersdelete) | Delete an account, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/disabled`](#post-apibulkadminusersdisabled) | Disable or enable a user, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/reset-two-factor`](#post-apibulkadminusersreset-two-factor) | Turn off an account's two-factor, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/revoke-access`](#post-apibulkadminusersrevoke-access) | Revoke a user's access, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/role`](#post-apibulkadminusersrole) | Change an account's role, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/users/sessions/revoke`](#post-apibulkadminuserssessionsrevoke) | Sign out one of an account's devices, for many at once | browser session or API key, admins |

### GET /api/admin/theme-defaults

Default themes

The theme list everyone without their own list sees. themes is null while the built-in ones apply.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_get_theme_defaults`](mcp.md#admin_get_theme_defaults) |
| Errors | 401, 403, 404, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `builtins` | integer | yes |  | Which built-in themes that list has seen (see builtins on POST /api/account/themes). |
| `themes` | array of [`Theme`](#theme) or null | yes |  | null while the built-in themes apply. |

### POST /api/admin/theme-defaults

Replace the default themes

Replace the theme list everyone without their own list sees, 1 to 40 themes in the Theme schema of the API reference. null puts the built-in themes back.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_theme_defaults`](mcp.md#admin_set_theme_defaults) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `builtins` | integer | no |  | Which built-in themes this list has seen: the highest since in Theme.builtIn (static/js/theme.js) when it was saved. Built-in themes added after that join the list the browser shows. Left out: 0, the first ones. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `themes` | array of [`Theme`](#theme) or null | no |  | 1 to 40 themes. null goes back to the built-in ones. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `builtins` | integer | yes |  | Which built-in themes that list has seen (see builtins on POST /api/account/themes). |
| `ok` | boolean | yes |  |  |
| `themes` | array of [`Theme`](#theme) or null | yes |  |  |

### GET /api/admin/users

List users

Accounts, a page at a time, including disabled ones, with each one's role and which is the owner. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_list_users`](mcp.md#admin_list_users) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | query | string | no |  | Keep accounts whose name or username contains this, in any case. |
| `sort` | query | string | no | one of `display_name`, `role`, `disabled`, `created_at` | Field to sort the whole list by. Rows without it go last. |
| `dir` | query | string | no | one of `asc`, `desc` | Default asc. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `total` | integer | yes |  |  |
| `users` | array of [`User`](#user) | yes |  |  |

### POST /api/admin/users

Create an account

Make a new account and its first login link, to hand to the person. role is admin, user or viewer (reads only). account_expires_at, when given, is when the account stops working. Answers with user_id and the link.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_create_user`](mcp.md#admin_create_user) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `account_expires_at` | string (date-time) or null | no |  | When the account stops working (RFC 3339). Left out: never. expires_in_minutes is the login link's expiry, not this. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `display_name` | string | yes | 1 to 80 characters |  |
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |
| `role` | string | yes | one of `admin`, `user`, `viewer` |  |

**Reply 201**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_at` | string (date-time) or null | yes |  | null when it never expires. |
| `expires_in_seconds` | integer | yes |  | 0 when it never expires. |
| `login_path` | string | yes |  | Path of the login link. A credential: treat it like a password. |
| `max_uses` | integer | yes |  | 0 when unlimited. |
| `unlimited` | boolean | yes |  |  |
| `user_id` | integer | yes |  |  |

### POST /api/admin/users/{user_id}/cancel-login-link

Stop an account's login link working

Void an account's live login link without making a new one: a link that went somewhere it should not have.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_cancel_login_link`](mcp.md#admin_cancel_login_link) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `removed` | integer | yes |  | Links removed: 0 or 1. |

### POST /api/admin/users/{user_id}/delete

Delete an account

Remove an account for good, with its sessions, login link and API keys.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_delete_user`](mcp.md#admin_delete_user) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `removed` | [`Revoked`](#revoked) | yes |  |  |

### POST /api/admin/users/{user_id}/disabled

Disable or enable a user

Switch an account off (disabled: true) or back on (false). Disabling signs it out everywhere and voids its live login link; its API keys stop working while it is off. Reversible. The owner and your own account cannot be disabled.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_user_disabled`](mcp.md#admin_set_user_disabled) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `disabled` | boolean | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/admin/users/{user_id}/expiry

Set when an account expires

expires_at is when the account stops working (RFC 3339), or null for never. Once it passes the account counts as disabled: sign-in, its sessions and its keys stop, and housekeeping disables it and revokes its access. A later expiry does not re-enable an account housekeeping already disabled. It must be in the future. The owner's account never expires; only the owner sets an admin's.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_user_expiry`](mcp.md#admin_set_user_expiry) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_at` | string (date-time) or null | yes |  | When the account stops working (RFC 3339), or null for never. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_at` | string (date-time) or null | yes |  | null when it never expires. |
| `ok` | boolean | yes |  |  |

### POST /api/admin/users/{user_id}/reissue-login

A new login link for an account

For someone who lost their device or their link: a fresh login link for their account.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_reissue_login`](mcp.md#admin_reissue_login) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_at` | string (date-time) or null | yes |  | null when it never expires. |
| `expires_in_seconds` | integer | yes |  | 0 when it never expires. |
| `login_path` | string | yes |  | Path of the login link. A credential: treat it like a password. |
| `max_uses` | integer | yes |  | 0 when unlimited. |
| `unlimited` | boolean | yes |  |  |

### POST /api/admin/users/{user_id}/rename

Rename someone

Change another account's display name (1-80 characters). Only the owner renames an admin. expect, when given, is the name you think it has now; if it changed meanwhile the rename is refused with the current name.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_rename_user`](mcp.md#admin_rename_user) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `display_name` | string | yes | 1 to 80 characters | 1-80 characters. |
| `expect` | string or null | no |  | The name you think they have. Answers 409 if it is another. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/admin/users/{user_id}/reset-two-factor

Turn off an account's two-factor

For someone who lost both their phone and their recovery codes: switches their two-factor sign-in off so a login link lets them in again. Only the owner resets an admin's; the owner's is reset from a shell.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_reset_two_factor`](mcp.md#admin_reset_two_factor) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/admin/users/{user_id}/revoke-access

Revoke a user's access

For an account that may be compromised: ends every session and voids its live login link and every API key it holds, at once.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_access`](mcp.md#admin_revoke_access) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `revoked` | [`Revoked`](#revoked) | yes |  |  |

### POST /api/admin/users/{user_id}/role

Change an account's role

Make an account an admin, an ordinary user, or a viewer, who may read but change nothing. Only the owner takes an admin's rights away; the owner is always an admin, and nobody changes their own role. Becoming an admin ends the account's sessions and live login link.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_user_role`](mcp.md#admin_set_user_role) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `role` | string | yes | one of `admin`, `user`, `viewer` |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `role` | string | yes |  |  |

### GET /api/admin/users/{user_id}/sessions

An account's signed-in devices

Where an account is signed in: every session, with when it started and was last seen. Addresses are shown to the owner only, and only the owner sees the owner's.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_list_user_sessions`](mcp.md#admin_list_user_sessions) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `sessions` | array of [`Session`](#session) | yes |  |  |
| `total` | integer | yes |  |  |

### POST /api/admin/users/{user_id}/sessions/{session_id}/revoke

Sign out one of an account's devices

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_user_session`](mcp.md#admin_revoke_user_session) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `session_id` | path | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/bulk/admin/users/cancel-login-link

Stop an account's login link working, for many at once

Runs POST /api/admin/users/{user_id}/cancel-login-link once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_cancel_login_link_bulk`](mcp.md#admin_cancel_login_link_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/delete

Delete an account, for many at once

Runs POST /api/admin/users/{user_id}/delete once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_delete_user_bulk`](mcp.md#admin_delete_user_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/disabled

Disable or enable a user, for many at once

Runs POST /api/admin/users/{user_id}/disabled once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_user_disabled_bulk`](mcp.md#admin_set_user_disabled_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].disabled` | boolean | no |  |  |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |
| `shared.disabled` | boolean | no |  |  |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/reset-two-factor

Turn off an account's two-factor, for many at once

Runs POST /api/admin/users/{user_id}/reset-two-factor once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_reset_two_factor_bulk`](mcp.md#admin_reset_two_factor_bulk) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/revoke-access

Revoke a user's access, for many at once

Runs POST /api/admin/users/{user_id}/revoke-access once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_access_bulk`](mcp.md#admin_revoke_access_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/role

Change an account's role, for many at once

Runs POST /api/admin/users/{user_id}/role once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_user_role_bulk`](mcp.md#admin_set_user_role_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].role` | string | no | one of `admin`, `user`, `viewer` |  |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |
| `shared.role` | string | no | one of `admin`, `user`, `viewer` |  |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/users/sessions/revoke

Sign out one of an account's devices, for many at once

Runs POST /api/admin/users/{user_id}/sessions/{session_id}/revoke once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_user_session_bulk`](mcp.md#admin_revoke_user_session_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].session_id` | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |
| `items[].user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

## Invite links

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/admin/invite-links`](#get-apiadmininvite-links) | List invite links | browser session or API key, admins |
| POST | [`/api/admin/invite-links`](#post-apiadmininvite-links) | Create an invite link | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/delete`](#post-apiadmininvite-linkslink_iddelete) | Delete an invite link | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/limits`](#post-apiadmininvite-linkslink_idlimits) | Change an invite link's places and expiry | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/rename`](#post-apiadmininvite-linkslink_idrename) | Rename an invite link | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/restore`](#post-apiadmininvite-linkslink_idrestore) | Undo deleting an invite link | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/revoke`](#post-apiadmininvite-linkslink_idrevoke) | Switch off an invite link | browser session or API key, admins |
| POST | [`/api/admin/invite-links/{link_id}/switch-on`](#post-apiadmininvite-linkslink_idswitch-on) | Switch an invite link back on | browser session or API key, admins |
| POST | [`/api/bulk/admin/invite-links/delete`](#post-apibulkadmininvite-linksdelete) | Delete an invite link, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/invite-links/restore`](#post-apibulkadmininvite-linksrestore) | Undo deleting an invite link, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/invite-links/revoke`](#post-apibulkadmininvite-linksrevoke) | Switch off an invite link, for many at once | browser session or API key, admins |
| POST | [`/api/bulk/admin/invite-links/switch-on`](#post-apibulkadmininvite-linksswitch-on) | Switch an invite link back on, for many at once | browser session or API key, admins |
| POST | [`/api/join/claim`](#post-apijoinclaim) | Sign in as an existing account | anyone |
| POST | [`/api/join/create`](#post-apijoincreate) | Join as a new account | anyone |
| POST | [`/api/join/info`](#post-apijoininfo) | What an invite link is | anyone |
| POST | [`/api/join/match`](#post-apijoinmatch) | Accounts matching a name | anyone |

### GET /api/admin/invite-links

List invite links

Group invite links, a page at a time: label, places used and left, expiry, and whether each still works. The links themselves are never shown again after creation. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_list_invite_links`](mcp.md#admin_list_invite_links) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | query | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | query | string | no |  | Keep links whose label or maker contains this, in any case. |
| `sort` | query | string | no | one of `label`, `used`, `expires_at` | Field to sort the whole list by. Rows without it go last. |
| `dir` | query | string | no | one of `asc`, `desc` | Default asc. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `invite_links` | array of [`InviteLink`](#invitelink) | yes |  |  |
| `next` | integer or null | yes |  | Offset of the next page, or null on the last one. |
| `total` | integer | yes |  |  |

### POST /api/admin/invite-links

Create an invite link

One link for a group: each person who opens it joins as a new account with role (user, the default, or viewer), up to max_uses people, for days days.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_create_invite_link`](mcp.md#admin_create_invite_link) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `days` | integer | yes | 1 to 365 |  |
| `label` | string | yes | 1 to 80 characters |  |
| `max_uses` | integer | yes | 1 to 1000 |  |
| `role` | string | no | one of `user`, `viewer`; default `user` | The role each person who joins gets: user, or viewer (reads only). |

**Reply 201**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_by` | integer | yes |  |  |
| `created_by_name` | string | no |  |  |
| `expires_at` | string (date-time) | yes |  |  |
| `id` | string | yes |  |  |
| `join_path` | string | yes |  | The link to hand out, shown in this response only. A credential: treat it like a password. |
| `label` | string | yes |  |  |
| `live` | boolean | yes |  |  |
| `max_uses` | integer | yes |  |  |
| `remaining` | integer | yes |  |  |
| `revoked` | boolean | yes |  |  |
| `role` | string | yes | one of `user`, `viewer` | The role each person who joins gets. |
| `used` | integer | yes |  |  |

### POST /api/admin/invite-links/{link_id}/delete

Delete an invite link

Take an invite link off the list and stop it working. The accounts it made stay.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_delete_invite_link`](mcp.md#admin_delete_invite_link) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/admin/invite-links/{link_id}/limits

Change an invite link's places and expiry

Set how many people the link may take in (never fewer than have joined) and, when days is given, a new expiry counted from now. The link then counts as this key's: revoking the key switches it off.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_set_invite_link_limits`](mcp.md#admin_set_invite_link_limits) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `days` | integer or null | no | 1 to 365 | Counted from now. Leave out to keep the expiry. |
| `max_uses` | integer | yes | 1 to 1000 | Not fewer than have joined. |

**Reply 200**: Success

[`InviteLink`](#invitelink).

### POST /api/admin/invite-links/{link_id}/rename

Rename an invite link

Change the label people see on an invite link's join page (1-80 characters). expect, when given, is the label you think it has now.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_rename_invite_link`](mcp.md#admin_rename_invite_link) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `expect` | string or null | no |  | The label you think it has. Answers 409 if it has another. |
| `label` | string | yes | 1 to 80 characters | 1-80 characters. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `label` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

### POST /api/admin/invite-links/{link_id}/restore

Undo deleting an invite link

Bring back an invite link deleted in the last few minutes, working as before. The link then counts as this key's: revoking the key switches it off.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_restore_invite_link`](mcp.md#admin_restore_invite_link) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

[`InviteLink`](#invitelink).

### POST /api/admin/invite-links/{link_id}/revoke

Switch off an invite link

Stop an invite link from letting anyone else join or sign in through it. Accounts it already made keep working. Cannot be switched back on.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_invite_link`](mcp.md#admin_revoke_invite_link) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/admin/invite-links/{link_id}/switch-on

Switch an invite link back on

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_switch_on_invite_link`](mcp.md#admin_switch_on_invite_link) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `link_id` | path | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

[`InviteLink`](#invitelink).

### POST /api/bulk/admin/invite-links/delete

Delete an invite link, for many at once

Runs POST /api/admin/invite-links/{link_id}/delete once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_delete_invite_link_bulk`](mcp.md#admin_delete_invite_link_bulk) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/invite-links/restore

Undo deleting an invite link, for many at once

Runs POST /api/admin/invite-links/{link_id}/restore once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_restore_invite_link_bulk`](mcp.md#admin_restore_invite_link_bulk) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/invite-links/revoke

Switch off an invite link, for many at once

Runs POST /api/admin/invite-links/{link_id}/revoke once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_revoke_invite_link_bulk`](mcp.md#admin_revoke_invite_link_bulk) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/admin/invite-links/switch-on

Switch an invite link back on, for many at once

Runs POST /api/admin/invite-links/{link_id}/switch-on once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, admins |
| MCP tool | [`admin_switch_on_invite_link_bulk`](mcp.md#admin_switch_on_invite_link_bulk) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/join/claim

Sign in as an existing account

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `credential` | string | yes |  |  |
| `display_name` | string | no |  |  |
| `pin` | string | no |  |  |
| `user_id` | integer | no |  |  |

**Reply 200**: Success

[`Joined`](#joined).

### POST /api/join/create

Join as a new account

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `credential` | string | yes |  |  |
| `display_name` | string | no |  |  |
| `pin` | string | no |  |  |
| `user_id` | integer | no |  |  |

**Reply 201**: Success

[`Joined`](#joined).

### POST /api/join/info

What an invite link is

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `credential` | string | yes |  |  |
| `display_name` | string | no |  |  |
| `pin` | string | no |  |  |
| `user_id` | integer | no |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `label` | string | yes |  |  |
| `remaining` | integer | yes |  |  |

### POST /api/join/match

Accounts matching a name

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `credential` | string | yes |  |  |
| `display_name` | string | no |  |  |
| `pin` | string | no |  |  |
| `user_id` | integer | no |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `can_create` | boolean | yes |  |  |
| `matches` | array of object | yes |  |  |
| `matches[].display_name` | string | yes |  |  |
| `matches[].has_pin` | boolean | yes |  |  |
| `matches[].id` | integer | yes |  |  |

## Owner

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/owner/jobs`](#get-apiownerjobs) | Background jobs | browser session only, the owner |
| POST | [`/api/owner/jobs/{name}/run`](#post-apiownerjobsnamerun) | Run a job now | browser session only, the owner |
| GET | [`/api/owner/logs/days`](#get-apiownerlogsdays) | The days on record | browser session only, the owner |
| GET | [`/api/owner/logs/days/{date}`](#get-apiownerlogsdaysdate) | One day's entries | browser session only, the owner |
| POST | [`/api/owner/logs/days/{date}/delete`](#post-apiownerlogsdaysdatedelete) | Delete one day | browser session only, the owner |
| GET | [`/api/owner/logs/days/{date}/download`](#get-apiownerlogsdaysdatedownload) | One day's entries as a file | browser session only, the owner |
| GET | [`/api/owner/logs/ip-names`](#get-apiownerlogsip-names) | What the owner calls addresses | browser session only, the owner |
| POST | [`/api/owner/logs/ip-names`](#post-apiownerlogsip-names) | Name an address or a range, or rename one | browser session only, the owner |
| POST | [`/api/owner/logs/ip-names/delete`](#post-apiownerlogsip-namesdelete) | Take a name off an address or a range | browser session only, the owner |
| POST | [`/api/owner/logs/prune`](#post-apiownerlogsprune) | Apply the retention rules now | browser session only, the owner |
| GET | [`/api/owner/logs/stream`](#get-apiownerlogsstream) | New log entries as they happen | browser session only, the owner |
| GET | [`/api/owner/settings`](#get-apiownersettings) | config.yaml, as a form | browser session only, the owner |
| POST | [`/api/owner/settings`](#post-apiownersettings) | Save settings | browser session only, the owner |
| POST | [`/api/owner/users/{user_id}/password-access`](#post-apiownerusersuser_idpassword-access) | Allow or block an account's password | browser session only, the owner |

### GET /api/owner/jobs

Background jobs

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 404, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `jobs` | array of [`Job`](#job) | yes |  |  |

### POST /api/owner/jobs/{name}/run

Run a job now

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `name` | path | string | yes |  | The job's name, from GET /api/owner/jobs. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 202**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/owner/logs/days

The days on record

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `q` | query | string | no |  | The Logs screen's search. |

**Reply 200**: Success

[`LogDays`](#logdays).

### GET /api/owner/logs/days/{date}

One day's entries

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `date` | path | string | yes |  | The day's date, from GET /api/owner/logs/days. |
| `q` | query | string | no |  | The Logs screen's search. |
| `before` | query | integer or null | no | at least 0 |  |
| `limit` | query | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |

**Reply 200**: Success

[`LogDay`](#logday).

### POST /api/owner/logs/days/{date}/delete

Delete one day

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `date` | path | string | yes |  | The day's date, from GET /api/owner/logs/days. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `deleted` | boolean | yes |  |  |

### GET /api/owner/logs/days/{date}/download

One day's entries as a file

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `date` | path | string | yes |  | The day's date, from GET /api/owner/logs/days. |

**Reply 200**: Success

Text, as `application/x-ndjson`.

### GET /api/owner/logs/ip-names

What the owner calls addresses

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 403, 404, 429, 503 |

**Reply 200**: Success

[`IPNames`](#ipnames).

### POST /api/owner/logs/ip-names

Name an address or a range, or rename one

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `name` | string | yes | 1 to 80 characters |  |
| `target` | string | yes | e.g. `10.0.0.0/24` | An address, or a range such as 10.0.0.0/24. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `name` | string | yes |  |  |
| `target` | string | yes |  |  |

### POST /api/owner/logs/ip-names/delete

Take a name off an address or a range

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `target` | string | yes | e.g. `10.0.0.0/24` | An address, or a range such as 10.0.0.0/24. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `deleted` | boolean | yes |  |  |

### POST /api/owner/logs/prune

Apply the retention rules now

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

[`PruneResult`](#pruneresult).

### GET /api/owner/logs/stream

New log entries as they happen

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `q` | query | string | no |  | The Logs screen's search. |

**Reply 200**: Success

Text, as `text/event-stream`.

### GET /api/owner/settings

config.yaml, as a form

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 404, 429, 503 |

**Reply 200**: Success

[`SettingsSnapshot`](#settingssnapshot).

### POST /api/owner/settings

Save settings

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 409, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `changes` | object | yes |  | Setting paths and their new values. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `version` | string | yes |  |  |

**Reply 200**: Success

[`SettingsResult`](#settingsresult).

### POST /api/owner/users/{user_id}/password-access

Allow or block an account's password

|  |  |
|---|---|
| Who may call | browser session only, the owner |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `user_id` | path | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `allowed` | boolean | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |
| `password_blocked` | boolean | yes |  |  |

## Monitoring

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/api.json`](#get-apiapijson) | This document | browser session or API key, anyone signed in |
| GET | [`/api/api.md`](#get-apiapimd) | The API reference as Markdown tables for people | browser session or API key, anyone signed in |
| GET | [`/api/health`](#get-apihealth) | Is the process up | anyone |
| GET | [`/api/mcp.json`](#get-apimcpjson) | The MCP tools: every tool /mcp has, its arguments, and the route it stands for | browser session or API key, anyone signed in |
| GET | [`/api/mcp.md`](#get-apimcpmd) | The MCP tools as Markdown tables for people | browser session or API key, anyone signed in |
| GET | [`/api/ready`](#get-apiready) | Can the server do real work | anyone |
| GET | [`/mcp`](#get-mcp) | Refused: this server has no stream | anyone |
| POST | [`/mcp`](#post-mcp) | Model Context Protocol, at the key's role | API key only, anyone signed in |
| DELETE | [`/mcp`](#delete-mcp) | Refused: this server keeps no sessions | anyone |

### GET /api/api.json

This document

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The API reference itself; GET /api/mcp.json is the MCP one. |
| Errors | 401, 404, 429, 503 |

**Reply 200**: Success

An empty object.

### GET /api/api.md

The API reference as Markdown tables for people

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The API reference itself, for people; GET /api/api.json is the same as JSON. |
| Errors | 401, 404, 429, 503 |

**Reply 200**: Success

Text, as `text/markdown`.

### GET /api/health

Is the process up

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/mcp.json

The MCP tools: every tool /mcp has, its arguments, and the route it stands for

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The MCP reference itself; tools/list gives the same list. |
| Errors | 401, 404, 429, 503 |

**Reply 200**: Success

An empty object.

### GET /api/mcp.md

The MCP tools as Markdown tables for people

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The MCP reference itself, for people; tools/list gives the same list. |
| Errors | 401, 404, 429, 503 |

**Reply 200**: Success

Text, as `text/markdown`.

### GET /api/ready

Can the server do real work

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /mcp

Refused: this server has no stream

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 404, 429, 503 |

**Reply 405**: Success

### POST /mcp

Model Context Protocol, at the key's role

|  |  |
|---|---|
| Who may call | API key only, anyone signed in |
| MCP tool | None, HTTP only: The MCP endpoint itself. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

An empty object.

**Reply 200**: Success

An empty object.

### DELETE /mcp

Refused: this server keeps no sessions

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 404, 429, 503 |

**Reply 405**: Success

## Backup

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/backup`](#get-apibackup) | Download a backup of your boards | browser session or API key, anyone signed in |
| POST | [`/api/backup/board`](#post-apibackupboard) | Restore a board from a backup | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/lists/{list_id}/export`](#get-apiboardsboard_idlistslist_idexport) | Download a column | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/lists/{list_id}/import`](#post-apiboardsboard_idlistslist_idimport) | Add cards to a column from a file | browser session or API key, anyone signed in |

### GET /api/backup

Download a backup of your boards

Every board you are on, as one JSON file to keep: each as much of it as you can see, with its columns, labels and cards. Boards reached through a single shared card are left out.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 401, 429, 503 |

**Reply 200**: Success

An empty object.

### POST /api/backup/board

Restore a board from a backup

Make a new board you own from one board of a backup file, with its columns and labels but no cards yet: add those a batch at a time with POST /api/boards/{board_id}/lists/{list_id}/import, using the list ids answered (in the backup's order). Never overwrites anything.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A backup file's contents, which the page sends in steps; deliberately not a tool. |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | object or null | yes |  |  |
| `board.carry_over` | boolean or null | no |  |  |
| `board.center_columns` | boolean | no |  |  |
| `board.color` | string | yes |  |  |
| `board.default_view` | string | yes |  |  |
| `board.labels` | array of object | yes |  |  |
| `board.labels[].color` | string | yes |  |  |
| `board.labels[].name` | string | yes |  |  |
| `board.labels[].ref` | integer | yes |  |  |
| `board.layout` | string | yes |  |  |
| `board.lists` | array of object | yes |  |  |
| `board.lists[].archived` | boolean | yes |  |  |
| `board.lists[].cards` | array of object | yes |  |  |
| `board.lists[].cards[].archived` | boolean | yes |  |  |
| `board.lists[].cards[].assignees` | array of string | no |  |  |
| `board.lists[].cards[].checklist` | array of object | yes |  |  |
| `board.lists[].cards[].checklist[].done` | boolean | yes |  |  |
| `board.lists[].cards[].checklist[].text` | string | yes |  |  |
| `board.lists[].cards[].day` | string | no |  |  |
| `board.lists[].cards[].description` | string | yes |  |  |
| `board.lists[].cards[].doing` | boolean | no |  |  |
| `board.lists[].cards[].done` | boolean | yes |  |  |
| `board.lists[].cards[].done_at` | string (date-time) or null | yes |  |  |
| `board.lists[].cards[].due_all_day` | boolean | yes |  |  |
| `board.lists[].cards[].due_at` | string (date-time) or null | yes |  |  |
| `board.lists[].cards[].labels` | array of integer | yes |  |  |
| `board.lists[].cards[].price` | string | no |  |  |
| `board.lists[].cards[].rank` | number | no |  |  |
| `board.lists[].cards[].scheduled` | boolean | no |  |  |
| `board.lists[].cards[].start_at` | string (date-time) or null | yes |  |  |
| `board.lists[].cards[].start_timed` | boolean | yes |  |  |
| `board.lists[].cards[].title` | string | yes |  |  |
| `board.lists[].shopping` | boolean | no |  |  |
| `board.lists[].show_updated` | string | no |  |  |
| `board.lists[].stage_tabs` | string | no |  |  |
| `board.lists[].stage_view` | string | no |  |  |
| `board.lists[].title` | string | yes |  |  |
| `board.notes` | string | no |  |  |
| `board.numbering` | string | yes |  |  |
| `board.purpose` | string | no |  |  |
| `board.purpose_enabled` | boolean | no |  |  |
| `board.shopping` | boolean | no |  |  |
| `board.show_updated` | boolean | no |  |  |
| `board.slug` | string | yes |  |  |
| `board.stage_doing` | string | no |  |  |
| `board.stage_done` | string | no |  |  |
| `board.stage_scheduled` | string | no |  |  |
| `board.stage_set` | string | no |  |  |
| `board.stage_tabs` | boolean | no |  |  |
| `board.stage_todo` | string | no |  |  |
| `board.theme` | object or null | yes |  |  |
| `board.theme.accent` | string | yes |  |  |
| `board.theme.background` | string | no |  |  |
| `board.theme.button` | object or null | no |  |  |
| `board.theme.button.case` | string | no | one of `normal`, `upper` |  |
| `board.theme.button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `board.theme.button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `board.theme.button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `board.theme.card` | string | no |  |  |
| `board.theme.column` | string | no |  |  |
| `board.theme.dark` | object or null | no |  |  |
| `board.theme.dark.accent` | string | yes |  | The main color: what you can press. |
| `board.theme.dark.background` | string | no |  | The page. Left out: the mode's own. |
| `board.theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `board.theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `board.theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `board.theme.light` | object or null | no |  |  |
| `board.theme.light.accent` | string | yes |  | The main color: what you can press. |
| `board.theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `board.theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `board.theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `board.theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `board.theme.mode` | string | yes |  |  |
| `board.theme.overrides` | map of string | no |  |  |
| `board.theme.second` | string | no |  |  |
| `board.theme.surface` | string | no |  |  |
| `board.title` | string | yes |  |  |
| `csrf_token` | string | no |  |  |

**Reply 200**: Success

An empty object.

### GET /api/boards/{board_id}/lists/{list_id}/export

Download a column

One column as a JSON file: its cards you can see, and the labels they use.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/lists/{list_id}/import

Add cards to a column from a file

Append the cards of a column's file (or a batch of a backup board's cards) to a column, with the labels they use. Needs the rights to add a card there. Never overwrites anything. Answers how many cards were added.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A backup file's contents, which the page sends in steps; deliberately not a tool. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `cards` | array of object | yes |  |  |
| `cards[].archived` | boolean | yes |  |  |
| `cards[].assignees` | array of string | no |  |  |
| `cards[].checklist` | array of object | yes |  |  |
| `cards[].checklist[].done` | boolean | yes |  |  |
| `cards[].checklist[].text` | string | yes |  |  |
| `cards[].day` | string | no |  |  |
| `cards[].description` | string | yes |  |  |
| `cards[].doing` | boolean | no |  |  |
| `cards[].done` | boolean | yes |  |  |
| `cards[].done_at` | string (date-time) or null | yes |  |  |
| `cards[].due_all_day` | boolean | yes |  |  |
| `cards[].due_at` | string (date-time) or null | yes |  |  |
| `cards[].labels` | array of integer | yes |  |  |
| `cards[].price` | string | no |  |  |
| `cards[].rank` | number | no |  |  |
| `cards[].scheduled` | boolean | no |  |  |
| `cards[].start_at` | string (date-time) or null | yes |  |  |
| `cards[].start_timed` | boolean | yes |  |  |
| `cards[].title` | string | yes |  |  |
| `csrf_token` | string | no |  |  |
| `labels` | array of object | yes |  |  |
| `labels[].color` | string | yes |  |  |
| `labels[].name` | string | yes |  |  |
| `labels[].ref` | integer | yes |  |  |

**Reply 200**: Success

An empty object.

## Boards

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/boards`](#get-apiboards) | List my boards | browser session or API key, anyone signed in |
| POST | [`/api/boards`](#post-apiboards) | Create a board | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}`](#get-apiboardsboard_id) | Read a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}`](#post-apiboardsboard_id) | Change a board's settings | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/archive`](#post-apiboardsboard_idarchive) | Archive or unarchive a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/carry-over`](#post-apiboardsboard_idcarry-over) | Carry unfinished cards over to today | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/delete`](#post-apiboardsboard_iddelete) | Delete a board | browser session only, anyone signed in |
| POST | [`/api/boards/{board_id}/duplicate`](#post-apiboardsboard_idduplicate) | Duplicate a board | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/export`](#get-apiboardsboard_idexport) | Download a board as a spreadsheet | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/labels`](#post-apiboardsboard_idlabels) | Add a label | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/labels/{label_id}`](#post-apiboardsboard_idlabelslabel_id) | Change a label | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/labels/{label_id}/delete`](#post-apiboardsboard_idlabelslabel_iddelete) | Delete a label | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/leave`](#post-apiboardsboard_idleave) | Leave a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/lists`](#post-apiboardsboard_idlists) | Add a column | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/lists/{list_id}`](#post-apiboardsboard_idlistslist_id) | Rename, move or archive a column | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/lists/{list_id}/delete`](#post-apiboardsboard_idlistslist_iddelete) | Delete a column | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/my-view`](#post-apiboardsboard_idmy-view) | Your own view of a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/my-view/collapsed`](#post-apiboardsboard_idmy-viewcollapsed) | Your collapsed columns | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/notes`](#get-apiboardsboard_idnotes) | Read a board's purpose and notes | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/notes`](#post-apiboardsboard_idnotes) | Change a board's purpose or notes | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/stamp`](#get-apiboardsboard_idstamp) | A board's revision | browser session or API key, anyone signed in |

### GET /api/boards

List my boards

Every board you own or are a member of (id, slug, title, your role, folder, and whether it is archived for everyone -- read-only -- or archived_for_me), your home-page folders, and single cards shared with you from boards you are not on.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_boards`](mcp.md#list_boards) |
| Errors | 401, 429, 503 |

**Reply 200**: Success

An empty object.

### POST /api/boards

Create a board

A new board owned by you, with the given columns in order (for example ["To do", "Doing", "Done"]); a board always gets at least three, so fewer are padded out. Returns its id and slug. A key held to some boards cannot make one.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_board`](mcp.md#create_board) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `lists` | array of string | no |  | Its columns, in order, for example ["To do", "Doing", "Done"]. A board always gets at least three, so fewer are padded out. |
| `title` | string | yes | at most 120 characters | The new board's title. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `id` | integer | yes |  |  |
| `slug` | string | yes |  | Its URL name, the part after /b/. |

### GET /api/boards/{board_id}

Read a board

A board as you can see it: its notes, its purpose if the board has that section on (board.purpose says what the board is for and how to work on it, so read it first), columns (board.lists), labels, members, and the cards you may see, with each card's column, labels, assignees, dates, checklist and rev. people maps user ids to names. access is what you may do.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_board`](mcp.md#get_board) |
| Errors | 400, 401, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `include_archived` | query | boolean or null | no |  | Include archived cards. The page gets them unless this is false; an agent's get_board leaves them out unless it is true. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}

Change a board's settings

Change any of a board's settings; leave out what should not change: its title, URL name (slug), layout (free, three, staged or calendar), default view, colour, theme, card numbering, where finished cards and the add-card box go, the stage names and buttons of a staged board, carry-over on a calendar board, shopping, ranking, and its purpose and notes. Needs edit rights on the whole board. The purpose and notes are for members, not share links; the quick-share and anonymous-comment switches are the owner's, from a browser. Answers the board's new revs.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_board`](mcp.md#update_board) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `add_placement` | string or null | no | one of ``, `top`, `bottom` | Where the add-card box sits in a column: top, bottom, or "" for each person's account setting. |
| `allow_anon_comments` | boolean or null | no |  | Let guests comment without giving a name. The owner's, from a browser only: refused through a key. |
| `allow_quick_share` | boolean or null | no |  | Let the board's cards be shared by quick link. The owner's, from a browser only: refused through a key. |
| `carry_over` | boolean or null | no |  | On a calendar board, move unfinished cards from past days to today. |
| `center_columns` | boolean or null | no |  | Centre the columns on wide screens. |
| `clear_theme` | boolean | no |  | true drops the board's own theme: everyone sees their own again. |
| `color` | string or null | no | one of ``, `blue`, `green`, `orange`, `red`, `purple`, `pink`, `teal`, `gray`, `yellow` | The board's colour; empty for none. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `default_view` | string or null | no | one of `kanban`, `todo`, `timeline` | The view the board opens in for someone who has not picked one. |
| `done_placement` | string or null | no | one of ``, `top`, `bottom` | Where finished cards go in a column: top, bottom, or "" for each person's account setting. |
| `layout` | string or null | no | one of ``, `three`, `staged`, `calendar` | "" (free: any number of columns), "three" (pinned to three columns: to do, doing, done), "staged" (each column has its own To do / Doing / Done, with an All view ranked board-wide), or "calendar" (cards filed by day; columns titled with dates are filed under those days). |
| `list_titles` | array of string | no |  | With layout "three": the titles of the columns added if the board has fewer than three open ones. |
| `notes` | string or null | no |  | Markdown: links, documents, an index. Members only. |
| `notes_rev` | integer or null | no |  | The notes_rev the notes were read at, as purpose_rev. |
| `numbering` | string or null | no | one of ``, `on`, `off` | Card numbers for everyone on the board: on, off, or "" to leave it to each person's account setting. |
| `purpose` | string or null | no |  | Markdown: what the board is for and how to work on it. Only while the Purpose section is on (409 purpose_off otherwise). Members only. |
| `purpose_enabled` | boolean or null | no |  | Switch the board's Purpose section on or off. Members only. |
| `purpose_rev` | integer or null | no |  | The purpose_rev the purpose was read at: if it moved since, 409 board_text_stale instead of writing over someone's change. |
| `shopping` | boolean or null | no |  | A shopping board: cards carry products and prices. |
| `show_rank` | boolean or null | no |  | Show each card's place in the board-wide ranking. |
| `show_updated` | boolean or null | no |  | Every card face shows when it was last changed, in its bottom-right corner. A column's own show_updated can override it. |
| `slug` | string or null | no |  | The board's URL name (the part after /b/). Empty picks one from the title. 409 slug_taken if another board has it. |
| `stage_doing` | string or null | no |  | What the Doing stage is called on this board; empty for the default. |
| `stage_done` | string or null | no |  | What the Done stage is called on this board; empty for the default. |
| `stage_scheduled` | string or null | no |  | With stage_set review: what the Scheduled stage is called; empty for the default. |
| `stage_set` | string or null | no | one of ``, `review` | On a staged or calendar board, its stages: "" for To do / Doing / Done, or "review" for Ready for review / Working / Scheduled / Resolved (cards' doing, scheduled and done flags). |
| `stage_tabs` | boolean or null | no |  | On a staged board, give every column its own To do / Doing / Done buttons. |
| `stage_todo` | string or null | no |  | What the To do stage is called on this board; empty for the default. |
| `theme` | object or null | no |  | The board's own theme, shown to everyone on it. |
| `theme.accent` | string | yes |  |  |
| `theme.background` | string | no |  |  |
| `theme.button` | object or null | no |  |  |
| `theme.button.case` | string | no | one of `normal`, `upper` |  |
| `theme.button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `theme.button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `theme.button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `theme.card` | string | no |  |  |
| `theme.column` | string | no |  |  |
| `theme.dark` | object or null | no |  |  |
| `theme.dark.accent` | string | yes |  | The main color: what you can press. |
| `theme.dark.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.light` | object or null | no |  |  |
| `theme.light.accent` | string | yes |  | The main color: what you can press. |
| `theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.mode` | string | yes |  |  |
| `theme.overrides` | map of string | no |  |  |
| `theme.second` | string | no |  |  |
| `theme.surface` | string | no |  |  |
| `title` | string or null | no | at most 120 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `notes_rev` | integer | yes |  |  |
| `purpose_rev` | integer | yes |  |  |
| `rev` | integer | yes |  |  |
| `slug` | string | yes |  |  |

### POST /api/boards/{board_id}/archive

Archive or unarchive a board

scope "everyone" (the owner only): the board leaves every member's home page and becomes read-only -- every change is refused with board_archived -- until it is unarchived the same way. scope "me" (anyone on the board, but not a key held to some boards): it only leaves your own home page. list_boards says which applies (archived, archived_for_me).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`archive_board`](mcp.md#archive_board) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | yes |  | true archives, false unarchives. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `scope` | string | yes | one of `everyone`, `me` | everyone (the owner only) or me (just your own home page). |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/carry-over

Carry unfinished cards over to today

On a calendar board with carry-over on, move the unfinished cards of past days to today (yours, within a day of the server's). Asking again the same day moves nothing; a board with carry-over off, or not a calendar, moves nothing. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`carry_over`](mcp.md#carry_over) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `today` | string | yes |  | Your today, as 2026-11-07: within a day of the server's. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `moved` | integer | yes |  | How many cards moved to today. |

### POST /api/boards/{board_id}/delete

Delete a board

Move one of your own boards to your board trash, where it stays until you restore it or delete it for good. The owner only, from a browser.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/duplicate

Duplicate a board

A new board owned by you, copied from one you can open: its settings, notes, labels, columns, and the cards you can see with their checklists. Comments, card pictures, members and share links are not copied. Returns the copy's id and slug. A key held to some boards cannot make one.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`duplicate_board`](mcp.md#duplicate_board) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `title` | string | yes | at most 120 characters | The copy's title. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `id` | integer | yes |  |  |
| `slug` | string | yes |  | Its URL name, the part after /b/. |

### GET /api/boards/{board_id}/export

Download a board as a spreadsheet

The board as an xlsx or csv spreadsheet (format): one row per card you can see, column by column, in board order. archived=1 adds archived cards and columns; tz is the timezone timed dates are written in.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `format` | query | string | no |  | xlsx or csv. |
| `archived` | query | string | no |  | 1 adds archived cards and columns, with an Archived column. |
| `tz` | query | string | no |  | The timezone timed dates are written in (an IANA name); defaults to your own, else UTC. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/boards/{board_id}/labels

Add a label

Add a label to a board, to put on cards with update_card's label_ids. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_label`](mcp.md#create_label) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string | yes |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `name` | string or null | no | at most 40 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string | yes |  |  |
| `id` | integer | yes |  |  |
| `name` | string | yes |  |  |

### POST /api/boards/{board_id}/labels/{label_id}

Change a label

Rename a label or change its colour; leave out what should not change. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_label`](mcp.md#update_label) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `label_id` | path | integer | yes |  | A label's id, from get_board (board.labels). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string or null | no |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `name` | string or null | no | at most 40 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/labels/{label_id}/delete

Delete a label

Delete a label; it comes off every card it was on. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_label`](mcp.md#delete_label) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `label_id` | path | integer | yes |  | A label's id, from get_board (board.labels). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/leave

Leave a board

Take yourself off a board you are a member of; it leaves your home page and you lose access until someone adds you again. The owner cannot leave (400 owner_cannot_leave).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`leave_board`](mcp.md#leave_board) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/lists

Add a column

Add a column (list) to a board, at the end or at position. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_list`](mcp.md#create_list) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `position` | number or null | no |  | Sort position among the columns; lower comes first. Leave out to put it at the end. |
| `title` | string | yes | at most 120 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean | yes |  |  |
| `id` | integer | yes |  |  |
| `position` | number | yes |  |  |
| `shopping` | boolean | yes |  |  |
| `show_updated` | string | yes |  |  |
| `stage_tabs` | string | yes |  |  |
| `stage_view` | string | yes |  |  |
| `title` | string | yes |  |  |
| `width` | integer | yes |  |  |

### POST /api/boards/{board_id}/lists/{list_id}

Rename, move or archive a column

Change a column's title, position or width, archive it (archived: true hides it; false brings it back), make it a shopping column, or on a staged board fix the stage it shows for everyone (stage_view; empty follows the navbar) or give it its own To do/Doing/Done buttons (stage_tabs: on, off, or empty to follow the board's setting). Leave out what should not change. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_list`](mcp.md#update_list) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides the column; false brings it back. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `position` | number or null | no |  | Sort position among the columns; lower comes first. |
| `shopping` | boolean or null | no |  | A shopping column: its cards carry products and prices. |
| `show_updated` | string or null | no | one of ``, `on`, `off` | Whether this column's cards show when they were last changed: on, off, or empty to follow the board's show_updated. |
| `stage_tabs` | string or null | no | one of ``, `on`, `off` | On a staged board, this column's own To do / Doing / Done buttons: on, off, or empty to follow the board's setting. |
| `stage_view` | string or null | no | one of ``, `todo`, `doing`, `scheduled`, `done` | On a staged board, the stage this column shows for everyone; empty follows the navbar. |
| `title` | string or null | no | at most 120 characters |  |
| `width` | integer or null | no |  | The column's width in pixels (200-800); 0 puts it back to the standard width. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/lists/{list_id}/delete

Delete a column

Delete a column and its share links; its cards go to the board's trash, where they can be restored into another column. Archiving it (update_list, archived: true) is the gentler way to put one away. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_list`](mcp.md#delete_list) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/my-view

Your own view of a board

How a board looks to you alone: greyscale status icons (plain_icons), or null to follow your account default. Nobody else on the board sees it.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_my_board_view`](mcp.md#set_my_board_view) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `plain_icons` | boolean or null | no |  | true for greyscale status icons on this board, false for coloured; null or left out follows your account default. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `plain_icons` | boolean or null | yes |  |  |

### POST /api/boards/{board_id}/my-view/collapsed

Your collapsed columns

Which columns of a board you keep collapsed, for you alone: the whole set each time.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_my_collapsed_lists`](mcp.md#set_my_collapsed_lists) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `list_ids` | array of integer | no |  | Every column you keep collapsed on this board (the whole set; ids from get_board's board.lists). Empty expands them all. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `collapsed_lists` | array of integer | yes |  | The ids kept: ones that are not columns of this board are dropped. |

### GET /api/boards/{board_id}/notes

Read a board's purpose and notes

Just the board's notes and purpose, without its cards. notes holds links, documents and reference material. purpose says what the board is for and how to work on it, and is only there when purpose_enabled is true. Both are markdown, and may be empty. Read them before working on a board. The revs go back to update_board_notes.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_board_notes`](mcp.md#get_board_notes) |
| Errors | 400, 401, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/notes

Change a board's purpose or notes

Replace a board's notes, its purpose, or both; leave out what should not change. A board's purpose can only be written while its Purpose section is on, which a person does in Board settings (409 purpose_off otherwise). Each is the whole new markdown text, so to add a link, send the old notes with it added. Each needs the rev it was read at, from get_board_notes or get_board: if someone edited the text since, the change is refused (409 board_text_stale) rather than written over theirs; read it again and redo the change on top. Needs edit rights on the board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_board_notes`](mcp.md#update_board_notes) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `notes` | string or null | no |  | Markdown: links, documents, an index. The whole new text. |
| `notes_rev` | integer or null | no |  | Required with notes: the notes_rev they were read at. |
| `purpose` | string or null | no |  | Markdown: what the board is for and how to work on it. The whole new text. |
| `purpose_rev` | integer or null | no |  | Required with purpose: the purpose_rev it was read at. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `notes_rev` | integer | yes |  |  |
| `purpose_rev` | integer | yes |  |  |

### GET /api/boards/{board_id}/stamp

A board's revision

The board's revision alone, which the page polls to know when to read the board again.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: The board page's cheap change poll; get_board reads the board itself. |
| Errors | 400, 401, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `rev` | integer | yes |  | Moves whenever anything the caller can see on the board changes. |

## Cards

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/boards/{board_id}/cards`](#post-apiboardsboard_idcards) | Add a card | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/cards/{card_id}`](#get-apiboardsboard_idcardscard_id) | Read a card | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}`](#post-apiboardsboard_idcardscard_id) | Change a card | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/checklist`](#post-apiboardsboard_idcardscard_idchecklist) | Add a checklist item | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/checklist/{item_id}`](#post-apiboardsboard_idcardscard_idchecklistitem_id) | Tick or edit a checklist item | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/checklist/{item_id}/delete`](#post-apiboardsboard_idcardscard_idchecklistitem_iddelete) | Delete a checklist item | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/copy`](#post-apiboardsboard_idcardscard_idcopy) | Copy a card to a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/delete`](#post-apiboardsboard_idcardscard_iddelete) | Delete a card | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/cards/{card_id}/image`](#get-apiboardsboard_idcardscard_idimage) | A card's picture | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/image`](#post-apiboardsboard_idcardscard_idimage) | Put a picture on a card | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/image/delete`](#post-apiboardsboard_idcardscard_idimagedelete) | Take a card's picture off | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/cards/{card_id}/links`](#get-apiboardsboard_idcardscard_idlinks) | List a card's mirrors | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete`](#post-apiboardsboard_idcardscard_idlinkslink_iddelete) | Stop mirroring a card | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/cards/{card_id}/product-image`](#get-apiboardsboard_idcardscard_idproduct-image) | A card's product photo | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/product/delete`](#post-apiboardsboard_idcardscard_idproductdelete) | Take a card's product off | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/product/refresh`](#post-apiboardsboard_idcardscard_idproductrefresh) | Look a card's product up again | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/send`](#post-apiboardsboard_idcardscard_idsend) | Send a card to another board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/lists/{list_id}/archive-cards`](#post-apiboardsboard_idlistslist_idarchive-cards) | Archive a column's cards | browser session or API key, anyone signed in |
| POST | [`/api/bulk/boards/cards`](#post-apibulkboardscards) | Change a card, for many at once | browser session or API key, anyone signed in |
| POST | [`/api/bulk/boards/cards/delete`](#post-apibulkboardscardsdelete) | Delete a card, for many at once | browser session or API key, anyone signed in |
| GET | [`/api/cards`](#get-apicards) | Find cards | browser session or API key, anyone signed in |

### POST /api/boards/{board_id}/cards

Add a card

Add a card to a column. The other fields are optional and set in the same call. Needs edit rights on that column.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_card`](mcp.md#create_card) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `assignees` | array of integer or null | no |  | User ids, from get_board's members or list_team. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `description` | string or null | no |  | Markdown. |
| `doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `done` | boolean or null | no |  | true makes it complete from the start. |
| `due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `label_ids` | array of integer or null | no |  | Label ids, from get_board (board.labels). |
| `list_id` | integer | yes |  | The column's id, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position within the column; lower comes first. Leave out to put it at the end. |
| `rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). At most one of doing, scheduled and done. |
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string | yes |  |  |

**Reply 200**: Success

An empty object.

### GET /api/boards/{board_id}/cards/{card_id}

Read a card

One card in full: title, description, column, labels, assignees, dates, checklist, its place in the board's ranking (rank_place, 0 for none), and rev (pass rev back to update_card when changing the title or description).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_card`](mcp.md#get_card) |
| Errors | 400, 401, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}

Change a card

Change any of a card's fields; leave out what should not change. Moving it (list_id, position) or archiving it needs edit rights on its column, and on the whole board to move it to another column. done marks it complete; doing and rank_place are the Doing stage and board-wide importance of a staged board. archived: true hides it from the board (reversible). Pass the card's rev when changing title or description: if someone edited it since, the change is refused (409 card_stale, with the card as it now stands) rather than written over theirs.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_card`](mcp.md#update_card) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides it from the board (reversible); false brings it back. |
| `assignees` | array of integer or null | no |  | The full set of user ids; replaces the current assignees. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `description` | string or null | no |  | Markdown. |
| `doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `done` | boolean or null | no |  | Marks it complete, or not. |
| `due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `label_ids` | array of integer or null | no |  | The full set of label ids; replaces the current labels. |
| `list_id` | integer or null | no |  | The column to move it to, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position within the column; lower comes first. |
| `price` | string or null | no |  | The price shown on the card, as text; an empty string clears it. |
| `rank` | number or null | no |  | The stored rank value itself (the rank from get_card), for putting a card back exactly as it was; rank_place is the usual way. Needs edit rights on the board. |
| `rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `rev` | integer or null | no |  | The rev from get_card / get_board. Pass it when changing title or description: if someone edited the card since, the change is refused rather than written over theirs. |
| `scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). true clears doing and reopens a done card. At most one of doing, scheduled and done. |
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string or null | no |  |  |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/checklist

Add a checklist item

Add an item to the end of a card's checklist. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`add_checklist_item`](mcp.md#add_checklist_item) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `text` | string | yes |  |  |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}

Tick or edit a checklist item

Tick an item off (done: true) or back on, change its text, or move it. Item ids are in the card's checklist. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_checklist_item`](mcp.md#update_checklist_item) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `item_id` | path | integer | yes |  | A checklist item's id, from get_card. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `done` | boolean or null | no |  | true ticks it off, false back on. |
| `position` | number or null | no |  | Its place in the checklist; lower comes first. |
| `text` | string or null | no |  |  |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}/delete

Delete a checklist item

Remove an item from a card's checklist, for good. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_checklist_item`](mcp.md#delete_checklist_item) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `item_id` | path | integer | yes |  | A checklist item's id, from get_card. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/copy

Copy a card to a board

Copy a card onto any board (its own included) with its content, comments and picture, optionally kept in step with the original (mirror). Needs edit rights where the copy goes, and on the card for a mirror. Answers the copy, with card_path and board_title.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`copy_card`](mcp.md#copy_card) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `list_id` | integer | yes |  | The column on that board it goes into, from get_board (board.lists). |
| `mirror` | string | no | one of ``, `one_way`, `two_way` | Copy only. Empty for a plain copy; one_way keeps the copy following the original (the copy's synced fields become read-only); two_way keeps both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments. |
| `position` | number or null | no |  | An exact place in the column, for putting a card back where it was; wins over top. |
| `to_board` | string or integer | yes |  | The board it goes to: id or URL slug, from list_boards. |
| `top` | boolean | no |  | Put it at the top of the column instead of the end. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/delete

Delete a card

Move a card to the board's trash. Not permanent: restore_card brings it back, until purge_card or empty_trash removes it for good. Needs edit rights on its column.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_card`](mcp.md#delete_card) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/boards/{board_id}/cards/{card_id}/image

A card's picture

The picture on a card, as the image file itself.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/boards/{board_id}/cards/{card_id}/image

Put a picture on a card

Store a picture on a card (base64 in data), replacing any it had, and answer with the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `data` | string | yes |  | The picture's bytes, base64: a PNG, JPEG, GIF or WebP. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/image/delete

Take a card's picture off

Remove the picture from a card and answer with the card. The card itself is not touched.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`remove_card_image`](mcp.md#remove_card_image) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### GET /api/boards/{board_id}/cards/{card_id}/links

List a card's mirrors

The cards this one is kept in step with: each link's id, mode (one_way / two_way), direction (out: this card's changes flow there; in: this card follows that one; both), and the other card, named only when you can open it.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_card_links`](mcp.md#list_card_links) |
| Errors | 400, 401, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `links` | array of object | yes |  |  |
| `links[].board_id` | integer | yes |  |  |
| `links[].board_title` | string | yes |  |  |
| `links[].card_id` | integer | yes |  |  |
| `links[].card_path` | string | yes |  |  |
| `links[].card_title` | string | yes |  |  |
| `links[].direction` | string | yes |  |  |
| `links[].id` | integer | yes |  |  |
| `links[].list_title` | string | yes |  |  |
| `links[].mode` | string | yes |  |  |

### POST /api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete

Stop mirroring a card

End one of a card's mirror links (link_id from list_card_links). Both cards stay as they are. Needs edit rights on the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`unlink_card`](mcp.md#unlink_card) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `link_id` | path | integer | yes |  | A mirror link's id, from list_card_links. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/boards/{board_id}/cards/{card_id}/product-image

A card's product photo

The photo of the product looked up for a card, as the image file itself.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | None, HTTP only: A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/boards/{board_id}/cards/{card_id}/product/delete

Take a card's product off

Remove the product (name, price, photo) looked up for a card. The link in its text stays. Answers with the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`remove_product`](mcp.md#remove_product) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/product/refresh

Look a card's product up again

Read the shop link in a card's title or description (or the one it was last looked up from) again now, and store the product's name, price and photo on the card. Works for any shop's link, shopping column or not. Answers with the card; product.status is failed when the page could not be read.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`refresh_product`](mcp.md#refresh_product) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/send

Send a card to another board

Move a card to another board, whole: it keeps its link, comments and picture. Labels are matched by name there (made if missing), assignees kept only if they are on that board, and per-person shares dropped. Needs edit rights on both boards. Answers the card where it landed, with card_path and board_title.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`send_card`](mcp.md#send_card) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `list_id` | integer | yes |  | The column on that board it goes into, from get_board (board.lists). |
| `mirror` | string | no | one of ``, `one_way`, `two_way` | Copy only. Empty for a plain copy; one_way keeps the copy following the original (the copy's synced fields become read-only); two_way keeps both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments. |
| `position` | number or null | no |  | An exact place in the column, for putting a card back where it was; wins over top. |
| `to_board` | string or integer | yes |  | The board it goes to: id or URL slug, from list_boards. |
| `top` | boolean | no |  | Put it at the top of the column instead of the end. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/lists/{list_id}/archive-cards

Archive a column's cards

Archive a column's open cards, all of them or the ones named in card_ids: clearing out a Done column, say. Archived cards are hidden, not deleted (update_card with archived: false brings one back). Needs edit rights on that column.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`archive_list_cards`](mcp.md#archive_list_cards) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `card_ids` | array of integer | no |  | The cards to archive; leave out to archive every open card in the column. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | array of integer | yes |  | The ids of the cards archived. |

### POST /api/bulk/boards/cards

Change a card, for many at once

Runs POST /api/boards/{board_id}/cards/{card_id} once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`update_card_bulk`](mcp.md#update_card_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].archived` | boolean or null | no |  | true hides it from the board (reversible); false brings it back. |
| `items[].assignees` | array of integer or null | no |  | The full set of user ids; replaces the current assignees. |
| `items[].board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `items[].card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `items[].day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `items[].description` | string or null | no |  | Markdown. |
| `items[].doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `items[].done` | boolean or null | no |  | Marks it complete, or not. |
| `items[].due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `items[].label_ids` | array of integer or null | no |  | The full set of label ids; replaces the current labels. |
| `items[].list_id` | integer or null | no |  | The column to move it to, from get_board (board.lists). |
| `items[].position` | number or null | no |  | Sort position within the column; lower comes first. |
| `items[].price` | string or null | no |  | The price shown on the card, as text; an empty string clears it. |
| `items[].rank` | number or null | no |  | The stored rank value itself (the rank from get_card), for putting a card back exactly as it was; rank_place is the usual way. Needs edit rights on the board. |
| `items[].rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `items[].rev` | integer or null | no |  | The rev from get_card / get_board. Pass it when changing title or description: if someone edited the card since, the change is refused rather than written over theirs. |
| `items[].scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). true clears doing and reopens a done card. At most one of doing, scheduled and done. |
| `items[].start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `items[].title` | string or null | no |  |  |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |
| `shared.archived` | boolean or null | no |  | true hides it from the board (reversible); false brings it back. |
| `shared.assignees` | array of integer or null | no |  | The full set of user ids; replaces the current assignees. |
| `shared.day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `shared.description` | string or null | no |  | Markdown. |
| `shared.doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `shared.done` | boolean or null | no |  | Marks it complete, or not. |
| `shared.due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `shared.label_ids` | array of integer or null | no |  | The full set of label ids; replaces the current labels. |
| `shared.list_id` | integer or null | no |  | The column to move it to, from get_board (board.lists). |
| `shared.position` | number or null | no |  | Sort position within the column; lower comes first. |
| `shared.price` | string or null | no |  | The price shown on the card, as text; an empty string clears it. |
| `shared.rank` | number or null | no |  | The stored rank value itself (the rank from get_card), for putting a card back exactly as it was; rank_place is the usual way. Needs edit rights on the board. |
| `shared.rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `shared.rev` | integer or null | no |  | The rev from get_card / get_board. Pass it when changing title or description: if someone edited the card since, the change is refused rather than written over theirs. |
| `shared.scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). true clears doing and reopens a done card. At most one of doing, scheduled and done. |
| `shared.start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `shared.title` | string or null | no |  |  |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### POST /api/bulk/boards/cards/delete

Delete a card, for many at once

Runs POST /api/boards/{board_id}/cards/{card_id}/delete once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_card_bulk`](mcp.md#delete_card_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `items[].card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

### GET /api/cards

Find cards

Search the cards you can see, across all your boards or one (board_id). query matches title and description (case-insensitive). Done and archived cards are left out unless asked for. Returns compact rows, soonest due first, with total and whether the list was cut at limit; use get_card for the full card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`find_cards`](mcp.md#find_cards) |
| Errors | 400, 401, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `query` | query | string | no |  | Matches title and description, in any case. |
| `board` | query | string or integer | no |  | Search one board only: its id or URL slug, from list_boards. |
| `assigned_to_me` | query | boolean | no |  | Only cards assigned to you. |
| `due_before` | query | string | no |  | Only cards due before this: "2026-10-30" or an RFC 3339 timestamp. |
| `include_done` | query | boolean | no |  | Include done cards. |
| `include_archived` | query | boolean | no |  | Include archived cards. |
| `limit` | query | integer | no | 1 to 200; default `50` | At most this many rows (default 50). |

**Reply 200**: Success

An empty object.

## Comments

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/boards/{board_id}/cards/{card_id}/comments`](#get-apiboardsboard_idcardscard_idcomments) | Read a card's comments | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/comments`](#post-apiboardsboard_idcardscard_idcomments) | Comment on a card | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/comments/{comment_id}`](#post-apiboardsboard_idcardscard_idcommentscomment_id) | Edit a comment | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/comments/{comment_id}/remove`](#post-apiboardsboard_idcardscard_idcommentscomment_idremove) | Remove a comment | browser session or API key, anyone signed in |

### GET /api/boards/{board_id}/cards/{card_id}/comments

Read a card's comments

A card's comments, oldest first: who wrote each (by: a member's name, guest:Name for a guest, guest: alone for an anonymous one) and when, and whether you may post one (can_comment).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_comments`](mcp.md#list_comments) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `can_comment` | boolean | yes |  | Whether the caller may post one. |
| `comments` | array of object | yes |  | Oldest first. |

### POST /api/boards/{board_id}/cards/{card_id}/comments

Comment on a card

Post a comment on a card, signed with your name. Markdown, up to 4000 characters. Needs edit rights on the card, or a card shared with you to comment on.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`add_comment`](mcp.md#add_comment) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}

Edit a comment

Change the text of a comment you wrote. Comment ids are in list_comments.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`edit_comment`](mcp.md#edit_comment) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | path | integer | yes |  | A comment's id, from list_comments. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters; replaces the comment's text. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}/remove

Remove a comment

Delete a comment for good: one you wrote, or anyone's on a board you manage. This cannot be undone.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`remove_comment`](mcp.md#remove_comment) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | path | integer | yes |  | A comment's id, from list_comments. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

## Folders

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/boards/{board_id}/folder`](#post-apiboardsboard_idfolder) | File a board in a folder | browser session or API key, anyone signed in |
| POST | [`/api/folders`](#post-apifolders) | Make a folder | browser session or API key, anyone signed in |
| POST | [`/api/folders/order`](#post-apifoldersorder) | Order your folders | browser session or API key, anyone signed in |
| POST | [`/api/folders/{folder_id}`](#post-apifoldersfolder_id) | Rename a folder | browser session or API key, anyone signed in |
| POST | [`/api/folders/{folder_id}/delete`](#post-apifoldersfolder_iddelete) | Delete a folder | browser session or API key, anyone signed in |
| POST | [`/api/folders/{folder_id}/parent`](#post-apifoldersfolder_idparent) | Move a folder | browser session or API key, anyone signed in |

### POST /api/boards/{board_id}/folder

File a board in a folder

File one of your boards in one of your folders on your home page, or take it out (folder_id 0).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_board_folder`](mcp.md#set_board_folder) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `folder_id` | integer | yes |  | The folder to file it in; 0: not in a folder. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/folders

Make a folder

Make a folder on your own home page, at the top or inside another of yours (parent_id).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_folder`](mcp.md#create_folder) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `name` | string | yes |  |  |
| `parent_id` | integer | no |  | The folder to make it in; 0 or absent: at the top. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `id` | integer | yes |  |  |
| `name` | string | yes |  |  |
| `parent_id` | integer | yes |  |  |
| `position` | number | yes |  |  |

### POST /api/folders/order

Order your folders

Put your folders in this order, first to last.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`order_folders`](mcp.md#order_folders) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `ids` | array of integer | yes |  | Your folder ids, first to last. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/folders/{folder_id}

Rename a folder

Rename one of your home page folders.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`rename_folder`](mcp.md#rename_folder) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `folder_id` | path | integer | yes |  | A folder's id, from list_boards (folders). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `name` | string | yes |  |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `id` | integer | yes |  |  |
| `name` | string | yes |  |  |
| `parent_id` | integer | yes |  |  |
| `position` | number | yes |  |  |

### POST /api/folders/{folder_id}/delete

Delete a folder

Remove one of your folders. Boards in it are not touched: they go back to the top of your home page.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_folder`](mcp.md#delete_folder) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `folder_id` | path | integer | yes |  | A folder's id, from list_boards (folders). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/folders/{folder_id}/parent

Move a folder

Put one of your folders inside another of yours, or back at the top (parent_id 0).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`move_folder`](mcp.md#move_folder) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `folder_id` | path | integer | yes |  | A folder's id, from list_boards (folders). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `parent_id` | integer | yes |  | The folder to put it in; 0: at the top. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `id` | integer | yes |  |  |
| `name` | string | yes |  |  |
| `parent_id` | integer | yes |  |  |
| `position` | number | yes |  |  |

## Google Calendar

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/boards/{board_id}/my-view/google`](#post-apiboardsboard_idmy-viewgoogle) | Sync a board to your Google Calendar | browser session or API key, anyone signed in |
| GET | [`/api/google`](#get-apigoogle) | Your Google Calendar connection | browser session or API key, anyone signed in |
| POST | [`/api/google`](#post-apigoogle) | Choose your Google calendar | browser session or API key, anyone signed in |
| GET | [`/api/google/calendars`](#get-apigooglecalendars) | Your Google calendars | browser session or API key, anyone signed in |
| GET | [`/api/google/callback`](#get-apigooglecallback) | Google's answer to a connection | anyone |
| GET | [`/api/google/connect`](#get-apigoogleconnect) | Connect your Google Calendar | browser session only, anyone signed in |
| POST | [`/api/google/disconnect`](#post-apigoogledisconnect) | Disconnect your Google Calendar | browser session or API key, anyone signed in |

### POST /api/boards/{board_id}/my-view/google

Sync a board to your Google Calendar

Turns syncing this board's dated cards with your Google Calendar on or off, for you alone. Off deletes the events it made for this board. 404 not_connected: connect Google first.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_my_board_google_sync`](mcp.md#set_my_board_google_sync) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `sync` | boolean | yes |  | true copies this board's dated cards to your Google Calendar and back; false stops, and takes its events off your calendar. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `sync` | boolean | yes |  |  |

### GET /api/google

Your Google Calendar connection

Whether Google Calendar sync is available on this server, whether your account is connected (and which Google account and calendar), the boards you sync, and how the last sync went. Connecting happens in a browser, in Settings: Google asks you there.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_google_calendar`](mcp.md#get_google_calendar) |
| Errors | 401, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `available` | boolean | yes |  | Whether this server is set up for Google Calendar at all. |
| `boards` | array of integer | yes |  | Ids of the boards you sync. |
| `calendar_id` | string | no |  | The calendar cards go to; primary is the account's main one. |
| `connected` | boolean | yes |  |  |
| `email` | string | no |  | The connected Google account. |
| `last_sync` | string (date-time) or null | no |  |  |
| `problem` | string | no |  | Why the last sync failed: revoked (connect again) or failed (it will try again). |

### POST /api/google

Choose your Google calendar

Which calendar your synced cards go on. Events already made move to it on the next sync. 404 google_off or not_connected.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_google_calendar`](mcp.md#set_google_calendar) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `calendar_id` | string | yes | 1 to 300 characters | A calendar id from list_google_calendars; primary is the account's main calendar. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `available` | boolean | yes |  | Whether this server is set up for Google Calendar at all. |
| `boards` | array of integer | yes |  | Ids of the boards you sync. |
| `calendar_id` | string | no |  | The calendar cards go to; primary is the account's main one. |
| `connected` | boolean | yes |  |  |
| `email` | string | no |  | The connected Google account. |
| `last_sync` | string (date-time) or null | no |  |  |
| `problem` | string | no |  | Why the last sync failed: revoked (connect again) or failed (it will try again). |

### GET /api/google/calendars

Your Google calendars

The calendars in your connected Google account that you can add events to. 404 google_off or not_connected; 409 google_unreachable: Google did not answer, or refused (connect again).

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_google_calendars`](mcp.md#list_google_calendars) |
| Errors | 401, 404, 409, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `calendars` | array of object | yes |  |  |
| `calendars[].id` | string | yes |  |  |
| `calendars[].primary` | boolean | no |  |  |
| `calendars[].summary` | string | yes |  |  |

### GET /api/google/callback

Google's answer to a connection

Where Google sends you back after you answer. Takes the answer only in the browser that began connecting, within ten minutes, once; then redirects to the Google Calendar screen.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `code` | query | string | no |  |  |
| `state` | query | string | no |  |  |
| `error` | query | string | no |  |  |

**Reply 200**: Success

### GET /api/google/connect

Connect your Google Calendar

Redirects to Google, which asks you to let this server manage events on your calendars. 404 google_off: the server has no Google OAuth client set up.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 401, 403, 404, 429, 503 |

**Reply 200**: Success

### POST /api/google/disconnect

Disconnect your Google Calendar

Stops syncing and forgets your Google account here (and asks Google to revoke the access). remove_events also deletes the events it made; otherwise they stay on your calendar, no longer synced.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`disconnect_google_calendar`](mcp.md#disconnect_google_calendar) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `remove_events` | boolean | no |  | true also deletes the events this server made on your calendar; otherwise they stay as they are. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

## Notifications

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/boards/{board_id}/cards/{card_id}/subscription`](#get-apiboardsboard_idcardscard_idsubscription) | Your notifications for a card | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/subscription`](#post-apiboardsboard_idcardscard_idsubscription) | Set your notifications for a card | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/subscription`](#get-apiboardsboard_idsubscription) | Your notifications for a board | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/subscription`](#post-apiboardsboard_idsubscription) | Set your notifications for a board | browser session or API key, anyone signed in |
| GET | [`/api/notify/devices`](#get-apinotifydevices) | Your browsers that take notifications | browser session only, anyone signed in |
| POST | [`/api/notify/devices`](#post-apinotifydevices) | Add or check in a browser for notifications | browser session only, anyone signed in |
| POST | [`/api/notify/devices/{device_id}/delete`](#post-apinotifydevicesdevice_iddelete) | Remove a browser from notifications | browser session only, anyone signed in |
| GET | [`/api/notify/prefs`](#get-apinotifyprefs) | Your notification settings | browser session or API key, anyone signed in |
| POST | [`/api/notify/prefs`](#post-apinotifyprefs) | Save your notification settings | browser session or API key, anyone signed in |
| POST | [`/api/notify/test`](#post-apinotifytest) | Send yourself a test notification | browser session or API key, anyone signed in |

### GET /api/boards/{board_id}/cards/{card_id}/subscription

Your notifications for a card

Which of the card's events reach you and where that comes from (source: your defaults, the board's override or this card's), with whether push is on and whether anything would reach you.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_card_subscription`](mcp.md#get_card_subscription) |
| Errors | 400, 401, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/cards/{card_id}/subscription

Set your notifications for a card

Choose which of the card's events reach you, or clear your choice (events null) so the board's or your defaults apply again. Answers as get_card_subscription does.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_card_subscription`](mcp.md#set_card_subscription) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `events` | object or null | yes |  | Which events to be told about; null or absent clears the override, so your defaults apply again. |
| `events.activity` | boolean | no |  |  |
| `events.assigned` | boolean | yes |  |  |
| `events.changes` | boolean | yes |  |  |
| `events.due` | boolean | yes |  |  |

**Reply 200**: Success

An empty object.

### GET /api/boards/{board_id}/subscription

Your notifications for a board

Which of the board's events reach you (due dates, being assigned, changes, activity) and where that comes from (source: your defaults or this board's override), with whether push is on and whether anything would reach you.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_board_subscription`](mcp.md#get_board_subscription) |
| Errors | 400, 401, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/subscription

Set your notifications for a board

Choose which of the board's events reach you, or clear your choice (events null) so your defaults apply again. Answers as get_board_subscription does.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_board_subscription`](mcp.md#set_board_subscription) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `events` | object or null | yes |  | Which events to be told about; null or absent clears the override, so your defaults apply again. |
| `events.activity` | boolean | no |  |  |
| `events.assigned` | boolean | yes |  |  |
| `events.changes` | boolean | yes |  |  |
| `events.due` | boolean | yes |  |  |

**Reply 200**: Success

An empty object.

### GET /api/notify/devices

Your browsers that take notifications

The browsers your notifications go to (id, label, when added and last reached), and the server's web push public key (empty when web push is off).

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 401, 403, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `devices` | array of object | yes |  |  |
| `public_key` | string | yes |  |  |

### POST /api/notify/devices

Add or check in a browser for notifications

Save this browser's web push subscription (endpoint and keys) so notifications reach it, or check it in again. push_disabled: web push is off; push_endpoint_refused: not a push service the server sends to; push_too_many_devices: remove one first.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `auth` | string | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `endpoint` | string | yes |  |  |
| `label` | string | no |  |  |
| `p256dh` | string | yes |  |  |

**Reply 200**: Success

An empty object.

### POST /api/notify/devices/{device_id}/delete

Remove a browser from notifications

Stop notifications going to one of your browsers.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `device_id` | path | string | yes | at least 1 characters |  |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/notify/prefs

Your notification settings

How and when you are told about things: Pushover (the key only masked) or this browser's push, timezone, reminder lead times, the hour for all-day reminders, the events you hear about as an assignee or watcher, whether shares with you are announced, and whether it is all paused. Also whether the server can send push at all and whether anything would reach you.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`get_notify_prefs`](mcp.md#get_notify_prefs) |
| Errors | 401, 429, 503 |

**Reply 200**: Success

An empty object.

### POST /api/notify/prefs

Save your notification settings

Replace your notification settings (display settings are kept). An empty pushover_user_key keeps the saved one; "-" removes it. notify_via left out stays as it is. Answers the settings as get_notify_prefs does.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`set_notify_prefs`](mcp.md#set_notify_prefs) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `all_day_reminder_hour` | integer | no | 0 to 23 | The hour a card due on a day (no time) is reminded about. |
| `assignee_defaults` | object | yes |  | What to be told about cards you are assigned to. |
| `assignee_defaults.activity` | boolean | no |  |  |
| `assignee_defaults.assigned` | boolean | yes |  |  |
| `assignee_defaults.changes` | boolean | yes |  |  |
| `assignee_defaults.due` | boolean | yes |  |  |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `notify_via` | string | no |  | Which ways of reaching you are used. Left out, it stays as it is. |
| `paused` | boolean | no |  |  |
| `pushover_device` | string | no |  |  |
| `pushover_user_key` | string | no |  | Empty or absent keeps the saved key; "-" removes it. |
| `reminder_lead_minutes` | array of integer | no |  | How long before a due time to be reminded, in minutes. |
| `shared_with_me` | boolean | no |  |  |
| `timezone` | string | no |  | An IANA zone such as Europe/Paris, for when reminders fire. |
| `watch_defaults` | object | yes |  | What to be told about every other card you can see. |
| `watch_defaults.activity` | boolean | no |  |  |
| `watch_defaults.assigned` | boolean | yes |  |  |
| `watch_defaults.changes` | boolean | yes |  |  |
| `watch_defaults.due` | boolean | yes |  |  |

**Reply 200**: Success

An empty object.

### POST /api/notify/test

Send yourself a test notification

Send yourself a test notification the way your settings say. push_disabled: the server sends none; push_no_key: nothing set up to reach you; push_rejected: Pushover said no (detail has its words); push_unreachable: the push service could not be reached.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`send_test_notification`](mcp.md#send_test_notification) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

## Shared

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/shared`](#get-apishared) | What a share link opens | anyone |
| GET | [`/api/shared/{board_id}`](#get-apisharedboard_id) | Read a board (share link) | anyone |
| POST | [`/api/shared/{board_id}`](#post-apisharedboard_id) | Change a board's settings (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards`](#post-apisharedboard_idcards) | Add a card (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}`](#post-apisharedboard_idcardscard_id) | Change a card (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/checklist`](#post-apisharedboard_idcardscard_idchecklist) | Add a checklist item (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/checklist/{item_id}`](#post-apisharedboard_idcardscard_idchecklistitem_id) | Tick or edit a checklist item (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/checklist/{item_id}/delete`](#post-apisharedboard_idcardscard_idchecklistitem_iddelete) | Delete a checklist item (share link) | anyone |
| GET | [`/api/shared/{board_id}/cards/{card_id}/comments`](#get-apisharedboard_idcardscard_idcomments) | Read a card's comments (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/comments`](#post-apisharedboard_idcardscard_idcomments) | Comment on a card (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/comments/{comment_id}`](#post-apisharedboard_idcardscard_idcommentscomment_id) | Edit a comment (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/comments/{comment_id}/remove`](#post-apisharedboard_idcardscard_idcommentscomment_idremove) | Remove a comment (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/delete`](#post-apisharedboard_idcardscard_iddelete) | Delete a card (share link) | anyone |
| GET | [`/api/shared/{board_id}/cards/{card_id}/image`](#get-apisharedboard_idcardscard_idimage) | A card's picture (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/image`](#post-apisharedboard_idcardscard_idimage) | Put a picture on a card (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/image/delete`](#post-apisharedboard_idcardscard_idimagedelete) | Take a card's picture off (share link) | anyone |
| GET | [`/api/shared/{board_id}/cards/{card_id}/product-image`](#get-apisharedboard_idcardscard_idproduct-image) | A card's product photo (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/product/delete`](#post-apisharedboard_idcardscard_idproductdelete) | Take a card's product off (share link) | anyone |
| POST | [`/api/shared/{board_id}/cards/{card_id}/product/refresh`](#post-apisharedboard_idcardscard_idproductrefresh) | Look a card's product up again (share link) | anyone |
| POST | [`/api/shared/{board_id}/carry-over`](#post-apisharedboard_idcarry-over) | Carry unfinished cards over to today (share link) | anyone |
| GET | [`/api/shared/{board_id}/export`](#get-apisharedboard_idexport) | Download a board as a spreadsheet (share link) | anyone |
| POST | [`/api/shared/{board_id}/labels`](#post-apisharedboard_idlabels) | Add a label (share link) | anyone |
| POST | [`/api/shared/{board_id}/labels/{label_id}`](#post-apisharedboard_idlabelslabel_id) | Change a label (share link) | anyone |
| POST | [`/api/shared/{board_id}/labels/{label_id}/delete`](#post-apisharedboard_idlabelslabel_iddelete) | Delete a label (share link) | anyone |
| POST | [`/api/shared/{board_id}/lists`](#post-apisharedboard_idlists) | Add a column (share link) | anyone |
| POST | [`/api/shared/{board_id}/lists/{list_id}`](#post-apisharedboard_idlistslist_id) | Rename, move or archive a column (share link) | anyone |
| POST | [`/api/shared/{board_id}/lists/{list_id}/archive-cards`](#post-apisharedboard_idlistslist_idarchive-cards) | Archive a column's cards (share link) | anyone |
| POST | [`/api/shared/{board_id}/lists/{list_id}/delete`](#post-apisharedboard_idlistslist_iddelete) | Delete a column (share link) | anyone |
| GET | [`/api/shared/{board_id}/lists/{list_id}/export`](#get-apisharedboard_idlistslist_idexport) | Download a column (share link) | anyone |
| POST | [`/api/shared/{board_id}/lists/{list_id}/import`](#post-apisharedboard_idlistslist_idimport) | Add cards to a column from a file (share link) | anyone |
| GET | [`/api/shared/{board_id}/stamp`](#get-apisharedboard_idstamp) | A board's revision (share link) | anyone |
| GET | [`/api/shared/{board_id}/trash`](#get-apisharedboard_idtrash) | List a board's trash (share link) | anyone |
| POST | [`/api/shared/{board_id}/trash/empty`](#post-apisharedboard_idtrashempty) | Empty a board's trash (share link) | anyone |
| POST | [`/api/shared/{board_id}/trash/{card_id}/delete`](#post-apisharedboard_idtrashcard_iddelete) | Delete a card for good (share link) | anyone |
| POST | [`/api/shared/{board_id}/trash/{card_id}/restore`](#post-apisharedboard_idtrashcard_idrestore) | Restore a card from the trash (share link) | anyone |

### GET /api/shared

What a share link opens

For a guest holding a share link in X-Share-Token: the board (and its title), column or card it opens, its mode and label, and whether guests may comment. A quick card link answers board_id, card_id, mode and quick only.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 404, 429, 503 |

**Reply 200**: Success

An empty object.

### GET /api/shared/{board_id}

Read a board (share link)

getBoard for a guest holding a share link in X-Share-Token: what the link shows.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `include_archived` | query | boolean or null | no |  | Include archived cards. The page gets them unless this is false; an agent's get_board leaves them out unless it is true. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}

Change a board's settings (share link)

updateBoard for a guest holding an edit share link in X-Share-Token. Never the purpose or notes.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `add_placement` | string or null | no | one of ``, `top`, `bottom` | Where the add-card box sits in a column: top, bottom, or "" for each person's account setting. |
| `allow_anon_comments` | boolean or null | no |  | Let guests comment without giving a name. The owner's, from a browser only: refused through a key. |
| `allow_quick_share` | boolean or null | no |  | Let the board's cards be shared by quick link. The owner's, from a browser only: refused through a key. |
| `carry_over` | boolean or null | no |  | On a calendar board, move unfinished cards from past days to today. |
| `center_columns` | boolean or null | no |  | Centre the columns on wide screens. |
| `clear_theme` | boolean | no |  | true drops the board's own theme: everyone sees their own again. |
| `color` | string or null | no | one of ``, `blue`, `green`, `orange`, `red`, `purple`, `pink`, `teal`, `gray`, `yellow` | The board's colour; empty for none. |
| `default_view` | string or null | no | one of `kanban`, `todo`, `timeline` | The view the board opens in for someone who has not picked one. |
| `done_placement` | string or null | no | one of ``, `top`, `bottom` | Where finished cards go in a column: top, bottom, or "" for each person's account setting. |
| `layout` | string or null | no | one of ``, `three`, `staged`, `calendar` | "" (free: any number of columns), "three" (pinned to three columns: to do, doing, done), "staged" (each column has its own To do / Doing / Done, with an All view ranked board-wide), or "calendar" (cards filed by day; columns titled with dates are filed under those days). |
| `list_titles` | array of string | no |  | With layout "three": the titles of the columns added if the board has fewer than three open ones. |
| `notes` | string or null | no |  | Markdown: links, documents, an index. Members only. |
| `notes_rev` | integer or null | no |  | The notes_rev the notes were read at, as purpose_rev. |
| `numbering` | string or null | no | one of ``, `on`, `off` | Card numbers for everyone on the board: on, off, or "" to leave it to each person's account setting. |
| `purpose` | string or null | no |  | Markdown: what the board is for and how to work on it. Only while the Purpose section is on (409 purpose_off otherwise). Members only. |
| `purpose_enabled` | boolean or null | no |  | Switch the board's Purpose section on or off. Members only. |
| `purpose_rev` | integer or null | no |  | The purpose_rev the purpose was read at: if it moved since, 409 board_text_stale instead of writing over someone's change. |
| `shopping` | boolean or null | no |  | A shopping board: cards carry products and prices. |
| `show_rank` | boolean or null | no |  | Show each card's place in the board-wide ranking. |
| `show_updated` | boolean or null | no |  | Every card face shows when it was last changed, in its bottom-right corner. A column's own show_updated can override it. |
| `slug` | string or null | no |  | The board's URL name (the part after /b/). Empty picks one from the title. 409 slug_taken if another board has it. |
| `stage_doing` | string or null | no |  | What the Doing stage is called on this board; empty for the default. |
| `stage_done` | string or null | no |  | What the Done stage is called on this board; empty for the default. |
| `stage_scheduled` | string or null | no |  | With stage_set review: what the Scheduled stage is called; empty for the default. |
| `stage_set` | string or null | no | one of ``, `review` | On a staged or calendar board, its stages: "" for To do / Doing / Done, or "review" for Ready for review / Working / Scheduled / Resolved (cards' doing, scheduled and done flags). |
| `stage_tabs` | boolean or null | no |  | On a staged board, give every column its own To do / Doing / Done buttons. |
| `stage_todo` | string or null | no |  | What the To do stage is called on this board; empty for the default. |
| `theme` | object or null | no |  | The board's own theme, shown to everyone on it. |
| `theme.accent` | string | yes |  |  |
| `theme.background` | string | no |  |  |
| `theme.button` | object or null | no |  |  |
| `theme.button.case` | string | no | one of `normal`, `upper` |  |
| `theme.button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `theme.button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `theme.button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `theme.card` | string | no |  |  |
| `theme.column` | string | no |  |  |
| `theme.dark` | object or null | no |  |  |
| `theme.dark.accent` | string | yes |  | The main color: what you can press. |
| `theme.dark.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.light` | object or null | no |  |  |
| `theme.light.accent` | string | yes |  | The main color: what you can press. |
| `theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.mode` | string | yes |  |  |
| `theme.overrides` | map of string | no |  |  |
| `theme.second` | string | no |  |  |
| `theme.surface` | string | no |  |  |
| `title` | string or null | no | at most 120 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `notes_rev` | integer | yes |  |  |
| `purpose_rev` | integer | yes |  |  |
| `rev` | integer | yes |  |  |
| `slug` | string | yes |  |  |

### POST /api/shared/{board_id}/cards

Add a card (share link)

createCard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `assignees` | array of integer or null | no |  | User ids, from get_board's members or list_team. |
| `day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `description` | string or null | no |  | Markdown. |
| `doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `done` | boolean or null | no |  | true makes it complete from the start. |
| `due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `label_ids` | array of integer or null | no |  | Label ids, from get_board (board.labels). |
| `list_id` | integer | yes |  | The column's id, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position within the column; lower comes first. Leave out to put it at the end. |
| `rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). At most one of doing, scheduled and done. |
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string | yes |  |  |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}

Change a card (share link)

patchCard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides it from the board (reversible); false brings it back. |
| `assignees` | array of integer or null | no |  | The full set of user ids; replaces the current assignees. |
| `day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `description` | string or null | no |  | Markdown. |
| `doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `done` | boolean or null | no |  | Marks it complete, or not. |
| `due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `label_ids` | array of integer or null | no |  | The full set of label ids; replaces the current labels. |
| `list_id` | integer or null | no |  | The column to move it to, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position within the column; lower comes first. |
| `price` | string or null | no |  | The price shown on the card, as text; an empty string clears it. |
| `rank` | number or null | no |  | The stored rank value itself (the rank from get_card), for putting a card back exactly as it was; rank_place is the usual way. Needs edit rights on the board. |
| `rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `rev` | integer or null | no |  | The rev from get_card / get_board. Pass it when changing title or description: if someone edited the card since, the change is refused rather than written over theirs. |
| `scheduled` | boolean or null | no |  | On a staged board with stage_set review, the Scheduled stage (after Working, i.e. doing). true clears doing and reopens a done card. At most one of doing, scheduled and done. |
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string or null | no |  |  |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/checklist

Add a checklist item (share link)

addChecklistItem for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `text` | string | yes |  |  |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/checklist/{item_id}

Tick or edit a checklist item (share link)

updateChecklistItem for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `item_id` | path | integer | yes |  | A checklist item's id, from get_card. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `done` | boolean or null | no |  | true ticks it off, false back on. |
| `position` | number or null | no |  | Its place in the checklist; lower comes first. |
| `text` | string or null | no |  |  |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/checklist/{item_id}/delete

Delete a checklist item (share link)

deleteChecklistItem for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `item_id` | path | integer | yes |  | A checklist item's id, from get_card. |

**Reply 200**: Success

An empty object.

### GET /api/shared/{board_id}/cards/{card_id}/comments

Read a card's comments (share link)

listComments for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `can_comment` | boolean | yes |  | Whether the caller may post one. |
| `comments` | array of object | yes |  | Oldest first. |

### POST /api/shared/{board_id}/cards/{card_id}/comments

Comment on a card (share link)

addComment for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/comments/{comment_id}

Edit a comment (share link)

editComment for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | path | integer | yes |  | A comment's id, from list_comments. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters; replaces the comment's text. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/comments/{comment_id}/remove

Remove a comment (share link)

removeComment for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | path | integer | yes |  | A comment's id, from list_comments. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/shared/{board_id}/cards/{card_id}/delete

Delete a card (share link)

deleteCard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/shared/{board_id}/cards/{card_id}/image

A card's picture (share link)

cardImage for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/shared/{board_id}/cards/{card_id}/image

Put a picture on a card (share link)

setCardImage for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `data` | string | yes |  | The picture's bytes, base64: a PNG, JPEG, GIF or WebP. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/image/delete

Take a card's picture off (share link)

removeCardImage for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

### GET /api/shared/{board_id}/cards/{card_id}/product-image

A card's product photo (share link)

productImage for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/shared/{board_id}/cards/{card_id}/product/delete

Take a card's product off (share link)

removeProduct for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/cards/{card_id}/product/refresh

Look a card's product up again (share link)

refreshProduct for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/carry-over

Carry unfinished cards over to today (share link)

carryOver for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `today` | string | yes |  | Your today, as 2026-11-07: within a day of the server's. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `moved` | integer | yes |  | How many cards moved to today. |

### GET /api/shared/{board_id}/export

Download a board as a spreadsheet (share link)

exportBoard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `format` | query | string | no |  | xlsx or csv. |
| `archived` | query | string | no |  | 1 adds archived cards and columns, with an Archived column. |
| `tz` | query | string | no |  | The timezone timed dates are written in (an IANA name); defaults to your own, else UTC. |

**Reply 200**: Success

Raw bytes, as `application/octet-stream`.

### POST /api/shared/{board_id}/labels

Add a label (share link)

createLabel for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string | yes |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `name` | string or null | no | at most 40 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string | yes |  |  |
| `id` | integer | yes |  |  |
| `name` | string | yes |  |  |

### POST /api/shared/{board_id}/labels/{label_id}

Change a label (share link)

updateLabel for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `label_id` | path | integer | yes |  | A label's id, from get_board (board.labels). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `color` | string or null | no |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `name` | string or null | no | at most 40 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/shared/{board_id}/labels/{label_id}/delete

Delete a label (share link)

deleteLabel for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `label_id` | path | integer | yes |  | A label's id, from get_board (board.labels). |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/shared/{board_id}/lists

Add a column (share link)

createList for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `position` | number or null | no |  | Sort position among the columns; lower comes first. Leave out to put it at the end. |
| `title` | string | yes | at most 120 characters |  |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean | yes |  |  |
| `id` | integer | yes |  |  |
| `position` | number | yes |  |  |
| `shopping` | boolean | yes |  |  |
| `show_updated` | string | yes |  |  |
| `stage_tabs` | string | yes |  |  |
| `stage_view` | string | yes |  |  |
| `title` | string | yes |  |  |
| `width` | integer | yes |  |  |

### POST /api/shared/{board_id}/lists/{list_id}

Rename, move or archive a column (share link)

updateList for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides the column; false brings it back. |
| `position` | number or null | no |  | Sort position among the columns; lower comes first. |
| `shopping` | boolean or null | no |  | A shopping column: its cards carry products and prices. |
| `show_updated` | string or null | no | one of ``, `on`, `off` | Whether this column's cards show when they were last changed: on, off, or empty to follow the board's show_updated. |
| `stage_tabs` | string or null | no | one of ``, `on`, `off` | On a staged board, this column's own To do / Doing / Done buttons: on, off, or empty to follow the board's setting. |
| `stage_view` | string or null | no | one of ``, `todo`, `doing`, `scheduled`, `done` | On a staged board, the stage this column shows for everyone; empty follows the navbar. |
| `title` | string or null | no | at most 120 characters |  |
| `width` | integer or null | no |  | The column's width in pixels (200-800); 0 puts it back to the standard width. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/shared/{board_id}/lists/{list_id}/archive-cards

Archive a column's cards (share link)

archiveListCards for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `card_ids` | array of integer | no |  | The cards to archive; leave out to archive every open card in the column. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | array of integer | yes |  | The ids of the cards archived. |

### POST /api/shared/{board_id}/lists/{list_id}/delete

Delete a column (share link)

deleteList for a guest holding an edit share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/shared/{board_id}/lists/{list_id}/export

Download a column (share link)

exportColumn for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Reply 200**: Success

An empty object.

### POST /api/shared/{board_id}/lists/{list_id}/import

Add cards to a column from a file (share link)

importColumn for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | path | integer | yes |  | A column's id, from get_board (board.lists). |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `cards` | array of object | yes |  |  |
| `cards[].archived` | boolean | yes |  |  |
| `cards[].assignees` | array of string | no |  |  |
| `cards[].checklist` | array of object | yes |  |  |
| `cards[].checklist[].done` | boolean | yes |  |  |
| `cards[].checklist[].text` | string | yes |  |  |
| `cards[].day` | string | no |  |  |
| `cards[].description` | string | yes |  |  |
| `cards[].doing` | boolean | no |  |  |
| `cards[].done` | boolean | yes |  |  |
| `cards[].done_at` | string (date-time) or null | yes |  |  |
| `cards[].due_all_day` | boolean | yes |  |  |
| `cards[].due_at` | string (date-time) or null | yes |  |  |
| `cards[].labels` | array of integer | yes |  |  |
| `cards[].price` | string | no |  |  |
| `cards[].rank` | number | no |  |  |
| `cards[].scheduled` | boolean | no |  |  |
| `cards[].start_at` | string (date-time) or null | yes |  |  |
| `cards[].start_timed` | boolean | yes |  |  |
| `cards[].title` | string | yes |  |  |
| `csrf_token` | string | no |  |  |
| `labels` | array of object | yes |  |  |
| `labels[].color` | string | yes |  |  |
| `labels[].name` | string | yes |  |  |
| `labels[].ref` | integer | yes |  |  |

**Reply 200**: Success

An empty object.

### GET /api/shared/{board_id}/stamp

A board's revision (share link)

getBoardStamp for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `rev` | integer | yes |  | Moves whenever anything the caller can see on the board changes. |

### GET /api/shared/{board_id}/trash

List a board's trash (share link)

listTrash for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `cards` | array of object | yes |  | The cards in the trash, each with deleted_at and deleted_by. |

### POST /api/shared/{board_id}/trash/empty

Empty a board's trash (share link)

emptyTrash for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `removed` | integer | yes |  | How many cards were deleted for good. |

### POST /api/shared/{board_id}/trash/{card_id}/delete

Delete a card for good (share link)

purgeCard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/shared/{board_id}/trash/{card_id}/restore

Restore a card from the trash (share link)

restoreCard for a guest holding a share link in X-Share-Token.

|  |  |
|---|---|
| Who may call | anyone |
| Errors | 400, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Reply 200**: Success

An empty object.

## Sharing

| Method | Path | Summary | Who may call |
|---|---|---|---|
| POST | [`/api/boards/{board_id}/cards/{card_id}/quick-share`](#post-apiboardsboard_idcardscard_idquick-share) | Get a card's public link | browser session only, anyone signed in |
| POST | [`/api/boards/{board_id}/cards/{card_id}/share-user`](#post-apiboardsboard_idcardscard_idshare-user) | Share a card with someone | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/members`](#post-apiboardsboard_idmembers) | Add, change or remove a board member | browser session or API key, anyone signed in |
| GET | [`/api/boards/{board_id}/shares`](#get-apiboardsboard_idshares) | List a board's share links | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/shares`](#post-apiboardsboard_idshares) | Make a share link | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/shares/{link_id}/delete`](#post-apiboardsboard_idshareslink_iddelete) | Delete a share link | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/shares/{link_id}/revoke`](#post-apiboardsboard_idshareslink_idrevoke) | Revoke a share link | browser session or API key, anyone signed in |

### POST /api/boards/{board_id}/cards/{card_id}/quick-share

Get a card's public link

The card's quick public link, made the first time it is asked for. For the board's owner and editors, from a browser session.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `share_path` | string | yes |  |  |

### POST /api/boards/{board_id}/cards/{card_id}/share-user

Share a card with someone

Share one card with an account, to view or to edit, without putting them on the board; mode empty stops sharing it with them. Only the board's owner may. Answers the card.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`share_card_with_user`](mcp.md#share_card_with_user) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `mode` | string | yes | one of `view`, `edit`, `` | view or edit; empty stops sharing the card with them. |
| `user_id` | integer | yes |  | The account to share the card with, from list_team. |

**Reply 200**: Success

An empty object.

### POST /api/boards/{board_id}/members

Add, change or remove a board member

Put an account on the board as an editor or a viewer, change their role, or take them off it (role empty). Only the board's owner may. The owner's own place cannot be changed.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`add_board_member`](mcp.md#add_board_member) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `role` | string | yes | one of `editor`, `viewer`, `` | editor or viewer; empty removes them from the board. |
| `user_id` | integer | yes |  | The account to add, change or remove, from list_team. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### GET /api/boards/{board_id}/shares

List a board's share links

The board's share links, live or not: id, what each opens (the board, a column or one card), mode, label, when it was made, expires, was revoked and last used, whether it is a quick link, and who made it. Never the link itself, which is shown once when it is made. Only the board's owner may.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_shares`](mcp.md#list_shares) |
| Errors | 400, 401, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

Array of object.

### POST /api/boards/{board_id}/shares

Make a share link

Make a link that opens the board, one column (list_id) or one card (card_id) to anyone holding it, to view, comment or edit, for days days (0: no end). share_path is the link, shown only this once. Only the board's owner may. From a browser session any link; from an API key only a view link to the whole board (mode view, no card_id or list_id), and only an admin-role key: anything else from a key is refused with 403.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`create_view_share`](mcp.md#create_view_share) |
| Errors | 400, 401, 403, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `card_id` | integer | no |  | A card's id, for a link to that card alone (0: not a card link). An API key cannot set it. |
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `days` | integer | no |  | How many days the link works, up to 3650 (0: no end). |
| `label` | string | no |  | A name for the link, up to 80 characters, shown in list_shares. |
| `list_id` | integer | no |  | A column's id, for a link to that column alone (0: not a column link). An API key cannot set it. |
| `mode` | string | yes | one of `view`, `comment`, `edit` | view, comment or edit. An API key may make view links only. |
| `visible_list_ids` | array of integer | no |  | A board link only: the columns (from get_board, board.lists) whose cards it shows; the rest show their titles only. Empty: every column. |

**Reply 201**: Success

An empty object.

### POST /api/boards/{board_id}/shares/{link_id}/delete

Delete a share link

Remove one of the board's share links from the list (link_id from list_shares); a live one stops working. Only the board's owner may.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`delete_share`](mcp.md#delete_share) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `link_id` | path | string | yes | at least 1 characters | A share link's id, from list_shares. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/shares/{link_id}/revoke

Revoke a share link

Stop one of the board's share links working (link_id from list_shares). It stays in the list, marked revoked. Only the board's owner may.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`revoke_share`](mcp.md#revoke_share) |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `link_id` | path | string | yes | at least 1 characters | A share link's id, from list_shares. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

## Trash

| Method | Path | Summary | Who may call |
|---|---|---|---|
| GET | [`/api/board-trash`](#get-apiboard-trash) | Your deleted boards | browser session only, anyone signed in |
| POST | [`/api/board-trash/empty`](#post-apiboard-trashempty) | Empty your board trash | browser session only, anyone signed in |
| POST | [`/api/board-trash/{board_id}/delete`](#post-apiboard-trashboard_iddelete) | Delete a board for good | browser session only, anyone signed in |
| POST | [`/api/board-trash/{board_id}/restore`](#post-apiboard-trashboard_idrestore) | Restore a deleted board | browser session only, anyone signed in |
| GET | [`/api/boards/{board_id}/trash`](#get-apiboardsboard_idtrash) | List a board's trash | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/trash/empty`](#post-apiboardsboard_idtrashempty) | Empty a board's trash | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/trash/{card_id}/delete`](#post-apiboardsboard_idtrashcard_iddelete) | Delete a card for good | browser session or API key, anyone signed in |
| POST | [`/api/boards/{board_id}/trash/{card_id}/restore`](#post-apiboardsboard_idtrashcard_idrestore) | Restore a card from the trash | browser session or API key, anyone signed in |
| POST | [`/api/bulk/boards/trash/restore`](#post-apibulkboardstrashrestore) | Restore a card from the trash, for many at once | browser session or API key, anyone signed in |

### GET /api/board-trash

Your deleted boards

The boards you deleted, which stay until you restore them or delete them for good.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 401, 403, 429, 503 |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `boards` | array of object | yes |  |  |
| `boards[].color` | string | yes |  |  |
| `boards[].deleted_at` | string (date-time) | yes |  |  |
| `boards[].id` | integer | yes |  |  |
| `boards[].title` | string | yes |  |  |

### POST /api/board-trash/empty

Empty your board trash

Delete every board in your board trash for good. This cannot be undone. Answers how many went.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `removed` | integer | yes |  |  |

### POST /api/board-trash/{board_id}/delete

Delete a board for good

Delete one of your deleted boards for good, with its cards, comments and pictures. This cannot be undone.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | integer | yes |  | The deleted board's id, from GET /api/board-trash. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/board-trash/{board_id}/restore

Restore a deleted board

Bring one of your deleted boards back, for everyone who was on it. Answers its id and slug.

|  |  |
|---|---|
| Who may call | browser session only, anyone signed in |
| Errors | 400, 401, 403, 404, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | integer | yes |  | The deleted board's id, from GET /api/board-trash. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `id` | integer | yes |  |  |
| `slug` | string | yes |  | Its URL name, the part after /b/. |

### GET /api/boards/{board_id}/trash

List a board's trash

The board's deleted cards, with when and by whom each was deleted. restore_card brings one back. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`list_trash`](mcp.md#list_trash) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `cards` | array of object | yes |  | The cards in the trash, each with deleted_at and deleted_by. |

### POST /api/boards/{board_id}/trash/empty

Empty a board's trash

Permanently delete every card in the board's trash. This cannot be undone. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`empty_trash`](mcp.md#empty_trash) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `removed` | integer | yes |  | How many cards were deleted for good. |

### POST /api/boards/{board_id}/trash/{card_id}/delete

Delete a card for good

Permanently delete one card that is in the board's trash. This cannot be undone. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`purge_card`](mcp.md#purge_card) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

### POST /api/boards/{board_id}/trash/{card_id}/restore

Restore a card from the trash

Bring a deleted card back out of the board's trash. Card ids are in list_trash. Needs edit rights on the whole board.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`restore_card`](mcp.md#restore_card) |
| Errors | 400, 401, 403, 404, 409, 429, 503 |

**Parameters**

| Name | In | Type | Required | Limits | Description |
|---|---|---|---|---|---|
| `board_id` | path | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | path | integer | yes |  | A card's id, from get_board or find_cards. |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |

**Reply 200**: Success

An empty object.

### POST /api/bulk/boards/trash/restore

Restore a card from the trash, for many at once

Runs POST /api/boards/{board_id}/trash/{card_id}/restore once per item: items holds each one's id and fields, shared the fields they have in common (an item's own win). Each runs as that route would, in order, and one failing does not stop the rest; the reply says how each went. At most 100 items.

|  |  |
|---|---|
| Who may call | browser session or API key, anyone signed in |
| MCP tool | [`restore_card_bulk`](mcp.md#restore_card_bulk) |
| Errors | 400, 401, 403, 429, 503 |

**Request body**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `csrf_token` | string | no |  | From GET /api/me. Browser sessions only. |
| `items` | array of object | yes |  | One entry per thing to change: its id, and any field that differs from shared. |
| `items[].board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `items[].card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `shared` | object | no |  | Fields every item gets, unless the item sets its own. |

**Reply 200**: Success

[`BulkReply`](#bulkreply).

## Schemas

The shapes the routes above refer to by name.

### APIKey

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board_access` | string | yes |  | Empty: every board. only/except: just, or all but, the boards listed. |
| `board_count` | integer | yes |  |  |
| `board_ids` | array of integer | no |  | The boards, in the key holder's own list only. |
| `created_at` | string (date-time) | yes |  |  |
| `created_by` | integer | no |  | The admin who made this key for its owner. Absent if the owner made it. |
| `created_by_name` | string | no |  |  |
| `display_name` | string | no |  | The key's owner. On the admin list only. |
| `expired` | boolean | yes |  |  |
| `expires_at` | string (date-time) | no |  |  |
| `id` | string | yes |  |  |
| `last_used_at` | string (date-time) | no |  |  |
| `live` | boolean | yes |  |  |
| `may_mint_keys` | boolean | yes |  | Whether this key may make keys for other accounts. Only the owner switches it on. |
| `may_sign_in` | boolean | yes |  | Whether this key may make a login link for its own account. |
| `minted_by_key` | string | yes |  | The key that made this one, or empty. Revoking that key revokes this one. |
| `name` | string | yes |  |  |
| `revoked` | boolean | yes |  |  |
| `revoked_at` | string (date-time) | no |  |  |
| `role` | string | yes | one of `admin`, `user`, `viewer` |  |
| `user_id` | integer | yes |  |  |

### BulkReply

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `done` | integer | yes |  | How many items worked. |
| `failed` | integer | yes |  | How many did not. |
| `results` | array of [`BulkResult`](#bulkresult) | yes |  | One per item, in the order they were sent. |

### BulkResult

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `error` | string | no |  | When it failed: the route's words for why. |
| `index` | integer | yes |  | The item's place in items, from 0. |
| `ok` | boolean | yes |  |  |
| `reply` | object | no |  | When it worked: the route's own reply, if it has one. |
| `request_id` | string | no |  | On a 5xx: the reference that finds this failure in the log. |
| `status` | integer | no |  | When it failed: the status the route would have answered. |

### Error

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `error` | string | yes |  |  |
| `request_id` | string | no |  | On a 5xx: the reference that finds this failure in the log. |

### IPNames

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `names` | array of object | yes |  |  |
| `names[].created_at` | string (date-time) | yes |  |  |
| `names[].name` | string | yes |  |  |
| `names[].target` | string | yes |  |  |

### InviteLink

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_by` | integer | yes |  |  |
| `created_by_name` | string | no |  |  |
| `expires_at` | string (date-time) | yes |  |  |
| `id` | string | yes |  |  |
| `label` | string | yes |  |  |
| `live` | boolean | yes |  |  |
| `max_uses` | integer | yes |  |  |
| `remaining` | integer | yes |  |  |
| `revoked` | boolean | yes |  |  |
| `role` | string | yes | one of `user`, `viewer` | The role each person who joins gets. |
| `used` | integer | yes |  |  |

### Job

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `interrupted` | boolean | yes |  |  |
| `interval_seconds` | integer | yes |  |  |
| `last_error` | string | no |  |  |
| `last_finished` | string (date-time) | no |  |  |
| `last_note` | string | no |  |  |
| `last_ok` | boolean or null | no |  |  |
| `last_started` | string (date-time) | no |  |  |
| `name` | string | yes |  |  |
| `next_due` | string (date-time) | yes |  |  |
| `running` | boolean | yes |  |  |
| `runs` | integer | yes |  |  |

### Joined

One of these:

1. An object:

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes |  |  |
| `ok` | boolean | yes |  |  |

2. An object:

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `challenge` | string | yes |  |  |
| `two_factor` | boolean | yes |  |  |

### LogDay

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `date` | string | yes |  |  |
| `entries` | array of object | yes |  |  |
| `next` | integer or null | yes |  |  |

### LogDays

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `days` | array of object | yes |  |  |
| `dropped` | integer | yes |  |  |
| `settings` | object | yes |  |  |
| `total_bytes` | integer | yes |  |  |

### LoginLink

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `expires_at` | string (date-time) or null | yes |  | null when it never expires. |
| `last_used_at` | string (date-time) or null | yes |  |  |
| `max_uses` | integer | yes |  | Sign-ins it allows. 0 when unlimited. |
| `unlimited` | boolean | yes |  |  |
| `uses` | integer | yes |  | Sign-ins so far. |

### Maintenance

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `error` | string | yes |  |  |
| `maintenance` | string | yes | one of `read_only`, `full` |  |

### Me

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `account_role` | string | yes | one of `admin`, `user`, `viewer` |  |
| `announcement` | string | yes |  | server.announcement. Empty when there is none. |
| `api_doc_url` | string | yes |  | Where to read the API reference as a page for people: config/api.md in app.repo, or /docs/api when there is no repo. |
| `api_key_id` | string | no |  | API keys only. |
| `api_key_name` | string | no |  | API keys only. |
| `api_keys_enabled` | boolean | yes |  |  |
| `auth` | string | yes |  |  |
| `authenticated` | boolean | yes |  |  |
| `capabilities` | array of string | yes |  | The optional parts of that contract this app has on now: viewer_role, account_expiry, key_minting, invite_links, key_sign_in. |
| `contract` | integer | yes |  | Which version of the portal contract this app speaks (ProjectPortal's CROSS_PROJECT.md). |
| `csrf_token` | string | no |  | Browser sessions only. |
| `description` | string | yes |  | app.description. Empty when there is none. |
| `display` | object | yes |  | How you want times and boards shown, as POST /api/account/display sets it. |
| `display_name` | string | yes |  |  |
| `has_password` | boolean | yes |  |  |
| `has_pin` | boolean | yes |  |  |
| `id` | integer | yes |  |  |
| `invite_links_enabled` | boolean | yes |  | Whether invite links are on. |
| `is_owner` | boolean | yes |  |  |
| `login_link` | [`LoginLink`](#loginlink) or null | yes |  | Your own live login link, or null. |
| `logs_enabled` | boolean | yes |  |  |
| `logs_ui_clicks` | boolean | yes |  |  |
| `maintenance` | string | yes | one of `none`, `read_only`, `full` |  |
| `may_mint_keys` | boolean | no |  | API keys only: whether this key may make keys for other accounts. |
| `mcp_doc_url` | string | yes |  | Where to read the MCP tools as a page for people: config/mcp.md in app.repo, or /docs/mcp when there is no repo. |
| `mcp_enabled` | boolean | yes |  | server.mcp: whether POST /mcp answers. |
| `mcp_url` | string | yes |  | Where to read the MCP reference: config/mcp.json in app.repo, or /api/mcp.json when there is no repo. Only with server.mcp on. |
| `openapi_url` | string | yes |  | Where to read this API's reference: config/api.json in app.repo, or /api/api.json when there is no repo. |
| `password_blocked` | boolean | yes |  |  |
| `passwords_enabled` | boolean | yes |  |  |
| `repo` | string | yes |  | app.repo without a trailing / or .git: where the source lives. Empty when it is not set. |
| `role` | string | yes | one of `admin`, `user`, `viewer` | The effective role: never higher than an API key's own. A viewer may read, and change nothing. |
| `show_about` | boolean | yes |  | app.show_about: whether the menu has About. |
| `theme_id` | string | yes |  | The theme you use. Empty: the first. |
| `themes` | array of [`Theme`](#theme) or null | yes |  | Your themes (see the Theme schema): your own list, else the app's defaults, else null for the built-in ones. |
| `themes_builtins` | integer | yes |  | Which built-in themes that list has seen (see builtins on POST /api/account/themes). |
| `themes_custom` | boolean | yes |  | Whether you have your own list rather than following the defaults. |
| `two_factor` | boolean | yes |  |  |
| `two_factor_enabled` | boolean | yes |  |  |
| `username` | string | yes |  |  |
| `version` | string | yes |  | The running build. dev when none was stamped. |

### Member

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes |  |  |
| `id` | integer | yes |  |  |

### NewAPIKey

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board_access` | string | yes |  | Empty: every board. only/except: just, or all but, the boards listed. |
| `board_count` | integer | yes |  |  |
| `board_ids` | array of integer | no |  | The boards, in the key holder's own list only. |
| `created_at` | string (date-time) | yes |  |  |
| `created_by` | integer | no |  | The admin who made this key for its owner. Absent if the owner made it. |
| `created_by_name` | string | no |  |  |
| `credential` | string | yes |  | Shown in this response only, and never recoverable. |
| `display_name` | string | no |  | The key's owner. On the admin list only. |
| `expired` | boolean | yes |  |  |
| `expires_at` | string (date-time) | no |  |  |
| `id` | string | yes |  |  |
| `last_used_at` | string (date-time) | no |  |  |
| `live` | boolean | yes |  |  |
| `may_mint_keys` | boolean | yes |  | Whether this key may make keys for other accounts. Only the owner switches it on. |
| `may_sign_in` | boolean | yes |  | Whether this key may make a login link for its own account. |
| `minted_by_key` | string | yes |  | The key that made this one, or empty. Revoking that key revokes this one. |
| `name` | string | yes |  |  |
| `revoked` | boolean | yes |  |  |
| `revoked_at` | string (date-time) | no |  |  |
| `role` | string | yes | one of `admin`, `user`, `viewer` |  |
| `user_id` | integer | yes |  |  |

### PruneResult

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `bytes_after` | integer | yes |  |  |
| `bytes_before` | integer | yes |  |  |
| `compressed` | integer | yes |  |  |
| `deleted` | integer | yes |  |  |

### Revoked

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `api_keys` | integer | yes |  |  |
| `login_links` | integer | yes |  |  |
| `passwords` | integer | yes |  |  |
| `sessions` | integer | yes |  |  |

### Session

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `current` | boolean | yes |  |  |
| `expires_at` | string (date-time) | yes |  |  |
| `id` | string | yes |  |  |
| `ip` | string | no |  |  |
| `ip_name` | string | no |  | What the owner calls this address on the Logs screen. The owner's own list only. |
| `label` | string | no |  | The name you gave this device. |
| `last_seen_at` | string (date-time) | no |  |  |
| `user_agent` | string | no |  |  |

### SettingsResult

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `applied` | array of string or null | yes |  |  |
| `changes` | array of object or null | yes |  |  |
| `restart` | array of string or null | yes |  |  |
| `scripts` | array of string or null | yes |  |  |
| `unchanged` | array of string or null | yes |  |  |
| `version` | string | yes |  |  |

### SettingsSnapshot

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `error` | object or null | yes |  |  |
| `error.at` | string (date-time) | yes |  |  |
| `error.message` | string | yes |  |  |
| `error.origin` | string | yes |  |  |
| `file` | string | yes |  |  |
| `in_container` | boolean | yes |  |  |
| `secret_file` | string | yes |  |  |
| `sections` | array of object | yes |  |  |
| `sections[].help` | string | yes |  |  |
| `sections[].keys` | array of object | yes |  |  |
| `sections[].keys[].apply` | string | yes |  |  |
| `sections[].keys[].default` | any | yes |  |  |
| `sections[].keys[].file_only` | boolean | yes |  | A path: change it in config.yaml, never here. |
| `sections[].keys[].help` | string | yes |  |  |
| `sections[].keys[].in_file` | boolean | yes |  |  |
| `sections[].keys[].key` | string | yes |  |  |
| `sections[].keys[].locked` | boolean | yes |  |  |
| `sections[].keys[].name` | string | yes |  |  |
| `sections[].keys[].pending_restart` | boolean | yes |  |  |
| `sections[].keys[].running` | any | yes |  |  |
| `sections[].keys[].type` | string | yes |  |  |
| `sections[].keys[].value` | any | yes |  |  |
| `sections[].name` | string | yes |  |  |
| `stale` | boolean | yes |  |  |
| `version` | string | yes |  | Send it back when saving: a changed file is refused with a 409. |
| `writable` | boolean | yes |  |  |

### SignIn

One of these:

1. An object:

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ok` | boolean | yes |  |  |

2. An object:

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `challenge` | string | yes |  |  |
| `two_factor` | boolean | yes |  |  |

### Theme

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `accent` | string | yes |  | The main color: what you can press. |
| `background` | string | no |  | The page. Left out: the mode's own. |
| `button` | object or null | no |  | How the theme's buttons look. A choice left out is the default (rounded, normal, raised, normal). |
| `button.case` | string | no | one of `normal`, `upper` |  |
| `button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `dark` | object or null | no |  | A light theme's colors for dark mode, kept while it is light. Switching the theme to dark swaps them in. A light theme's light (or a dark theme's dark) is dropped. |
| `dark.accent` | string | yes |  | The main color: what you can press. |
| `dark.background` | string | no |  | The page. Left out: the mode's own. |
| `dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `id` | string | yes |  | Lower-case letters, digits, - and _, up to 32. |
| `labels` | array of object | no |  |  |
| `labels[].color` | string | yes |  |  |
| `labels[].name` | string | yes |  |  |
| `light` | object or null | no |  | A dark theme's colors for light mode, kept while it is dark. Switching the theme to light swaps them in. |
| `light.accent` | string | yes |  | The main color: what you can press. |
| `light.background` | string | no |  | The page. Left out: the mode's own. |
| `light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `mode` | string | yes | one of `light`, `dark` |  |
| `name` | string | yes |  | 1 to 60 characters. |
| `overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, accentText, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, bar (the bar across the top), barText, info, infoBorder, border, rowOdd and rowEven (alternating table rows), column, face, statusDoing, statusDoingFg, statusDone, statusDoneFg, statusPlain, statusPlainFg (a card's lit Doing/Done icons and the greyscale ones), and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `surface` | string | no |  | Cards. Left out: the mode's own. |

### User

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `created_at` | string (date-time) | yes |  |  |
| `disabled` | boolean | yes |  | Switched off, or its expiry has passed. |
| `display_name` | string | yes |  |  |
| `expired` | boolean | yes |  | Whether expires_at has passed. |
| `expires_at` | string (date-time) or null | yes |  | When the account stops working, or null for never. |
| `has_password` | boolean | yes |  |  |
| `id` | integer | yes |  |  |
| `login_link` | [`LoginLink`](#loginlink) or null | yes |  | The account's live login link, or null. |
| `owner` | boolean | yes |  |  |
| `password_blocked` | boolean | yes |  |  |
| `role` | string | yes | one of `admin`, `user`, `viewer` |  |
| `two_factor` | boolean | yes |  |  |
| `username` | string | yes |  | Empty until a password is set. |
