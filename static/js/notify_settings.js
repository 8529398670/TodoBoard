// The account page's notification settings: the Pushover user key, when
// reminders arrive, and the defaults every board and card starts from. The
// key itself is never sent back to the browser once saved -- only a hint of
// it -- so the field is left blank to keep it and typed into to replace it.
const NotifySettings = {
  prefs: null,

  // Lead times offered for due reminders, in minutes.
  leads: [ 10080 , 2880 , 1440 , 180 , 60 , 15 ],

  async init() {
    try {
      this.prefs = await Api.request( "/api/notify/prefs" );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    this.render();
    if ( window.location.hash === "#notifications" ) Dom.get( "notifications" ).scrollIntoView();
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

  render() {
    const prefs = this.prefs;
    Dom.show( Dom.get( "notify-server-off" ) , !prefs.push_enabled );
    const root = Dom.get( "notify-form-root" );
    Dom.clear( root );
    const form = Dom.el( "form" , { class: "stack" , attrs: { autocomplete: "off" } } );

    const key = Dom.el( "input" , { attrs: { maxlength: "30" , autocomplete: "off" , spellcheck: "false" ,
      placeholder: prefs.has_pushover_key ? I18n.fill( "notify.key_saved_placeholder" , { hint: prefs.pushover_key_hint } ) : T( "notify.key_placeholder" ) } } );
    const device = Dom.el( "input" , { attrs: { maxlength: "25" , placeholder: T( "notify.device_placeholder" ) } } );
    device.value = prefs.pushover_device || "";
    const row = Dom.el( "div" , { class: "form-row" } );
    row.appendChild( Ui.field( "notify.key_label" , key ) );
    row.appendChild( Ui.field( "notify.device_label" , device ) );
    form.appendChild( row );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "notify.key_help" ) } ) );

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
      const input = Dom.el( "input" , { attrs: { type: "checkbox" } } );
      input.checked = prefs.reminder_lead_minutes.indexOf( minutes ) >= 0;
      leadBoxes[ minutes ] = input;
      leads.appendChild( Dom.el( "label" , { class: "check" , children: [ input , Dom.el( "span" , { text: this.leadLabel( minutes ) } ) ] } ) );
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
    actions.appendChild( Dom.el( "button" , { text: T( "notify.save" ) , attrs: { type: "submit" } } ) );
    actions.appendChild( Ui.button( "notify.test" , () => this.test() , "secondary" ) );
    if ( prefs.has_pushover_key ) {
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
