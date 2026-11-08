// "Sign in on another device": a login link for yourself, the same kind an
// admin hands out -- once by default, or several times until it expires.
// Making one replaces the link you had; this browser stays signed in. The
// link you could still use shows here with a Cancel button. Only offered to a
// session, because the server refuses an API key here.
const LoginLink = {
  me: null,

  init( me ) {
    if ( me.auth !== "session" ) return;
    this.me = me;
    Dom.show( Dom.get( "own-login-panel" ) , true );
    LoginLinks.fill( Dom.get( "own-login-uses" ) , Dom.get( "own-login-expires" ) );
    Dom.get( "own-login-button" ).addEventListener( "click" , this.onCreate.bind( this ) );
    Dom.get( "own-login-copy" ).addEventListener( "click" , function () {
      Dom.copy( Dom.get( "own-login-value" ) );
    } );
    Dom.get( "own-login-cancel" ).addEventListener( "click" , this.onCancel.bind( this ) );
    this.renderLive();
  },

  // The link this account could still sign in with, if any.
  renderLive() {
    const link = this.me.login_link;
    Dom.text( Dom.get( "own-login-live-text" ) , link ? LoginLinks.live( link , "my_live" ) : "" );
    Dom.show( Dom.get( "own-login-live" ) , Boolean( link ) );
  },

  async onCreate() {
    const button = Dom.get( "own-login-button" );
    const body = LoginLinks.body( Dom.get( "own-login-uses" ).value , Dom.get( "own-login-expires" ).value );
    let issued = null;
    button.disabled = true;
    try {
      issued = await Api.createMyLoginLink( body );
    } catch ( error ) {
      App.showError( error.message );
      return;
    } finally {
      button.disabled = false;
    }
    // A path, not a URL: the browser knows the public origin, the server
    // behind a proxy may not (see App.showLoginLink).
    Dom.text( Dom.get( "own-login-value" ) , window.location.origin + issued.login_path );
    Dom.text( Dom.get( "own-login-expiry" ) , I18n.fill( "own_login.link_notice" , { summary: LoginLinks.summary( issued ) } ) );
    Dom.show( Dom.get( "own-login-box" ) , true );
    Dom.text( button , I18n.get( "own_login.again_button" ) );
    this.me.login_link = { max_uses: issued.max_uses , unlimited: issued.unlimited , uses: 0 , expires_at: issued.expires_at };
    this.renderLive();
  },

  async onCancel() {
    const question = I18n.get( "login_links.my_cancel_confirm" );
    if ( question !== "" && !window.confirm( question ) ) return;
    try {
      await Api.cancelMyLoginLink();
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    this.me.login_link = null;
    this.renderLive();
    Dom.show( Dom.get( "own-login-box" ) , false );
    Dom.text( Dom.get( "own-login-value" ) , "" );
    Dom.toast( I18n.get( "login_links.my_cancelled_toast" ) , "ok" );
  },
};
