package models

import (
	regexp "regexp"
	strings "strings"
)

const (
	ThemeLight = "light"
	ThemeDark  = "dark"
)

// Theme is how a page looks: light or dark, and optionally its own colours.
// A colour left "" keeps what the mode already uses. Colours are stored only
// as #rrggbb, so nothing a user types ends up inside a style as anything
// other than a colour.
//
// A person's theme lives in UserPrefs and follows them to every device; a
// board's lives on the Board and is seen by everyone on it, unless they have
// chosen to keep their own (UserPrefs.OwnThemeOnBoards).
type Theme struct {
	Mode       string `json:"mode"`
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Column     string `json:"column"`
	Card       string `json:"card"`
}

var themeColorPattern = regexp.MustCompile( `^#[0-9a-f]{6}$` )

// Validate cleans what can be cleaned and refuses the rest. Mode "" is light:
// light is the default and stays it.
func ( theme *Theme ) Validate() ( err error ) {
	if theme.Mode == "" { theme.Mode = ThemeLight }
	if theme.Mode != ThemeLight && theme.Mode != ThemeDark { return ErrBadInput }
	for _ , color := range []*string{ &theme.Accent , &theme.Background , &theme.Surface , &theme.Column , &theme.Card } {
		*color = strings.ToLower( strings.TrimSpace( *color ) )
		if *color != "" && themeColorPattern.MatchString( *color ) == false { return ErrBadInput }
	}
	return
}
