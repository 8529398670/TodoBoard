// Per-board and per-card notification subscriptions. Each overrides the
// person's defaults (set on the account page) for that board or card; "use
// my defaults" removes the override again. The server resolves card, then
// board, then defaults -- the "source" it reports says which one applied.
const Subscribe = {
  openBoard( ctx , back ) {
    Panel.open( "notify.board_title" , ( body ) => {
      // render redraws body from scratch, so it gets a box of its own below
      // the back button.
      const box = Dom.el( "div" );
      body.appendChild( box );
      this.render( box , {
        load: function () { return ctx.client.boardSubscription(); },
        save: function ( events ) { return ctx.client.setBoardSubscription( events ); },
        intro: T( "notify.board_intro" ),
      } );
    } , back );
  },

  openCard( ctx , card ) {
    Panel.open( "notify.card_title" , ( body ) => {
      this.render( body , {
        load: function () { return ctx.client.cardSubscription( card.id ); },
        save: function ( events ) { return ctx.client.setCardSubscription( card.id , events ); },
        intro: I18n.fill( "notify.card_intro" , { card: card.title } ),
      } );
    } );
  },

  async render( body , source ) {
    Dom.clear( body );
    let state;
    try {
      state = await source.load();
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    if ( !state.push_enabled ) body.appendChild( Dom.el( "p" , { class: "notice notice-warn" , text: T( "notify.server_off" ) } ) );
    else if ( !state.has_pushover_key ) {
      body.appendChild( Dom.el( "p" , { class: "notice notice-warn" , children: [
        Dom.el( "span" , { text: T( "notify.no_key" ) + " " } ),
        Dom.el( "a" , { text: T( "notify.account_link" ) , attrs: { href: "/account#notifications" } } ),
      ] } ) );
    }
    body.appendChild( Dom.el( "p" , { class: "muted" , text: source.intro } ) );

    const events = Object.assign( {} , state.events );
    const fields = Dom.el( "div" , { class: "check-list" } );
    [ "due" , "assigned" , "changes" ].forEach( function ( name ) {
      fields.appendChild( Ui.checkbox( "notify.event_" + name , events[ name ] , function ( checked ) { events[ name ] = checked; } ) );
    } );
    body.appendChild( fields );
    body.appendChild( Dom.el( "p" , { class: "muted" , text: T( "notify.source_" + state.source ) } ) );
    body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Dom.el( "button" , { class: "small" , text: T( "board.save" ) , attrs: { type: "button" } , on: { click: async () => {
        try {
          await source.save( events );
          Ui.toast( T( "notify.saved" ) , "ok" );
          this.render( body , source );
        } catch ( error ) { Ui.error( error ); }
      } } } ),
      Ui.button( "notify.use_defaults" , async () => {
        try {
          await source.save( null );
          this.render( body , source );
        } catch ( error ) { Ui.error( error ); }
      } ),
    ] } ) );
  },
};
