// How the page looks: light, dark, or another theme.
//
// Light is the default and stays it -- never the OS setting. Dark is
// Bootstrap's own dark mode (<html data-bs-theme="dark">). Light is the
// attribute being absent, and Bootstrap's color-mode script, which would
// follow prefers-color-scheme, is deliberately not included.
//
// A theme is a mode and a few colors. It can also carry its colors for the
// other mode (a light theme's "dark", a dark theme's "light"), so switching
// it between light and dark swaps them in -- see setMode(). colors() works out everything else
// the page needs from those -- text, links, borders, tints -- each pushed until
// it is readable. Its overrides replace any of the worked-out colors by name,
// and dropping one puts the readable color back.
//
// Which themes there are, and which one someone uses, is kept on their
// account (models/themes.go): their own list, or until they change something,
// the defaults an admin set, or until an admin does, the built-in ones below.
// The Appearance screen (themes_editor.js) edits both lists.
//
// The theme in use is also kept in this browser, whole, so the first paint --
// and the pages before signing in -- are already in it. The account is the
// authority: load( me ) replaces what was kept as soon as /api/me answers.
//
// This file is loaded in <head>, before the stylesheet's first paint, so a
// saved dark theme never flashes light on the way in.
//
// TodoBoard adds three things to the template's themes:
//   - A board can have a theme of its own (Board.theme), which everyone on it
//     sees unless they ticked "Always use my own theme on boards"
//     (ownOnBoards). setBoard() takes it; the board page's last one is kept per
//     page address, so a themed board paints in its theme from the start.
//   - Boards have columns and card faces: two more worked-out colors, column
//     and face, as --app-column and --app-face (board.css).
//   - Before the Appearance screen each account had one theme, and a board's
//     theme is still one theme on its own (models.DisplayTheme): fromDisplay()
//     reads either as a theme, and toDisplay() writes a board's back.
//     An account's old theme shows as "My theme" in the list until the person
//     saves a list of their own, which takes it in.
// A guest on a share link has no account: their pick is kept in this browser.
//
// Brand colors live in ThemePalettes, named, whether a theme uses them or not,
// so the rest are there for whoever needs them next.
const ThemePalettes = {
  // Wright State University. Primary: green and gold. The rest are its accents.
  wsu: {
    green: "#046a38",
    gold: "#cba052",
    dark_green: "#00482b",
    deep_gold: "#ae8643",
    yellow: "#f8e08e",
    leaf: "#719949",
    beige: "#dcd59a",
    black: "#101820",
    faded_gold: "#e7e3c6",
    off_white: "#f2f1f0",
  },
};

const Theme = {
  storageKey: "theme",
  // What redraws when the theme changes: the Appearance screen.
  onChange: [],

  // The built-in themes: what everyone has until an admin saves defaults. accent
  // is the main color: what you can press. second says where you are: the
  // active nav item, headings, progress, info. background and surface are the
  // page and the cards, when the mode's will not do. Names come from
  // language.yaml (theme.preset_<id>) until a list is saved; a saved list keeps
  // its own names.
  //
  // Every one has colors for both modes: the ones it starts in, and the other
  // mode's under its name, which the Light/Dark switch swaps in (setMode).
  // Default comes first and starts light: it is what a new browser shows.
  builtIn: [
    { id: "default" , mode: "light" , accent: "#1f5fd6" , second: "#0b7a75" ,
      dark: { accent: "#6ea0ff" , second: "#5fd3c4" } },
    { id: "ocean" , mode: "light" , accent: "#0b7285" , second: "#bd5418" , background: "#eef6f8" , surface: "#ffffff" ,
      dark: { accent: "#3bc9db" , second: "#ff922b" , background: "#0b1a1f" , surface: "#12262d" } },
    { id: "forest" , mode: "light" , accent: "#28843b" , second: "#8a6a12" , background: "#f1f6ef" , surface: "#ffffff" ,
      dark: { accent: "#69db7c" , second: "#e0b84c" , background: "#0f1a12" , surface: "#17251a" } },
    { id: "sepia" , mode: "light" , accent: "#9c5a1c" , second: "#2f6f6a" , background: "#f6f0e4" , surface: "#fbf7ef" ,
      dark: { accent: "#e0a15e" , second: "#6cc4bb" , background: "#1c1712" , surface: "#262019" } },
    { id: "grape" , mode: "light" , accent: "#7048c8" , second: "#c0267e" , background: "#f5f2fc" , surface: "#ffffff" ,
      dark: { accent: "#b197fc" , second: "#f783ac" , background: "#15111f" , surface: "#1e182c" } },
    { id: "pink" , mode: "light" , accent: "#c2255c" , second: "#6b3fc0" , background: "#fdf0f5" , surface: "#ffffff" ,
      dark: { accent: "#f783ac" , second: "#b197fc" , background: "#1f1018" , surface: "#2a1621" } },
    // Wright State University: Green on Off White; switched to dark, Gold on Black with Dark Green panels.
    { id: "wsu" , mode: "light" , accent: ThemePalettes.wsu.green , second: ThemePalettes.wsu.gold , background: ThemePalettes.wsu.off_white , surface: "#ffffff" ,
      dark: { accent: ThemePalettes.wsu.gold , second: ThemePalettes.wsu.yellow , background: ThemePalettes.wsu.black , surface: ThemePalettes.wsu.dark_green } },
    { id: "midnight" , mode: "dark" , accent: "#8c7cff" , second: "#ffb86b" , background: "#0e1020" , surface: "#171a30" ,
      light: { accent: "#5b4bd6" , second: "#c26a00" , background: "#f3f3fb" , surface: "#ffffff" } },
    { id: "slate" , mode: "dark" , accent: "#4fb3a9" , second: "#e6b450" , background: "#15191e" , surface: "#1d232a" ,
      light: { accent: "#2b7a72" , second: "#a0711a" , background: "#f2f4f6" , surface: "#ffffff" } },
    { id: "cerise" , mode: "dark" , accent: "#f06595" , second: "#ffa8d0" , background: "#1f0d16" , surface: "#2a1220" ,
      light: { accent: "#b0164c" , second: "#e64980" , background: "#fdf2f6" , surface: "#ffffff" } },
  ],

  // What a theme leaves out.
  modes: {
    light: { background: "#f7f8fa" , surface: "#ffffff" , text: "#212529" },
    dark: { background: "#101215" , surface: "#191c21" , text: "#dee2e6" },
  },

  // Bootstrap's other button colors, each worked out per theme: a fill, the
  // label on it (Fg), and the outline version's label and border on a card
  // (Text). primary is the accent and has its own fields above.
  variants: [ "secondary" , "success" , "danger" , "warning" , "info" , "light" , "dark" ],
  // What success, danger and warning start from in each mode, before the
  // theme's accent is mixed in (buttonHarmony percent) so they belong to it.
  semantic: {
    light: { success: "#198754" , danger: "#dc3545" , warning: "#ffc107" },
    dark: { success: "#3fbf7f" , danger: "#ef6b75" , warning: "#ffcd39" },
  },
  buttonHarmony: 12,

  // A theme's button: shape, weight, depth and case. Missing or unknown
  // choices fall back to buttonDefault: rounded, normal weight, raised, as written.
  buttonDefault: { shape: "rounded" , weight: "normal" , depth: "raised" , case: "normal" },
  buttonChoices: {
    shape: { square: { "--app-btn-radius": "0" } , soft: { "--app-btn-radius": "0.375rem" } ,
      rounded: { "--app-btn-radius": "0.75rem" } , pill: { "--app-btn-radius": "50rem" } },
    weight: { normal: { "--app-btn-weight": "400" } , medium: { "--app-btn-weight": "500" } , bold: { "--app-btn-weight": "700" } },
    depth: {
      flat: { "--app-btn-shadow": "0 0 0 transparent" , "--app-btn-image": "none" },
      raised: { "--app-btn-shadow": "0 1px 2px rgba(0, 0, 0, 0.2), 0 3px 8px rgba(0, 0, 0, 0.14)" , "--app-btn-image": "none" },
      glossy: { "--app-btn-shadow": "inset 0 1px 0 rgba(255, 255, 255, 0.3), 0 1px 3px rgba(0, 0, 0, 0.25)" ,
        "--app-btn-image": "linear-gradient(180deg, rgba(255, 255, 255, 0.24), rgba(255, 255, 255, 0) 55%, rgba(0, 0, 0, 0.08))" },
    },
    case: { normal: { "--app-btn-transform": "none" , "--app-btn-spacing": "normal" } ,
      upper: { "--app-btn-transform": "uppercase" , "--app-btn-spacing": "0.06em" } },
  },

  // TodoBoard: how much of the text color darkens (or lightens) the page for
  // a board column, and a dark card face off its card.
  boardTints: { column: 5 , face: 4 },

  // How much of the second color goes into each layer, in percent, so a nested
  // card stands off its card and the card off the page.
  tints: { page: 3.5 , surface: 1.75 , nested: 8.75 , chrome: 10 , info: 14 , infoBorder: 45 },

  // Field of colors() -> custom property in style.css, which hands them on to
  // Bootstrap. Every one of them can be overridden.
  vars: {
    text: "--app-text",
    accent: "--app-accent",
    accentFg: "--app-accent-fg",
    second: "--app-second",
    secondFg: "--app-second-fg",
    secondText: "--app-second-text",
    link: "--app-link",
    navLink: "--app-nav-link",
    background: "--app-bg",
    surface: "--app-surface",
    nested: "--app-nested",
    chrome: "--app-chrome",
    info: "--app-info",
    infoBorder: "--app-info-border",
    border: "--app-border",
    column: "--app-column",
    face: "--app-face",
  },

  // Pairs that must stay readable: [ front , back ] fields of colors(). check()
  // and scripts/check_themes.js both use them. Names are language.yaml keys
  // (theme.check_<name>).
  pairs: {
    text_card: [ "text" , "surface" ],
    text_nested: [ "text" , "nested" ],
    text_info: [ "text" , "info" ],
    link: [ "link" , "nested" ],
    button: [ "accentFg" , "accent" ],
    heading: [ "secondText" , "nested" ],
    active_nav: [ "secondFg" , "second" ],
    sidebar_link: [ "navLink" , "chrome" ],
    text_column: [ "text" , "column" ],
    text_face: [ "text" , "face" ],
  },

  // The themes on offer, and the one in use (its id).
  list: [],
  // Whether list is the person's own rather than the defaults.
  custom: false,
  current: "default",
  // TodoBoard: the open board's own theme (null: none), whether this person
  // keeps theirs on boards anyway, and whether there is an account to save to.
  board: null,
  boardRaw: "",
  ownOnBoards: false,
  signedIn: false,
  // WCAG AA for text.
  minContrast: 4.5,

  // A built-in theme with its name filled in, ready to be saved in a list.
  named( theme ) {
    const copy = this.copy( theme );
    if ( !copy.name ) copy.name = ( typeof I18n === "undefined" ? "" : I18n.get( "theme.preset_" + theme.id ) ) || theme.id;
    return copy;
  },

  copy( theme ) {
    const copy = Object.assign( {} , theme );
    copy.overrides = Object.assign( {} , theme.overrides || {} );
    if ( theme.button ) copy.button = Object.assign( {} , theme.button );
    // TodoBoard: the labels that come with the theme.
    if ( Array.isArray( theme.labels ) ) copy.labels = theme.labels.map( ( label ) => Object.assign( {} , label ) );
    Object.keys( this.modes ).forEach( ( mode ) => {
      if ( theme[ mode ] ) copy[ mode ] = Object.assign( {} , theme[ mode ] , { overrides: Object.assign( {} , theme[ mode ].overrides || {} ) } );
    } );
    return copy;
  },

  // The fields that belong to one mode. The name and the button style are the
  // theme's whatever the mode.
  modeFields: [ "accent" , "second" , "background" , "surface" , "overrides" ],

  // Switches a theme to light or dark, in place. Its colors for the mode it
  // leaves are kept under that mode's name, and any it has for the mode it
  // goes to come back, so WSU switched to dark is WSU's dark colors and
  // switched back is its light ones again. A theme with nothing kept for the
  // new mode keeps its colors, as before.
  setMode( theme , mode ) {
    if ( theme.mode === mode || !( mode in this.modes ) ) return theme;
    const leaving = {};
    this.modeFields.forEach( ( field ) => { if ( theme[ field ] !== undefined ) leaving[ field ] = theme[ field ]; } );
    const coming = theme[ mode ];
    if ( coming ) {
      this.modeFields.forEach( ( field ) => {
        if ( coming[ field ] !== undefined ) theme[ field ] = coming[ field ];
        else delete theme[ field ];
      } );
      theme.overrides = Object.assign( {} , theme.overrides || {} );
    }
    delete theme[ mode ];
    theme[ theme.mode ] = leaving;
    theme.mode = mode;
    return theme;
  },

  find( id ) {
    return this.list.find( function ( theme ) { return theme.id === id; } ) || this.list[ 0 ] || this.builtIn[ 0 ];
  },

  // What was kept in this browser: a whole theme, or (from before themes could
  // be edited) a built-in one's id.
  restore() {
    let saved = null;
    try {
      const raw = window.localStorage.getItem( this.storageKey ) || "";
      if ( raw.charAt( 0 ) === "{" ) saved = JSON.parse( raw );
      // An id, or from before Light and Dark were one theme, "light" or "dark":
      // Default in that mode.
      else if ( raw in this.modes ) saved = this.setMode( this.copy( this.builtIn[ 0 ] ) , raw );
      else if ( raw !== "" ) saved = this.builtIn.find( function ( theme ) { return theme.id === raw; } ) || null;
    } catch ( error ) {
      // Storage blocked (private window, previews) or unreadable: stay light.
    }
    // Before theme lists, TodoBoard kept the account's one theme here, and
    // earlier still "dark" under "theme" (read above as Default in dark).
    if ( !this.valid( saved ) ) saved = this.fromDisplay( this.read( "theme.user" , {} ).theme , "mine" , "" );
    if ( !this.valid( saved ) ) saved = this.builtIn[ 0 ];
    this.ownOnBoards = this.read( "theme.own" , false ) === true || this.read( "theme.user" , {} ).own === true;
    this.list = this.builtIn.slice();
    if ( !this.builtIn.some( ( theme ) => theme.id === saved.id ) ) this.list.push( saved );
    this.current = saved.id;
    const key = this.boardKey();
    if ( key ) this.takeBoard( this.read( key , null ) );
    this.apply( this.resolved() );
  },

  // A JSON value kept in this browser, or fallback.
  read( key , fallback ) {
    try {
      const raw = window.localStorage.getItem( key );
      return raw ? JSON.parse( raw ) : fallback;
    } catch ( error ) {
      return fallback;
    }
  },

  write( key , value ) {
    try {
      if ( value === null ) window.localStorage.removeItem( key );
      else window.localStorage.setItem( key , JSON.stringify( value ) );
    } catch ( error ) {
      // Not remembered for next time, but still applied now.
    }
  },

  // The board's last theme is kept per page address. A share link's address
  // is a credential, so only a hash of it is kept.
  boardKey() {
    const path = window.location.pathname;
    if ( !/^\/(b|s)\//.test( path ) ) return "";
    let hash = 0;
    for ( let index = 0; index < path.length; index += 1 ) hash = ( hash * 31 + path.charCodeAt( index ) ) | 0;
    return "theme.board." + ( hash >>> 0 ).toString( 36 );
  },

  isColor( value ) {
    return typeof value === "string" && /^#[0-9a-f]{6}$/i.test( value );
  },

  // A DisplayTheme (a board's, or an account's from before theme lists) as a
  // theme, with any colors it keeps for the other mode. Its colors may be missing: the mode's built-in fills them, and the
  // second color is the main one, as those themes had only one. column and
  // card, from before overrides, become the column and face overrides.
  fromDisplay( display , id , name ) {
    if ( !display || typeof display !== "object" ) return null;
    const mode = display.mode === "dark" ? "dark" : "light";
    const base = this.setMode( this.copy( this.builtIn[ 0 ] ) , mode );
    const color = ( value ) => ( this.isColor( value ) ? value.toLowerCase() : "" );
    const accent = color( display.accent ) || base.accent;
    const theme = { id: id , name: name , mode: mode , accent: accent , second: color( display.second ) || color( display.accent ) || base.second };
    if ( color( display.background ) ) theme.background = color( display.background );
    if ( color( display.surface ) ) theme.surface = color( display.surface );
    theme.overrides = {};
    Object.keys( display.overrides || {} ).forEach( ( field ) => {
      if ( color( display.overrides[ field ] ) ) theme.overrides[ field ] = color( display.overrides[ field ] );
    } );
    if ( color( display.column ) && !theme.overrides.column ) theme.overrides.column = color( display.column );
    if ( color( display.card ) && !theme.overrides.face ) theme.overrides.face = color( display.card );
    if ( display.button && typeof display.button === "object" ) theme.button = Object.assign( {} , display.button );
    // The colors kept for the other mode, so the Light/Dark switch swaps them in.
    const other = mode === "light" ? "dark" : "light";
    const kept = display[ other ];
    if ( kept && typeof kept === "object" && color( kept.accent ) && color( kept.second ) ) {
      const colors = { accent: color( kept.accent ) , second: color( kept.second ) , overrides: {} };
      if ( color( kept.background ) ) colors.background = color( kept.background );
      if ( color( kept.surface ) ) colors.surface = color( kept.surface );
      Object.keys( kept.overrides || {} ).forEach( ( field ) => {
        if ( color( kept.overrides[ field ] ) ) colors.overrides[ field ] = color( kept.overrides[ field ] );
      } );
      theme[ other ] = colors;
    }
    return theme;
  },

  // A theme as a board's DisplayTheme, to save on the board.
  toDisplay( theme ) {
    const display = { mode: theme.mode , accent: theme.accent , second: theme.second ,
      background: theme.background || "" , surface: theme.surface || "" , column: "" , card: "" };
    if ( Object.keys( theme.overrides || {} ).length > 0 ) display.overrides = Object.assign( {} , theme.overrides );
    if ( theme.button ) display.button = Object.assign( {} , theme.button );
    // The other mode's colors, kept (Light/Dark switches back to them).
    const other = theme.mode === "light" ? "dark" : "light";
    const kept = theme[ other ];
    if ( kept && this.isColor( kept.accent ) && this.isColor( kept.second ) ) {
      const colors = { accent: kept.accent , second: kept.second };
      if ( kept.background ) colors.background = kept.background;
      if ( kept.surface ) colors.surface = kept.surface;
      if ( Object.keys( kept.overrides || {} ).length > 0 ) colors.overrides = Object.assign( {} , kept.overrides );
      display[ other ] = colors;
    }
    return display;
  },

  // Whether an old theme was a plain mode, with no colors of its own.
  plainDisplay( display ) {
    return [ "accent" , "second" , "background" , "surface" , "column" , "card" ].every( ( field ) => !display[ field ] )
      && Object.keys( display.overrides || {} ).length === 0 && !display.button;
  },

  // Enough of a theme to paint, so a broken stored value never breaks the page.
  valid( theme ) {
    const hex = /^#[0-9a-f]{6}$/i;
    return Boolean( theme && typeof theme === "object" && ( theme.mode === "light" || theme.mode === "dark" )
      && hex.test( theme.accent || "" ) && hex.test( theme.second || "" ) );
  },

  // Keeps the theme in use for the next first paint. Plain Default in light is
  // kept as nothing at all, which is what a new browser starts with anyway.
  keep( theme ) {
    try {
      let plain = theme.id === this.builtIn[ 0 ].id && theme.mode === "light"
        && JSON.stringify( this.colors( theme ) ) === JSON.stringify( this.colors( this.builtIn[ 0 ] ) );
      if ( plain && JSON.stringify( this.buttonStyle( theme ) ) !== JSON.stringify( this.buttonStyle( this.builtIn[ 0 ] ) ) ) plain = false;
      if ( plain ) window.localStorage.removeItem( this.storageKey );
      else window.localStorage.setItem( this.storageKey , JSON.stringify( theme ) );
      // Read by restore() only before theme lists; this browser has moved on.
      window.localStorage.removeItem( "theme.user" );
    } catch ( error ) {
      // Not remembered for next time, but still applied now.
    }
    this.write( "theme.own" , this.ownOnBoards ? true : null );
  },

  apply( theme ) {
    const root = document.documentElement;
    // data-theme too: board.css keys its own dark colors on it.
    if ( theme.mode === "dark" ) {
      root.setAttribute( "data-bs-theme" , "dark" );
      root.setAttribute( "data-theme" , "dark" );
    } else {
      root.removeAttribute( "data-bs-theme" );
      root.removeAttribute( "data-theme" );
    }
    const colors = this.colors( theme );
    Object.keys( this.vars ).forEach( ( field ) => root.style.setProperty( this.vars[ field ] , colors[ field ] ) );
    const style = this.buttonStyle( theme );
    Object.keys( style ).forEach( ( name ) => root.style.setProperty( name , style[ name ] ) );
  },

  // A theme's button choices, each known one or the default.
  button( theme ) {
    const chosen = ( theme && theme.button ) || {};
    const result = {};
    Object.keys( this.buttonDefault ).forEach( ( key ) => {
      result[ key ] = chosen[ key ] in this.buttonChoices[ key ] ? chosen[ key ] : this.buttonDefault[ key ];
    } );
    return result;
  },

  // The custom properties style.css reads for every button's shape.
  buttonStyle( theme ) {
    const button = this.button( theme );
    return Object.assign( {} , ...Object.keys( button ).map( ( key ) => this.buttonChoices[ key ][ button[ key ] ] ) );
  },

  changed() {
    this.onChange.forEach( ( listener ) => listener() );
  },

  // The account's themes, from /api/me (or a save's reply). A person who never
  // picked one on their account keeps what this browser had, if it is on offer.
  load( me ) {
    const kept = this.current;
    this.signedIn = true;
    this.custom = Boolean( me.themes_custom );
    this.list = Array.isArray( me.themes ) && me.themes.length > 0
      ? me.themes.map( ( theme ) => this.copy( theme ) )
      : this.builtIn.map( ( theme ) => this.named( theme ) );
    let id = me.theme_id;
    // TodoBoard: the one theme the account had before theme lists. A plain
    // light or dark is that built-in theme; anything else joins the list as
    // "My theme" until the person saves a list of their own, which takes it
    // in. The server keeps the old one, so going back to the defaults (or
    // undoing the first change) offers it again and nothing is lost. A
    // save's reply has no display; the last one known stands in.
    const display = me.display || ( me.themes_custom === undefined ? null : this.lastDisplay );
    if ( me.display ) this.lastDisplay = me.display;
    if ( display ) this.ownOnBoards = Boolean( display.own_theme_on_boards );
    const old = display && !this.custom ? display.theme : null;
    if ( old && this.plainDisplay( old ) ) {
      // Default in that mode. The list is the built-ins here, a fresh copy.
      if ( !id ) {
        id = this.builtIn[ 0 ].id;
        if ( old.mode === "dark" ) this.setMode( this.find( id ) , "dark" );
      }
    } else if ( old && !this.list.some( ( theme ) => theme.id === "mine" ) ) {
      this.list.push( this.fromDisplay( old , "mine" , ( typeof I18n === "undefined" ? "" : I18n.get( "theme.mine_name" ) ) || "My theme" ) );
      if ( !id ) id = "mine";
    }
    this.current = this.find( id || kept ).id;
    this.show();
  },

  // A guest on a share link: the built-in themes, picked in this browser.
  loadGuest() {
    const kept = this.find( this.current );
    this.signedIn = false;
    this.custom = false;
    this.ownOnBoards = false;
    this.list = this.builtIn.map( ( theme ) => this.named( theme ) );
    if ( !this.builtIn.some( ( theme ) => theme.id === kept.id ) ) this.list.push( kept );
    this.show();
  },

  // Whether boards should keep this person's own theme; from the Appearance screen.
  setOwnOnBoards( own ) {
    this.ownOnBoards = Boolean( own );
    if ( this.lastDisplay ) this.lastDisplay.own_theme_on_boards = this.ownOnBoards;
    this.show();
  },

  // The open board's theme (a DisplayTheme), or null when it has none.
  setBoard( display ) {
    const raw = display ? JSON.stringify( display ) : "";
    if ( raw === this.boardRaw ) return;
    this.takeBoard( display );
    const key = this.boardKey();
    if ( key ) this.write( key , display || null );
    this.show();
  },

  takeBoard( display ) {
    this.boardRaw = display ? JSON.stringify( display ) : "";
    this.board = display ? this.fromDisplay( display , "board" , "" ) : null;
  },

  // The theme on the page: the board's, unless there is none or this person
  // keeps their own.
  resolved() {
    if ( this.board && !this.ownOnBoards ) return this.board;
    return this.find( this.current );
  },

  // Paints the theme in use and keeps it for next time.
  show() {
    const theme = this.find( this.current );
    this.apply( this.resolved() );
    this.keep( theme );
    this.changed();
  },

  // Picking is undoable, and quick picks merge into one step.
  pick( id ) {
    const previous = this.current;
    if ( previous === id ) return Promise.resolve();
    const choose = async ( chosen ) => {
      this.current = this.find( chosen ).id;
      this.show();
      if ( this.signedIn ) await Api.setTheme( this.current );
    };
    // The board page has no Undo; a pick there is one press to put back.
    if ( typeof Undo === "undefined" ) return choose( id );
    return Undo.run( {
      label: I18n.get( "undo.theme" ),
      merge: "theme",
      screen: null,
      do: () => choose( id ),
      undo: () => choose( previous ),
    } );
  },

  // Saves someone's whole list (null: back to the defaults) and which one they
  // use, and takes the server's answer as the new state.
  async store( list , id ) {
    const reply = await Api.setThemes( list , id );
    this.load( reply );
  },

  // Every color the page needs, from a theme's few, before its overrides.
  // Fills keep the color as chosen. Anything read as text is pushed until it
  // is readable, on the nested card, the most tinted place it can land: WSU's
  // gold is gold on a progress bar and darker in a heading.
  derived( theme ) {
    const mode = this.modes[ theme.mode ] || this.modes.light;
    const background = theme.background || mode.background;
    const surface = theme.surface || mode.surface;
    const second = theme.second;
    const card = this.mix( second , surface , this.tints.surface );
    const nested = this.mix( second , surface , this.tints.nested );
    const chrome = this.mix( second , surface , this.tints.chrome );
    const result = {
      text: mode.text,
      accent: theme.accent,
      accentFg: this.readableOn( theme.accent ),
      second: second,
      secondFg: this.readableOn( second ),
      secondText: this.asText( second , nested ),
      link: this.asText( theme.accent , nested ),
      navLink: this.asText( theme.accent , chrome ),
      background: this.mix( second , background , this.tints.page ),
      surface: card,
      nested: nested,
      chrome: chrome,
      info: this.mix( second , card , this.tints.info ),
      infoBorder: this.mix( second , card , this.tints.infoBorder ),
      // From the card, not the page: most borders sit on a card, and a card can be
      // a long way from its page (WSU's dark green panels on black).
      border: this.mix( mode.text , card , 20 ),
    };
    // TodoBoard's boards: a column a shade off the page, and card faces on it
    // (in dark, a shade lighter than a card, as they sit on a dark column).
    result.column = this.mix( mode.text , result.background , this.boardTints.column );
    result.face = theme.mode === "dark" ? this.mix( mode.text , card , this.boardTints.face ) : card;
    // The other buttons. A fill stays close to its color but is moved until
    // black or white reads on it; an outline's label is read on the card.
    const semantic = this.semantic[ theme.mode ] || this.semantic.light;
    const fills = {
      secondary: this.mix( second , this.mix( mode.text , card , 58 ) , 15 ),
      success: this.mix( theme.accent , semantic.success , this.buttonHarmony ),
      danger: this.mix( theme.accent , semantic.danger , this.buttonHarmony ),
      warning: this.mix( theme.accent , semantic.warning , this.buttonHarmony ),
      info: second,
      light: this.mix( second , "#f8f9fa" , 10 ),
      dark: this.mix( second , "#212529" , 12 ),
    };
    this.variants.forEach( ( variant ) => {
      const name = this.buttonField( variant );
      const fill = this.asFill( fills[ variant ] );
      result[ name ] = fill;
      result[ name + "Fg" ] = this.readableOn( fill );
      result[ name + "Text" ] = this.asText( fill , card );
    } );
    return result;
  },

  // "danger" -> "btnDanger", the field of colors() for that button's fill.
  buttonField( variant ) {
    return "btn" + variant.charAt( 0 ).toUpperCase() + variant.slice( 1 );
  },

  // This color, moved darker (or lighter) until black or white reads on it.
  asFill( color ) {
    const label = this.readableOn( color );
    const away = label === "#ffffff" ? "#000000" : "#ffffff";
    for ( let step = 0 ; step <= 100 ; step += 2 ) {
      const tried = this.mix( away , color , step );
      if ( this.contrast( tried , label ) >= this.minContrast ) return tried;
    }
    return away;
  },

  // What the page uses: the worked-out colors, with the theme's overrides on top.
  colors( theme ) {
    const colors = this.derived( theme );
    Object.keys( theme.overrides || {} ).forEach( ( field ) => {
      if ( field in colors ) colors[ field ] = theme.overrides[ field ];
    } );
    return colors;
  },

  // The pairs that are too hard to read in this theme, worst first:
  // [ { name , front , back , ratio } ]. Only an override can cause one; the
  // built-in themes are held to none by scripts/check_themes.js.
  check( theme ) {
    const colors = this.colors( theme );
    return Object.keys( this.pairs ).map( ( name ) => {
      const [ front , back ] = this.pairs[ name ];
      return { name: name , front: front , back: back , ratio: this.contrast( colors[ front ] , colors[ back ] ) };
    } ).filter( ( pair ) => pair.ratio < this.minContrast ).sort( ( a , b ) => a.ratio - b.ratio );
  },

  // percent of over, the rest under.
  mix( over , under , percent ) {
    const channel = function ( color , offset ) { return parseInt( color.slice( offset , offset + 2 ) , 16 ); };
    return "#" + [ 1 , 3 , 5 ].map( function ( offset ) {
      const mixed = channel( over , offset ) * percent / 100 + channel( under , offset ) * ( 100 - percent ) / 100;
      return Math.round( mixed ).toString( 16 ).padStart( 2 , "0" );
    } ).join( "" );
  },

  // Black or white, for a label on this color.
  readableOn( color ) {
    return this.contrast( color , "#ffffff" ) >= this.contrast( color , "#0b0d10" ) ? "#ffffff" : "#0b0d10";
  },

  // This color, moved toward black or white until text in it reads on ground.
  asText( color , ground ) {
    const away = this.luminance( ground ) > 0.4 ? "#000000" : "#ffffff";
    for ( let step = 0 ; step <= 100 ; step += 2 ) {
      const tried = this.mix( away , color , step );
      if ( this.contrast( tried , ground ) >= this.minContrast ) return tried;
    }
    return away;
  },

  // Relative luminance (WCAG) of #rrggbb.
  luminance( hex ) {
    const channel = function ( offset ) {
      const value = parseInt( hex.slice( offset , offset + 2 ) , 16 ) / 255;
      return value <= 0.03928 ? value / 12.92 : Math.pow( ( value + 0.055 ) / 1.055 , 2.4 );
    };
    return 0.2126 * channel( 1 ) + 0.7152 * channel( 3 ) + 0.0722 * channel( 5 );
  },

  contrast( first , second ) {
    const a = this.luminance( first ) , b = this.luminance( second );
    return ( Math.max( a , b ) + 0.05 ) / ( Math.min( a , b ) + 0.05 );
  },
};

// Every button variant's three colors are custom properties, and pairs that
// must read: its label on its fill, and the outline's label on a card.
Theme.variants.forEach( ( variant ) => {
  const name = Theme.buttonField( variant );
  Theme.vars[ name ] = "--app-btn-" + variant;
  Theme.vars[ name + "Fg" ] = "--app-btn-" + variant + "-fg";
  Theme.vars[ name + "Text" ] = "--app-btn-" + variant + "-text";
  Theme.pairs[ "button_" + variant ] = [ name + "Fg" , name ];
  Theme.pairs[ "outline_" + variant ] = [ name + "Text" , "surface" ];
} );

Theme.restore();
