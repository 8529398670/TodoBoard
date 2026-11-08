// Kanban: one column per list, cards dragged within and between columns.
// SortableJS does the dragging (touch included); on drop the card gets the
// midpoint of its new neighbours' positions, so a move writes one card.
const KanbanView = {
  render( root , ctx ) {
    const three = ctx.isThree();
    if ( ctx.isStaged() ) Dom.get( "stage-slot" ).appendChild( Stages.bar( ctx ) );
    if ( ctx.isAllView() ) {
      this.renderAll( root , ctx );
      return;
    }
    const scroller = Dom.el( "div" , { class: "kanban scroll-x" + ( three ? " three" : "" ) + ( ctx.isStaged() ? " staged" : "" ) + ( ctx.board.center_columns && !three ? " centered" : "" ) } );
    const lists = ctx.lists();
    lists.forEach( ( list ) => scroller.appendChild( this.column( list , ctx ) ) );
    if ( ctx.canEdit && !three && !ctx.fixedColumns ) scroller.appendChild( this.addListColumn( ctx ) );
    if ( ctx.swipeColumns() && lists.length > 1 ) root.appendChild( this.dots( scroller , lists ) );
    root.appendChild( scroller );

    if ( !ctx.canEdit || typeof Sortable === "undefined" ) return;

    Dom.all( ".kanban-cards:not(.is-obscured)" , scroller ).forEach( ( container ) => {
      Sortable.create( container , {
        group: "cards",
        animation: 150,
        forceFallback: true,
        delay: 150,
        delayOnTouchOnly: true,
        ghostClass: "drag-ghost",
        // Holding a card at the screen edge scrolls on to the next column,
        // which is how a card crosses columns one screen wide.
        scroll: true,
        scrollSensitivity: 60,
        // Archived cards stay where they are shown; the number field and
        // status icons are for typing and clicking, not for picking a card up.
        draggable: ".card-face:not(.is-archived)",
        filter: ".card-number, .status-icons",
        preventOnFilter: false,
        onStart: function () { ctx.setDragging( true ); },
        onEnd: function ( event ) {
          ctx.setDragging( false );
          KanbanView.dropped( event , ctx );
        },
      } );
    } );

    // A three-column board's columns are fixed in place, and a single
    // shared column has nowhere to go.
    if ( three || ctx.fixedColumns ) return;
    Sortable.create( scroller , {
      animation: 150,
      forceFallback: true,
      handle: ".column-handle",
      draggable: ".kanban-column[data-list-id]",
      delay: 150,
      delayOnTouchOnly: true,
      onStart: function () { ctx.setDragging( true ); },
      onEnd: function ( event ) {
        ctx.setDragging( false );
        if ( event.oldIndex === event.newIndex ) return;
        const column = event.item;
        const listId = Number( column.getAttribute( "data-list-id" ) );
        const neighbour = function ( node ) {
          return node && node.hasAttribute( "data-list-id" ) ? Number( node.getAttribute( "data-position" ) ) : null;
        };
        const position = CardBits.positionBetween( neighbour( column.previousElementSibling ) , neighbour( column.nextElementSibling ) );
        ctx.mutate( function () { return ctx.client.updateList( listId , { position: position } ); } );
      },
    } );
  },

  // Where you are among full-width columns: a dot per column (tap one to go
  // there) and the current column's name. Only drawn on a phone held upright
  // with "one full-width column at a time" on -- CSS hides it otherwise.
  dots( scroller , lists ) {
    const bar = Dom.el( "nav" , { class: "column-dots" , attrs: { "aria-label": T( "board.columns_nav" ) } } );
    const label = Dom.el( "span" , { class: "column-dots-label" } );
    const dots = lists.map( function ( list , index ) {
      return Dom.el( "button" , {
        class: "column-dot",
        attrs: { type: "button" , "aria-label": list.title , title: list.title },
        on: { click: function () {
          const column = scroller.children[ index ];
          if ( column ) column.scrollIntoView( { behavior: "smooth" , block: "nearest" , inline: "start" } );
        } },
      } );
    } );
    dots.forEach( function ( dot ) { bar.appendChild( dot ); } );
    bar.appendChild( label );
    const update = function () {
      // The column whose left edge is nearest the scroller's.
      const left = scroller.getBoundingClientRect().left;
      let index = 0;
      let best = Infinity;
      lists.forEach( function ( list , i ) {
        const column = scroller.children[ i ];
        const distance = column ? Math.abs( column.getBoundingClientRect().left - left ) : Infinity;
        if ( distance < best ) {
          best = distance;
          index = i;
        }
      } );
      dots.forEach( function ( dot , i ) {
        dot.classList.toggle( "active" , i === index );
        dot.setAttribute( "aria-current" , i === index ? "true" : "false" );
      } );
      Dom.text( label , ( index + 1 ) + " / " + lists.length + " · " + lists[ index ].title );
    };
    scroller.addEventListener( "scroll" , update , { passive: true } );
    // After board.js restores the scroll position.
    window.requestAnimationFrame( update );
    update();
    return bar;
  },

  dropped( event , ctx ) {
    const item = event.item;
    const cardId = Number( item.getAttribute( "data-card-id" ) );
    const listId = Number( event.to.getAttribute( "data-list-id" ) );
    if ( event.from === event.to && event.oldIndex === event.newIndex ) return;
    // Neighbours are the open cards either side; archived ones shown at the
    // end of a column do not count.
    const open = function ( node ) {
      return node && node.hasAttribute( "data-card-id" ) && !node.classList.contains( "is-archived" );
    };
    const neighbour = function ( node ) {
      return open( node ) ? Number( node.getAttribute( "data-position" ) ) : null;
    };
    const position = CardBits.positionBetween( neighbour( item.previousElementSibling ) , neighbour( item.nextElementSibling ) );
    const fields = { list_id: listId , position: position };
    // Into a column showing another stage, the card takes that stage. Only
    // staged columns name one; TodoView shares this and has none.
    const stage = event.to.getAttribute( "data-stage" );
    const card = ctx.card( cardId );
    if ( stage && card && Stages.of( card ) !== stage ) Object.assign( fields , Stages.fields( stage ) );
    ctx.mutate( function () { return ctx.client.patchCard( cardId , fields ); } );
  },

  // Dragging in the All list changes rank and nothing else, and finished
  // cards hold no place to change.
  rankDragging( container , ctx ) {
    if ( !ctx.canRank ) return;
    Sortable.create( container , {
      animation: 150,
      forceFallback: true,
      delay: 150,
      delayOnTouchOnly: true,
      ghostClass: "drag-ghost",
      draggable: ".card-face",
      filter: ".card-number, .status-icons",
      preventOnFilter: false,
      onStart: function () { ctx.setDragging( true ); },
      onEnd: function ( event ) {
        ctx.setDragging( false );
        if ( event.oldIndex === event.newIndex ) return;
        KanbanView.ranked( event.item , ctx );
      },
    } );
  },

  // The place a card dropped in the All list has taken. Its neighbours'
  // places are the board's, which also counts cards in the other stage, so
  // the drop is read from them rather than from where it sits in this list.
  ranked( item , ctx ) {
    const cardId = Number( item.getAttribute( "data-card-id" ) );
    const card = ctx.card( cardId );
    if ( !card ) return;
    const placeOf = function ( node ) { return node ? Number( node.getAttribute( "data-rank-place" ) ) || 0 : 0; };
    const own = card.rank_place || 0;
    const before = item.previousElementSibling;
    const above = placeOf( before );
    const below = placeOf( item.nextElementSibling );
    let place;
    // Moving down, the cards it passes each shift up one.
    if ( below ) place = own && own < below ? below - 1 : below;
    else if ( above ) place = own && own < above ? above : above + 1;
    else if ( !before ) place = 1;
    // Under an unranked card is no rank at all.
    else place = 0;
    if ( place === own ) {
      BoardPage.render();
      return;
    }
    ctx.mutate( function () { return ctx.client.patchCard( cardId , { rank_place: place } ); } );
  },

  column( list , ctx ) {
    const hidden = ctx.hiddenCount( list.id );
    if ( hidden !== null ) return this.obscuredColumn( list , hidden );
    if ( ctx.isCollapsed( list.id ) ) return this.collapsedColumn( list , ctx );
    // On a staged board a column is the one stage it is showing: counted,
    // numbered and added to as such.
    const staged = ctx.isStaged();
    const stage = staged ? ctx.stageIn( list.id ) : null;
    const cards = staged ? ctx.cardsAt( list.id , stage ) : ctx.cardsIn( list.id );
    const column = Dom.el( "section" , { class: "kanban-column" , attrs: { "data-list-id": list.id , "data-position": list.position } } );
    if ( staged ) column.setAttribute( "data-stage-key" , String( list.id ) );

    const header = Dom.el( "header" , { class: "column-header" } );
    const title = Dom.el( "h2" , { class: "column-title" + ( ctx.canEdit ? " column-handle" : "" ) } );
    Markdown.renderInline( list.title , title );
    header.appendChild( title );
    // A column set to a stage of its own says so, since it will not follow
    // the bar.
    const own = staged ? ctx.columnView( list.id ) : "";
    if ( own ) {
      const name = Stages.label( ctx , own );
      header.appendChild( Dom.el( "span" , { class: "column-view-badge" , text: name , attrs: { title: I18n.fill( "board.column_view_badge" , { stage: name } ) } } ) );
    }
    header.appendChild( Dom.el( "span" , { class: "count" , text: String( cards.length ) } ) );
    if ( ctx.canCollapse() ) {
      header.appendChild( Ui.iconButton( "arrows-collapse-vertical" , T( "board.column_collapse" ) , function () { BoardPage.setCollapsed( list.id , true ); } , "column-collapse" ) );
    }
    header.appendChild( ListMenu.button( list , ctx ) );
    column.appendChild( header );

    const container = Dom.el( "div" , { class: "kanban-cards" , attrs: { "data-list-id": list.id } } );
    if ( staged ) container.setAttribute( "data-stage" , stage );
    const shown = staged ? ctx.shownAt( list.id , stage ) : ctx.shownIn( list.id );
    shown.forEach( ( card ) => container.appendChild( this.face( card , ctx , cards ) ) );
    const add = ctx.canEdit ? QuickAdd.forCards( list , ctx , "board.add_card" , stage ) : null;
    if ( add && ctx.addOnTop() ) column.appendChild( add );
    column.appendChild( container );
    if ( add && !ctx.addOnTop() ) column.appendChild( add );
    // A free-lists board keeps each column's own width (wide screens only;
    // CSS ignores it on a phone and on a three-column board).
    if ( !ctx.isThree() && list.width ) column.style.setProperty( "--col-w" , list.width + "px" );
    if ( ctx.canEdit && !ctx.isThree() && !ctx.fixedColumns ) column.appendChild( this.resizer( column , list , ctx ) );
    return column;
  },

  // A collapsed column: a strip with its title and count, and anywhere on it
  // opens it again. It stays a column in the row, so it still drags by its
  // title and dots() still finds the others by position.
  //
  // Letting go of a drag over the strip is a click on it too, and Sortable
  // only swallows that click if the drag passed over nothing. So a click that
  // ended away from where the press began is a drag, and opens nothing. The
  // keyboard presses nothing first, so its click always counts.
  collapsedColumn( list , ctx ) {
    const count = ctx.isStaged() ? ctx.cardsAt( list.id , ctx.stageIn( list.id ) ).length : ctx.cardsIn( list.id ).length;
    let pressed = null;
    const strip = Dom.el( "section" , {
      class: "kanban-column is-collapsed",
      attrs: { "data-list-id": list.id , "data-position": list.position },
      on: {
        pointerdown: function ( event ) { pressed = { x: event.clientX , y: event.clientY }; },
        click: function ( event ) {
          const dragged = pressed && Math.hypot( event.clientX - pressed.x , event.clientY - pressed.y ) > 8;
          pressed = null;
          if ( !dragged ) BoardPage.setCollapsed( list.id , false );
        },
      },
    } );
    strip.appendChild( Ui.iconButton( "arrows-expand-vertical" , T( "board.column_expand" ) , function () {} , "column-expand" ) );
    strip.appendChild( Dom.el( "span" , { class: "count" , text: String( count ) } ) );
    const title = Dom.el( "h2" , { class: "column-title" + ( ctx.canEdit && !ctx.fixedColumns ? " column-handle" : "" ) } );
    Markdown.renderInline( list.title , title );
    strip.appendChild( title );
    return strip;
  },

  // The All list: one stage of every list, ranked ones first, in one centred
  // list in place of the columns. It is no list itself, so it has no menu or
  // add box.
  renderAll( root , ctx ) {
    const cards = ctx.allCards();
    const stage = ctx.stageIn( "all" );
    const view = Dom.el( "section" , { class: "all-view" , attrs: { "data-stage-key": "all" } } );
    view.appendChild( Stages.tabs( cards , ctx ) );
    const container = Dom.el( "div" , { class: "kanban-cards all-cards" } );
    cards.filter( function ( card ) { return Stages.of( card ) === stage; } ).forEach( ( card ) => {
      const face = this.face( card , ctx , null , true );
      face.setAttribute( "data-rank-place" , String( card.rank_place || 0 ) );
      const list = ctx.board.lists.find( function ( item ) { return item.id === card.list_id; } );
      if ( list ) {
        const from = Dom.el( "span" , { class: "card-source" } );
        Markdown.renderInline( list.title , from );
        face.appendChild( from );
      }
      container.appendChild( face );
    } );
    view.appendChild( container );
    root.appendChild( view );
    if ( stage !== "done" && typeof Sortable !== "undefined" ) this.rankDragging( container , ctx );
  },

  // The strip on a column's right edge: drag it to widen or narrow the
  // column for everyone on the board, double-click it for the standard width.
  resizer( column , list , ctx ) {
    const handle = Dom.el( "div" , { class: "column-resize" , attrs: { "aria-hidden": "true" , title: T( "board.column_resize" ) } } );
    let startX = 0;
    let startWidth = 0;
    let width = 0;
    handle.addEventListener( "pointerdown" , function ( event ) {
      if ( event.button !== 0 ) return;
      event.preventDefault();
      event.stopPropagation();
      handle.setPointerCapture( event.pointerId );
      startX = event.clientX;
      startWidth = width = column.getBoundingClientRect().width;
      column.classList.add( "is-resizing" );
      ctx.setDragging( true );
    } );
    handle.addEventListener( "pointermove" , function ( event ) {
      if ( !handle.hasPointerCapture( event.pointerId ) ) return;
      width = Math.min( 800 , Math.max( 200 , startWidth + event.clientX - startX ) );
      column.style.setProperty( "--col-w" , width + "px" );
    } );
    const finish = function ( event ) {
      if ( !handle.hasPointerCapture( event.pointerId ) ) return;
      handle.releasePointerCapture( event.pointerId );
      column.classList.remove( "is-resizing" );
      ctx.setDragging( false );
      const rounded = Math.round( width );
      if ( Math.abs( rounded - startWidth ) < 2 ) return;
      ctx.mutate( function () { return ctx.client.updateList( list.id , { width: rounded } ); } );
    };
    handle.addEventListener( "pointerup" , finish );
    handle.addEventListener( "pointercancel" , finish );
    handle.addEventListener( "dblclick" , function () {
      column.style.removeProperty( "--col-w" );
      if ( list.width ) ctx.mutate( function () { return ctx.client.updateList( list.id , { width: 0 } ); } );
    } );
    return handle;
  },

  // A column a share link shows by title only. The server sent how many
  // cards it holds and nothing else, so these are blank bars, blurred.
  obscuredColumn( list , count ) {
    const column = Dom.el( "section" , { class: "kanban-column is-obscured" , attrs: { "data-list-id": list.id , "data-position": list.position } } );
    const header = Dom.el( "header" , { class: "column-header" } );
    const title = Dom.el( "h2" , { class: "column-title" } );
    Markdown.renderInline( list.title , title );
    header.appendChild( title );
    header.appendChild( Dom.el( "span" , { class: "count" , text: String( count ) } ) );
    column.appendChild( header );
    column.appendChild( Dom.el( "p" , { class: "visually-hidden" , text: T( "share.column_hidden" ) } ) );
    column.appendChild( KanbanView.obscuredCards( count ) );
    return column;
  },

  obscuredCards( count ) {
    const container = Dom.el( "div" , { class: "kanban-cards is-obscured" , attrs: { "aria-hidden": "true" , title: T( "share.column_hidden" ) } } );
    // Widths from the index, so a redraw does not shimmer.
    const widths = [ 82 , 64 , 91 , 55 , 73 , 60 , 87 ];
    for ( let index = 0; index < count; index += 1 ) {
      const face = Dom.el( "div" , { class: "card-face obscured-card" } );
      const bar = Dom.el( "span" , { class: "obscured-bar" } );
      bar.style.width = widths[ index % widths.length ] + "%";
      face.appendChild( bar );
      if ( index % 2 === 0 ) {
        const short = Dom.el( "span" , { class: "obscured-bar" } );
        short.style.width = widths[ ( index + 3 ) % widths.length ] / 2 + "%";
        face.appendChild( short );
      }
      container.appendChild( face );
    }
    return container;
  },

  // cards is the card's column (open cards), for its number. ranking is for
  // the All list, where the number is the card's rank instead.
  face( card , ctx , cards , ranking ) {
    const face = Dom.el( "article" , {
      class: "card-face has-status" + ( card.done ? " is-done" : "" ) + ( card.archived ? " is-archived" : "" ),
      attrs: { "data-card-id": card.id , "data-position": card.position , tabindex: "0" , role: "button" },
      on: {
        click: function ( event ) { if ( !event.target.closest( "a" ) ) ctx.openCard( card.id ); },
        keydown: function ( event ) { if ( event.key === "Enter" ) ctx.openCard( card.id ); },
        contextmenu: function ( event ) { CardMenu.open( event , card , ctx ); },
      },
    } );
    face.appendChild( card.archived ? CardBits.restoreIcon( card , ctx ) : CardBits.statusIcons( card , ctx ) );
    const labels = CardBits.labels( card , ctx.board );
    if ( labels ) face.appendChild( labels );
    const photo = CardBits.thumbnail( card , ctx , "card-product-thumb" );
    if ( photo ) face.appendChild( photo );

    const title = Dom.el( "div" , { class: "card-title" } );
    if ( cards && !card.archived && ctx.numbered() ) title.appendChild( CardBits.numberField( card , cards , ctx ) );
    const rank = ranking ? CardBits.rankField( card , ctx ) : ( ctx.showRank() ? CardBits.rank( card , ctx ) : null );
    if ( rank ) title.appendChild( rank );
    const text = Dom.el( "span" );
    Markdown.renderInline( card.title , text );
    title.appendChild( text );
    face.appendChild( title );

    if ( CardBits.hasDescription( card ) ) {
      const preview = Dom.el( "div" , { class: "card-body-preview" } );
      Markdown.render( card.description , preview );
      face.appendChild( preview );
    }
    const checklist = CardBits.checklistItems( card );
    if ( checklist ) face.appendChild( checklist );

    const meta = Dom.el( "div" , { class: "card-meta" } );
    [
      CardBits.price( card ),
      CardBits.due( card ),
      CardBits.checklist( card ),
      CardBits.comments( card ),
      CardBits.assignees( card , ctx.people ),
    ].filter( Boolean ).forEach( function ( node ) { meta.appendChild( node ); } );
    if ( meta.childNodes.length ) face.appendChild( meta );
    return face;
  },

  addListColumn( ctx ) {
    const column = Dom.el( "section" , { class: "kanban-column add-column" } );
    column.appendChild( QuickAdd.build( "list" , "board.add_list" , "board.list_title_placeholder" , function ( title ) {
      return ctx.mutate( function () { return ctx.client.createList( title ); } );
    } , 120 ) );
    return column;
  },
};

// "+ Add a card" that turns into an input on click. Enter adds and keeps the
// input open for the next one, which is how people add a list's worth of
// cards in a row.
const QuickAdd = {
  // Boxes whose cards go to the other end from the usual one, by key. It
  // lasts while the box stays open -- across the re-render after each add --
  // and is forgotten when the box is cancelled or opened afresh.
  flipped: {},
  // When each box's arrow was last pressed, by key. Kept here and not in the
  // button's closure, which a poll can replace between two clicks.
  lastFlip: {},
  // Two presses this close together are a double click. Timed by hand: phones
  // do not reliably send dblclick.
  doubleClickMs: 350,
  reopening: false,

  // A direction locked for one column, "top" or "bottom", or "" for none. It
  // is this browser's, and outlasts the box and the page until it is changed.
  locked( ctx , list ) {
    let saved = "";
    try { saved = window.localStorage.getItem( "add_lock:" + ctx.board.id + ":" + list.id ) || ""; } catch ( error ) { saved = ""; }
    return saved === "top" || saved === "bottom" ? saved : "";
  },

  setLocked( ctx , list , direction ) {
    const name = "add_lock:" + ctx.board.id + ":" + list.id;
    try {
      if ( direction ) window.localStorage.setItem( name , direction );
      else window.localStorage.removeItem( name );
    } catch ( error ) { /* not remembered */ }
  },

  // A column's add-card box. With add-on-top the box sits above the cards
  // and what it adds goes first; otherwise both go last. Its arrow button
  // sends what is added to the other end instead, for as long as it is on.
  // Double-clicking the arrow locks the direction it was showing -- the two
  // clicks flip it there and back -- and any single click after that unlocks
  // it and flips it.
  // A picture pasted into the box is held, shown small, and put on the
  // card once it is added.
  // stage, on a staged board, is the one the column is showing: new cards
  // start there.
  forCards( list , ctx , buttonKey , stage ) {
    const key = "card-" + list.id;
    const onTop = function () {
      const lock = QuickAdd.locked( ctx , list );
      return lock ? lock === "top" : ctx.addOnTop() !== !!QuickAdd.flipped[ key ];
    };
    const flip = Ui.iconButton( "arrow-bar-up" , "" , function () {
      const now = Date.now();
      const double = now - ( QuickAdd.lastFlip[ key ] || 0 ) < QuickAdd.doubleClickMs;
      // A third quick click starts over rather than counting as another double.
      QuickAdd.lastFlip[ key ] = double ? 0 : now;
      const next = !onTop();
      QuickAdd.flipped[ key ] = next !== ctx.addOnTop();
      QuickAdd.setLocked( ctx , list , double ? ( next ? "top" : "bottom" ) : "" );
      show();
      if ( double ) flip.closest( ".quick-add-form" ).querySelector( ".quick-add-input" ).focus();
    } , "quick-add-flip" );
    const show = function () {
      const top = onTop();
      const lock = QuickAdd.locked( ctx , list );
      const label = T( lock ? "board.add_locked_" + lock : ( top ? "board.add_to_top" : "board.add_to_bottom" ) );
      flip.firstChild.className = "bi bi-" + ( top ? "arrow-bar-up" : "arrow-bar-down" );
      flip.classList.toggle( "active" , !!QuickAdd.flipped[ key ] );
      flip.classList.toggle( "is-locked" , !!lock );
      flip.setAttribute( "aria-pressed" , lock || QuickAdd.flipped[ key ] ? "true" : "false" );
      flip.setAttribute( "aria-label" , label );
      flip.setAttribute( "title" , label );
    };
    show();
    const wrap = this.build( key , buttonKey , "board.card_title_placeholder" , function ( title , image ) {
      const position = onTop() ? CardBits.edgePosition( ctx.cardsIn( list.id ) , true ) : undefined;
      return ctx.mutate( async function () {
        const card = await ctx.client.createCard( list.id , title || T( "board.image_card_title" ) , position , stage && stage !== "todo" ? Stages.fields( stage ) : undefined );
        if ( image ) await ctx.client.uploadImage( card.id , await CardBits.shrinkImage( image ) );
        return card;
      } );
    } , undefined , true , flip );
    wrap.querySelector( ".quick-add-open" ).addEventListener( "click" , show );
    return wrap;
  },

  // key identifies this form across re-renders ("card-12", "list"), so after
  // an add -- which re-renders the board -- the new copy can be reopened.
  // maxlength is for lists and checklist items; card titles have none.
  // withImage lets a pasted picture ride along: submit( text , file ).
  // extra, if given, is one more control shown beside Add and Cancel.
  build( key , buttonKey , placeholderKey , submit , maxlength , withImage , extra ) {
    const wrap = Dom.el( "div" , { class: "quick-add" , attrs: { "data-quick-key": key } } );
    const opener = Dom.el( "button" , { class: Ui.buttonClass( "secondary small quick-add-open" ) , text: T( buttonKey ) , attrs: { type: "button" } } );
    const form = Dom.el( "form" , { class: "quick-add-form" , attrs: { autocomplete: "off" } } );
    form.hidden = true;
    // Card titles have no practical length limit, so the box grows with
    // them; Enter adds, and for card titles Shift+Enter adds a line. Lists
    // and checklist items (the ones with a maxlength) stay on one line.
    const input = Dom.el( "textarea" , { class: "quick-add-input form-control" , attrs: { rows: "1" , placeholder: T( placeholderKey ) , "aria-label": T( placeholderKey ) } } );
    input.addEventListener( "input" , function () {
      const flat = maxlength ? Ui.oneLine( input.value ) : input.value;
      if ( flat !== input.value ) input.value = flat;
      Ui.autoGrow( input );
    } );
    if ( maxlength ) input.setAttribute( "maxlength" , String( maxlength ) );
    let image = null;
    const preview = Dom.el( "div" , { class: "quick-add-image" } );
    preview.hidden = true;
    const holdImage = function ( file ) {
      image = file;
      Dom.clear( preview );
      preview.hidden = !file;
      if ( !file ) return;
      const url = URL.createObjectURL( file );
      preview.appendChild( Dom.el( "img" , { attrs: { src: url , alt: "" } } ) );
      preview.appendChild( Ui.closeButton( T( "board.image_remove" ) , function () { URL.revokeObjectURL( url ); holdImage( null ); input.focus(); } , "btn-close-small" ) );
    };
    if ( withImage ) {
      input.addEventListener( "paste" , function ( event ) {
        const file = CardBits.pastedImage( event );
        if ( !file || CardBits.pasteHasText( event ) ) return;
        event.preventDefault();
        holdImage( file );
      } );
    }
    const cancel = Ui.button( "board.cancel" , function () { holdImage( null ); delete QuickAdd.flipped[ key ]; form.hidden = true; opener.hidden = false; } );
    form.appendChild( preview );
    form.appendChild( input );
    form.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Dom.el( "button" , { class: Ui.buttonClass( "small" ) , text: T( "board.add" ) , attrs: { type: "submit" } } ),
      cancel,
    ].concat( extra ? [ extra ] : [] ) } ) );
    opener.addEventListener( "click" , function () {
      if ( !QuickAdd.reopening ) delete QuickAdd.flipped[ key ];
      opener.hidden = true;
      form.hidden = false;
      input.focus();
    } );
    form.addEventListener( "submit" , async function ( event ) {
      event.preventDefault();
      const value = input.value.trim();
      const held = image;
      if ( !value && !held ) return;
      input.value = "";
      holdImage( null );
      Ui.autoGrow( input );
      await submit( value , held );
      const fresh = document.querySelector( "[data-quick-key=\"" + key + "\"] .quick-add-open" );
      if ( fresh && fresh !== opener ) {
        QuickAdd.reopening = true;
        fresh.click();
        QuickAdd.reopening = false;
      }
      else input.focus();
    } );
    input.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Escape" ) cancel.click();
      if ( event.key === "Enter" && !event.isComposing && !( event.shiftKey && !maxlength ) ) {
        event.preventDefault();
        form.requestSubmit();
      }
    } );
    wrap.appendChild( opener );
    wrap.appendChild( form );
    return wrap;
  },
};

// A card's right-click menu: the card editor's everyday actions without
// opening it. Delete moves the card to the trash, and is undoable like any
// other (see history.js).
const CardMenu = {
  open( event , card , ctx ) {
    const items = [ { key: "board.card_menu_open" , action: function () { ctx.openCard( card.id ); } } ];
    if ( card.access === "edit" && !card.archived ) {
      items.push( { key: card.done ? "board.card_menu_undone" : "board.card_menu_done" , action: function () {
        CardBits.toggleDone( card , ctx );
      } } );
    }
    this.sendItems( card , ctx ).forEach( function ( item ) { items.push( item ); } );
    const link = ctx.cardLink ? ctx.cardLink( card ) : "";
    if ( link ) items.push( { key: "card.copy_link" , action: function () { Ui.copyText( link ); } } );
    items.push( { key: "export.copy_card_markdown" , action: function () { Ui.copyText( CardExport.markdown( ctx , card ) ); } } );
    items.push( { key: "export.card_button" , action: function () { CardExport.open( ctx , card ); } } );
    if ( ctx.board.allow_quick_share && ctx.canEdit && !ctx.isGuest && !card.archived ) {
      items.push( { key: "share.quick_menu" , action: function () { Sharing.quickShare( ctx , card ); } } );
    }
    if ( ctx.canEdit ) {
      items.push( { key: card.archived ? "card.unarchive" : "card.archive" , action: function () {
        ctx.mutate( function () { return ctx.client.patchCard( card.id , { archived: !card.archived } ); } );
      } } );
      items.push( { key: "card.delete" , danger: true , action: async function () {
        await ctx.mutate( function () { return ctx.client.deleteCard( card.id ); } );
        Ui.toast( T( "trash.moved" ) , "ok" );
      } } );
    }
    ContextMenu.open( event , items );
  },

  // Send to top and bottom of the card's own list, for board editors: a
  // position change is a move, which a card shared on its own cannot make.
  // Not on the All list, which is ordered across lists, and an end the card
  // already shows at is left out. On a staged board the ends are of the
  // stage on show, and the whole list's end is the stage's end too.
  sendItems( card , ctx ) {
    if ( !ctx.canEdit || card.archived || ctx.isAllView() ) return [];
    const shown = ctx.isStaged() ? ctx.cardsAt( card.list_id , Stages.of( card ) ) : ctx.cardsIn( card.list_id );
    const others = ctx.cardsIn( card.list_id ).filter( function ( other ) { return other.id !== card.id; } );
    const send = function ( top ) {
      ctx.mutate( function () { return ctx.client.patchCard( card.id , { position: CardBits.edgePosition( others , top ) } ); } );
    };
    const items = [];
    if ( shown.length && shown[ 0 ].id !== card.id ) items.push( { key: "board.card_menu_top" , action: function () { send( true ); } } );
    if ( shown.length && shown[ shown.length - 1 ].id !== card.id ) items.push( { key: "board.card_menu_bottom" , action: function () { send( false ); } } );
    return items;
  },
};

// A list's menu: copy its card titles as a Markdown list, or every card in
// full (anyone), archive
// every card on it (editors, a column link's included), share it (the
// owner), and rename, archive or delete it (board editors).
const ListMenu = {
  button( list , ctx ) {
    return Ui.iconButton( "three-dots" , T( "board.list_menu" ) , function ( event ) {
      event.stopPropagation();
      ListMenu.open( list , ctx );
    } , "list-menu-button" );
  },

  // The column's open card titles, in order, as "- Title" or "1. Title".
  // A title is one line of Markdown already, so it goes in as written.
  markdownList( list , ctx , numbered ) {
    return ctx.cardsIn( list.id ).map( function ( card , index ) {
      return ( numbered ? ( index + 1 ) + ". " : "- " ) + card.title.replace( /\s*\n\s*/g , " " );
    } ).join( "\n" );
  },

  // Tick the cards to archive from one column. Nothing is ticked to start
  // with; "Select all" ticks the lot.
  pickToArchive( list , ctx ) {
    const cards = ctx.isStaged() ? ctx.cardsAt( list.id , ctx.stageIn( list.id ) ) : ctx.cardsIn( list.id );
    Panel.open( "board.archive_pick_title" , function ( body ) {
      const form = Dom.el( "form" , { class: "archive-pick" , attrs: { autocomplete: "off" } } );
      const boxes = [];
      const submit = Dom.el( "button" , { class: "btn btn-primary" , attrs: { type: "submit" } } );
      const update = function () {
        const picked = boxes.filter( function ( box ) { return box.checked; } ).length;
        Dom.text( submit , I18n.fill( "board.archive_picked" , { n: picked } ) );
        submit.disabled = picked === 0;
        all.querySelector( "input" ).checked = picked === boxes.length;
      };
      const all = Ui.checkbox( "board.select_all" , false , function ( checked ) {
        boxes.forEach( function ( box ) { box.checked = checked; } );
        update();
      } );
      all.classList.add( "archive-pick-all" );
      form.appendChild( all );
      const items = Dom.el( "ul" , { class: "archive-pick-list" } );
      cards.forEach( function ( card ) {
        const box = Dom.el( "input" , { class: "form-check-input" , attrs: { type: "checkbox" , value: String( card.id ) } } );
        box.addEventListener( "change" , update );
        boxes.push( box );
        const title = Dom.el( "span" );
        Markdown.renderInline( card.title , title );
        items.appendChild( Dom.el( "li" , { children: [ Dom.el( "label" , { class: "check" , children: [ box , title ] } ) ] } ) );
      } );
      form.appendChild( items );
      form.appendChild( submit );
      form.addEventListener( "submit" , async function ( event ) {
        event.preventDefault();
        const ids = boxes.filter( function ( box ) { return box.checked; } ).map( function ( box ) { return Number( box.value ); } );
        if ( !ids.length ) return;
        Panel.close();
        await ctx.mutate( function () { return ctx.client.archiveCards( list.id , ids ); } );
      } );
      body.appendChild( form );
      update();
    } , function () { ListMenu.open( list , BoardPage.context() ); } );
  },

  open( list , ctx ) {
    // Coming back from picking cards to archive, list can be from before a
    // poll, and this menu shows what it is set to.
    list = ctx.board.lists.find( function ( item ) { return item.id === list.id; } ) || list;
    Panel.open( "board.list_menu" , function ( body ) {
      const copy = Dom.el( "section" , { class: "panel-section" } );
      copy.appendChild( Dom.el( "h3" , { text: T( "board.copy_heading" ) } ) );
      copy.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "board.copy_bullets" , function () { Ui.copyText( ListMenu.markdownList( list , ctx , false ) ); } ),
        Ui.button( "board.copy_numbered" , function () { Ui.copyText( ListMenu.markdownList( list , ctx , true ) ); } ),
        Ui.button( "board.copy_everything" , function () { Ui.copyText( BoardExport.listNumbered( ctx , list , true ) ); } ),
        Ui.button( "board.copy_everything_bullets" , function () { Ui.copyText( BoardExport.listNumbered( ctx , list , false ) ); } ),
      ] } ) );
      body.appendChild( copy );
      // The column as a file, and cards back from one (appended).
      const file = Dom.el( "section" , { class: "panel-section" } );
      file.appendChild( Dom.el( "h3" , { text: T( "backup.column_heading" ) } ) );
      const fileActions = Dom.el( "div" , { class: "row-actions" } );
      fileActions.appendChild( Ui.button( "backup.column_export" , function () { Backup.exportColumn( ctx , list ); } ) );
      if ( ctx.canEdit ) fileActions.appendChild( Ui.button( "backup.column_import" , function () { Backup.importColumn( ctx , list ); } ) );
      file.appendChild( fileActions );
      body.appendChild( file );
      // Clearing out a column -- Done, usually. Archived cards come back with
      // "Show archived", undo, or Board settings.
      // On a staged board that means what the column is showing, not the
      // stages out of sight behind it.
      const staged = ctx.isStaged();
      const open = staged ? ctx.cardsAt( list.id , ctx.stageIn( list.id ) ) : ctx.cardsIn( list.id );
      if ( ctx.canEdit && open.length ) {
        body.appendChild( Dom.el( "section" , { class: "panel-section" , children: [
          Dom.el( "div" , { class: "row-actions" , children: [
            Ui.button( staged ? "board.archive_shown" : "board.archive_cards" , async function () {
              if ( !window.confirm( I18n.fill( "board.archive_cards_confirm" , { n: open.length , list: list.title } ) ) ) return;
              Panel.close();
              const ids = staged ? open.map( function ( card ) { return card.id; } ) : undefined;
              await ctx.mutate( function () { return ctx.client.archiveCards( list.id , ids ); } );
            } ),
            Ui.button( "board.archive_pick" , function () { ListMenu.pickToArchive( list , ctx ); } ),
          ] } ),
        ] } ) );
      }
      if ( ctx.canManage && !ctx.isGuest ) {
        body.appendChild( Dom.el( "section" , { class: "panel-section" , children: [
          Ui.button( "board.share_list" , function () { Sharing.openList( ctx , list ); } ),
        ] } ) );
      }
      if ( ctx.canCollapse() ) {
        body.appendChild( Dom.el( "section" , { class: "panel-section" , children: [
          Ui.button( "board.column_collapse" , function () {
            Panel.close();
            BoardPage.setCollapsed( list.id , true );
          } ),
        ] } ) );
      }
      // Renaming, archiving and deleting change the board, which a column
      // link cannot.
      if ( !ctx.canEdit || ctx.fixedColumns ) return;
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const input = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "120" , required: "" } } );
      input.value = list.title;
      form.appendChild( Ui.field( "board.list_title_label" , input ) );
      form.appendChild( Ui.checkbox( "board.list_shopping" , list.shopping , async function ( checked ) {
        Dom.text( status , T( "card.autosave_saving" ) );
        const result = await ctx.mutate( function () { return ctx.client.updateList( list.id , { shopping: checked } ); } );
        Dom.text( status , result ? T( "card.autosave_saved" ) : "" );
      } ) );
      if ( staged ) {
        const stageView = Ui.select( [ { value: "" , text: T( "board.column_view_follow" ) } ].concat( Stages.order.map( function ( stage ) {
          return { value: stage , text: Stages.label( ctx , stage ) };
        } ) ) , list.stage_view || "" );
        stageView.addEventListener( "change" , async function () {
          Dom.text( status , T( "card.autosave_saving" ) );
          const result = await ctx.mutate( function () { return ctx.client.updateList( list.id , { stage_view: stageView.value } ); } );
          Dom.text( status , result ? T( "card.autosave_saved" ) : "" );
        } );
        form.appendChild( Ui.field( "board.column_view_label" , stageView ) );
        form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "board.column_view_hint" ) } ) );
      }
      // The name saves itself after a pause in typing, or when it is left;
      // Enter saves and closes the menu.
      const status = Dom.el( "p" , { class: "muted small-print autosave-status" , attrs: { "aria-live": "polite" } } );
      form.appendChild( status );
      let saved = list.title;
      let timer = null;
      const save = async function () {
        window.clearTimeout( timer );
        const value = input.value.trim();
        if ( !value || value === saved ) return;
        saved = value;
        Dom.text( status , T( "card.autosave_saving" ) );
        const result = await ctx.mutate( function () { return ctx.client.updateList( list.id , { title: value } ); } );
        Dom.text( status , result ? T( "card.autosave_saved" ) : "" );
      };
      input.addEventListener( "input" , function () {
        window.clearTimeout( timer );
        Dom.text( status , T( "card.autosave_pending" ) );
        timer = window.setTimeout( save , 800 );
      } );
      input.addEventListener( "change" , save );
      form.addEventListener( "submit" , async function ( event ) {
        event.preventDefault();
        await save();
        Panel.close();
      } );
      body.appendChild( form );
      if ( ctx.isThree() ) return;
      body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "board.archive_list" , async function () {
          await ctx.mutate( function () { return ctx.client.updateList( list.id , { archived: true } ); } );
          Panel.close();
        } ),
        Ui.button( "board.delete_list" , async function () {
          if ( !window.confirm( I18n.fill( "board.delete_list_confirm" , { name: list.title } ) ) ) return;
          await ctx.mutate( function () { return ctx.client.deleteList( list.id ); } );
          Panel.close();
        } , "secondary small danger" ),
      ] } ) );
    } );
  },
};
