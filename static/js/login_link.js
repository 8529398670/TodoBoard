// "Sign in on another device": a one-time login link for yourself, the same
// kind an admin hands out. Making one replaces any link of yours not yet
// used; this browser stays signed in. Only offered to a session, because the
// server refuses an API key here.
const LoginLink = {
  init( me ) {
    if ( me.auth !== "session" ) return;
    Dom.show( Dom.get( "own-login-panel" ) , true );
    Dom.get( "own-login-button" ).addEventListener( "click" , this.onCreate.bind( this ) );
    Dom.get( "own-login-copy" ).addEventListener( "click" , function () {
      Dom.copy( Dom.get( "own-login-value" ) );
    } );
  },

  async onCreate() {
    let issued = null;
    try {
      issued = await Api.createMyLoginLink();
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    // A path, not a URL: the browser knows the public origin, the server
    // behind a proxy may not (see App.showLoginLink).
    Dom.text( Dom.get( "own-login-value" ) , window.location.origin + issued.login_path );
    Dom.text( Dom.get( "own-login-expiry" ) , I18n.fill( "own_login.link_notice" , { minutes: Math.round( issued.expires_in_seconds / 60 ) } ) );
    Dom.show( Dom.get( "own-login-box" ) , true );
    Dom.text( Dom.get( "own-login-button" ) , I18n.get( "own_login.again_button" ) );
  },
};
