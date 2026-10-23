// Your own settings, all on the Account page: theme, how dates and times
// read, and whether cards are numbered by default. They are the account's,
// so they follow the person to every device. A guest on a share link has no
// account page, so the board page opens the theme (kept in this browser) and
// times for them in a panel (open).
const UserSettings = {
  // The Account page card.
  render( root , me ) {
    Dom.clear( root );
    root.appendChild( this.appearance( me ) );
    root.appendChild( this.time( me ) );
    root.appendChild( this.cards( me ) );
  },

  // A guest's panel on the board page.
  open( me ) {
    Panel.open( "user_settings.title" , ( body ) => {
      body.appendChild( this.appearance( me ) );
      body.appendChild( this.time( me ) );
    } );
  },

  // The person's own theme. Signed in, it is the account's and saved to the
  // server; a guest's stays in this browser.
  appearance( me ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "user_settings.appearance_heading" ) } ) );
    const signedIn = !!( me && me.authenticated );
    section.appendChild( ThemeEditor.render( Theme.user , function ( theme ) {
      if ( !signedIn ) {
        Theme.setUser( theme , false );
        return;
      }
      UserSettings.save( me , theme ? { theme: theme } : { clear_theme: true } );
    } ) );
    if ( signedIn ) {
      section.appendChild( Ui.checkbox( "user_settings.own_theme_on_boards" , me.display && me.display.own_theme_on_boards , function ( checked ) {
        UserSettings.save( me , { own_theme_on_boards: checked } );
      } ) );
      section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "user_settings.own_theme_hint" ) } ) );
    } else {
      section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "user_settings.guest_theme_hint" ) } ) );
    }
    return section;
  },

  // Saves the account's display settings. The server takes them as one
  // record, so what is not being changed is sent as it stands.
  async save( me , changes ) {
    const display = me.display || {};
    const body = Object.assign( {
      timezone: display.timezone || "",
      hour12: typeof display.hour12 === "boolean" ? display.hour12 : null,
      number_cards: !!display.number_cards,
      plain_icons: !!display.plain_icons,
    } , changes );
    try {
      const saved = await Api.post( "/api/account/display" , body );
      me.display = saved;
      Dates.configure( saved );
      Theme.setUser( saved.theme , saved.own_theme_on_boards );
      Ui.toast( T( "user_settings.saved" ) , "ok" );
    } catch ( error ) {
      Ui.error( error );
    }
  },

  time( me ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "user_settings.time_heading" ) } ) );
    if ( !me || !me.authenticated ) {
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "user_settings.guest_hint" ) } ) );
      return section;
    }
    const display = me.display || {};
    const zone = Ui.select( [ { value: "" , text: I18n.fill( "user_settings.timezone_browser" , { zone: Dates.browserZone() } ) } ]
      .concat( Dates.timezones().map( function ( name ) { return { value: name , text: name }; } ) ) , display.timezone || "" );
    const clock = Ui.select( [
      { value: "" , text: T( "user_settings.clock_auto" ) },
      { value: "12" , text: T( "user_settings.clock_12" ) },
      { value: "24" , text: T( "user_settings.clock_24" ) },
    ] , display.hour12 === true ? "12" : display.hour12 === false ? "24" : "" );

    const save = function () {
      UserSettings.save( me , { timezone: zone.value , hour12: clock.value === "" ? null : clock.value === "12" } );
    };
    zone.addEventListener( "change" , save );
    clock.addEventListener( "change" , save );

    const row = Dom.el( "div" , { class: "form-row" } );
    row.appendChild( Ui.field( "user_settings.timezone_label" , zone ) );
    row.appendChild( Ui.field( "user_settings.clock_label" , clock ) );
    section.appendChild( row );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "user_settings.time_hint" ) } ) );
    return section;
  },

  cards( me ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "user_settings.cards_heading" ) } ) );
    section.appendChild( Ui.checkbox( "user_settings.number_cards" , me.display && me.display.number_cards , function ( checked ) {
      UserSettings.save( me , { number_cards: checked } );
    } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "user_settings.number_cards_hint" ) } ) );
    section.appendChild( Ui.checkbox( "user_settings.plain_icons" , me.display && me.display.plain_icons , function ( checked ) {
      UserSettings.save( me , { plain_icons: checked } );
    } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "user_settings.plain_icons_hint" ) } ) );
    return section;
  },

  // Wire every [data-user-settings] button on the page.
  bind( getMe ) {
    Dom.all( "[data-user-settings]" ).forEach( function ( button ) {
      button.addEventListener( "click" , function () { UserSettings.open( getMe() ); } );
    } );
  },
};
