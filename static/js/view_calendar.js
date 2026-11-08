// The calendar: what a board with the calendar layout shows in place of its
// columns. Every card is filed under a day (card.day, "2026-11-07"), and each
// day has its own To do, Doing and Done, from the card's flags as on a staged
// board. Three ways to look: a month of day cells with counts, a week of day
// columns, or one day. The board's lists still hold the cards, so switching
// the layout back shows them where they were; new cards go into the first
// list. Cards with no day wait in a tray under the days, to be dragged onto
// one.
//
// Days are strings throughout. Turning one into a Date (parse) gives a
// holder at local midnight, only for stepping and formatting -- the same
// convention Dates uses for calendar days.
const CalendarView = {
  anchor: "",
  // The stage each day column is showing, by day.
  dayStages: {},
  // Boards already carried over today in this tab, by id.
  carried: {},

  modes: [ "month" , "week" , "day" ],

  parse( day ) {
    const parts = day.split( "-" ).map( Number );
    return new Date( parts[ 0 ] , parts[ 1 ] - 1 , parts[ 2 ] );
  },

  iso( date ) { return Dates.isoDay( date ); },

  today() { return this.iso( Dates.today() ); },

  // Monday of the week a day is in.
  weekStart( date ) { return Dates.addDays( date , -( ( date.getDay() + 6 ) % 7 ) ); },

  format( date , options ) { return new Intl.DateTimeFormat( undefined , options ).format( date ); },

  mode( ctx ) {
    let saved = "";
    try { saved = window.localStorage.getItem( "calendar-mode:" + ctx.board.id ) || ""; } catch ( error ) { saved = ""; }
    if ( this.modes.indexOf( saved ) >= 0 ) return saved;
    return Stages.query.matches ? "day" : "week";
  },

  setMode( ctx , mode ) {
    try { window.localStorage.setItem( "calendar-mode:" + ctx.board.id , mode ); } catch ( error ) { /* this tab only */ }
    BoardPage.render();
  },

  step( ctx , direction ) {
    const mode = this.mode( ctx );
    const date = this.parse( this.anchor );
    if ( mode === "month" ) {
      date.setDate( 1 );
      date.setMonth( date.getMonth() + direction );
    } else {
      date.setDate( date.getDate() + direction * ( mode === "week" ? 7 : 1 ) );
    }
    this.anchor = this.iso( date );
    BoardPage.render();
  },

  // Open cards by day, each day's in board order; "" holds the undated.
  byDay( ctx ) {
    const days = {};
    ctx.cards.forEach( function ( card ) {
      if ( card.archived ) return;
      const key = card.day || "";
      ( days[ key ] = days[ key ] || [] ).push( card );
    } );
    Object.keys( days ).forEach( function ( key ) { days[ key ].sort( function ( a , b ) { return a.position - b.position; } ); } );
    return days;
  },

  render( root , ctx ) {
    this.carryOver( ctx );
    if ( !this.anchor ) this.anchor = this.today();
    const mode = this.mode( ctx );
    const days = this.byDay( ctx );
    root.appendChild( this.toolbar( ctx , mode ) );
    root.appendChild( mode === "month" ? this.month( ctx , days ) : this.columns( ctx , mode , days ) );
    if ( mode === "week" ) this.showToday( root , mode );
    const tray = this.tray( ctx , days[ "" ] || [] );
    if ( tray ) root.appendChild( tray );
    if ( ctx.canEdit && typeof Sortable !== "undefined" ) this.sortable( root , ctx );
  },

  // Once a day per tab, on opening the board: unfinished cards from past
  // days move to today, if the board carries over. The server does nothing
  // when there is nothing to move, and the poll brings the moves to others.
  carryOver( ctx ) {
    const today = this.today();
    if ( !ctx.board.carry_over || !ctx.canEdit || this.carried[ ctx.board.id ] === today ) return;
    this.carried[ ctx.board.id ] = today;
    ctx.client.carryOver( today ).then( function ( result ) {
      if ( result && result.moved > 0 ) ctx.reload();
    } ).catch( function () { /* tried again tomorrow, or on reload */ } );
  },

  toolbar( ctx , mode ) {
    const bar = Dom.el( "div" , { class: "calendar-toolbar" } );
    const nav = Dom.el( "div" , { class: "calendar-nav" } );
    nav.appendChild( Ui.iconButton( "chevron-left" , T( "calendar.previous" ) , () => this.step( ctx , -1 ) , "calendar-step" ) );
    nav.appendChild( Ui.button( "calendar.today" , () => {
      this.anchor = this.today();
      BoardPage.render();
    } , "secondary small" ) );
    nav.appendChild( Ui.iconButton( "chevron-right" , T( "calendar.next" ) , () => this.step( ctx , 1 ) , "calendar-step" ) );
    bar.appendChild( nav );
    bar.appendChild( Dom.el( "h2" , { class: "calendar-title" , text: this.title( mode ) } ) );
    const group = Dom.el( "div" , { class: "btn-group calendar-modes" , attrs: { role: "group" , "aria-label": T( "calendar.mode_label" ) } } );
    this.modes.forEach( ( name ) => {
      const active = name === mode;
      group.appendChild( Dom.el( "button" , {
        class: Ui.buttonClass( "secondary small" + ( active ? " active" : "" ) ),
        text: T( "calendar.mode_" + name ),
        attrs: { type: "button" , "aria-pressed": active ? "true" : "false" },
        on: { click: () => this.setMode( ctx , name ) },
      } ) );
    } );
    bar.appendChild( group );
    return bar;
  },

  title( mode ) {
    const date = this.parse( this.anchor );
    if ( mode === "month" ) return this.format( date , { month: "long" , year: "numeric" } );
    if ( mode === "day" ) return this.format( date , { weekday: "long" , month: "long" , day: "numeric" , year: "numeric" } );
    const start = this.weekStart( date );
    const end = Dates.addDays( start , 6 );
    try {
      return new Intl.DateTimeFormat( undefined , { month: "short" , day: "numeric" , year: "numeric" } ).formatRange( start , end );
    } catch ( error ) {
      return this.format( start , { month: "short" , day: "numeric" } ) + " – " + this.format( end , { month: "short" , day: "numeric" , year: "numeric" } );
    }
  },

  // The first time a week is shown, scroll its today (or its first day)
  // into view; after that the board keeps whatever scroll the person left.
  // Later than the render, which puts the old scroll back itself.
  showToday( root , mode ) {
    const key = mode + ":" + this.weekStart( this.parse( this.anchor ) ).getTime();
    if ( this.scrolledFor === key ) return;
    this.scrolledFor = key;
    window.setTimeout( function () {
      const scroller = root.querySelector( ".calendar-days" );
      const today = root.querySelector( ".calendar-day.is-today" );
      if ( scroller ) scroller.scrollLeft = today ? Math.max( today.offsetLeft - scroller.offsetLeft - 16 , 0 ) : 0;
    } , 0 );
  },

  // A week of day columns, or the one day, centred.
  columns( ctx , mode , days ) {
    const first = mode === "week" ? this.weekStart( this.parse( this.anchor ) ) : this.parse( this.anchor );
    const scroller = Dom.el( "div" , { class: "kanban scroll-x staged calendar-days" + ( mode === "day" ? " centered calendar-one-day" : "" ) } );
    for ( let index = 0; index < ( mode === "week" ? 7 : 1 ); index += 1 ) {
      const day = this.iso( Dates.addDays( first , index ) );
      scroller.appendChild( this.column( ctx , day , days[ day ] || [] , mode ) );
    }
    return scroller;
  },

  column( ctx , day , cards , mode ) {
    const today = this.today();
    const stage = this.dayStages[ day ] || "todo";
    const shown = cards.filter( function ( card ) { return Stages.of( card ) === stage; } );
    const column = Dom.el( "section" , { class: "kanban-column calendar-day" + ( day === today ? " is-today" : "" ) + ( day < today ? " is-past" : "" ) , attrs: { "data-day": day } } );
    const header = Dom.el( "header" , { class: "column-header" } );
    const date = this.parse( day );
    const heading = Dom.el( "h2" , { class: "column-title" } );
    const label = this.format( date , mode === "day" ? { weekday: "long" , month: "short" , day: "numeric" } : { weekday: "short" , day: "numeric" } );
    if ( mode === "week" ) {
      heading.appendChild( Dom.el( "button" , { class: "calendar-day-link" , text: label , attrs: { type: "button" , title: T( "calendar.open_day" ) } , on: { click: () => {
        this.anchor = day;
        this.setMode( ctx , "day" );
      } } } ) );
    } else {
      heading.textContent = label;
    }
    header.appendChild( heading );
    if ( day === today ) header.appendChild( Dom.el( "span" , { class: "calendar-today-badge" , text: T( "calendar.today" ) } ) );
    header.appendChild( Dom.el( "span" , { class: "count" , text: String( cards.length ) } ) );
    column.appendChild( header );

    const counts = { todo: 0 , doing: 0 , done: 0 };
    cards.forEach( function ( card ) { counts[ Stages.of( card ) ] += 1; } );
    column.appendChild( Stages.buttons( ctx , "stage-tabs" , "board.stage_tabs" , Stages.order , stage , counts , ( picked ) => {
      this.dayStages[ day ] = picked;
      BoardPage.render();
    } ) );

    const container = Dom.el( "div" , { class: "kanban-cards" , attrs: { "data-day": day , "data-stage": stage } } );
    shown.forEach( function ( card ) { container.appendChild( KanbanView.face( card , ctx ) ); } );
    if ( !shown.length ) container.appendChild( Dom.el( "p" , { class: "muted small-print calendar-empty" , text: T( "calendar.empty" ) } ) );
    column.appendChild( container );
    if ( ctx.canEdit ) column.appendChild( this.quickAdd( ctx , day , stage ) );
    return column;
  },

  // The list new cards go into: the board's first. A board with none gets
  // one first.
  async homeList( ctx ) {
    const lists = ctx.lists();
    if ( lists.length ) return lists[ 0 ].id;
    const list = await ctx.client.createList( T( "calendar.list_title" ) );
    return list.id;
  },

  quickAdd( ctx , day , stage ) {
    return QuickAdd.build( "day-" + day , "board.add_card" , "board.card_title_placeholder" , ( title , image ) => {
      return ctx.mutate( async () => {
        const listId = await this.homeList( ctx );
        const fields = Object.assign( { day: day } , stage !== "todo" ? Stages.fields( stage ) : {} );
        const card = await ctx.client.createCard( listId , title || T( "board.image_card_title" ) , undefined , fields );
        if ( image ) await ctx.client.uploadImage( card.id , await CardBits.shrinkImage( image ) );
        return card;
      } );
    } , undefined , true );
  },

  // A month of day cells, Monday first, with the days either side that
  // fill its weeks. A cell shows its counts and first open cards; clicking
  // it opens the day.
  month( ctx , days ) {
    const today = this.today();
    const anchor = this.parse( this.anchor );
    const first = new Date( anchor.getFullYear() , anchor.getMonth() , 1 );
    const last = new Date( anchor.getFullYear() , anchor.getMonth() + 1 , 0 );
    const start = this.weekStart( first );
    const grid = Dom.el( "div" , { class: "calendar-month" , attrs: { role: "grid" } } );
    for ( let index = 0; index < 7; index += 1 ) {
      grid.appendChild( Dom.el( "div" , { class: "calendar-weekday" , text: this.format( Dates.addDays( start , index ) , { weekday: "short" } ) } ) );
    }
    for ( let date = start; date <= last || date.getDay() !== 1; date = Dates.addDays( date , 1 ) ) {
      const day = this.iso( date );
      const cards = days[ day ] || [];
      const counts = { todo: 0 , doing: 0 , done: 0 };
      cards.forEach( function ( card ) { counts[ Stages.of( card ) ] += 1; } );
      const cell = Dom.el( "button" , {
        class: "calendar-cell" + ( day === today ? " is-today" : "" ) + ( date.getMonth() !== anchor.getMonth() ? " other-month" : "" ) + ( day < today ? " is-past" : "" ),
        attrs: { type: "button" , "data-day": day , "aria-label": this.format( date , { weekday: "long" , month: "long" , day: "numeric" } ) + ( cards.length ? " · " + I18n.fill( "calendar.cards_count" , { n: cards.length } ) : "" ) },
        on: { click: () => {
          this.anchor = day;
          this.setMode( ctx , "day" );
        } },
      } );
      cell.appendChild( Dom.el( "span" , { class: "calendar-date" , text: String( date.getDate() ) } ) );
      const badges = Dom.el( "span" , { class: "calendar-counts" } );
      Stages.order.forEach( function ( stage ) {
        if ( !counts[ stage ] ) return;
        badges.appendChild( Dom.el( "span" , { class: "calendar-count stage-" + stage , text: String( counts[ stage ] ) , attrs: { title: Stages.label( ctx , stage ) } } ) );
      } );
      cell.appendChild( badges );
      const open = cards.filter( function ( card ) { return !card.done; } );
      const titles = Dom.el( "span" , { class: "calendar-titles" } );
      open.slice( 0 , 3 ).forEach( function ( card ) {
        const line = Dom.el( "span" , { class: "calendar-card-title" + ( card.doing ? " is-doing" : "" ) } );
        Markdown.renderInline( card.title , line );
        titles.appendChild( line );
      } );
      if ( open.length > 3 ) titles.appendChild( Dom.el( "span" , { class: "muted" , text: I18n.fill( "calendar.more" , { n: open.length - 3 } ) } ) );
      cell.appendChild( titles );
      grid.appendChild( cell );
    }
    return grid;
  },

  // Cards with no day, to drag onto one; open them to set a day too.
  tray( ctx , cards ) {
    if ( !cards.length ) return null;
    const section = Dom.el( "section" , { class: "calendar-tray" } );
    section.appendChild( Dom.el( "h2" , { class: "calendar-tray-title" , text: I18n.fill( "calendar.undated" , { n: cards.length } ) } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( ctx.canEdit ? "calendar.undated_hint" : "calendar.undated_hint_view" ) } ) );
    const container = Dom.el( "div" , { class: "kanban-cards calendar-tray-cards" , attrs: { "data-day": "" } } );
    cards.forEach( function ( card ) { container.appendChild( KanbanView.face( card , ctx ) ); } );
    section.appendChild( container );
    return section;
  },

  // Drag a card to another day (and the stage that day is showing), or
  // within a day to reorder it.
  sortable( root , ctx ) {
    Dom.all( ".kanban-cards[data-day]" , root ).forEach( ( container ) => {
      Sortable.create( container , {
        group: "calendar-cards",
        animation: 150,
        forceFallback: true,
        delay: 150,
        delayOnTouchOnly: true,
        ghostClass: "drag-ghost",
        draggable: ".card-face",
        filter: ".status-icons",
        preventOnFilter: false,
        onStart: function () { ctx.setDragging( true ); },
        onEnd: function ( event ) {
          ctx.setDragging( false );
          CalendarView.dropped( event , ctx );
        },
      } );
    } );
  },

  dropped( event , ctx ) {
    const item = event.item;
    const to = event.to;
    if ( event.from === to && event.oldIndex === event.newIndex ) return;
    const cardId = Number( item.getAttribute( "data-card-id" ) );
    const card = ctx.card( cardId );
    if ( !card ) return;
    const neighbour = function ( node ) {
      return node && node.hasAttribute( "data-card-id" ) ? Number( node.getAttribute( "data-position" ) ) : null;
    };
    const fields = { position: CardBits.positionBetween( neighbour( item.previousElementSibling ) , neighbour( item.nextElementSibling ) ) };
    const day = to.getAttribute( "data-day" ) || "";
    if ( day !== ( card.day || "" ) ) fields.day = day;
    const stage = to.getAttribute( "data-stage" );
    if ( stage && stage !== Stages.of( card ) ) Object.assign( fields , Stages.fields( stage ) );
    ctx.mutate( function () { return ctx.client.patchCard( cardId , fields ); } );
  },
};
