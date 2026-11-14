package routes

import (
	fiber "github.com/gofiber/fiber/v3"

	models "todoboard/server/models"
	security "todoboard/server/security"
)

type displayRequest struct {
	Timezone         string        `json:"timezone,omitempty"`
	Hour12           *bool         `json:"hour12,omitempty"`
	NumberCards      *bool         `json:"number_cards,omitempty"`
	PlainIcons       *bool         `json:"plain_icons,omitempty"`
	DoneOnTop        *bool         `json:"done_on_top,omitempty"`
	AddOnTop         *bool         `json:"add_on_top,omitempty"`
	Theme            *models.DisplayTheme `json:"theme,omitempty"`
	ClearTheme       bool          `json:"clear_theme,omitempty"`
	OwnThemeOnBoards *bool         `json:"own_theme_on_boards,omitempty"`
	SwipeColumns     *bool         `json:"swipe_columns,omitempty"`
	FolderSort       *string       `json:"folder_sort,omitempty"`
}

func displayPayload( prefs *models.UserPrefs ) ( payload fiber.Map ) {
	payload = fiber.Map{ "timezone": prefs.DisplayTimezone , "hour12": prefs.Hour12 , "number_cards": prefs.NumberCards , "plain_icons": prefs.PlainIcons , "done_on_top": prefs.DoneOnTop , "add_on_top": prefs.AddOnTop , "theme": prefs.Theme , "own_theme_on_boards": prefs.OwnThemeOnBoards , "swipe_columns": prefs.SwipeColumns == nil || *prefs.SwipeColumns , "folder_sort": prefs.FolderSort , "label_colors": prefs.LabelColors }
	if prefs.LabelColors == nil { payload[ "label_colors" ] = []models.SavedColor{} }
	return
}

type labelColorsRequest struct {
	Colors []models.SavedColor `json:"colors" jsonschema_description:"Your whole list of saved label colours, in order; it replaces the one stored."`
}

// saveLabelColors replaces the person's own label colours with the list
// sent, in its order: the board's label editor adds to it, the account page
// renames and removes. Labels already using a removed colour keep it.
//
// @route("POST /api/account/label-colors", summary="Save your label colours",
//     tag="Account", errors=(400,),
//     description="Replace your own saved label colours with this list, in its order. Labels already using a "
//         "colour you leave out keep it. Answers your display settings.",
//     mcp="set_label_colors",
// )
func ( handlers *Handlers ) saveLabelColors( c fiber.Ctx , in *labelColorsRequest ) ( reply *fiber.Map , err error ) {
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = get_err
		return
	}
	prefs.LabelColors = in.Colors
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , prefs ); save_err != nil {
		err = refuse( fiber.StatusBadRequest , "invalid_label_colors" )
		return
	}
	payload := displayPayload( prefs )
	reply = &payload
	return
}

// saveDisplay stores how this person wants times shown: a timezone ("" for
// the browser's own) and 12- or 24-hour clocks (null for the locale's) --
// and, when sent, whether cards are numbered and status icons grey by
// default, the person's theme (clear_theme back to plain light), and
// whether boards with a theme of their own keep showing the person's, phone
// swipe columns, and how home page folders are ordered.
//
// @route("POST /api/account/display", summary="Save your display settings",
//     tag="Account", errors=(400,),
//     description="How times and boards are shown to you. timezone (empty for the browser's own) and hour12 "
//         "(null for the locale's) are always set from what is sent; every other field changes only when sent: "
//         "number_cards, plain_icons (grey status icons), done_on_top, add_on_top, theme (clear_theme goes back "
//         "to plain light), own_theme_on_boards (keep your theme on boards that have their own), swipe_columns "
//         "(phones) and folder_sort. Answers your display settings.",
//     mcp="set_display",
// )
func ( handlers *Handlers ) saveDisplay( c fiber.Ctx , in *displayRequest ) ( reply *fiber.Map , err error ) {
	body := in
	user := security.UserFrom( c )
	prefs , get_err := models.GetUserPrefs( handlers.Store , user.ID )
	if get_err != nil {
		err = get_err
		return
	}
	prefs.DisplayTimezone , prefs.Hour12 = body.Timezone , body.Hour12
	if body.Timezone != "" { prefs.Timezone = body.Timezone }
	if body.NumberCards != nil { prefs.NumberCards = *body.NumberCards }
	if body.PlainIcons != nil { prefs.PlainIcons = *body.PlainIcons }
	if body.DoneOnTop != nil { prefs.DoneOnTop = *body.DoneOnTop }
	if body.AddOnTop != nil { prefs.AddOnTop = *body.AddOnTop }
	if body.OwnThemeOnBoards != nil { prefs.OwnThemeOnBoards = *body.OwnThemeOnBoards }
	if body.SwipeColumns != nil { prefs.SwipeColumns = body.SwipeColumns }
	if body.FolderSort != nil { prefs.FolderSort = *body.FolderSort }
	if body.Theme != nil {
		if body.Theme.Validate() != nil {
			err = refuse( fiber.StatusBadRequest , "invalid_theme" )
			return
		}
		prefs.Theme = body.Theme
	}
	if body.ClearTheme { prefs.Theme = nil }
	if save_err := models.SaveUserPrefs( handlers.Store , user.ID , prefs ); save_err != nil {
		err = refuse( fiber.StatusBadRequest , "invalid_timezone" )
		return
	}
	payload := displayPayload( prefs )
	reply = &payload
	return
}
