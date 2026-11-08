// "Where you're signed in", on the Account screen: every device with a live
// session on this account, a name for each, and a way to sign one out from
// here. The Users screen uses the same rows for an admin's view of someone's
// devices. app.js calls Sessions.init() once it knows who is signed in.
//
// Signing a device out cannot be undone: it is the button for a lost phone,
// and an undo would let the session back in. So it asks first. This device
// has no button here; that is what Log out is for.
const Sessions = {
  ready: false,
  // Whose devices an admin has open under the Users table.
  shownFor: null,

  async init( me ) {
    if ( me.auth !== "session" ) return;
    this.ready = true;
    Dom.show( Dom.get( "sessions-panel" ) , true );
    Dom.get( "sessions-revoke-others" ).addEventListener( "click" , () => this.onRevokeOthers() );
    Dom.get( "user-sessions-close" ).addEventListener( "click" , () => this.closeFor() );
    window.addEventListener( "hashchange" , () => {
      if ( Nav.current() === "account" ) this.refresh();
    } );
    await this.refresh();
  },

  // A short name out of a User-Agent header. It only has to help someone tell
  // their own devices apart; the order matters, since Edge and Chrome both
  // say Safari, and Android says Linux.
  browser( agent ) {
    const known = [ [ /Edg\// , "Edge" ] , [ /OPR\// , "Opera" ] , [ /Firefox\// , "Firefox" ] , [ /Chrome\// , "Chrome" ] , [ /Safari\// , "Safari" ] ];
    const match = known.find( function ( entry ) { return entry[ 0 ].test( agent ); } );
    return match ? match[ 1 ] : "";
  },

  system( agent ) {
    const known = [ [ /Android/ , "Android" ] , [ /iPhone|iPad/ , "iOS" ] , [ /Mac OS X/ , "macOS" ] , [ /Windows/ , "Windows" ] , [ /Linux/ , "Linux" ] ];
    const match = known.find( function ( entry ) { return entry[ 0 ].test( agent ); } );
    return match ? match[ 1 ] : "";
  },

  device( session ) {
    const agent = session.user_agent || "";
    const browser = this.browser( agent );
    const system = this.system( agent );
    if ( browser && system ) {
      return I18n.get( "sessions.device_name" ).replace( "{browser}" , browser ).replace( "{system}" , system );
    }
    return browser || system || agent || I18n.get( "sessions.unknown_device" );
  },

  when( value ) {
    if ( !value ) return I18n.get( "sessions.unknown" );
    return new Date( value ).toLocaleString( undefined , { dateStyle: "medium" , timeStyle: "short" } );
  },

  // What you call a device, in place of the name worked out for it. That name
  // stays underneath, so a label cannot hide what a device says it is.
  nameCell( session ) {
    const device = this.device( session );
    const line = [ Dom.el( "span" , { text: session.label || device } ) ];
    if ( session.current ) line.push( Dom.el( "span" , { class: "ms-2" , children: [ Dom.badge( I18n.get( "sessions.this_device" ) , "success" ) ] } ) );
    const cell = [ Dom.el( "div" , { children: line } ) ];
    if ( session.label ) cell.push( Dom.el( "div" , { class: "small text-body-secondary" , text: device } ) );
    return cell;
  },

  // Names one of your own devices. Empty puts the worked-out name back.
  rename( session ) {
    Panel.open( "sessions.rename_title" , ( body ) => {
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const input = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "80" , placeholder: this.device( session ) } } );
      input.value = session.label || "";
      form.appendChild( Ui.field( "sessions.rename_label" , input ) );
      form.appendChild( Dom.el( "div" , { class: "form-text mb-3" , text: I18n.get( "sessions.rename_help" ) } ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: I18n.get( "account.save_button" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        try {
          await Api.labelSession( session.id , input.value.trim() , session.label || "" );
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        Panel.close();
        await this.refresh();
      } );
      body.appendChild( form );
      window.setTimeout( function () { input.focus(); } , 0 );
    } );
  },

  // revoke is the call that ends this session and redraws its table: the
  // button is the same on your own list and on an admin's view of someone's.
  revokeButton( revoke ) {
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-danger text-nowrap",
      text: I18n.get( "sessions.revoke_button" ),
      attrs: { type: "button" },
      on: { click: async () => {
        // An empty confirm message means "do not ask", following the same
        // language.yaml rule as every other string here.
        const question = I18n.get( "sessions.revoke_confirm" );
        if ( question !== "" && !window.confirm( question ) ) return;
        try {
          await revoke();
        } catch ( error ) { App.showError( error.message ); }
      } },
    } );
  },

  async onRevokeOthers() {
    const question = I18n.get( "sessions.revoke_others_confirm" );
    if ( question !== "" && !window.confirm( question ) ) return;
    try {
      await Api.revokeOtherSessions();
      await this.refresh();
    } catch ( error ) { App.showError( error.message ); }
  },

  async refresh() {
    if ( !this.ready ) return;
    let page = { sessions: [] };
    try {
      page = await Api.sessions();
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    const body = Dom.get( "sessions-table" ).querySelector( "tbody" );
    Dom.clear( body );
    page.sessions.forEach( ( session ) => {
      const actions = [ Dom.el( "button" , {
        class: "btn btn-sm btn-outline-secondary text-nowrap",
        text: I18n.get( "sessions.rename_button" ),
        attrs: { type: "button" },
        on: { click: () => this.rename( session ) },
      } ) ];
      if ( !session.current ) {
        actions.push( this.revokeButton( async () => {
          await Api.revokeSession( session.id );
          await this.refresh();
        } ) );
      }
      body.appendChild( this.row( session , actions ) );
    } );
    Dom.show( Dom.get( "sessions-revoke-others" ) , page.sessions.length > 1 );
  },

  row( session , actions ) {
    return Dom.el( "tr" , { children: [
      Dom.el( "td" , { children: this.nameCell( session ) } ),
      Dom.el( "td" , { text: session.ip || I18n.get( "sessions.unknown" ) } ),
      Dom.el( "td" , { text: this.when( session.last_seen_at ) } ),
      Dom.el( "td" , { text: this.when( session.created_at ) } ),
      Dom.el( "td" , { children: [ Dom.el( "div" , { class: "row-actions" , children: actions } ) ] } ),
    ] } );
  },

  // An admin's view of one account's devices, under the Users table: for ending
  // one of them, where revoke access would end everything. Addresses reach
  // the owner only; the server leaves them out for other admins.
  async showFor( user ) {
    this.shownFor = user;
    Dom.text( Dom.get( "user-sessions-heading" ) , I18n.get( "sessions.user_heading" ).replace( "{name}" , user.display_name ) );
    const body = Dom.get( "user-sessions-table" ).querySelector( "tbody" );
    // Someone else's devices must not sit under this name while it loads.
    Dom.clear( body );
    Dom.show( Dom.get( "user-sessions" ) , true );
    let page = { sessions: [] };
    try {
      page = await Api.userSessions( user.id );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    if ( !this.shownFor || this.shownFor.id !== user.id ) return;
    page.sessions.forEach( ( session ) => {
      body.appendChild( this.row( session , [ this.revokeButton( async () => {
        await Api.revokeUserSession( user.id , session.id );
        await this.showFor( user );
      } ) ] ) );
    } );
    Dom.show( Dom.get( "user-sessions-empty" ) , page.sessions.length === 0 );
    Dom.get( "user-sessions" ).scrollIntoView( { block: "nearest" } );
  },

  closeFor() {
    this.shownFor = null;
    Dom.show( Dom.get( "user-sessions" ) , false );
  },
};
