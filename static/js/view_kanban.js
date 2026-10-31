// Kanban: one column per list, cards dragged within and between columns.
// SortableJS does the dragging (touch included); on drop the card gets the
// midpoint of its new neighbours' positions, so a move writes one card.
const KanbanView = {
  render( root , ctx ) {
    const three = ctx.isThree();
    const scroller = Dom.el( "div" , { class: "kanban scroll-x" + ( three ? " three" : "" ) } );
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
    ctx.mutate( function () { return ctx.client.patchCard( cardId , { list_id: listId , position: position } ); } );
  },

  column( list , ctx ) {
    const hidden = ctx.hiddenCount( list.id );
    if ( hidden !== null ) return this.obscuredColumn( list , hidden );
    const cards = ctx.cardsIn( list.id );
    const column = Dom.el( "section" , { class: "kanban-column" , attrs: { "data-list-id": list.id , "data-position": list.position } } );

    const header = Dom.el( "header" , { class: "column-header" } );
    const title = Dom.el( "h2" , { class: "column-title" + ( ctx.canEdit ? " column-handle" : "" ) } );
    Markdown.renderInline( list.title , title );
    header.appendChild( title );
    header.appendChild( Dom.el( "span" , { class: "count" , text: String( cards.length ) } ) );
    header.appendChild( ListMenu.button( list , ctx ) );
    column.appendChild( header );

    const container = Dom.el( "div" , { class: "kanban-cards" , attrs: { "data-list-id": list.id } } );
    ctx.shownIn( list.id ).forEach( ( card ) => container.appendChild( this.face( card , ctx , cards ) ) );
    const add = ctx.canEdit ? QuickAdd.forCards( list , ctx , "board.add_card" ) : null;
    if ( add && ctx.addOnTop() ) column.appendChild( add );
    column.appendChild( container );
    if ( add && !ctx.addOnTop() ) column.appendChild( add );
    return column;
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

  // cards is the card's column (open cards), for its number.
  face( card , ctx , cards ) {
    const face = Dom.el( "article" , {
      class: "card-face has-status" + ( card.done ? " is-done" : "" ) + ( card.archived ? " is-archived" : "" ),
      attrs: { "data-card-id": card.id , "data-position": card.position , tabindex: "0" , role: "button" },
      on: {
        click: function () { ctx.openCard( card.id ); },
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
  // A column's add-card box. With add-on-top the box sits above the cards
  // and what it adds goes first; otherwise both go last.
  // A picture pasted into the box is held, shown small, and put on the
  // card once it is added.
  forCards( list , ctx , buttonKey ) {
    return this.build( "card-" + list.id , buttonKey , "board.card_title_placeholder" , function ( title , image ) {
      const position = ctx.addOnTop() ? CardBits.edgePosition( ctx.cardsIn( list.id ) , true ) : undefined;
      return ctx.mutate( async function () {
        const card = await ctx.client.createCard( list.id , title || T( "board.image_card_title" ) , position );
        if ( image ) await ctx.client.uploadImage( card.id , await CardBits.shrinkImage( image ) );
        return card;
      } );
    } , undefined , true );
  },

  // key identifies this form across re-renders ("card-12", "list"), so after
  // an add -- which re-renders the board -- the new copy can be reopened.
  // maxlength is for lists and checklist items; card titles have none.
  // withImage lets a pasted picture ride along: submit( text , file ).
  build( key , buttonKey , placeholderKey , submit , maxlength , withImage ) {
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
    const cancel = Ui.button( "board.cancel" , function () { holdImage( null ); form.hidden = true; opener.hidden = false; } );
    form.appendChild( preview );
    form.appendChild( input );
    form.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Dom.el( "button" , { class: Ui.buttonClass( "small" ) , text: T( "board.add" ) , attrs: { type: "submit" } } ),
      cancel,
    ] } ) );
    opener.addEventListener( "click" , function () {
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
      if ( fresh && fresh !== opener ) fresh.click();
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
    items.push( { key: "export.copy_card_markdown" , action: function () { Ui.copyText( CardExport.markdown( ctx , card ) ); } } );
    items.push( { key: "export.card_button" , action: function () { CardExport.open( ctx , card ); } } );
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
    const cards = ctx.cardsIn( list.id );
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
    Panel.open( "board.list_menu" , function ( body ) {
      const copy = Dom.el( "section" , { class: "panel-section" } );
      copy.appendChild( Dom.el( "h3" , { text: T( "board.copy_heading" ) } ) );
      copy.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "board.copy_bullets" , function () { Ui.copyText( ListMenu.markdownList( list , ctx , false ) ); } ),
        Ui.button( "board.copy_numbered" , function () { Ui.copyText( ListMenu.markdownList( list , ctx , true ) ); } ),
        Ui.button( "board.copy_everything" , function () { Ui.copyText( BoardExport.listNumbered( ctx , list ) ); } ),
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
      const open = ctx.cardsIn( list.id );
      if ( ctx.canEdit && open.length ) {
        body.appendChild( Dom.el( "section" , { class: "panel-section" , children: [
          Dom.el( "div" , { class: "row-actions" , children: [
            Ui.button( "board.archive_cards" , async function () {
              if ( !window.confirm( I18n.fill( "board.archive_cards_confirm" , { n: open.length , list: list.title } ) ) ) return;
              Panel.close();
              await ctx.mutate( function () { return ctx.client.archiveCards( list.id ); } );
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
