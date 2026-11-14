# TodoBoard MCP tools

This file is generated: do not edit it. Change the tools next to their handlers and run `scripts/api_docs.sh`. Every route is in [api.md](api.md).

Every tool POST /mcp offers. Written by scripts/api_docs.sh from the registered tools; do not edit by hand. A key sees the tools its role allows (adminOnly ones need an admin key), and a tool whose feature is switched off answers like one that does not exist. route is the HTTP route the tool stands for, with the same rules; http_only are the routes a key may call that have no tool, and why.

|  |  |
|---|---|
| Endpoint | `POST /mcp` |
| Transport | Streamable HTTP, stateless: one JSON-RPC message per POST, a JSON reply |
| Auth | `Authorization: Bearer <API key>` |

| Tool | Route | Who | What |
|---|---|---|---|
| [`add_board_member`](#add_board_member) | [`POST /api/boards/{board_id}/members`](api.md#post-apiboardsboard_idmembers) | any key | Add, change or remove a board member |
| [`add_checklist_item`](#add_checklist_item) | [`POST /api/boards/{board_id}/cards/{card_id}/checklist`](api.md#post-apiboardsboard_idcardscard_idchecklist) | any key | Add a checklist item |
| [`add_comment`](#add_comment) | [`POST /api/boards/{board_id}/cards/{card_id}/comments`](api.md#post-apiboardsboard_idcardscard_idcomments) | any key | Comment on a card |
| [`admin_cancel_login_link`](#admin_cancel_login_link) | [`POST /api/admin/users/{user_id}/cancel-login-link`](api.md#post-apiadminusersuser_idcancel-login-link) | admin keys | Stop an account's login link working (admin) |
| [`admin_create_invite_link`](#admin_create_invite_link) | [`POST /api/admin/invite-links`](api.md#post-apiadmininvite-links) | admin keys | Create an invite link (admin) |
| [`admin_create_user`](#admin_create_user) | [`POST /api/admin/users`](api.md#post-apiadminusers) | admin keys | Create an account (admin) |
| [`admin_delete_invite_link`](#admin_delete_invite_link) | [`POST /api/admin/invite-links/{link_id}/delete`](api.md#post-apiadmininvite-linkslink_iddelete) | admin keys | Delete an invite link (admin) |
| [`admin_delete_user`](#admin_delete_user) | [`POST /api/admin/users/{user_id}/delete`](api.md#post-apiadminusersuser_iddelete) | admin keys | Delete an account (admin) |
| [`admin_get_theme_defaults`](#admin_get_theme_defaults) | [`GET /api/admin/theme-defaults`](api.md#get-apiadmintheme-defaults) | admin keys | Default themes (admin) |
| [`admin_list_invite_links`](#admin_list_invite_links) | [`GET /api/admin/invite-links`](api.md#get-apiadmininvite-links) | admin keys | List invite links (admin) |
| [`admin_list_user_sessions`](#admin_list_user_sessions) | [`GET /api/admin/users/{user_id}/sessions`](api.md#get-apiadminusersuser_idsessions) | admin keys | An account's signed-in devices (admin) |
| [`admin_list_users`](#admin_list_users) | [`GET /api/admin/users`](api.md#get-apiadminusers) | admin keys | List users (admin) |
| [`admin_reissue_login`](#admin_reissue_login) | [`POST /api/admin/users/{user_id}/reissue-login`](api.md#post-apiadminusersuser_idreissue-login) | admin keys | A new login link for an account (admin) |
| [`admin_rename_invite_link`](#admin_rename_invite_link) | [`POST /api/admin/invite-links/{link_id}/rename`](api.md#post-apiadmininvite-linkslink_idrename) | admin keys | Rename an invite link (admin) |
| [`admin_rename_user`](#admin_rename_user) | [`POST /api/admin/users/{user_id}/rename`](api.md#post-apiadminusersuser_idrename) | admin keys | Rename someone (admin) |
| [`admin_reset_two_factor`](#admin_reset_two_factor) | [`POST /api/admin/users/{user_id}/reset-two-factor`](api.md#post-apiadminusersuser_idreset-two-factor) | admin keys | Turn off an account's two-factor (admin) |
| [`admin_restore_invite_link`](#admin_restore_invite_link) | [`POST /api/admin/invite-links/{link_id}/restore`](api.md#post-apiadmininvite-linkslink_idrestore) | admin keys | Undo deleting an invite link (admin) |
| [`admin_revoke_access`](#admin_revoke_access) | [`POST /api/admin/users/{user_id}/revoke-access`](api.md#post-apiadminusersuser_idrevoke-access) | admin keys | Revoke a user's access (admin) |
| [`admin_revoke_invite_link`](#admin_revoke_invite_link) | [`POST /api/admin/invite-links/{link_id}/revoke`](api.md#post-apiadmininvite-linkslink_idrevoke) | admin keys | Switch off an invite link (admin) |
| [`admin_revoke_user_session`](#admin_revoke_user_session) | [`POST /api/admin/users/{user_id}/sessions/{session_id}/revoke`](api.md#post-apiadminusersuser_idsessionssession_idrevoke) | admin keys | Sign out one of an account's devices (admin) |
| [`admin_set_invite_link_limits`](#admin_set_invite_link_limits) | [`POST /api/admin/invite-links/{link_id}/limits`](api.md#post-apiadmininvite-linkslink_idlimits) | admin keys | Change an invite link's places and expiry (admin) |
| [`admin_set_theme_defaults`](#admin_set_theme_defaults) | [`POST /api/admin/theme-defaults`](api.md#post-apiadmintheme-defaults) | admin keys | Replace the default themes (admin) |
| [`admin_set_user_disabled`](#admin_set_user_disabled) | [`POST /api/admin/users/{user_id}/disabled`](api.md#post-apiadminusersuser_iddisabled) | admin keys | Disable or enable a user (admin) |
| [`admin_set_user_role`](#admin_set_user_role) | [`POST /api/admin/users/{user_id}/role`](api.md#post-apiadminusersuser_idrole) | admin keys | Change an account's role (admin) |
| [`admin_switch_on_invite_link`](#admin_switch_on_invite_link) | [`POST /api/admin/invite-links/{link_id}/switch-on`](api.md#post-apiadmininvite-linkslink_idswitch-on) | admin keys | Switch an invite link back on (admin) |
| [`archive_board`](#archive_board) | [`POST /api/boards/{board_id}/archive`](api.md#post-apiboardsboard_idarchive) | any key | Archive or unarchive a board |
| [`archive_list_cards`](#archive_list_cards) | [`POST /api/boards/{board_id}/lists/{list_id}/archive-cards`](api.md#post-apiboardsboard_idlistslist_idarchive-cards) | any key | Archive a column's cards |
| [`cancel_my_login_link`](#cancel_my_login_link) | [`POST /api/account/login-link/cancel`](api.md#post-apiaccountlogin-linkcancel) | any key | Stop your own login link working |
| [`carry_over`](#carry_over) | [`POST /api/boards/{board_id}/carry-over`](api.md#post-apiboardsboard_idcarry-over) | any key | Carry unfinished cards over to today |
| [`copy_card`](#copy_card) | [`POST /api/boards/{board_id}/cards/{card_id}/copy`](api.md#post-apiboardsboard_idcardscard_idcopy) | any key | Copy a card to a board |
| [`create_board`](#create_board) | [`POST /api/boards`](api.md#post-apiboards) | any key | Create a board |
| [`create_card`](#create_card) | [`POST /api/boards/{board_id}/cards`](api.md#post-apiboardsboard_idcards) | any key | Add a card |
| [`create_folder`](#create_folder) | [`POST /api/folders`](api.md#post-apifolders) | any key | Make a folder |
| [`create_label`](#create_label) | [`POST /api/boards/{board_id}/labels`](api.md#post-apiboardsboard_idlabels) | any key | Add a label |
| [`create_list`](#create_list) | [`POST /api/boards/{board_id}/lists`](api.md#post-apiboardsboard_idlists) | any key | Add a column |
| [`create_my_login_link`](#create_my_login_link) | [`POST /api/account/login-link`](api.md#post-apiaccountlogin-link) | any key | A login link for yourself |
| [`delete_card`](#delete_card) | [`POST /api/boards/{board_id}/cards/{card_id}/delete`](api.md#post-apiboardsboard_idcardscard_iddelete) | any key | Delete a card |
| [`delete_checklist_item`](#delete_checklist_item) | [`POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}/delete`](api.md#post-apiboardsboard_idcardscard_idchecklistitem_iddelete) | any key | Delete a checklist item |
| [`delete_folder`](#delete_folder) | [`POST /api/folders/{folder_id}/delete`](api.md#post-apifoldersfolder_iddelete) | any key | Delete a folder |
| [`delete_label`](#delete_label) | [`POST /api/boards/{board_id}/labels/{label_id}/delete`](api.md#post-apiboardsboard_idlabelslabel_iddelete) | any key | Delete a label |
| [`delete_list`](#delete_list) | [`POST /api/boards/{board_id}/lists/{list_id}/delete`](api.md#post-apiboardsboard_idlistslist_iddelete) | any key | Delete a column |
| [`delete_share`](#delete_share) | [`POST /api/boards/{board_id}/shares/{link_id}/delete`](api.md#post-apiboardsboard_idshareslink_iddelete) | any key | Delete a share link |
| [`duplicate_board`](#duplicate_board) | [`POST /api/boards/{board_id}/duplicate`](api.md#post-apiboardsboard_idduplicate) | any key | Duplicate a board |
| [`edit_comment`](#edit_comment) | [`POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}`](api.md#post-apiboardsboard_idcardscard_idcommentscomment_id) | any key | Edit a comment |
| [`empty_trash`](#empty_trash) | [`POST /api/boards/{board_id}/trash/empty`](api.md#post-apiboardsboard_idtrashempty) | any key | Empty a board's trash |
| [`find_cards`](#find_cards) | [`GET /api/cards`](api.md#get-apicards) | any key | Find cards |
| [`get_board`](#get_board) | [`GET /api/boards/{board_id}`](api.md#get-apiboardsboard_id) | any key | Read a board |
| [`get_board_notes`](#get_board_notes) | [`GET /api/boards/{board_id}/notes`](api.md#get-apiboardsboard_idnotes) | any key | Read a board's purpose and notes |
| [`get_board_subscription`](#get_board_subscription) | [`GET /api/boards/{board_id}/subscription`](api.md#get-apiboardsboard_idsubscription) | any key | Your notifications for a board |
| [`get_card`](#get_card) | [`GET /api/boards/{board_id}/cards/{card_id}`](api.md#get-apiboardsboard_idcardscard_id) | any key | Read a card |
| [`get_card_subscription`](#get_card_subscription) | [`GET /api/boards/{board_id}/cards/{card_id}/subscription`](api.md#get-apiboardsboard_idcardscard_idsubscription) | any key | Your notifications for a card |
| [`get_notify_prefs`](#get_notify_prefs) | [`GET /api/notify/prefs`](api.md#get-apinotifyprefs) | any key | Your notification settings |
| [`label_my_session`](#label_my_session) | [`POST /api/account/sessions/{session_id}/label`](api.md#post-apiaccountsessionssession_idlabel) | any key | Name one of your devices |
| [`leave_board`](#leave_board) | [`POST /api/boards/{board_id}/leave`](api.md#post-apiboardsboard_idleave) | any key | Leave a board |
| [`list_boards`](#list_boards) | [`GET /api/boards`](api.md#get-apiboards) | any key | List my boards |
| [`list_card_links`](#list_card_links) | [`GET /api/boards/{board_id}/cards/{card_id}/links`](api.md#get-apiboardsboard_idcardscard_idlinks) | any key | List a card's mirrors |
| [`list_comments`](#list_comments) | [`GET /api/boards/{board_id}/cards/{card_id}/comments`](api.md#get-apiboardsboard_idcardscard_idcomments) | any key | Read a card's comments |
| [`list_my_sessions`](#list_my_sessions) | [`GET /api/account/sessions`](api.md#get-apiaccountsessions) | any key | Your signed-in devices |
| [`list_shares`](#list_shares) | [`GET /api/boards/{board_id}/shares`](api.md#get-apiboardsboard_idshares) | any key | List a board's share links |
| [`list_team`](#list_team) | [`GET /api/team`](api.md#get-apiteam) | any key | List team |
| [`list_trash`](#list_trash) | [`GET /api/boards/{board_id}/trash`](api.md#get-apiboardsboard_idtrash) | any key | List a board's trash |
| [`move_folder`](#move_folder) | [`POST /api/folders/{folder_id}/parent`](api.md#post-apifoldersfolder_idparent) | any key | Move a folder |
| [`order_folders`](#order_folders) | [`POST /api/folders/order`](api.md#post-apifoldersorder) | any key | Order your folders |
| [`purge_card`](#purge_card) | [`POST /api/boards/{board_id}/trash/{card_id}/delete`](api.md#post-apiboardsboard_idtrashcard_iddelete) | any key | Delete a card for good |
| [`refresh_product`](#refresh_product) | [`POST /api/boards/{board_id}/cards/{card_id}/product/refresh`](api.md#post-apiboardsboard_idcardscard_idproductrefresh) | any key | Look a card's product up again |
| [`remove_card_image`](#remove_card_image) | [`POST /api/boards/{board_id}/cards/{card_id}/image/delete`](api.md#post-apiboardsboard_idcardscard_idimagedelete) | any key | Take a card's picture off |
| [`remove_comment`](#remove_comment) | [`POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}/remove`](api.md#post-apiboardsboard_idcardscard_idcommentscomment_idremove) | any key | Remove a comment |
| [`remove_product`](#remove_product) | [`POST /api/boards/{board_id}/cards/{card_id}/product/delete`](api.md#post-apiboardsboard_idcardscard_idproductdelete) | any key | Take a card's product off |
| [`rename_account`](#rename_account) | [`POST /api/account/rename`](api.md#post-apiaccountrename) | any key | Rename your account |
| [`rename_folder`](#rename_folder) | [`POST /api/folders/{folder_id}`](api.md#post-apifoldersfolder_id) | any key | Rename a folder |
| [`restore_card`](#restore_card) | [`POST /api/boards/{board_id}/trash/{card_id}/restore`](api.md#post-apiboardsboard_idtrashcard_idrestore) | any key | Restore a card from the trash |
| [`revoke_my_other_sessions`](#revoke_my_other_sessions) | [`POST /api/account/sessions/revoke-others`](api.md#post-apiaccountsessionsrevoke-others) | any key | Sign out all your devices |
| [`revoke_my_session`](#revoke_my_session) | [`POST /api/account/sessions/{session_id}/revoke`](api.md#post-apiaccountsessionssession_idrevoke) | any key | Sign out one of your devices |
| [`revoke_share`](#revoke_share) | [`POST /api/boards/{board_id}/shares/{link_id}/revoke`](api.md#post-apiboardsboard_idshareslink_idrevoke) | any key | Revoke a share link |
| [`send_card`](#send_card) | [`POST /api/boards/{board_id}/cards/{card_id}/send`](api.md#post-apiboardsboard_idcardscard_idsend) | any key | Send a card to another board |
| [`send_test_notification`](#send_test_notification) | [`POST /api/notify/test`](api.md#post-apinotifytest) | any key | Send yourself a test notification |
| [`set_board_folder`](#set_board_folder) | [`POST /api/boards/{board_id}/folder`](api.md#post-apiboardsboard_idfolder) | any key | File a board in a folder |
| [`set_board_subscription`](#set_board_subscription) | [`POST /api/boards/{board_id}/subscription`](api.md#post-apiboardsboard_idsubscription) | any key | Set your notifications for a board |
| [`set_card_subscription`](#set_card_subscription) | [`POST /api/boards/{board_id}/cards/{card_id}/subscription`](api.md#post-apiboardsboard_idcardscard_idsubscription) | any key | Set your notifications for a card |
| [`set_display`](#set_display) | [`POST /api/account/display`](api.md#post-apiaccountdisplay) | any key | Save your display settings |
| [`set_label_colors`](#set_label_colors) | [`POST /api/account/label-colors`](api.md#post-apiaccountlabel-colors) | any key | Save your label colours |
| [`set_my_board_view`](#set_my_board_view) | [`POST /api/boards/{board_id}/my-view`](api.md#post-apiboardsboard_idmy-view) | any key | Your own view of a board |
| [`set_my_collapsed_lists`](#set_my_collapsed_lists) | [`POST /api/boards/{board_id}/my-view/collapsed`](api.md#post-apiboardsboard_idmy-viewcollapsed) | any key | Your collapsed columns |
| [`set_notify_prefs`](#set_notify_prefs) | [`POST /api/notify/prefs`](api.md#post-apinotifyprefs) | any key | Save your notification settings |
| [`set_theme`](#set_theme) | [`POST /api/account/theme`](api.md#post-apiaccounttheme) | any key | Pick your theme |
| [`set_themes`](#set_themes) | [`POST /api/account/themes`](api.md#post-apiaccountthemes) | any key | Replace your own themes |
| [`share_card_with_user`](#share_card_with_user) | [`POST /api/boards/{board_id}/cards/{card_id}/share-user`](api.md#post-apiboardsboard_idcardscard_idshare-user) | any key | Share a card with someone |
| [`unlink_card`](#unlink_card) | [`POST /api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete`](api.md#post-apiboardsboard_idcardscard_idlinkslink_iddelete) | any key | Stop mirroring a card |
| [`update_board`](#update_board) | [`POST /api/boards/{board_id}`](api.md#post-apiboardsboard_id) | any key | Change a board's settings |
| [`update_board_notes`](#update_board_notes) | [`POST /api/boards/{board_id}/notes`](api.md#post-apiboardsboard_idnotes) | any key | Change a board's purpose or notes |
| [`update_card`](#update_card) | [`POST /api/boards/{board_id}/cards/{card_id}`](api.md#post-apiboardsboard_idcardscard_id) | any key | Change a card |
| [`update_checklist_item`](#update_checklist_item) | [`POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}`](api.md#post-apiboardsboard_idcardscard_idchecklistitem_id) | any key | Tick or edit a checklist item |
| [`update_label`](#update_label) | [`POST /api/boards/{board_id}/labels/{label_id}`](api.md#post-apiboardsboard_idlabelslabel_id) | any key | Change a label |
| [`update_list`](#update_list) | [`POST /api/boards/{board_id}/lists/{list_id}`](api.md#post-apiboardsboard_idlistslist_id) | any key | Rename, move or archive a column |
| [`whoami`](#whoami) | [`GET /api/me`](api.md#get-apime) | any key | Who am I |

## add_board_member

Put an account on the board as an editor or a viewer, change their role, or take them off it (role empty). Only the board's owner may. The owner's own place cannot be changed.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/members`](api.md#post-apiboardsboard_idmembers) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `role` | string | yes | one of `editor`, `viewer`, `` | editor or viewer; empty removes them from the board. |
| `user_id` | integer | yes |  | The account to add, change or remove, from list_team. |

## add_checklist_item

Add an item to the end of a card's checklist. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/checklist`](api.md#post-apiboardsboard_idcardscard_idchecklist) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `text` | string | yes |  |  |

## add_comment

Post a comment on a card, signed with your name. Markdown, up to 4000 characters. Needs edit rights on the card, or a card shared with you to comment on.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/comments`](api.md#post-apiboardsboard_idcardscard_idcomments) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters. |

## admin_cancel_login_link

Void an account's live login link without making a new one: a link that went somewhere it should not have. Devices it already signed in stay signed in; admin_revoke_access ends those too. Answers with removed: 0 or 1.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/cancel-login-link`](api.md#post-apiadminusersuser_idcancel-login-link) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_create_invite_link

One link for a group: each person who opens it joins as a new user, up to max_uses people, for days days. Answers with join_url (shown only now) and the link's details. Revoking or deleting this API key switches the link off.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links`](api.md#post-apiadmininvite-links) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `days` | integer | yes | 1 to 365 |  |
| `label` | string | yes | 1 to 80 characters |  |
| `max_uses` | integer | yes | 1 to 1000 |  |

## admin_create_user

Make a new account and its first login link, to hand to the person. role is admin or user. Answers with user_id and the link. Answers with login_url (open it to sign in; it works once unless max_uses says otherwise), expires_at and max_uses. Any older link the account could still use stops working; its signed-in devices stay signed in. Revoking or deleting this API key voids the link while it is unused, and signs out every device it signed in.

|  |  |
|---|---|
| Route | [`POST /api/admin/users`](api.md#post-apiadminusers) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes | 1 to 80 characters |  |
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |
| `role` | string | yes | one of `admin`, `user` |  |

## admin_delete_invite_link

Take an invite link off the list and stop it working. The accounts it made stay. admin_restore_invite_link undoes it for a few minutes.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/delete`](api.md#post-apiadmininvite-linkslink_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

## admin_delete_user

Remove an account for good, with its sessions, login link and API keys. Cannot be undone: admin_set_user_disabled is the reversible version. Only the owner deletes an admin; nobody deletes the owner or themselves.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/delete`](api.md#post-apiadminusersuser_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_get_theme_defaults

The theme list everyone without their own list sees. themes is null while the built-in ones apply.

|  |  |
|---|---|
| Route | [`GET /api/admin/theme-defaults`](api.md#get-apiadmintheme-defaults) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## admin_list_invite_links

Group invite links, a page at a time: label, places used and left, expiry, and whether each still works. The links themselves are never shown again after creation. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Route | [`GET /api/admin/invite-links`](api.md#get-apiadmininvite-links) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `dir` | string | no | one of `asc`, `desc` | Default asc. |
| `limit` | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | string | no |  | Keep links whose label or maker contains this, in any case. |
| `sort` | string | no | one of `label`, `used`, `expires_at` | Field to sort the whole list by. Rows without it go last. |

## admin_list_user_sessions

Where an account is signed in: every session, with when it started and was last seen. Addresses are shown to the owner only, and only the owner sees the owner's.

|  |  |
|---|---|
| Route | [`GET /api/admin/users/{user_id}/sessions`](api.md#get-apiadminusersuser_idsessions) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `limit` | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_list_users

Accounts, a page at a time, including disabled ones, with each one's role and which is the owner. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Route | [`GET /api/admin/users`](api.md#get-apiadminusers) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `dir` | string | no | one of `asc`, `desc` | Default asc. |
| `limit` | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | integer | no | at least 0 | Rows to skip: the next value of the previous page. |
| `q` | string | no |  | Keep accounts whose name or username contains this, in any case. |
| `sort` | string | no | one of `display_name`, `role`, `disabled`, `created_at` | Field to sort the whole list by. Rows without it go last. |

## admin_reissue_login

For someone who lost their device or their link: a fresh login link for their account. Answers with login_url (open it to sign in; it works once unless max_uses says otherwise), expires_at and max_uses. Any older link the account could still use stops working; its signed-in devices stay signed in. Revoking or deleting this API key voids the link while it is unused, and signs out every device it signed in.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/reissue-login`](api.md#post-apiadminusersuser_idreissue-login) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_rename_invite_link

Change the label people see on an invite link's join page (1-80 characters). expect, when given, is the label you think it has now.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/rename`](api.md#post-apiadmininvite-linkslink_idrename) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expect` | string or null | no |  | The label you think it has. Answers 409 if it has another. |
| `label` | string | yes | 1 to 80 characters | 1-80 characters. |
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

## admin_rename_user

Change another account's display name (1-80 characters). Only the owner renames an admin. expect, when given, is the name you think it has now; if it changed meanwhile the rename is refused with the current name.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/rename`](api.md#post-apiadminusersuser_idrename) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes | 1 to 80 characters | 1-80 characters. |
| `expect` | string or null | no |  | The name you think they have. Answers 409 if it is another. |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_reset_two_factor

For someone who lost both their phone and their recovery codes: switches their two-factor sign-in off so a login link lets them in again. Only the owner resets an admin's; the owner's is reset from a shell.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/reset-two-factor`](api.md#post-apiadminusersuser_idreset-two-factor) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_restore_invite_link

Bring back an invite link deleted in the last few minutes, working as before. The link then counts as this key's: revoking the key switches it off.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/restore`](api.md#post-apiadmininvite-linkslink_idrestore) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

## admin_revoke_access

For an account that may be compromised: ends every session and voids its live login link and every API key it holds, at once. The account stays enabled; admin_reissue_login lets its owner back in. The owner and your own account cannot be revoked.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/revoke-access`](api.md#post-apiadminusersuser_idrevoke-access) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_revoke_invite_link

Stop an invite link from letting anyone else join or sign in through it. Accounts it already made keep working. Cannot be switched back on.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/revoke`](api.md#post-apiadmininvite-linkslink_idrevoke) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

## admin_revoke_user_session

End one of an account's sessions, rather than all of them with admin_revoke_access.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/sessions/{session_id}/revoke`](api.md#post-apiadminusersuser_idsessionssession_idrevoke) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `session_id` | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_set_invite_link_limits

Set how many people the link may take in (never fewer than have joined) and, when days is given, a new expiry counted from now. The link then counts as this key's: revoking the key switches it off.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/limits`](api.md#post-apiadmininvite-linkslink_idlimits) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `days` | integer or null | no | 1 to 365 | Counted from now. Leave out to keep the expiry. |
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |
| `max_uses` | integer | yes | 1 to 1000 | Not fewer than have joined. |

## admin_set_theme_defaults

Replace the theme list everyone without their own list sees, 1 to 40 themes in the Theme schema of the API reference. null puts the built-in themes back.

|  |  |
|---|---|
| Route | [`POST /api/admin/theme-defaults`](api.md#post-apiadmintheme-defaults) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `themes` | array of object or null | no |  | 1 to 40 themes. null goes back to the built-in ones. |
| `themes[].accent` | string | yes |  | The main color: what you can press. |
| `themes[].background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].button` | object or null | no |  | How the theme's buttons look. A choice left out is the default (rounded, normal, raised, normal). |
| `themes[].button.case` | string | no | one of `normal`, `upper` |  |
| `themes[].button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `themes[].button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `themes[].button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `themes[].dark` | object or null | no |  | A light theme's colors for dark mode, kept while it is light. Switching the theme to dark swaps them in. A light theme's light (or a dark theme's dark) is dropped. |
| `themes[].dark.accent` | string | yes |  | The main color: what you can press. |
| `themes[].dark.background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `themes[].id` | string | yes |  | Lower-case letters, digits, - and _, up to 32. |
| `themes[].labels` | array of object | no |  |  |
| `themes[].labels[].color` | string | yes |  |  |
| `themes[].labels[].name` | string | yes |  |  |
| `themes[].light` | object or null | no |  | A dark theme's colors for light mode, kept while it is dark. Switching the theme to light swaps them in. |
| `themes[].light.accent` | string | yes |  | The main color: what you can press. |
| `themes[].light.background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `themes[].mode` | string | yes | one of `light`, `dark` |  |
| `themes[].name` | string | yes |  | 1 to 60 characters. |
| `themes[].overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].surface` | string | no |  | Cards. Left out: the mode's own. |

## admin_set_user_disabled

Switch an account off (disabled: true) or back on (false). Disabling signs it out everywhere and voids its live login link; its API keys stop working while it is off. Reversible. The owner and your own account cannot be disabled.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/disabled`](api.md#post-apiadminusersuser_iddisabled) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `disabled` | boolean | yes |  |  |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_set_user_role

Make an account an admin or an ordinary user. Only the owner takes an admin's rights away; the owner is always an admin, and nobody changes their own role. Becoming an admin ends the account's sessions and live login link.

|  |  |
|---|---|
| Route | [`POST /api/admin/users/{user_id}/role`](api.md#post-apiadminusersuser_idrole) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `role` | string | yes | one of `admin`, `user` |  |
| `user_id` | integer | yes | at least 1 | The account's id, from GET /api/admin/users (admin_list_users over MCP). |

## admin_switch_on_invite_link

Undo admin_revoke_invite_link: anyone still holding the link can use it again. The link then counts as this key's: revoking the key switches it off.

|  |  |
|---|---|
| Route | [`POST /api/admin/invite-links/{link_id}/switch-on`](api.md#post-apiadmininvite-linkslink_idswitch-on) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `link_id` | string | yes |  | The link's id, from GET /api/admin/invite-links (admin_list_invite_links over MCP). |

## archive_board

scope "everyone" (the owner only): the board leaves every member's home page and becomes read-only -- every change is refused with board_archived -- until it is unarchived the same way. scope "me" (anyone on the board, but not a key held to some boards): it only leaves your own home page. list_boards says which applies (archived, archived_for_me).

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/archive`](api.md#post-apiboardsboard_idarchive) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | yes |  | true archives, false unarchives. |
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `scope` | string | yes | one of `everyone`, `me` | everyone (the owner only) or me (just your own home page). |

## archive_list_cards

Archive a column's open cards, all of them or the ones named in card_ids: clearing out a Done column, say. Archived cards are hidden, not deleted (update_card with archived: false brings one back). Needs edit rights on that column.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/lists/{list_id}/archive-cards`](api.md#post-apiboardsboard_idlistslist_idarchive-cards) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_ids` | array of integer | no |  | The cards to archive; leave out to archive every open card in the column. |
| `list_id` | integer | yes |  | A column's id, from get_board (board.lists). |

## cancel_my_login_link

Void your account's live login link without making a new one. Devices it already signed in stay signed in. Answers with removed: 0 or 1.

|  |  |
|---|---|
| Route | [`POST /api/account/login-link/cancel`](api.md#post-apiaccountlogin-linkcancel) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## carry_over

On a calendar board with carry-over on, move the unfinished cards of past days to today (yours, within a day of the server's). Asking again the same day moves nothing; a board with carry-over off, or not a calendar, moves nothing. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/carry-over`](api.md#post-apiboardsboard_idcarry-over) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `today` | string | yes |  | Your today, as 2026-11-07: within a day of the server's. |

## copy_card

Copy a card onto any board (its own included) with its content, comments and picture, optionally kept in step with the original (mirror). Needs edit rights where the copy goes, and on the card for a mirror. Answers the copy, with card_path and board_title.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/copy`](api.md#post-apiboardsboard_idcardscard_idcopy) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `list_id` | integer | yes |  | The column on that board it goes into, from get_board (board.lists). |
| `mirror` | string | no | one of ``, `one_way`, `two_way` | Copy only. Empty for a plain copy; one_way keeps the copy following the original (the copy's synced fields become read-only); two_way keeps both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments. |
| `position` | number or null | no |  | An exact place in the column, for putting a card back where it was; wins over top. |
| `to_board` | string or integer | yes |  | The board it goes to: id or URL slug, from list_boards. |
| `top` | boolean | no |  | Put it at the top of the column instead of the end. |

## create_board

A new board owned by you, with the given columns in order (for example ["To do", "Doing", "Done"]); a board always gets at least three, so fewer are padded out. Returns its id and slug. A key held to some boards cannot make one.

|  |  |
|---|---|
| Route | [`POST /api/boards`](api.md#post-apiboards) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `lists` | array of string | no |  | Its columns, in order, for example ["To do", "Doing", "Done"]. A board always gets at least three, so fewer are padded out. |
| `title` | string | yes | at most 120 characters | The new board's title. |

## create_card

Add a card to a column. The other fields are optional and set in the same call. Needs edit rights on that column. If the card is made but its other fields are refused, the answer says so and names the new card: change it with update_card rather than adding it again.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards`](api.md#post-apiboardsboard_idcards) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `assignees` | array of integer or null | no |  | User ids, from get_board's members or list_team. |
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `day` | string or null | no |  | On a calendar board (layout calendar), the day the card is filed under: 2026-11-07, or an empty string for none. |
| `description` | string or null | no |  | Markdown. |
| `doing` | boolean or null | no |  | On a staged board, the Doing stage. true starts the card (reopening it if done); done: true moves it on to Done. |
| `done` | boolean or null | no |  | true makes it complete from the start. |
| `due_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `label_ids` | array of integer or null | no |  | Label ids, from get_board (board.labels). |
| `list_id` | integer | yes |  | The column's id, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position within the column; lower comes first. Leave out to put it at the end. |
| `rank_place` | integer or null | no | at least 0 | Importance across the whole board: 1 is the most important, and other cards shift to make room. 0 takes the card out of the ranking. Needs edit rights on the board. |
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string | yes |  |  |

## create_folder

Make a folder on your own home page, at the top or inside another of yours (parent_id).

|  |  |
|---|---|
| Route | [`POST /api/folders`](api.md#post-apifolders) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `name` | string | yes |  |  |
| `parent_id` | integer | no |  | The folder to make it in; 0 or absent: at the top. |

## create_label

Add a label to a board, to put on cards with update_card's label_ids. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/labels`](api.md#post-apiboardsboard_idlabels) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `color` | string | yes |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `name` | string or null | no | at most 40 characters |  |

## create_list

Add a column (list) to a board, at the end or at position. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/lists`](api.md#post-apiboardsboard_idlists) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `position` | number or null | no |  | Sort position among the columns; lower comes first. Leave out to put it at the end. |
| `title` | string | yes | at most 120 characters |  |

## create_my_login_link

A link that signs your own account in on a browser or another device. Only a key with your account's full role may ask: one scoped down to user on an admin account is refused. Answers with login_url (open it to sign in; it works once unless max_uses says otherwise), expires_at and max_uses. Any older link the account could still use stops working; its signed-in devices stay signed in. Revoking or deleting this API key voids the link while it is unused, and signs out every device it signed in.

|  |  |
|---|---|
| Route | [`POST /api/account/login-link`](api.md#post-apiaccountlogin-link) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expires_in_minutes` | integer or null | no | 0 to 525600 | 0 never expires. Default auth.login_token_ttl_seconds for a link that works once, auth.multi_use_login_token_ttl_seconds otherwise. |
| `max_uses` | integer or null | no | 0 to 1000 | Sign-ins the link allows: 1-1000, or 0 for unlimited. Default 1. |

## delete_card

Move a card to the board's trash. Not permanent: restore_card brings it back, until purge_card or empty_trash removes it for good. Needs edit rights on its column.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/delete`](api.md#post-apiboardsboard_idcardscard_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## delete_checklist_item

Remove an item from a card's checklist, for good. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}/delete`](api.md#post-apiboardsboard_idcardscard_idchecklistitem_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `item_id` | integer | yes |  | A checklist item's id, from get_card. |

## delete_folder

Remove one of your folders. Boards in it are not touched: they go back to the top of your home page.

|  |  |
|---|---|
| Route | [`POST /api/folders/{folder_id}/delete`](api.md#post-apifoldersfolder_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `folder_id` | integer | yes |  | A folder's id, from list_boards (folders). |

## delete_label

Delete a label; it comes off every card it was on. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/labels/{label_id}/delete`](api.md#post-apiboardsboard_idlabelslabel_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `label_id` | integer | yes |  | A label's id, from get_board (board.labels). |

## delete_list

Delete a column and its share links; its cards go to the board's trash, where they can be restored into another column. Archiving it (update_list, archived: true) is the gentler way to put one away. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/lists/{list_id}/delete`](api.md#post-apiboardsboard_idlistslist_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | integer | yes |  | A column's id, from get_board (board.lists). |

## delete_share

Remove one of the board's share links from the list (link_id from list_shares); a live one stops working. Only the board's owner may.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/shares/{link_id}/delete`](api.md#post-apiboardsboard_idshareslink_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `link_id` | string | yes | at least 1 characters | A share link's id, from list_shares. |

## duplicate_board

A new board owned by you, copied from one you can open: its settings, notes, labels, columns, and the cards you can see with their checklists. Comments, card pictures, members and share links are not copied. Returns the copy's id and slug. A key held to some boards cannot make one.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/duplicate`](api.md#post-apiboardsboard_idduplicate) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `title` | string | yes | at most 120 characters | The copy's title. |

## edit_comment

Change the text of a comment you wrote. Comment ids are in list_comments.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}`](api.md#post-apiboardsboard_idcardscard_idcommentscomment_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | integer | yes |  | A comment's id, from list_comments. |
| `text` | string | yes | at most 4000 characters | Markdown, up to 4000 characters; replaces the comment's text. |

## empty_trash

Permanently delete every card in the board's trash. This cannot be undone. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/trash/empty`](api.md#post-apiboardsboard_idtrashempty) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## find_cards

Search the cards you can see, across all your boards or one (board_id). query matches title and description (case-insensitive). Done and archived cards are left out unless asked for. Returns compact rows, soonest due first, with total and whether the list was cut at limit; use get_card for the full card.

|  |  |
|---|---|
| Route | [`GET /api/cards`](api.md#get-apicards) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `assigned_to_me` | boolean | no |  | Only cards assigned to you. |
| `board` | string or integer | no |  | Search one board only: its id or URL slug, from list_boards. |
| `due_before` | string | no |  | Only cards due before this: "2026-10-30" or an RFC 3339 timestamp. |
| `include_archived` | boolean | no |  | Include archived cards. |
| `include_done` | boolean | no |  | Include done cards. |
| `limit` | integer | no | 1 to 200; default `50` | At most this many rows (default 50). |
| `query` | string | no |  | Matches title and description, in any case. |

## get_board

A board as you can see it: its notes, its purpose if the board has that section on (board.purpose says what the board is for and how to work on it, so read it first), columns (board.lists), labels, members, and the cards you may see, with each card's column, labels, assignees, dates, checklist and rev. people maps user ids to names. access is what you may do. Archived cards are left out unless include_archived is true.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}`](api.md#get-apiboardsboard_id) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `include_archived` | boolean or null | no |  | Include archived cards. The page gets them unless this is false; an agent's get_board leaves them out unless it is true. |

## get_board_notes

Just the board's notes and purpose, without its cards. notes holds links, documents and reference material. purpose says what the board is for and how to work on it, and is only there when purpose_enabled is true. Both are markdown, and may be empty. Read them before working on a board. The revs go back to update_board_notes.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/notes`](api.md#get-apiboardsboard_idnotes) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## get_board_subscription

Which of the board's events reach you (due dates, being assigned, changes, activity) and where that comes from (source: your defaults or this board's override), with whether push is on and whether anything would reach you.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/subscription`](api.md#get-apiboardsboard_idsubscription) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## get_card

One card in full: title, description, column, labels, assignees, dates, checklist, its place in the board's ranking (rank_place, 0 for none), and rev (pass rev back to update_card when changing the title or description).

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/cards/{card_id}`](api.md#get-apiboardsboard_idcardscard_id) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## get_card_subscription

Which of the card's events reach you and where that comes from (source: your defaults, the board's override or this card's), with whether push is on and whether anything would reach you.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/cards/{card_id}/subscription`](api.md#get-apiboardsboard_idcardscard_idsubscription) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## get_notify_prefs

How and when you are told about things: Pushover (the key only masked) or this browser's push, timezone, reminder lead times, the hour for all-day reminders, the events you hear about as an assignee or watcher, whether shares with you are announced, and whether it is all paused. Also whether the server can send push at all and whether anything would reach you.

|  |  |
|---|---|
| Route | [`GET /api/notify/prefs`](api.md#get-apinotifyprefs) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## label_my_session

Give one of your devices a name (at most 80 characters; empty clears it). expect, when given, is the name you think it has now.

|  |  |
|---|---|
| Route | [`POST /api/account/sessions/{session_id}/label`](api.md#post-apiaccountsessionssession_idlabel) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `expect` | string or null | no |  | The name you think it has. Answers 409 if it has another. |
| `label` | string | yes | at most 80 characters | At most 80 characters. |
| `session_id` | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |

## leave_board

Take yourself off a board you are a member of; it leaves your home page and you lose access until someone adds you again. The owner cannot leave (400 owner_cannot_leave).

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/leave`](api.md#post-apiboardsboard_idleave) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## list_boards

Every board you own or are a member of (id, slug, title, your role, folder, and whether it is archived for everyone -- read-only -- or archived_for_me), your home-page folders, and single cards shared with you from boards you are not on.

|  |  |
|---|---|
| Route | [`GET /api/boards`](api.md#get-apiboards) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## list_card_links

The cards this one is kept in step with: each link's id, mode (one_way / two_way), direction (out: this card's changes flow there; in: this card follows that one; both), and the other card, named only when you can open it.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/cards/{card_id}/links`](api.md#get-apiboardsboard_idcardscard_idlinks) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## list_comments

A card's comments, oldest first: who wrote each (by: a member's name, guest:Name for a guest, guest: alone for an anonymous one) and when, and whether you may post one (can_comment).

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/cards/{card_id}/comments`](api.md#get-apiboardsboard_idcardscard_idcomments) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## list_my_sessions

Where your account is signed in, a page at a time: id, when it started and was last seen, the browser, and the name you gave it. Through an API key no device is current. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Route | [`GET /api/account/sessions`](api.md#get-apiaccountsessions) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `limit` | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | integer | no | at least 0 | Rows to skip: the next value of the previous page. |

## list_shares

The board's share links, live or not: id, what each opens (the board, a column or one card), mode, label, when it was made, expires, was revoked and last used, whether it is a quick link, and who made it. Never the link itself, which is shown once when it is made. Only the board's owner may.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/shares`](api.md#get-apiboardsboard_idshares) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## list_team

People with an active account, a page at a time: id and display name. Use the id wherever another tool asks for a user. next is the offset of the following page, or null on the last one.

|  |  |
|---|---|
| Route | [`GET /api/team`](api.md#get-apiteam) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `limit` | integer | no | 1 to 1000; default `200` | Rows to return. Default 200. |
| `offset` | integer | no | at least 0 | Rows to skip: the next value of the previous page. |

## list_trash

The board's deleted cards, with when and by whom each was deleted. restore_card brings one back. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`GET /api/boards/{board_id}/trash`](api.md#get-apiboardsboard_idtrash) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |

## move_folder

Put one of your folders inside another of yours, or back at the top (parent_id 0).

|  |  |
|---|---|
| Route | [`POST /api/folders/{folder_id}/parent`](api.md#post-apifoldersfolder_idparent) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `folder_id` | integer | yes |  | A folder's id, from list_boards (folders). |
| `parent_id` | integer | yes |  | The folder to put it in; 0: at the top. |

## order_folders

Put your folders in this order, first to last.

|  |  |
|---|---|
| Route | [`POST /api/folders/order`](api.md#post-apifoldersorder) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `ids` | array of integer | yes |  | Your folder ids, first to last. |

## purge_card

Permanently delete one card that is in the board's trash. This cannot be undone. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/trash/{card_id}/delete`](api.md#post-apiboardsboard_idtrashcard_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## refresh_product

Read the shop link in a card's title or description (or the one it was last looked up from) again now, and store the product's name, price and photo on the card. Works for any shop's link, shopping column or not. Answers with the card; product.status is failed when the page could not be read.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/product/refresh`](api.md#post-apiboardsboard_idcardscard_idproductrefresh) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## remove_card_image

Remove the picture from a card and answer with the card. The card itself is not touched.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/image/delete`](api.md#post-apiboardsboard_idcardscard_idimagedelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## remove_comment

Delete a comment for good: one you wrote, or anyone's on a board you manage. This cannot be undone.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/comments/{comment_id}/remove`](api.md#post-apiboardsboard_idcardscard_idcommentscomment_idremove) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `comment_id` | integer | yes |  | A comment's id, from list_comments. |

## remove_product

Remove the product (name, price, photo) looked up for a card. The link in its text stays. Answers with the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/product/delete`](api.md#post-apiboardsboard_idcardscard_idproductdelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## rename_account

Change your own display name (1-80 characters). expect, when given, is the name you think it has now; if someone changed it meanwhile the rename is refused with the current name, so nothing is overwritten unseen.

|  |  |
|---|---|
| Route | [`POST /api/account/rename`](api.md#post-apiaccountrename) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `display_name` | string | yes | 1 to 80 characters |  |
| `expect` | string or null | no |  |  |

## rename_folder

Rename one of your home page folders.

|  |  |
|---|---|
| Route | [`POST /api/folders/{folder_id}`](api.md#post-apifoldersfolder_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `folder_id` | integer | yes |  | A folder's id, from list_boards (folders). |
| `name` | string | yes |  |  |

## restore_card

Bring a deleted card back out of the board's trash. Card ids are in list_trash. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/trash/{card_id}/restore`](api.md#post-apiboardsboard_idtrashcard_idrestore) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |

## revoke_my_other_sessions

End every session your account has: through an API key there is no current device to keep, so every browser signs in again. API keys keep working. Answers with how many were removed.

|  |  |
|---|---|
| Route | [`POST /api/account/sessions/revoke-others`](api.md#post-apiaccountsessionsrevoke-others) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## revoke_my_session

End one of your signed-in sessions; that browser has to sign in again.

|  |  |
|---|---|
| Route | [`POST /api/account/sessions/{session_id}/revoke`](api.md#post-apiaccountsessionssession_idrevoke) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `session_id` | string | yes |  | The device's id, from GET /api/account/sessions or GET /api/admin/users/{user_id}/sessions (list_my_sessions or admin_list_user_sessions over MCP). |

## revoke_share

Stop one of the board's share links working (link_id from list_shares). It stays in the list, marked revoked. Only the board's owner may.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/shares/{link_id}/revoke`](api.md#post-apiboardsboard_idshareslink_idrevoke) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `link_id` | string | yes | at least 1 characters | A share link's id, from list_shares. |

## send_card

Move a card to another board, whole: it keeps its link, comments and picture. Labels are matched by name there (made if missing), assignees kept only if they are on that board, and per-person shares dropped. Needs edit rights on both boards. Answers the card where it landed, with card_path and board_title.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/send`](api.md#post-apiboardsboard_idcardscard_idsend) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `list_id` | integer | yes |  | The column on that board it goes into, from get_board (board.lists). |
| `mirror` | string | no | one of ``, `one_way`, `two_way` | Copy only. Empty for a plain copy; one_way keeps the copy following the original (the copy's synced fields become read-only); two_way keeps both in step. Synced: title, description, checklist, dates, price, picture, To do/Doing/Done, comments. |
| `position` | number or null | no |  | An exact place in the column, for putting a card back where it was; wins over top. |
| `to_board` | string or integer | yes |  | The board it goes to: id or URL slug, from list_boards. |
| `top` | boolean | no |  | Put it at the top of the column instead of the end. |

## send_test_notification

Send yourself a test notification the way your settings say. push_disabled: the server sends none; push_no_key: nothing set up to reach you; push_rejected: Pushover said no (detail has its words); push_unreachable: the push service could not be reached.

|  |  |
|---|---|
| Route | [`POST /api/notify/test`](api.md#post-apinotifytest) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## set_board_folder

File one of your boards in one of your folders on your home page, or take it out (folder_id 0).

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/folder`](api.md#post-apiboardsboard_idfolder) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `folder_id` | integer | yes |  | The folder to file it in; 0: not in a folder. |

## set_board_subscription

Choose which of the board's events reach you, or clear your choice (events null) so your defaults apply again. Answers as get_board_subscription does.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/subscription`](api.md#post-apiboardsboard_idsubscription) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `events` | object or null | yes |  | Which events to be told about; null or absent clears the override, so your defaults apply again. |
| `events.activity` | boolean | no |  |  |
| `events.assigned` | boolean | yes |  |  |
| `events.changes` | boolean | yes |  |  |
| `events.due` | boolean | yes |  |  |

## set_card_subscription

Choose which of the card's events reach you, or clear your choice (events null) so the board's or your defaults apply again. Answers as get_card_subscription does.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/subscription`](api.md#post-apiboardsboard_idcardscard_idsubscription) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `events` | object or null | yes |  | Which events to be told about; null or absent clears the override, so your defaults apply again. |
| `events.activity` | boolean | no |  |  |
| `events.assigned` | boolean | yes |  |  |
| `events.changes` | boolean | yes |  |  |
| `events.due` | boolean | yes |  |  |

## set_display

How times and boards are shown to you. timezone (empty for the browser's own) and hour12 (null for the locale's) are always set from what is sent; every other field changes only when sent: number_cards, plain_icons (grey status icons), done_on_top, add_on_top, theme (clear_theme goes back to plain light), own_theme_on_boards (keep your theme on boards that have their own), swipe_columns (phones) and folder_sort. Answers your display settings.

|  |  |
|---|---|
| Route | [`POST /api/account/display`](api.md#post-apiaccountdisplay) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `add_on_top` | boolean or null | no |  |  |
| `clear_theme` | boolean | no |  |  |
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
| `theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.light` | object or null | no |  |  |
| `theme.light.accent` | string | yes |  | The main color: what you can press. |
| `theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.mode` | string | yes |  |  |
| `theme.overrides` | map of string | no |  |  |
| `theme.second` | string | no |  |  |
| `theme.surface` | string | no |  |  |
| `timezone` | string | no |  |  |

## set_label_colors

Replace your own saved label colours with this list, in its order. Labels already using a colour you leave out keep it. Answers your display settings.

|  |  |
|---|---|
| Route | [`POST /api/account/label-colors`](api.md#post-apiaccountlabel-colors) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `colors` | array of object | yes |  | Your whole list of saved label colours, in order; it replaces the one stored. |
| `colors[].color` | string | yes |  |  |
| `colors[].name` | string | yes |  |  |

## set_my_board_view

How a board looks to you alone: greyscale status icons (plain_icons), or null to follow your account default. Nobody else on the board sees it.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/my-view`](api.md#post-apiboardsboard_idmy-view) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `plain_icons` | boolean or null | no |  | true for greyscale status icons on this board, false for coloured; null or left out follows your account default. |

## set_my_collapsed_lists

Which columns of a board you keep collapsed, for you alone: the whole set each time.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/my-view/collapsed`](api.md#post-apiboardsboard_idmy-viewcollapsed) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_ids` | array of integer | no |  | Every column you keep collapsed on this board (the whole set; ids from get_board's board.lists). Empty expands them all. |

## set_notify_prefs

Replace your notification settings (display settings are kept). An empty pushover_user_key keeps the saved one; "-" removes it. notify_via left out stays as it is. Answers the settings as get_notify_prefs does. Read them with get_notify_prefs first and send them back changed: what you leave out is cleared.

|  |  |
|---|---|
| Route | [`POST /api/notify/prefs`](api.md#post-apinotifyprefs) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `all_day_reminder_hour` | integer | no | 0 to 23 | The hour a card due on a day (no time) is reminded about. |
| `assignee_defaults` | object | yes |  | What to be told about cards you are assigned to. |
| `assignee_defaults.activity` | boolean | no |  |  |
| `assignee_defaults.assigned` | boolean | yes |  |  |
| `assignee_defaults.changes` | boolean | yes |  |  |
| `assignee_defaults.due` | boolean | yes |  |  |
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

## set_theme

Switch to another theme from your list (or the defaults), by its id.

|  |  |
|---|---|
| Route | [`POST /api/account/theme`](api.md#post-apiaccounttheme) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `theme_id` | string | yes |  | A theme's id. Empty for the first in the list. |

## set_themes

Replace your whole list of color themes (1 to 40) and pick the one you use. themes: null goes back to following the app's defaults. A theme is the Theme schema in the API reference (config/openapi.json): id, name, mode light|dark, accent, second, and optional colors as #rrggbb. whoami shows the current list.

|  |  |
|---|---|
| Route | [`POST /api/account/themes`](api.md#post-apiaccountthemes) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `theme_id` | string | no |  | The theme you use. |
| `themes` | array of object or null | no |  | Your whole list, 1 to 40 themes. null goes back to following the app's defaults. |
| `themes[].accent` | string | yes |  | The main color: what you can press. |
| `themes[].background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].button` | object or null | no |  | How the theme's buttons look. A choice left out is the default (rounded, normal, raised, normal). |
| `themes[].button.case` | string | no | one of `normal`, `upper` |  |
| `themes[].button.depth` | string | no | one of `flat`, `raised`, `glossy` |  |
| `themes[].button.shape` | string | no | one of `square`, `soft`, `rounded`, `pill` |  |
| `themes[].button.weight` | string | no | one of `normal`, `medium`, `bold` |  |
| `themes[].dark` | object or null | no |  | A light theme's colors for dark mode, kept while it is light. Switching the theme to dark swaps them in. A light theme's light (or a dark theme's dark) is dropped. |
| `themes[].dark.accent` | string | yes |  | The main color: what you can press. |
| `themes[].dark.background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `themes[].id` | string | yes |  | Lower-case letters, digits, - and _, up to 32. |
| `themes[].labels` | array of object | no |  |  |
| `themes[].labels[].color` | string | yes |  |  |
| `themes[].labels[].name` | string | yes |  |  |
| `themes[].light` | object or null | no |  | A dark theme's colors for light mode, kept while it is dark. Switching the theme to light swaps them in. |
| `themes[].light.accent` | string | yes |  | The main color: what you can press. |
| `themes[].light.background` | string | no |  | The page. Left out: the mode's own. |
| `themes[].light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `themes[].mode` | string | yes | one of `light`, `dark` |  |
| `themes[].name` | string | yes |  | 1 to 60 characters. |
| `themes[].overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `themes[].second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `themes[].surface` | string | no |  | Cards. Left out: the mode's own. |

## share_card_with_user

Share one card with an account, to view or to edit, without putting them on the board; mode empty stops sharing it with them. Only the board's owner may. Answers the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/share-user`](api.md#post-apiboardsboard_idcardscard_idshare-user) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `mode` | string | yes | one of `view`, `edit`, `` | view or edit; empty stops sharing the card with them. |
| `user_id` | integer | yes |  | The account to share the card with, from list_team. |

## unlink_card

End one of a card's mirror links (link_id from list_card_links). Both cards stay as they are. Needs edit rights on the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/links/{link_id}/delete`](api.md#post-apiboardsboard_idcardscard_idlinkslink_iddelete) |
| Read only | no |
| Destructive | yes |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `link_id` | integer | yes |  | A mirror link's id, from list_card_links. |

## update_board

Change any of a board's settings; leave out what should not change: its title, URL name (slug), layout (free, three, staged or calendar), default view, colour, theme, card numbering, where finished cards and the add-card box go, the stage names and buttons of a staged board, carry-over on a calendar board, shopping, ranking, and its purpose and notes. Needs edit rights on the whole board. The purpose and notes are for members, not share links; the quick-share and anonymous-comment switches are the owner's, from a browser. Answers the board's new revs. To change only the purpose or notes, update_board_notes is the safer call: it insists on the rev you read them at.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}`](api.md#post-apiboardsboard_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `add_placement` | string or null | no | one of ``, `top`, `bottom` | Where the add-card box sits in a column: top, bottom, or "" for each person's account setting. |
| `allow_anon_comments` | boolean or null | no |  | Let guests comment without giving a name. The owner's, from a browser only: refused through a key. |
| `allow_quick_share` | boolean or null | no |  | Let the board's cards be shared by quick link. The owner's, from a browser only: refused through a key. |
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
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
| `slug` | string or null | no |  | The board's URL name (the part after /b/). Empty picks one from the title. 409 slug_taken if another board has it. |
| `stage_doing` | string or null | no |  | What the Doing stage is called on this board; empty for the default. |
| `stage_done` | string or null | no |  | What the Done stage is called on this board; empty for the default. |
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
| `theme.dark.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.dark.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.dark.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.light` | object or null | no |  |  |
| `theme.light.accent` | string | yes |  | The main color: what you can press. |
| `theme.light.background` | string | no |  | The page. Left out: the mode's own. |
| `theme.light.overrides` | map of string | no |  | Worked-out colors replaced by name: text, accent, accentFg, second, secondFg, secondText, link, navLink, background, surface, nested, chrome, info, infoBorder, border, and for each button variant V (Secondary, Success, Danger, Warning, Info, Light, Dark) btnV, btnVFg and btnVText. Leaving one out puts the readable color back. |
| `theme.light.second` | string | yes |  | Says where you are: active nav item, headings, info. |
| `theme.light.surface` | string | no |  | Cards. Left out: the mode's own. |
| `theme.mode` | string | yes |  |  |
| `theme.overrides` | map of string | no |  |  |
| `theme.second` | string | no |  |  |
| `theme.surface` | string | no |  |  |
| `title` | string or null | no | at most 120 characters |  |

## update_board_notes

Replace a board's notes, its purpose, or both; leave out what should not change. A board's purpose can only be written while its Purpose section is on, which a person does in Board settings (409 purpose_off otherwise). Each is the whole new markdown text, so to add a link, send the old notes with it added. Each needs the rev it was read at, from get_board_notes or get_board: if someone edited the text since, the change is refused (409 board_text_stale) rather than written over theirs; read it again and redo the change on top. Needs edit rights on the board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/notes`](api.md#post-apiboardsboard_idnotes) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `notes` | string or null | no |  | Markdown: links, documents, an index. The whole new text. |
| `notes_rev` | integer or null | no |  | Required with notes: the notes_rev they were read at. |
| `purpose` | string or null | no |  | Markdown: what the board is for and how to work on it. The whole new text. |
| `purpose_rev` | integer or null | no |  | Required with purpose: the purpose_rev it was read at. |

## update_card

Change any of a card's fields; leave out what should not change. Moving it (list_id, position) or archiving it needs edit rights on its column, and on the whole board to move it to another column. done marks it complete; doing and rank_place are the Doing stage and board-wide importance of a staged board. archived: true hides it from the board (reversible). Pass the card's rev when changing title or description: if someone edited it since, the change is refused (409 card_stale, with the card as it now stands) rather than written over theirs.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}`](api.md#post-apiboardsboard_idcardscard_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides it from the board (reversible); false brings it back. |
| `assignees` | array of integer or null | no |  | The full set of user ids; replaces the current assignees. |
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
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
| `start_at` | string or null | no |  | 2026-10-30 for an all-day date, an RFC 3339 timestamp for a time, an empty string to clear. |
| `title` | string or null | no |  |  |

## update_checklist_item

Tick an item off (done: true) or back on, change its text, or move it. Item ids are in the card's checklist. Returns the card. Needs edit rights on the card.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/cards/{card_id}/checklist/{item_id}`](api.md#post-apiboardsboard_idcardscard_idchecklistitem_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `card_id` | integer | yes |  | A card's id, from get_board or find_cards. |
| `done` | boolean or null | no |  | true ticks it off, false back on. |
| `item_id` | integer | yes |  | A checklist item's id, from get_card. |
| `position` | number or null | no |  | Its place in the checklist; lower comes first. |
| `text` | string or null | no |  |  |

## update_label

Rename a label or change its colour; leave out what should not change. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/labels/{label_id}`](api.md#post-apiboardsboard_idlabelslabel_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `color` | string or null | no |  | A palette name (blue, green, orange, red, purple, pink, teal, gray, yellow) or a #rrggbb colour. |
| `label_id` | integer | yes |  | A label's id, from get_board (board.labels). |
| `name` | string or null | no | at most 40 characters |  |

## update_list

Change a column's title, position or width, archive it (archived: true hides it; false brings it back), make it a shopping column, or on a staged board fix the stage it shows for everyone (stage_view; empty follows the navbar) or give it its own To do/Doing/Done buttons (stage_tabs: on, off, or empty to follow the board's setting). Leave out what should not change. Needs edit rights on the whole board.

|  |  |
|---|---|
| Route | [`POST /api/boards/{board_id}/lists/{list_id}`](api.md#post-apiboardsboard_idlistslist_id) |
| Read only | no |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|
| `archived` | boolean or null | no |  | true hides the column; false brings it back. |
| `board` | string or integer | yes |  | The board's id, or its URL slug (the part after /b/), from list_boards. |
| `list_id` | integer | yes |  | A column's id, from get_board (board.lists). |
| `position` | number or null | no |  | Sort position among the columns; lower comes first. |
| `shopping` | boolean or null | no |  | A shopping column: its cards carry products and prices. |
| `stage_tabs` | string or null | no | one of ``, `on`, `off` | On a staged board, this column's own To do / Doing / Done buttons: on, off, or empty to follow the board's setting. |
| `stage_view` | string or null | no | one of ``, `todo`, `doing`, `done` | On a staged board, the stage this column shows for everyone; empty follows the navbar. |
| `title` | string or null | no | at most 120 characters |  |
| `width` | integer or null | no |  | The column's width in pixels (200-800); 0 puts it back to the standard width. |

## whoami

The account this API key belongs to, the role it acts with (role -- never higher than the key's own scope), and the account's own role.

|  |  |
|---|---|
| Route | [`GET /api/me`](api.md#get-apime) |
| Read only | yes |
| Destructive | no |

**Arguments**

| Field | Type | Required | Limits | Description |
|---|---|---|---|---|

## HTTP only

Routes an API key may call that have no tool, and why.

| Route | Why |
|---|---|
| [`GET /api/api.md`](api.md#get-apiapimd) | The API reference itself, for people; GET /api/openapi.json is the same as JSON. |
| [`GET /api/backup`](api.md#get-apibackup) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`GET /api/boards/{board_id}/cards/{card_id}/image`](api.md#get-apiboardsboard_idcardscard_idimage) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`GET /api/boards/{board_id}/cards/{card_id}/product-image`](api.md#get-apiboardsboard_idcardscard_idproduct-image) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`GET /api/boards/{board_id}/export`](api.md#get-apiboardsboard_idexport) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`GET /api/boards/{board_id}/lists/{list_id}/export`](api.md#get-apiboardsboard_idlistslist_idexport) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`GET /api/boards/{board_id}/stamp`](api.md#get-apiboardsboard_idstamp) | The board page's cheap change poll; get_board reads the board itself. |
| [`GET /api/mcp.json`](api.md#get-apimcpjson) | The MCP reference itself; tools/list gives the same list. |
| [`GET /api/mcp.md`](api.md#get-apimcpmd) | The MCP reference itself, for people; tools/list gives the same list. |
| [`GET /api/openapi.json`](api.md#get-apiopenapijson) | The API reference itself; GET /api/mcp.json is the MCP one. |
| [`POST /api/backup/board`](api.md#post-apibackupboard) | A backup file's contents, which the page sends in steps; deliberately not a tool. |
| [`POST /api/boards/{board_id}/cards/{card_id}/image`](api.md#post-apiboardsboard_idcardscard_idimage) | A file (a picture, a spreadsheet or a backup), which travels over HTTP rather than as a tool answer. |
| [`POST /api/boards/{board_id}/lists/{list_id}/import`](api.md#post-apiboardsboard_idlistslist_idimport) | A backup file's contents, which the page sends in steps; deliberately not a tool. |
| [`POST /api/logout`](api.md#post-apilogout) | Ends a browser session; a key has none to end. |
| [`POST /api/logs/events`](api.md#post-apilogsevents) | The browser's own activity reports (track.js), not an action. |
| [`POST /mcp`](api.md#post-mcp) | The MCP endpoint itself. |
