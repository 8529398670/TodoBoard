// Light is the default and stays it. Dark is only ever what someone picked
// with the toggle, remembered in this browser -- never the OS setting. This
// file is loaded in <head> so a saved choice applies before first paint.
( function () {
  try {
    if ( window.localStorage.getItem( "theme" ) === "dark" ) {
      document.documentElement.setAttribute( "data-theme" , "dark" );
    }
  } catch ( error ) {
    // Storage blocked (private mode): stay light.
  }
} )();

const Theme = {
  isDark() {
    return document.documentElement.getAttribute( "data-theme" ) === "dark";
  },

  toggle() {
    this.set( !this.isDark() );
  },

  set( dark ) {
    if ( dark ) document.documentElement.setAttribute( "data-theme" , "dark" );
    else document.documentElement.removeAttribute( "data-theme" );
    try {
      window.localStorage.setItem( "theme" , dark ? "dark" : "light" );
    } catch ( error ) {
      // Not remembered, but still switched for this page.
    }
  },

  // Wire every [data-theme-toggle] button on the page.
  bind() {
    Dom.all( "[data-theme-toggle]" ).forEach( function ( button ) {
      button.addEventListener( "click" , function () { Theme.toggle(); } );
    } );
  },
};
