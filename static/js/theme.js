// How the page looks: light or dark, and optionally its own colours.
//
// Light is the default and stays it -- never the OS setting. A person's
// theme is their account's (UserPrefs.Theme, so it follows them to every
// device); a board may have one of its own that everyone on it sees, unless
// they chose to keep theirs. A guest has no account, so theirs lives in this
// browser only.
//
// Dark is Bootstrap's own dark mode (<html data-bs-theme="dark">), with
// data-theme="dark" beside it for this app's own stylesheets. Light is both
// attributes being absent, and Bootstrap's colour-mode script, which would
// follow prefers-color-scheme, is deliberately not included.
//
// This file is loaded in <head>: the last theme seen for this page is kept
// in localStorage and applied here, before first paint, then replaced by
// the real one once the page has asked the server.
const Theme = {
  // Theme field -> the custom property it sets (style.css / board.css).
  vars: { accent: "--accent" , background: "--bg" , surface: "--card-bg" , column: "--column-bg" , card: "--face-bg" },

  user: null,
  board: null,
  ownOnBoards: false,

  read( key ) {
    try {
      const raw = window.localStorage.getItem( key );
      return raw ? JSON.parse( raw ) : null;
    } catch ( error ) {
      return null;
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

  // The board cache is per page address. A share link's address is a
  // credential, so only a hash of it is kept.
  boardKey() {
    const path = window.location.pathname;
    if ( !/^\/(b|s)\//.test( path ) ) return "";
    let hash = 0;
    for ( let index = 0; index < path.length; index += 1 ) hash = ( hash * 31 + path.charCodeAt( index ) ) | 0;
    return "theme.board." + ( hash >>> 0 ).toString( 36 );
  },

  restore() {
    const saved = this.read( "theme.user" ) || {};
    this.user = saved.theme || null;
    this.ownOnBoards = !!saved.own;
    // Before account themes, the toggle kept just "dark" here.
    try {
      if ( !this.user && window.localStorage.getItem( "theme" ) === "dark" ) this.user = { mode: "dark" };
    } catch ( error ) { /* stay light */ }
    const key = this.boardKey();
    if ( key ) this.board = this.read( key );
    this.refresh();
  },

  // The one that shows: the board's, unless there is none or the person
  // keeps their own.
  resolved() {
    if ( this.board && !this.ownOnBoards ) return this.board;
    return this.user;
  },

  refresh() {
    this.apply( this.resolved() );
  },

  // The person's own theme, from the server (or, for a guest, this browser).
  setUser( theme , ownOnBoards ) {
    this.user = theme || null;
    this.ownOnBoards = !!ownOnBoards;
    this.write( "theme.user" , { theme: this.user , own: this.ownOnBoards } );
    try { window.localStorage.removeItem( "theme" ); } catch ( error ) { /* nothing to clear */ }
    this.refresh();
  },

  // The open board's theme; null when it has none.
  setBoard( theme ) {
    this.board = theme || null;
    const key = this.boardKey();
    if ( key ) this.write( key , this.board );
    this.refresh();
  },

  isDark() {
    return document.documentElement.getAttribute( "data-theme" ) === "dark";
  },

  apply( theme ) {
    const root = document.documentElement;
    const dark = !!theme && theme.mode === "dark";
    if ( dark ) {
      root.setAttribute( "data-theme" , "dark" );
      root.setAttribute( "data-bs-theme" , "dark" );
    } else {
      root.removeAttribute( "data-theme" );
      root.removeAttribute( "data-bs-theme" );
    }
    Object.keys( this.vars ).forEach( ( field ) => {
      const value = theme && this.isColor( theme[ field ] ) ? theme[ field ] : "";
      if ( value ) root.style.setProperty( this.vars[ field ] , value );
      else root.style.removeProperty( this.vars[ field ] );
    } );
    // Text on the accent has to stay readable whatever accent was picked.
    if ( theme && this.isColor( theme.accent ) ) root.style.setProperty( "--accent-fg" , this.contrast( theme.accent , "#ffffff" ) >= this.contrast( theme.accent , "#0b0d10" ) ? "#ffffff" : "#0b0d10" );
    else root.style.removeProperty( "--accent-fg" );
    // A custom page colour gets borders drawn from it rather than the mode's.
    if ( theme && this.isColor( theme.background ) ) root.style.setProperty( "--border" , "color-mix(in srgb, var(--fg) 14%, " + theme.background + ")" );
    else root.style.removeProperty( "--border" );
  },

  isColor( value ) {
    return typeof value === "string" && /^#[0-9a-f]{6}$/i.test( value );
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

Theme.restore();
