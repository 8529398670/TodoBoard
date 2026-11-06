// The Notifications screen: this browser and the others that take
// notifications, the Pushover user key, when reminders arrive, and the
// defaults every board and card starts from, under a one-line summary of what
// is on. The key itself is never sent back to the browser once saved -- only a hint of
// it -- so the field is left blank to keep it and typed into to replace it.
const NotifySettings = {
  prefs: null,
  // What the server says about browsers: the key to subscribe against, and
  // every browser this person has set up.
  push: { public_key: "" , devices: [] },
  // Whether this browser is one of them.
  subscribed: false,

  // Lead times offered for due reminders, in minutes.
  leads: [ 10080 , 2880 , 1440 , 180 , 60 , 15 ],

  async init() {
    try {
      this.prefs = await Api.request( "/api/notify/prefs" );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    await this.loadBrowsers();
    // A board's Notifications panel links straight here (#notifications);
    // nav.js shows this screen for that hash.
    this.render();
  },

  // Never in the way of the rest of the screen: everything but this section
  // works without it.
  async loadBrowsers() {
    if ( !this.prefs.web_push_enabled ) return;
    try {
      this.push = await Api.request( "/api/notify/devices" );
      if ( PushClient.supported() ) {
        await PushClient.checkIn( this.push.public_key , this.push.devices );
        this.subscribed = !!( await PushClient.subscription() );
        this.push = await Api.request( "/api/notify/devices" );
      }
    } catch ( error ) {
      this.subscribed = false;
    }
  },

  // After a browser is added or removed: what can reach this person changed.
  async refreshBrowsers() {
    try {
      this.prefs = await Api.request( "/api/notify/prefs" );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    await this.loadBrowsers();
    this.render();
  },

  async setBrowser( on ) {
    try {
      if ( on ) await PushClient.enable( this.push.public_key );
      else await PushClient.disable();
    } catch ( error ) {
      App.showError( error.message );
    }
    await this.refreshBrowsers();
  },

  async removeDevice( device ) {
    try {
      if ( device.id === PushClient.savedId() ) await PushClient.disable();
      else await Api.post( "/api/notify/devices/" + encodeURIComponent( device.id ) + "/delete" );
    } catch ( error ) {
      App.showError( error.message );
    }
    await this.refreshBrowsers();
  },

  // This browser's switch, then every browser set up, this one included.
  browserSection() {
    const section = Dom.el( "section" , { class: "stack" } );
    section.appendChild( Dom.el( "h3" , { class: "h6 mb-0" , text: T( "notify.browser_heading" ) } ) );
    // In this order: an old iPhone should not be sent to its Home Screen for
    // something that will not work there either.
    const obstacle = PushClient.iosTooOld() ? "notify.browser_ios_old" : PushClient.needsHomeScreen() ? "notify.browser_ios" : !PushClient.supported() ? "notify.browser_unsupported" : "";
    if ( obstacle ) {
      section.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: T( obstacle ) } ) );
    } else if ( this.subscribed ) {
      section.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: T( this.prefs.notify_via === "pushover" ? "notify.browser_on_unused" : "notify.browser_on" ) } ) );
      section.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ Ui.button( "notify.browser_disable" , () => this.setBrowser( false ) , "secondary" ) ] } ) );
    } else if ( PushClient.blocked() ) {
      section.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: T( "notify.browser_blocked" ) } ) );
    } else {
      section.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: T( "notify.browser_intro" ) } ) );
      section.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ Ui.button( "notify.browser_enable" , () => this.setBrowser( true ) , "secondary" ) ] } ) );
    }
    if ( this.push.devices.length ) {
      const list = Dom.el( "ul" , { class: "link-list" } );
      const here = PushClient.savedId();
      this.push.devices.forEach( ( device ) => {
        const when = device.last_success ? I18n.fill( "notify.device_last" , { when: Dates.format( device.last_success ) } ) : T( "notify.device_never" );
        const name = device.label || T( "notify.device_unnamed" );
        const row = Dom.el( "li" , { class: "link-row" } );
        row.appendChild( Dom.el( "div" , { children: [
          Dom.el( "strong" , { text: device.id === here ? I18n.fill( "notify.device_this" , { name: name } ) : name } ),
          Dom.el( "div" , { class: "muted" , text: when } ),
        ] } ) );
        row.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ Ui.button( "notify.device_remove" , () => this.removeDevice( device ) , "secondary small danger" ) ] } ) );
        list.appendChild( row );
      } );
      section.appendChild( list );
    }
    return section;
  },

  leadLabel( minutes ) {
    if ( minutes % 1440 === 0 ) return I18n.fill( minutes === 1440 ? "notify.lead_day" : "notify.lead_days" , { n: minutes / 1440 } );
    if ( minutes % 60 === 0 ) return I18n.fill( minutes === 60 ? "notify.lead_hour" : "notify.lead_hours" , { n: minutes / 60 } );
    return I18n.fill( "notify.lead_minutes" , { n: minutes } );
  },

  timezones() {
    let zones = [];
    try { zones = Intl.supportedValuesOf( "timeZone" ); } catch ( error ) { zones = []; }
    if ( zones.indexOf( "UTC" ) < 0 ) zones.unshift( "UTC" );
    return zones;
  },

  // The screen's one line: is anything going to arrive, and when.
  summary() {
    const prefs = this.prefs;
    if ( !prefs.push_enabled ) return T( "notify.summary_server_off" );
    if ( !prefs.can_receive ) return T( "notify.summary_not_set_up" );
    if ( prefs.paused ) return T( "notify.summary_paused" );
    if ( !prefs.reminder_lead_minutes.length ) return T( "notify.summary_on" );
    return I18n.fill( "notify.summary_on_leads" , { leads: prefs.reminder_lead_minutes.map( ( minutes ) => this.leadLabel( minutes ) ).join( ", " ) } );
  },

  render() {
    const prefs = this.prefs;
    Dom.text( Dom.get( "notify-summary" ) , this.summary() );
    Dom.show( Dom.get( "notify-server-off" ) , !prefs.push_enabled );
    const root = Dom.get( "notify-form-root" );
    Dom.clear( root );
    const form = Dom.el( "form" , { class: "stack" , attrs: { autocomplete: "off" } } );

    if ( prefs.web_push_enabled ) form.appendChild( this.browserSection() );

    // A choice only when there are two ways to choose from. Someone left on
    // one the server has since turned off still sees it, to get back out.
    let via = null;
    if ( ( prefs.pushover_enabled && prefs.web_push_enabled ) || prefs.notify_via !== "both" ) {
      via = Ui.select( [ "both" , "pushover" , "devices" ].map( function ( value ) { return { value: value , text: T( "notify.via_" + value ) }; } ) , prefs.notify_via );
      form.appendChild( Dom.el( "div" , { class: "form-row" , children: [ Ui.field( "notify.via_label" , via ) ] } ) );
    }

    const key = Dom.el( "input" , { attrs: { maxlength: "30" , autocomplete: "off" , spellcheck: "false" ,
      placeholder: prefs.has_pushover_key ? I18n.fill( "notify.key_saved_placeholder" , { hint: prefs.pushover_key_hint } ) : T( "notify.key_placeholder" ) } } );
    const device = Dom.el( "input" , { attrs: { maxlength: "25" , placeholder: T( "notify.device_placeholder" ) } } );
    device.value = prefs.pushover_device || "";
    // Without an app token here a Pushover key would go nowhere.
    if ( prefs.pushover_enabled ) {
      const row = Dom.el( "div" , { class: "form-row" } );
      row.appendChild( Ui.field( "notify.key_label" , key ) );
      row.appendChild( Ui.field( "notify.device_label" , device ) );
      form.appendChild( row );
      form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "notify.key_help" ) } ) );
    }

    let browserZone = "UTC";
    try { browserZone = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"; } catch ( error ) { browserZone = "UTC"; }
    const zone = Ui.select( this.timezones().map( function ( name ) { return { value: name , text: name }; } ) ,
      prefs.timezone && prefs.timezone !== "UTC" ? prefs.timezone : browserZone );
    const hours = [];
    for ( let hour = 0; hour < 24; hour += 1 ) hours.push( { value: hour , text: String( hour ).padStart( 2 , "0" ) + ":00" } );
    const allDay = Ui.select( hours , prefs.all_day_reminder_hour );
    const timing = Dom.el( "div" , { class: "form-row" } );
    timing.appendChild( Ui.field( "notify.timezone_label" , zone ) );
    timing.appendChild( Ui.field( "notify.all_day_label" , allDay ) );
    form.appendChild( timing );

    const leadBoxes = {};
    const leads = Dom.el( "fieldset" , { class: "check-list" } );
    leads.appendChild( Dom.el( "legend" , { text: T( "notify.leads_label" ) } ) );
    this.leads.forEach( ( minutes ) => {
      const box = Ui.check( this.leadLabel( minutes ) , prefs.reminder_lead_minutes.indexOf( minutes ) >= 0 );
      leadBoxes[ minutes ] = box.querySelector( "input" );
      leads.appendChild( box );
    } );
    form.appendChild( leads );

    const sets = {};
    [ "assignee_defaults" , "watch_defaults" ].forEach( function ( name ) {
      const set = Object.assign( {} , prefs[ name ] );
      sets[ name ] = set;
      const group = Dom.el( "fieldset" , { class: "check-list" } );
      group.appendChild( Dom.el( "legend" , { text: T( "notify." + name ) } ) );
      [ "due" , "assigned" , "changes" ].forEach( function ( event ) {
        group.appendChild( Ui.checkbox( "notify.event_" + event , set[ event ] , function ( checked ) { set[ event ] = checked; } ) );
      } );
      form.appendChild( group );
    } );

    let sharedWithMe = prefs.shared_with_me;
    let paused = prefs.paused;
    const general = Dom.el( "div" , { class: "check-list" } );
    general.appendChild( Ui.checkbox( "notify.shared_with_me" , sharedWithMe , function ( checked ) { sharedWithMe = checked; } ) );
    general.appendChild( Ui.checkbox( "notify.paused" , paused , function ( checked ) { paused = checked; } ) );
    form.appendChild( general );

    const actions = Dom.el( "div" , { class: "row-actions" } );
    actions.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "notify.save" ) , attrs: { type: "submit" } } ) );
    actions.appendChild( Ui.button( "notify.test" , () => this.test() , "secondary" ) );
    if ( prefs.pushover_enabled && prefs.has_pushover_key ) {
      actions.appendChild( Ui.button( "notify.remove_key" , () => this.save( { pushover_user_key: "-" } ) , "secondary danger" ) );
    }
    form.appendChild( actions );

    form.addEventListener( "submit" , ( event ) => {
      event.preventDefault();
      const chosen = this.leads.filter( function ( minutes ) { return leadBoxes[ minutes ].checked; } );
      this.save( {
        pushover_user_key: key.value.trim(),
        pushover_device: device.value.trim(),
        timezone: zone.value,
        all_day_reminder_hour: Number( allDay.value ),
        reminder_lead_minutes: chosen,
        assignee_defaults: sets.assignee_defaults,
        watch_defaults: sets.watch_defaults,
        shared_with_me: sharedWithMe,
        paused: paused,
        notify_via: via ? via.value : prefs.notify_via,
      } );
    } );
    root.appendChild( form );
  },

  // Saves the whole form; a partial (removing the key) keeps the rest as is.
  async save( fields ) {
    const prefs = this.prefs;
    const body = Object.assign( {
      pushover_user_key: "",
      pushover_device: prefs.pushover_device,
      timezone: prefs.timezone,
      all_day_reminder_hour: prefs.all_day_reminder_hour,
      reminder_lead_minutes: prefs.reminder_lead_minutes,
      assignee_defaults: prefs.assignee_defaults,
      watch_defaults: prefs.watch_defaults,
      shared_with_me: prefs.shared_with_me,
      paused: prefs.paused,
      notify_via: prefs.notify_via,
    } , fields );
    try {
      this.prefs = await Api.post( "/api/notify/prefs" , body );
      this.render();
      Ui.toast( T( "notify.saved" ) , "ok" );
    } catch ( error ) {
      App.showError( error.message );
    }
  },

  async test() {
    try {
      await Api.post( "/api/notify/test" );
      Ui.toast( T( "notify.test_sent" ) , "ok" );
    } catch ( error ) {
      App.showError( error.message );
    }
  },
};
