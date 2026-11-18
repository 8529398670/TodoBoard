// The Google Calendar screen: connect your own Google account, pick the
// calendar cards go on, see how the last sync went, and disconnect. Which
// boards sync is each board's own choice (Board settings, Just for you).
// Connecting is a trip to Google and back: /api/google/connect sends the
// browser there, and Google's answer lands back here with ?google=<outcome>.
const GoogleSettings = {
  status: null,
  calendars: null,

  async init() {
    this.root = Dom.get( "google-root" );
    try {
      this.status = await Api.request( "/api/google" );
    } catch ( error ) {
      return;
    }
    // Only where it can be used, or is still connected from before.
    Dom.show( document.querySelector( '[data-nav="google"]' ) , this.status.available || this.status.connected );
    this.showOutcome();
    this.render();
  },

  // The answer Google sent back with, once, then off the address.
  showOutcome() {
    const params = new URLSearchParams( window.location.search );
    const outcome = params.get( "google" );
    if ( !outcome ) return;
    params.delete( "google" );
    const query = params.toString();
    window.history.replaceState( null , "" , window.location.pathname + ( query ? "?" + query : "" ) + window.location.hash );
    const ok = outcome === "connected";
    Ui.toast( T( "google.outcome_" + ( [ "connected" , "denied" ].indexOf( outcome ) >= 0 ? outcome : "failed" ) ) , ok ? "ok" : "error" );
  },

  async reload() {
    try {
      this.status = await Api.request( "/api/google" );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    this.render();
  },

  connectButton( key ) {
    return Ui.button( key , function () { window.location.href = "/api/google/connect"; } , "primary" );
  },

  render() {
    if ( !this.root ) return;
    const status = this.status;
    this.root.replaceChildren();
    if ( !status.available && !status.connected ) {
      this.root.appendChild( Dom.el( "p" , { class: "alert alert-warning" , text: T( "google.unavailable" ) } ) );
      return;
    }
    if ( !status.connected ) {
      this.root.appendChild( Dom.el( "p" , { class: "text-body-secondary" , text: T( "google.intro" ) } ) );
      this.root.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ this.connectButton( "google.connect" ) ] } ) );
      return;
    }
    this.root.appendChild( Dom.el( "p" , { text: I18n.fill( "google.connected_as" , { email: status.email || "?" } ) } ) );
    if ( status.problem === "revoked" ) {
      const alert = Dom.el( "div" , { class: "alert alert-warning" , children: [ Dom.el( "p" , { text: T( "google.problem_revoked" ) } ) ] } );
      if ( status.available ) alert.appendChild( this.connectButton( "google.reconnect" ) );
      this.root.appendChild( alert );
    } else if ( status.problem ) {
      this.root.appendChild( Dom.el( "p" , { class: "alert alert-warning" , text: T( "google.problem_failed" ) } ) );
    }
    if ( status.last_sync ) {
      this.root.appendChild( Dom.el( "p" , { class: "muted small-print" , text: I18n.fill( "google.last_sync" , { when: Dates.format( status.last_sync , false , true ) } ) } ) );
    }
    this.root.appendChild( Dom.el( "p" , { text: I18n.fill( status.boards.length === 1 ? "google.boards_one" : "google.boards_many" , { n: status.boards.length } ) } ) );
    this.root.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "google.boards_hint" ) } ) );
    if ( status.available ) this.root.appendChild( this.calendarField() );
    this.root.appendChild( this.disconnectForm() );
  },

  // Which calendar: the list comes from Google, so it loads on its own.
  calendarField() {
    const wrap = Dom.el( "div" , { class: "mb-3" } );
    const select = Ui.select( [ { value: this.status.calendar_id , text: this.status.calendar_id === "primary" ? T( "google.calendar_primary" ) : this.status.calendar_id } ] , this.status.calendar_id );
    select.disabled = true;
    wrap.appendChild( Ui.field( "google.calendar_label" , select ) );
    wrap.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "google.calendar_hint" ) } ) );
    const fill = ( calendars ) => {
      select.replaceChildren();
      calendars.forEach( ( calendar ) => {
        const value = calendar.primary ? "primary" : calendar.id;
        const node = Dom.el( "option" , { text: calendar.primary ? T( "google.calendar_primary" ) + " (" + calendar.summary + ")" : calendar.summary , attrs: { value: value } } );
        if ( value === this.status.calendar_id ) node.selected = true;
        select.appendChild( node );
      } );
      select.disabled = false;
    };
    if ( this.calendars ) {
      fill( this.calendars );
    } else {
      Api.request( "/api/google/calendars" ).then( ( reply ) => {
        this.calendars = reply.calendars;
        fill( this.calendars );
      } ).catch( function () { /* stays on the one saved */ } );
    }
    select.addEventListener( "change" , async () => {
      try {
        this.status = await Api.post( "/api/google" , { calendar_id: select.value } );
        Ui.toast( T( "google.calendar_saved" ) , "ok" );
      } catch ( error ) {
        App.showError( error.message );
      }
    } );
    return wrap;
  },

  disconnectForm() {
    const wrap = Dom.el( "div" , { class: "border-top pt-3" } );
    let removeEvents = false;
    wrap.appendChild( Ui.checkbox( "google.remove_events" , false , function ( checked ) { removeEvents = checked; } ) );
    wrap.appendChild( Dom.el( "div" , { class: "row-actions mt-2" , children: [ Ui.button( "google.disconnect" , async () => {
      if ( !( await Dom.confirm( T( removeEvents ? "google.disconnect_confirm_remove" : "google.disconnect_confirm" ) ) ) ) return;
      try {
        await Api.post( "/api/google/disconnect" , { remove_events: removeEvents } );
        this.calendars = null;
        Ui.toast( T( "google.disconnected" ) , "ok" );
        await this.reload();
      } catch ( error ) {
        App.showError( error.message );
      }
    } , "danger" ) ] } ) );
    return wrap;
  },
};
