// Your own settings, from the gear in any header: theme, and how dates and
// times read. Theme is per browser (theme.js); the clock settings are the
// account's, so they follow the person to every device. A guest on a share
// link has no account, so for them times follow the device.
const UserSettings = {
  open( me ) {
    Panel.open( "user_settings.title" , ( body ) => {
      body.appendChild( this.appearance() );
      body.appendChild( this.time( me ) );
    } );
  },

  appearance() {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "user_settings.appearance_heading" ) } ) );
    const theme = Ui.select( [
      { value: "light" , text: T( "user_settings.theme_light" ) },
      { value: "dark" , text: T( "user_settings.theme_dark" ) },
    ] , Theme.isDark() ? "dark" : "light" );
    theme.addEventListener( "change" , function () { Theme.set( theme.value === "dark" ); } );
    section.appendChild( Ui.field( "user_settings.theme_label" , theme ) );
    return section;
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

    const save = async function () {
      const hour12 = clock.value === "" ? null : clock.value === "12";
      try {
        const saved = await Api.post( "/api/account/display" , { timezone: zone.value , hour12: hour12 } );
        me.display = saved;
        Dates.configure( saved );
        Ui.toast( T( "user_settings.saved" ) , "ok" );
        if ( typeof UserSettings.onChange === "function" ) UserSettings.onChange();
      } catch ( error ) {
        Ui.error( error );
      }
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

  // Wire every [data-user-settings] button on the page.
  bind( getMe ) {
    Dom.all( "[data-user-settings]" ).forEach( function ( button ) {
      button.addEventListener( "click" , function () { UserSettings.open( getMe() ); } );
    } );
  },
};
