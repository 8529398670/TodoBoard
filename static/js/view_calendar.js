// The calendar: what a board with the calendar layout shows in place of its
// columns. Every card is filed under a day (card.day, "2026-11-07"), and each
// day has its own To do, Doing and Done, from the card's flags as on a staged
// board. Three ways to look: a month of day cells with counts, a week of day
// columns, or one day. The range buttons and, for a week or a day, the
// previous / Today / next buttons, the dates on show and the stage every day
// shows sit in the navbar, as a staged board's bar does. A week is any seven
// days in a row: it opens on Monday, Today puts today first, and the arrow
// keys slide it a day at a time. Where it was left is kept per board in this
// browser, so a refresh comes back to it. The board's lists still hold the cards, so switching
// the layout back shows them where they were; new cards go into the first
// list. Cards with no day wait in a tray under the days, to be dragged onto
// one.
//
// Days are strings throughout. Turning one into a Date (parse) gives a
// holder at local midnight, only for stepping and formatting -- the same
// convention Dates uses for calendar days.
const CalendarView = {
  anchor: "",
  // The first of the seven days the week shows; "" until a week is shown.
  weekFrom: "",
  // The board anchor and weekFrom belong to.
  boardId: null,
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
    // Back to a week that still holds the anchor keeps its first day; any
    // other week opens on the anchor's Monday.
    if ( mode === "week" && this.weekFrom ) {
      const offset = Math.round( ( this.parse( this.anchor ) - this.parse( this.weekFrom ) ) / 86400000 );
      if ( offset < 0 || offset > 6 ) this.weekFrom = "";
    }
    try { window.localStorage.setItem( "calendar-mode:" + ctx.board.id , mode ); } catch ( error ) { /* this tab only */ }
    this.remember( ctx );
    BoardPage.render();
  },

  // Where the calendar was left on this board, kept in this browser so a
  // refresh opens on the same day or week.
  remember( ctx ) {
    try {
      window.localStorage.setItem( "calendar-spot:" + ctx.board.id , JSON.stringify( { anchor: this.anchor , weekFrom: this.weekFrom } ) );
    } catch ( error ) { /* this tab only */ }
  },

  // On first showing a board (or another board in the same tab): its saved
  // spot, else today.
  recall( ctx ) {
    if ( this.boardId === ctx.board.id && this.anchor ) return;
    this.boardId = ctx.board.id;
    this.anchor = "";
    this.weekFrom = "";
    const valid = function ( day ) { return typeof day === "string" && /^\d{4}-\d{2}-\d{2}$/.test( day ) ? day : ""; };
    try {
      const saved = JSON.parse( window.localStorage.getItem( "calendar-spot:" + ctx.board.id ) || "null" );
      if ( saved ) {
        this.anchor = valid( saved.anchor );
        this.weekFrom = this.anchor ? valid( saved.weekFrom ) : "";
      }
    } catch ( error ) { /* today, then */ }
    if ( !this.anchor ) this.anchor = this.today();
  },

  // Moves by days, or for a month by months. A week moves its first day
  // and takes the anchor along, so Day opens where the week was.
  step( ctx , direction , days ) {
    const mode = this.mode( ctx );
    if ( mode === "month" ) {
      const date = this.parse( this.anchor );
      date.setDate( 1 );
      date.setMonth( date.getMonth() + direction );
      this.anchor = this.iso( date );
    } else if ( mode === "week" ) {
      this.weekFrom = this.iso( Dates.addDays( this.parse( this.firstDay() ) , direction * ( days || 7 ) ) );
      this.anchor = this.weekFrom;
    } else {
      this.anchor = this.iso( Dates.addDays( this.parse( this.anchor ) , direction ) );
    }
    this.remember( ctx );
    BoardPage.render();
  },

  // Back to today: the week then starts on it.
  goToday( ctx ) {
    this.anchor = this.today();
    if ( this.mode( ctx ) === "week" ) this.weekFrom = this.anchor;
    this.remember( ctx );
    BoardPage.render();
  },

  // The week's first day: where it was slid to, else the anchor's Monday.
  firstDay() {
    if ( !this.weekFrom ) this.weekFrom = this.iso( this.weekStart( this.parse( this.anchor ) ) );
    return this.weekFrom;
  },

  // The stage every day column shows: the navbar's.
  stage() {
    const stage = BoardPage.stage;
    return Stages.order.indexOf( stage ) >= 0 ? stage : "todo";
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
    this.recall( ctx );
    const mode = this.mode( ctx );
    const days = this.byDay( ctx );
    Dom.get( "stage-slot" ).appendChild( this.navbar( ctx , mode , days ) );
    // A month keeps its dates above the grid; a week or a day has them in
    // the navbar.
    if ( mode === "month" ) root.appendChild( this.toolbar( ctx , mode ) );
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

  // Previous, Today and next. Their tooltips name the keys that do the same.
  navButtons( ctx , mode ) {
    const nav = Dom.el( "div" , { class: "calendar-nav" } );
    const hint = mode === "month" ? "_month" : "";
    nav.appendChild( Ui.iconButton( "chevron-left" , T( "calendar.previous" + hint ) , () => this.step( ctx , -1 ) , "calendar-step" ) );
    const today = Ui.button( "calendar.today" , () => this.goToday( ctx ) , "secondary small" );
    today.title = T( mode === "week" ? "calendar.today_week_hint" : "calendar.today_hint" );
    nav.appendChild( today );
    nav.appendChild( Ui.iconButton( "chevron-right" , T( "calendar.next" + hint ) , () => this.step( ctx , 1 ) , "calendar-step" ) );
    return nav;
  },

  // A month's dates, above its grid.
  toolbar( ctx , mode ) {
    const bar = Dom.el( "div" , { class: "calendar-toolbar" } );
    bar.appendChild( this.navButtons( ctx , mode ) );
    bar.appendChild( Dom.el( "h2" , { class: "calendar-title" , text: this.title( mode ) } ) );
    return bar;
  },

  // In the navbar: the range, and for a week or a day the way to move, the
  // dates on show and the stage all its days show, with the cards in each
  // across the days on show.
  navbar( ctx , mode , days ) {
    const wrap = Dom.el( "div" , { class: "calendar-navbar" } );
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
    wrap.appendChild( group );
    if ( mode === "month" ) return wrap;
    const dates = Dom.el( "div" , { class: "calendar-navbar-dates" } );
    dates.appendChild( this.navButtons( ctx , mode ) );
    dates.appendChild( Dom.el( "span" , { class: "calendar-navbar-title" , text: this.title( mode , true ) } ) );
    wrap.appendChild( dates );
    const counts = { todo: 0 , doing: 0 , done: 0 };
    this.shownDays( mode ).forEach( function ( day ) {
      ( days[ day ] || [] ).forEach( function ( card ) { counts[ Stages.of( card ) ] += 1; } );
    } );
    wrap.appendChild( Stages.buttons( ctx , "stage-bar" , "board.stage_bar" , Stages.order , this.stage() , counts , function ( stage ) { BoardPage.setStage( stage ); } ) );
    return wrap;
  },

  // The days a week or a day shows, in order.
  shownDays( mode ) {
    if ( mode !== "week" ) return [ this.anchor ];
    const first = this.parse( this.firstDay() );
    const result = [];
    for ( let index = 0; index < 7; index += 1 ) result.push( this.iso( Dates.addDays( first , index ) ) );
    return result;
  },

  // short, for the navbar: the day column under it already spells out the
  // day in full.
  title( mode , short ) {
    const date = this.parse( this.anchor );
    if ( mode === "month" ) return this.format( date , { month: "long" , year: "numeric" } );
    if ( mode === "day" ) return this.format( date , short ? { weekday: "short" , month: "short" , day: "numeric" , year: "numeric" } : { weekday: "long" , month: "long" , day: "numeric" , year: "numeric" } );
    const start = this.parse( this.firstDay() );
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
    const key = mode + ":" + this.firstDay();
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
    const scroller = Dom.el( "div" , { class: "kanban scroll-x staged calendar-days" + ( mode === "day" ? " centered calendar-one-day" : "" ) } );
    this.shownDays( mode ).forEach( ( day ) => {
      scroller.appendChild( this.column( ctx , day , days[ day ] || [] , mode ) );
    } );
    return scroller;
  },

  column( ctx , day , cards , mode ) {
    const today = this.today();
    const stage = this.stage();
    // The day's cards in the stage on show: what is counted, numbered and
    // added to.
    const shown = cards.filter( function ( card ) { return Stages.of( card ) === stage; } );
    const column = Dom.el( "section" , { class: "kanban-column calendar-day" + ( day === today ? " is-today" : "" ) + ( day < today ? " is-past" : "" ) , attrs: { "data-day": day , "data-stage-key": "day-" + day } } );
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
    // Cards in the stage on show, of all the day's.
    header.appendChild( Dom.el( "span" , { class: "count" , text: shown.length === cards.length ? String( cards.length ) : shown.length + " / " + cards.length , attrs: { title: I18n.fill( "calendar.stage_count" , { n: shown.length , total: cards.length , stage: Stages.label( ctx , stage ) } ) } } ) );
    column.appendChild( header );

    const container = Dom.el( "div" , { class: "kanban-cards" , attrs: { "data-day": day , "data-stage": stage } } );
    shown.forEach( function ( card ) { container.appendChild( KanbanView.face( card , ctx , shown ) ); } );
    if ( !shown.length ) container.appendChild( Dom.el( "p" , { class: "muted small-print calendar-empty" , text: T( "calendar.empty" ) } ) );
    // The add box above the cards or under them, as the board says.
    const add = ctx.canEdit ? this.quickAdd( ctx , day , stage , shown ) : null;
    if ( add && ctx.addOnTop() ) column.appendChild( add );
    column.appendChild( container );
    if ( add && !ctx.addOnTop() ) column.appendChild( add );
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

  // A day's add box, with the board's arrow to add at the top or the
  // bottom. A locked direction is one for all the board's days, since the
  // days on show keep changing.
  quickAdd( ctx , day , stage , shown ) {
    return QuickAdd.forCards( null , ctx , "board.add_card" , stage , {
      key: "day-" + day,
      lockId: "calendar",
      cards: function () { return shown; },
      create: async ( title , position ) => {
        const listId = await this.homeList( ctx );
        const fields = Object.assign( { day: day } , stage !== "todo" ? Stages.fields( stage ) : {} );
        return ctx.client.createCard( listId , title , position , fields );
      },
    } );
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

  // Left and Right slide a week or a day by a day (Shift: a week), or a
  // month by a month; T goes back to today. Bound once, on the document,
  // with the guards the stage keys use.
  bindKeys() {
    document.addEventListener( "keydown" , ( event ) => {
      const arrow = event.key === "ArrowLeft" || event.key === "ArrowRight";
      const today = event.key === "t" || event.key === "T";
      if ( !arrow && !today ) return;
      if ( event.ctrlKey || event.metaKey || event.altKey || ( today && event.shiftKey ) ) return;
      const target = event.target;
      if ( target && target.closest && target.closest( "input, textarea, select, [contenteditable]:not([contenteditable=\"false\"])" ) ) return;
      if ( !BoardPage.data || BoardPage.view !== "kanban" || BoardPage.dragging ) return;
      const ctx = BoardPage.context();
      if ( !ctx.isCalendar() || ctx.cardOnly ) return;
      if ( document.querySelector( "dialog[open], .context-menu" ) ) return;
      event.preventDefault();
      if ( today ) {
        this.goToday( ctx );
        return;
      }
      const direction = event.key === "ArrowRight" ? 1 : -1;
      this.step( ctx , direction , event.shiftKey ? 7 : 1 );
    } );
  },
};
