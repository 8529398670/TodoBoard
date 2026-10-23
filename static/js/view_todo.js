// Todo: the same lists and cards, read as checklists. Each list is a section,
// each card a row with the same Doing / Done icons as a kanban card (see
// CardBits.statusIcons). Rows drag between sections like cards between
// columns.
const TodoView = {
  hideDone() {
    try { return window.localStorage.getItem( "todo_hide_done" ) === "1"; } catch ( error ) { return false; }
  },

  // Which rows have their body open, for this tab: card ids.
  openBodies() {
    try { return JSON.parse( window.sessionStorage.getItem( "todo_open_bodies" ) || "[]" ); } catch ( error ) { return []; }
  },

  toggleBody( cardId ) {
    const open = this.openBodies();
    const index = open.indexOf( cardId );
    if ( index >= 0 ) open.splice( index , 1 );
    else open.push( cardId );
    try { window.sessionStorage.setItem( "todo_open_bodies" , JSON.stringify( open ) ); } catch ( error ) { /* not remembered */ }
  },

  render( root , ctx ) {
    const wrap = Dom.el( "div" , { class: "todo" } );
    const toolbar = Dom.el( "div" , { class: "todo-toolbar" } );
    toolbar.appendChild( Ui.checkbox( "todo.hide_done" , this.hideDone() , function ( checked ) {
      try { window.localStorage.setItem( "todo_hide_done" , checked ? "1" : "0" ); } catch ( error ) { /* per-tab only */ }
      BoardPage.render();
    } ) );
    wrap.appendChild( toolbar );

    ctx.lists().forEach( ( list ) => wrap.appendChild( this.section( list , ctx ) ) );
    if ( ctx.canEdit && !ctx.isThree() && !ctx.listOnly ) {
      wrap.appendChild( QuickAdd.build( "list" , "board.add_list" , "board.list_title_placeholder" , function ( title ) {
        return ctx.mutate( function () { return ctx.client.createList( title ); } );
      } ) );
    }
    root.appendChild( wrap );

    if ( !ctx.canEdit || typeof Sortable === "undefined" ) return;
    Dom.all( ".todo-items" , wrap ).forEach( function ( container ) {
      Sortable.create( container , {
        group: "todo",
        animation: 150,
        forceFallback: true,
        handle: ".todo-handle",
        draggable: ".todo-row:not(.is-archived)",
        ghostClass: "drag-ghost",
        onStart: function () { ctx.setDragging( true ); },
        onEnd: function ( event ) {
          ctx.setDragging( false );
          KanbanView.dropped( event , ctx );
        },
      } );
    } );
  },

  section( list , ctx ) {
    const all = ctx.cardsIn( list.id );
    const shown = ctx.shownIn( list.id );
    const cards = this.hideDone() ? shown.filter( function ( card ) { return !card.done; } ) : shown;
    const done = all.filter( function ( card ) { return card.done; } ).length;

    const section = Dom.el( "section" , { class: "todo-section card" } );
    const header = Dom.el( "header" , { class: "todo-header" } );
    const title = Dom.el( "h2" );
    Markdown.renderInline( list.title , title );
    header.appendChild( title );
    header.appendChild( Dom.el( "span" , { class: "count" , text: I18n.fill( "todo.progress" , { done: done , total: all.length } ) } ) );
    header.appendChild( ListMenu.button( list , ctx ) );
    section.appendChild( header );

    if ( all.length ) {
      const bar = Dom.el( "div" , { class: "progress" , attrs: { role: "progressbar" , "aria-valuemin": "0" , "aria-valuemax": String( all.length ) , "aria-valuenow": String( done ) } } );
      const fill = Dom.el( "div" , { class: "progress-fill" } );
      fill.style.width = Math.round( 100 * done / all.length ) + "%";
      bar.appendChild( fill );
      section.appendChild( bar );
    }

    const items = Dom.el( "ul" , { class: "todo-items" , attrs: { "data-list-id": list.id } } );
    cards.forEach( ( card ) => items.appendChild( this.row( card , ctx , all ) ) );
    section.appendChild( items );

    if ( ctx.canEdit ) section.appendChild( QuickAdd.build( "card-" + list.id , "todo.add_item" , "board.card_title_placeholder" , function ( text ) {
      return ctx.mutate( function () { return ctx.client.createCard( list.id , text ); } );
    } ) );
    return section;
  },

  // cards is the row's column (open cards), for its number.
  row( card , ctx , cards ) {
    const row = Dom.el( "li" , { class: "todo-row" + ( card.done ? " is-done" : "" ) + ( card.archived ? " is-archived" : "" ) , attrs: { "data-card-id": card.id , "data-position": card.position } } );
    if ( ctx.canEdit && !card.archived ) row.appendChild( Dom.el( "span" , { class: "todo-handle" , text: T( "todo.drag_symbol" ) , attrs: { "aria-hidden": "true" } } ) );
    if ( ctx.numbered() && !card.archived ) row.appendChild( CardBits.numberField( card , cards , ctx ) );

    const body = Dom.el( "button" , { class: "todo-body" , attrs: { type: "button" } , on: { click: function () { ctx.openCard( card.id ); } } } );
    const title = Dom.el( "span" , { class: "todo-title" } );
    Markdown.renderInline( card.title , title );
    body.appendChild( title );
    const meta = Dom.el( "span" , { class: "card-meta" } );
    [ CardBits.labels( card , ctx.board ) , CardBits.due( card ) , CardBits.checklist( card ) , CardBits.assignees( card , ctx.people ) ]
      .filter( Boolean ).forEach( function ( node ) { meta.appendChild( node ); } );
    if ( meta.childNodes.length ) body.appendChild( meta );
    row.appendChild( body );
    row.appendChild( card.archived ? CardBits.restoreIcon( card , ctx ) : CardBits.statusIcons( card , ctx ) );

    // A card with a body gets a disclosure toggle; open, the body shows
    // rendered under the row.
    if ( CardBits.hasDescription( card ) ) {
      const open = this.openBodies().indexOf( card.id ) >= 0;
      row.appendChild( Dom.el( "button" , {
        class: "secondary small body-toggle",
        text: T( open ? "card.body_open_symbol" : "card.body_closed_symbol" ),
        attrs: { type: "button" , "aria-expanded": open ? "true" : "false" , "aria-label": T( "card.body_toggle" ) , title: T( "card.body_toggle" ) },
        on: { click: () => {
          this.toggleBody( card.id );
          BoardPage.render();
        } },
      } ) );
      if ( open ) {
        const full = Dom.el( "div" , { class: "todo-row-body" } );
        Markdown.render( card.description , full );
        row.appendChild( full );
      }
    }
    return row;
  },
};
