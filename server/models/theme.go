package models

import (
	regexp "regexp"
	strings "strings"
)

const (
	ThemeLight = "light"
	ThemeDark  = "dark"
)

// DisplayTheme is one theme kept on its own rather than in a list: a board's
// (Board.Theme, seen by everyone on it unless they keep their own, see
// UserPrefs.OwnThemeOnBoards), and the single theme an account had before the
// Appearance screen's lists (UserPrefs.Theme, read by theme.js as a theme in
// the list until the person saves a list of their own).
//
// It is a Theme (themes.go) without the id and name: a mode, the main and
// second colors, the page and card colors, overrides and a button style.
// Column and Card are how boards were themed before overrides: theme.js reads
// them as the "column" and "face" overrides, and a theme saved by today's
// editor leaves them empty. Every color may be "", which theme.js fills in
// from the mode, so a theme saved before "second" existed still shows.
// Colors are stored only as #rrggbb, so nothing a user types ends up inside a
// style as anything other than a color.
type DisplayTheme struct {
	Mode       string            `json:"mode"`
	Accent     string            `json:"accent"`
	Second     string            `json:"second,omitempty"`
	Background string            `json:"background"`
	Surface    string            `json:"surface"`
	Column     string            `json:"column"`
	Card       string            `json:"card"`
	Overrides  map[string]string `json:"overrides,omitempty"`
	Button     *ThemeButton      `json:"button,omitempty"`
}

var themeColorPattern = regexp.MustCompile( `^#[0-9a-f]{6}$` )

// Validate cleans what can be cleaned and refuses the rest. Mode "" is light:
// light is the default and stays it.
func ( theme *DisplayTheme ) Validate() ( err error ) {
	if theme.Mode == "" { theme.Mode = ThemeLight }
	if theme.Mode != ThemeLight && theme.Mode != ThemeDark { return ErrBadInput }
	for _ , color := range []*string{ &theme.Accent , &theme.Second , &theme.Background , &theme.Surface , &theme.Column , &theme.Card } {
		*color = strings.ToLower( strings.TrimSpace( *color ) )
		if *color != "" && themeColorPattern.MatchString( *color ) == false { return ErrBadInput }
	}
	overrides := map[string]string{}
	for field , color := range theme.Overrides {
		color = strings.ToLower( strings.TrimSpace( color ) )
		if ThemeOverrideFields[ field ] == false || themeColorPattern.MatchString( color ) == false { return ErrBadInput }
		overrides[ field ] = color
	}
	theme.Overrides = nil
	if len( overrides ) > 0 { theme.Overrides = overrides }
	if theme.Button , err = cleanThemeButton( "" , theme.Button ); err != nil { return ErrBadInput }
	return
}
