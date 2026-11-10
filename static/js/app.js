// Page logic for the home page: the boards, your account and settings, and
// the admin screens, one sidebar screen at a time (nav.js).
//
// The shape: load the language file, ask /api/me once, then branch on
// authenticated/role. Do not keep a second client-side notion of "am I logged
// in" -- the server is the authority, and guessing is how the two drift apart.
// When the page needs to know again it asks again: see fresh.js, and the 401
// handling in api.js.
//
// New screens belong in their own file under /js/ with their own init(),
// rather than being appended here until this becomes the monolith the rest of
// the project is organised to avoid.

const App = {
  currentUser: null,

  async init() {
    Panel.bind();
    try {
      await I18n.load();
      I18n.apply();
    } catch ( error ) {
      // Without the language file every label is blank, so say something in
      // the one place that can afford a hardcoded string.
      Dom.text( Dom.get( "error-banner" ) , "Could not load language file." );
      Dom.show( Dom.get( "error-banner" ) , true );
    }

    let me = { authenticated: false };
    try {
      me = await Api.me();
    } catch ( error ) {
      this.showError( I18n.get( "errors.network" ) );
    }

    this.currentUser = me.authenticated ? me : null;
    if ( me.authenticated ) {
      Dates.configure( me.display );
      LabelColors.set( me.display.label_colors );
      Theme.setUser( me.display.theme , me.display.own_theme_on_boards );
    }
    Dom.show( Dom.get( "main" ) , true );

    Dom.passwordToggles();
    // Not a plain #main link: the hash is which screen is showing.
    Dom.get( "skip-link" ).addEventListener( "click" , ( event ) => {
      event.preventDefault();
      Dom.get( "main" ).focus();
    } );

    if ( this.loginCredential() !== "" ) {
      this.renderLoginConfirm( me );
      return;
    }
    if ( Join.credential() !== "" ) {
      await Join.init( me );
      return;
    }
    if ( !me.authenticated ) {
      this.renderSignedOut();
      return;
    }
    await this.renderSignedIn( me );
  },

  renderSignedOut() {
    Dom.show( Dom.get( "signed-out" ) , true );
    // A failed login lands here with ?login=invalid rather than explaining
    // what was wrong with the link -- see routes/auth.go.
    const params = new URLSearchParams( window.location.search );
    if ( params.get( "login" ) === "invalid" ) {
      const banner = Dom.get( "invalid-link" );
      if ( I18n.get( "signed_out.invalid_link" ) !== "" ) Dom.show( banner , true );
    }
    // A sign-in that waited too long for its two-factor code.
    if ( params.get( "login" ) === "expired" ) {
      const banner = Dom.get( "sign-in-expired" );
      if ( I18n.get( "signed_out.sign_in_expired" ) !== "" ) Dom.show( banner , true );
    }
    // The session went while the page was open (signed out elsewhere, or it
    // expired): see Api.sessionEnded.
    if ( params.get( "login" ) === "ended" ) {
      Dom.show( Dom.get( "session-ended" ) , I18n.get( "signed_out.session_ended" ) !== "" );
    }
    if ( PushClient.homeScreenApp() ) this.bindPasteLogin();
    Password.initLogin();
  },

  // A Home Screen app keeps its own cookies and has no address bar, and a
  // tapped login link opens the browser instead, so a pasted link is the only
  // way to sign in to it. Pasting and pressing the button stands in for
  // renderLoginConfirm's button: nothing is spent until then.
  bindPasteLogin() {
    const form = Dom.get( "paste-login" );
    const button = Dom.get( "paste-login-button" );
    Dom.show( form , true );
    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      const credential = this.pastedCredential( Dom.get( "paste-login-value" ).value );
      Dom.show( Dom.get( "invalid-link" ) , false );
      button.disabled = true;
      try {
        const result = await Api.login( credential );
        if ( result.two_factor ) {
          TwoFactor.ask( result.challenge );
          return;
        }
        window.location.replace( "/" );
      } catch ( error ) {
        button.disabled = false;
        if ( error.status === 400 ) Dom.show( Dom.get( "invalid-link" ) , true );
        else this.showError( error.message );
      }
    } );
  },

  // Whole links and bare credentials both work: copying from the Account page
  // gives the first, and from a message it can be either.
  pastedCredential( text ) {
    const value = text.trim();
    const prefix = "/login/";
    const at = value.indexOf( prefix );
    const credential = at >= 0 ? value.slice( at + prefix.length ) : value;
    return this.decodedCredential( credential.split( /[?#]/ )[ 0 ] );
  },

  decodedCredential( text ) {
    try {
      return decodeURIComponent( text ).replace( /\/+$/ , "" );
    } catch ( error ) {
      return "";
    }
  },

  // A login link is /login/<credential>, and the server answers it with this
  // page. Nothing has been redeemed at that point.
  loginCredential() {
    const prefix = "/login/";
    const path = window.location.pathname;
    if ( path.indexOf( prefix ) !== 0 ) return "";
    return this.decodedCredential( path.slice( prefix.length ) );
  },

  // Opening a login link only shows this. The link is spent by the POST the
  // button sends, so a chat app's link preview or a mail scanner fetching the
  // URL cannot use it up, and someone already signed in is told they are
  // about to switch accounts rather than being switched silently.
  renderLoginConfirm( me ) {
    Dom.show( Dom.get( "login-confirm" ) , true );
    if ( me.authenticated ) {
      const warning = I18n.get( "login_confirm.switch_warning" );
      if ( warning !== "" ) {
        const box = Dom.get( "login-switch-warning" );
        Dom.text( box , warning.replace( "{name}" , me.display_name ) );
        Dom.show( box , true );
      }
    }

    const button = Dom.get( "login-confirm-button" );
    button.addEventListener( "click" , async () => {
      button.disabled = true;
      try {
        const result = await Api.login( this.loginCredential() );
        // The account has two-factor on: the link is accepted, and the code
        // finishes the sign-in (see two_factor.js).
        if ( result.two_factor ) {
          TwoFactor.ask( result.challenge );
          return;
        }
        // replace, not assign: the link should not stay in the back button.
        window.location.replace( "/" );
      } catch ( error ) {
        if ( error.status === 400 ) {
          window.location.replace( "/?login=invalid" );
          return;
        }
        // Anything else (rate limited, network) leaves the link unspent,
        // so let them try again.
        button.disabled = false;
        this.showError( error.message );
      }
    } );
  },

  // Says so on every screen, so nobody wonders why a change was refused and
  // the owner, whom the mode lets through, cannot leave it on by accident.
  showMaintenance( me ) {
    const banner = Dom.get( "maintenance-banner" );
    const on = me.maintenance === "read_only" || me.maintenance === "full";
    if ( on ) Dom.text( banner , I18n.get( "maintenance." + me.maintenance + "_banner" ) );
    Dom.show( banner , on );
  },

  // What the owner wants everyone to read. Dismissing it is remembered against
  // its text, so the next message shows up however the last one went.
  showAnnouncement( me ) {
    const text = me.announcement || "";
    let dismissed = "";
    try {
      dismissed = window.localStorage.getItem( "announcement-dismissed" ) || "";
    } catch ( error ) { /* storage blocked: it comes back on the next load */ }
    Dom.text( Dom.get( "announcement-text" ) , text );
    Dom.show( Dom.get( "announcement-banner" ) , text !== "" && text !== dismissed );
  },

  onDismissAnnouncement() {
    Dom.show( Dom.get( "announcement-banner" ) , false );
    try {
      window.localStorage.setItem( "announcement-dismissed" , Dom.get( "announcement-text" ).textContent );
    } catch ( error ) { /* storage blocked: it comes back on the next load */ }
  },

  async renderSignedIn( me ) {
    // The account screens used to be a page of their own at /account, and
    // links to it are still around (/account#notifications from a board's
    // Notifications panel): land on the same screen here.
    if ( window.location.pathname.indexOf( "/account" ) === 0 ) {
      window.history.replaceState( null , "" , "/" + ( window.location.hash || "#account" ) );
    }

    Dom.show( Dom.get( "app-loading" ) , true );
    Dom.show( Dom.get( "signed-in" ) , true );
    this.showMaintenance( me );
    this.showAnnouncement( me );
    Dom.get( "announcement-dismiss" ).addEventListener( "click" , () => this.onDismissAnnouncement() );
    this.labelProfile();
    Dom.get( "display-name" ).value = me.display_name;

    Dom.onSubmit( Dom.get( "rename-form" ) , this.onRename.bind( this ) );
    Dom.get( "logout-button" ).addEventListener( "click" , this.onLogout.bind( this ) );

    // Each screen owns its own file and its own init(). The boards come
    // first: they are what most visits are for.
    Track.init( me );
    Fresh.init( me );
    await Home.init( me );
    UserSettings.render( Dom.get( "user-settings-root" ) , me );
    // Side by side: the wait is the slowest screen, not all of them added up.
    const loading = [];
    loading.push( NotifySettings.init( me ) );
    Pin.init( me );
    LoginLink.init( me );
    Password.init( me );
    TwoFactor.init( me );
    loading.push( Sessions.init( me ) );
    Backup.initAccount( me );
    loading.push( Keys.init( me ) );
    Mcp.init( me );
    About.init( me );

    if ( me.role === "admin" ) {
      Dom.show( Dom.get( "admin-panel" ) , true );
      LoginLinks.fill( Dom.get( "new-uses" ) , Dom.get( "new-expires" ) );
      Dom.onSubmit( Dom.get( "create-user-form" ) , this.onCreateUser.bind( this ) );
      Dom.get( "copy-link-button" ).addEventListener( "click" , this.onCopyLink.bind( this ) );
      Qr.toggle( Dom.get( "login-link-qr-button" ) , Dom.get( "login-link-qr" ) , Dom.get( "login-link-value" ) );
      Dom.get( "copy-admin-key-button" ).addEventListener( "click" , function () { Dom.copy( Dom.get( "admin-key-value" ) ); } );
      Pager.search( "users-table" , "users-search" , () => this.refreshUsers() );
      Pager.sortable( "users-table" , () => this.refreshUsers() );
      // Only the owner gets the Logs, Jobs and server Settings screens; each
      // init checks is_owner.
      loading.push( this.refreshUsers() );
      loading.push( Invites.init( me ) );
      loading.push( Logs.init( me ) );
      loading.push( Jobs.init( me ) );
      loading.push( Settings.init( me ) );
      const version = I18n.get( "nav.version" );
      if ( version !== "" ) {
        Dom.text( Dom.get( "sidebar-version" ) , version.replace( "{version}" , () => me.version ) );
        Dom.show( Dom.get( "sidebar-version" ) , true );
      }
    }

    // allSettled, not all: Nav.init has to run whatever else went wrong, or
    // one screen's failure would leave no screen showing at all.
    const results = await Promise.allSettled( loading );
    results.forEach( ( result ) => {
      if ( result.status === "rejected" ) this.showError( result.reason.message );
    } );
    Dom.show( Dom.get( "app-loading" ) , false );

    // Last, so the sidebar offers exactly the screens the lines above un-hid.
    Nav.init();
  },

  async onRename( event ) {
    event.preventDefault();
    const value = Dom.get( "display-name" ).value.trim();
    const previous = this.currentUser.display_name;
    if ( value === previous ) {
      Dom.toast( I18n.get( "ui.unchanged" ) , "ok" );
      return;
    }
    try {
      await this.applyRename( value , previous );
    } catch ( error ) {
      this.showError( error.message );
      return;
    }
    // Let go of the field, so the next Ctrl+Z undoes the rename rather than
    // the typing.
    Undo.saved( event.target );
    // Both directions go through the API, so Ctrl+Z puts the old name back on
    // the account, not just on the screen. Each says which name it expects to
    // find, so a rename made in another tab since is refused (409) rather
    // than overwritten.
    Undo.record( {
      label: I18n.get( "undo.rename" ),
      toast: true,
      undo: () => this.applyRename( previous , value ),
      redo: () => this.applyRename( value , previous ),
    } );
  },

  async applyRename( value , expect ) {
    await Api.rename( value , expect );
    this.currentUser.display_name = value;
    Dom.get( "display-name" ).value = value;
    this.labelProfile();
  },

  // The sidebar's Account entry says, in its tooltip, who is signed in,
  // which the header used to spell out.
  labelProfile() {
    const user = this.currentUser;
    const label = I18n.get( "nav.account" );
    const who = user.display_name + " (" + user.role + ")";
    const link = Dom.get( "profile-link" );
    link.setAttribute( "title" , label ? label + " · " + who : who );
  },

  async onLogout() {
    await PushClient.leave();
    try {
      await Api.logout();
    } catch ( error ) {
      // Even if the call failed, reloading lands on the signed-out page,
      // which is the honest state to show.
    }
    window.location.href = "/";
  },

  async onCreateUser( event ) {
    event.preventDefault();
    const name = Dom.get( "new-name" ).value.trim();
    const role = Dom.get( "new-role" ).value;
    const options = LoginLinks.body( Dom.get( "new-uses" ).value , Dom.get( "new-expires" ).value );
    try {
      const created = await Api.createUser( name , role , options );
      this.showLoginLink( created , name );
      Dom.get( "new-name" ).value = "";
      await this.refreshUsers();
    } catch ( error ) {
      this.showError( error.message );
    }
  },

  // The server returns a path, not a full URL: behind a reverse proxy it does
  // not reliably know its own public origin, whereas the browser always does.
  // issued is the server's answer, which also says how often and how long
  // the link works; name says whose it is.
  showLoginLink( issued , name ) {
    Dom.text( Dom.get( "login-link-for" ) , I18n.fill( "admin.link_for" , { name: name } ) );
    Dom.text( Dom.get( "login-link-notice" ) , I18n.fill( "admin.link_notice" , { summary: LoginLinks.summary( issued ) } ) );
    Dom.text( Dom.get( "login-link-value" ) , window.location.origin + issued.login_path );
    Qr.hide( Dom.get( "login-link-qr" ) );
    const box = Dom.get( "login-link-box" );
    Dom.show( box , true );
    // The row that asked for it may be a long way down the Users table.
    box.scrollIntoView( { block: "nearest" } );
  },

  // New link for someone: how many times and how long, asked in the panel.
  // Making it stops the link they had.
  reissueFor( user ) {
    Panel.open( "admin.reissue_title" , ( body ) => {
      body.appendChild( Dom.el( "p" , { class: "text-body-secondary" , text: I18n.fill( "admin.reissue_confirm" , { name: user.display_name } ) } ) );
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const choices = LoginLinks.fields();
      choices.nodes.forEach( function ( node ) { form.appendChild( node ); } );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary mt-2" , text: I18n.get( "login_links.make_button" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        let issued;
        try {
          issued = await Api.reissueLogin( user.id , choices.body() );
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        Panel.close();
        this.showLoginLink( issued , user.display_name );
        await this.refreshUsers();
      } );
      body.appendChild( form );
    } );
  },

  // Rename someone. Only the owner renames an admin; the server says so.
  renameUser( user ) {
    Panel.open( "admin.rename_title" , ( body ) => {
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const input = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "80" , required: "" } } );
      input.value = user.display_name;
      form.appendChild( Ui.field( "admin.new_name_label" , input ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary mt-2" , text: I18n.get( "account.save_button" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        const name = input.value.trim();
        if ( name === user.display_name ) {
          Panel.close();
          return;
        }
        try {
          await Api.renameUser( user.id , name , user.display_name );
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        Panel.close();
        await this.refreshUsers();
        Dom.toast( I18n.get( "ui.saved" ) , "ok" );
      } );
      body.appendChild( form );
      window.setTimeout( function () { input.focus(); input.select(); } , 0 );
    } );
  },

  // Asks first when the language file has a question for it ("" skips
  // asking), then runs action, refreshes the table and says it is done.
  async confirmed( questionKey , user , action , toastKey , extra ) {
    const question = I18n.fill( questionKey , Object.assign( { name: user.display_name } , extra || {} ) );
    if ( !( await Dom.confirm( question ) ) ) return;
    try {
      await action();
    } catch ( error ) {
      this.showError( error.message );
      return;
    }
    await this.refreshUsers();
    if ( toastKey ) Dom.toast( I18n.fill( toastKey , Object.assign( { name: user.display_name } , extra || {} ) ) , "ok" );
  },

  // The less used actions on a user's row, in a menu behind ⋯ so the row
  // keeps its everyday buttons in sight.
  moreActions( user , isSelf , isProtected ) {
    const me = this.currentUser;
    const items = [];
    // Only the owner renames an admin.
    if ( user.role !== "admin" || me.is_owner ) {
      items.push( { key: "admin.rename_button" , action: () => this.renameUser( user ) } );
    }
    // Nobody changes their own role; the owner is always an admin; only the
    // owner takes an admin's rights away.
    if ( !isProtected && user.role === "user" ) {
      items.push( { key: "admin.make_admin_button" , action: () => this.confirmed( "admin.make_admin_confirm" , user , () => Api.setUserRole( user.id , "admin" ) , "admin.role_changed_toast" , { role: I18n.get( "admin.role_admin" ) } ) } );
    }
    if ( !isProtected && user.role === "admin" && me.is_owner ) {
      items.push( { key: "admin.make_user_button" , action: () => this.confirmed( "admin.make_user_confirm" , user , () => Api.setUserRole( user.id , "user" ) , "admin.role_changed_toast" , { role: I18n.get( "admin.role_user" ) } ) } );
    }
    // Devices: the owner's own are the owner's alone.
    if ( me.auth === "session" && ( !user.owner || me.is_owner ) ) {
      items.push( { key: "admin.sessions_button" , action: () => Sessions.showFor( user ) } );
    }
    if ( user.login_link ) {
      items.push( { key: "login_links.cancel_button" , action: () => this.confirmed( "login_links.cancel_confirm" , user , () => Api.cancelLoginLink( user.id ) , "login_links.cancelled_toast" ) } );
    }
    // Resetting 2FA follows deleting's rules: not yourself, not the owner,
    // and another admin's only by the owner.
    if ( user.two_factor && !isProtected && ( user.role !== "admin" || me.is_owner ) ) {
      items.push( { key: "admin.reset_two_factor_button" , danger: true , action: () => this.confirmed( "admin.reset_two_factor_confirm" , user , () => Api.resetTwoFactor( user.id ) , "admin.two_factor_reset_toast" ) } );
    }
    // Whether one account may use a password is the owner's call.
    if ( me.is_owner && me.passwords_enabled && !isSelf ) {
      items.push( user.password_blocked
        ? { key: "admin.allow_password_button" , action: () => this.confirmed( "admin.allow_password_confirm" , user , () => Api.setPasswordAccess( user.id , true ) , "admin.password_allowed_toast" ) }
        : { key: "admin.block_password_button" , danger: true , action: () => this.confirmed( "admin.block_password_confirm" , user , () => Api.setPasswordAccess( user.id , false ) , "admin.password_blocked_toast" ) } );
    }
    if ( items.length === 0 ) return null;
    return Ui.iconButton( "three-dots" , I18n.get( "admin.more_actions" ) , function ( event ) {
      ContextMenu.open( event , items );
    } , "btn-sm" );
  },

  async onCopyLink() {
    await Dom.copy( Dom.get( "login-link-value" ) );
  },

  // An API key that acts as user. Asked for in the panel, but shown in the
  // page behind it: it appears once, and a stray click closes a panel.
  newKeyFor( user ) {
    Panel.open( "admin.key_title" , function ( body ) {
      body.appendChild( Dom.el( "p" , { class: "muted small-print" , text: I18n.fill( "admin.key_hint" , { name: user.display_name } ) } ) );
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "80" , required: "" , placeholder: T( "api_keys.name_placeholder" ) } } );
      form.appendChild( Ui.field( "api_keys.name_label" , name ) );
      // Only an admin's key can be an admin key.
      const role = user.role === "admin" ? Ui.select( [
        { value: "user" , text: T( "api_keys.role_user" ) },
        { value: "admin" , text: T( "api_keys.role_admin" ) },
      ] , "user" ) : null;
      if ( role ) form.appendChild( Ui.field( "api_keys.role_label" , role ) );
      const days = Dom.el( "input" , { class: "form-control" , attrs: { type: "number" , min: "0" , max: "3650" , inputmode: "numeric" , placeholder: T( "api_keys.days_placeholder" ) } } );
      form.appendChild( Ui.field( "api_keys.days_label" , days ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "api_keys.create_button" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async function ( event ) {
        event.preventDefault();
        // Blank is the server's default; 0 is never, and has to be typed.
        const rawDays = days.value.trim();
        let created;
        try {
          created = await Api.createKeyFor( user.id , name.value.trim() , role ? role.value : "user" , rawDays === "" ? null : Number( rawDays ) );
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        Panel.close();
        App.showAdminKey( created.credential , user );
        await Keys.refreshAll();
      } );
      body.appendChild( form );
      window.setTimeout( function () { name.focus(); } , 0 );
    } );
  },

  showAdminKey( credential , user ) {
    Dom.text( Dom.get( "admin-key-notice" ) , I18n.fill( "admin.key_notice" , { name: user.display_name } ) );
    Dom.text( Dom.get( "admin-key-value" ) , credential );
    const box = Dom.get( "admin-key-box" );
    Dom.show( box , true );
    box.scrollIntoView( { block: "nearest" } );
  },

  async refreshUsers() {
    const body = Dom.get( "users-table" ).querySelector( "tbody" );
    await Pager.load( "users-table" , "users" , ( offset ) => Api.listUsers( offset , Pager.params( "users-table" ) ) , function ( user ) {
      const statusKey = user.disabled ? "admin.status_disabled" : "admin.status_active";
      const statusTone = user.disabled ? "secondary" : "success";

      const reissueButton = Dom.el( "button" , {
        class: "btn btn-sm btn-outline-secondary",
        text: I18n.get( "admin.reissue_button" ),
        on: { click: function () { App.reissueFor( user ); } },
      } );

      // The server is the authority on all of these; the rules below only
      // decide which buttons it would refuse, so they are not offered. Your
      // own row and the owner's never get revoke, disable, or delete.
      const isSelf = user.id === App.currentUser.id;
      const isProtected = isSelf || user.owner;

      // Your own keys are made under API keys, and the owner's by the owner.
      const keyButton = isProtected || user.disabled || !App.currentUser.api_keys_enabled ? null : Dom.el( "button" , {
        class: "btn btn-sm btn-outline-secondary",
        text: I18n.get( "admin.key_button" ),
        on: { click: function () { App.newKeyFor( user ); } },
      } );

      // Revoking is for "someone else may have this account's credentials":
      // it ends every session, login link, password and API key, and leaves the
      // account enabled. Not offered on your own row -- the server refuses
      // that, since it would sign you out with no way back in.
      const revokeButton = isProtected ? null : Dom.el( "button" , {
        class: "btn btn-sm btn-outline-danger",
        text: I18n.get( "admin.revoke_button" ),
        on: { click: async function () {
          const question = I18n.get( "admin.revoke_confirm" ).replace( "{name}" , () => user.display_name );
          if ( !( await Dom.confirm( question ) ) ) return;
          try {
            await Api.revokeAccess( user.id );
            await App.refreshUsers();
            if ( App.currentUser.api_keys_enabled ) await Keys.refreshAll();
            Dom.toast( I18n.get( "admin.revoked_toast" ).replace( "{name}" , () => user.display_name ) , "ok" );
          } catch ( error ) { App.showError( error.message ); }
        } },
      } );

      // Disabling asks first and is not undoable, like a revoke: it ends their
      // sessions and links, and enabling does not bring those back.
      const toggleButton = isProtected ? null : Dom.el( "button" , {
        class: "btn btn-sm btn-outline-secondary",
        text: I18n.get( user.disabled ? "admin.enable_button" : "admin.disable_button" ),
        on: { click: async function () {
          if ( !user.disabled ) {
            const question = I18n.get( "admin.disable_confirm" ).replace( "{name}" , () => user.display_name );
            if ( !( await Dom.confirm( question ) ) ) return;
          }
          try {
            await Api.setDisabled( user.id , !user.disabled );
            await App.refreshUsers();
            Dom.toast( I18n.get( user.disabled ? "admin.enabled_toast" : "admin.disabled_toast" ).replace( "{name}" , () => user.display_name ) , "ok" );
          } catch ( error ) { App.showError( error.message ); }
        } },
      } );

      // Deleting removes the account and everything it held, for good. Any
      // admin may delete a user; only the owner may delete another admin.
      const canDelete = !isProtected && ( user.role !== "admin" || App.currentUser.is_owner );
      const deleteButton = !canDelete ? null : Dom.el( "button" , {
        class: "btn btn-sm btn-outline-danger",
        text: I18n.get( "admin.delete_button" ),
        on: { click: async function () {
          const question = I18n.get( "admin.delete_confirm" ).replace( "{name}" , () => user.display_name );
          if ( !( await Dom.confirm( question ) ) ) return;
          try {
            await Api.deleteUser( user.id );
            await App.refreshUsers();
            if ( App.currentUser.api_keys_enabled ) await Keys.refreshAll();
          } catch ( error ) { App.showError( error.message ); }
        } },
      } );

      const ownerLabel = I18n.get( "admin.owner_label" );
      const roleText = user.owner && ownerLabel !== "" ? user.role + " (" + ownerLabel + ")" : user.role;

      // The name, with the username under it once they have set a password.
      const nameCell = [ Dom.el( "div" , { text: user.display_name } ) ];
      if ( user.username ) nameCell.push( Dom.el( "div" , { class: "small text-body-secondary" , text: user.username } ) );

      // Status: active or disabled, whether 2FA or a password is on, and the
      // login link they could still use.
      const badges = [ Dom.badge( I18n.get( statusKey ) , statusTone ) ];
      if ( user.two_factor && I18n.get( "admin.two_factor_badge" ) !== "" ) badges.push( Dom.badge( I18n.get( "admin.two_factor_badge" ) , "info" ) );
      if ( user.has_password && I18n.get( "admin.password_badge" ) !== "" ) {
        badges.push( Dom.badge( I18n.get( user.password_blocked ? "admin.password_blocked_badge" : "admin.password_badge" ) , user.password_blocked ? "warning" : "secondary" ) );
      }
      const statusCell = [ Dom.el( "div" , { class: "d-flex flex-wrap gap-1" , children: badges } ) ];
      if ( user.login_link ) statusCell.push( Dom.el( "div" , { class: "small text-body-secondary mt-1" , text: LoginLinks.live( user.login_link , "live" ) } ) );

      // Built with createElement + textContent, never an HTML string: a
      // display name is user-controlled input.
      body.appendChild( Dom.el( "tr" , { children: [
        Dom.el( "td" , { children: nameCell } ),
        Dom.el( "td" , { text: roleText } ),
        Dom.el( "td" , { children: statusCell } ),
        Dom.el( "td" , { class: "text-nowrap" , children: [ Dom.when( user.created_at ) ] } ),
        Dom.el( "td" , { children: [
          Dom.rowActions( user.display_name , [ reissueButton , keyButton , revokeButton , toggleButton , deleteButton , App.moreActions( user , isSelf , isProtected ) ] ),
        ] } ),
      ] } ) );
    } );
  },

  // message may be an error code the server answered with (pin_wrong,
  // invite_link_full, ...); language.yaml's errors section words those.
  showError( message ) {
    const banner = Dom.get( "error-banner" );
    const worded = message ? I18n.get( "errors." + message ) : "";
    Dom.text( banner , worded || message || I18n.get( "errors.generic" ) );
    Dom.flash( banner , 5000 );
    // It sits at the top of the screen, which may be scrolled well past.
    banner.scrollIntoView( { block: "nearest" } );
  },

  // A refusal about one field goes beside that field, and anything else to the
  // banner. fields maps error codes to input ids. A field that is not on screen
  // cannot show it, so that goes to the banner too.
  showFieldError( message , fields ) {
    const input = Dom.get( fields[ message ] );
    if ( !input || input.offsetParent === null ) {
      this.showError( message );
      return;
    }
    Dom.fieldError( input , Dom.worded( { message: message } ) );
    input.focus();
  },
};

document.addEventListener( "DOMContentLoaded" , function () {
  App.init();
} );
