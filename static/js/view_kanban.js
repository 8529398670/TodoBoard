// Kanban: one column per list, cards dragged within and between columns.
// SortableJS does the dragging (touch included); on drop the card gets the
// midpoint of its new neighbours' positions, so a move writes one card.
const KanbanView = {
  render( root , ctx ) {
    const three = ctx.isThree();
    const scroller = Dom.el( "div" , { class: "kanban scroll-x" + ( three ? " three" : "" ) } );
    const lists = ctx.lists();
    lists.forEach( ( list ) => scroller.appendChild( this.column( list , ctx ) ) );
    if ( ctx.canEdit && !three && !ctx.listOnly ) scroller.appendChild( this.addListColumn( ctx ) );
    root.appendChild( scroller );

    if ( !ctx.canEdit || typeof Sortable === "undefined" ) return;

    Dom.all( ".kanban-cards" , scroller ).forEach( ( container ) => {
      Sortable.create( container , {
        group: "cards",
        animation: 150,
        forceFallback: true,
        delay: 150,
        delayOnTouchOnly: true,
        ghostClass: "drag-ghost",
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
    if ( three || ctx.listOnly ) return;
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
    column.appendChild( container );

    if ( ctx.canEdit ) column.appendChild( QuickAdd.build( "card-" + list.id , "board.add_card" , "board.card_title_placeholder" , function ( title ) {
      return ctx.mutate( function () { return ctx.client.createCard( list.id , title ); } );
    } ) );
    return column;
  },

  // cards is the card's column (open cards), for its number.
  face( card , ctx , cards ) {
    const face = Dom.el( "article" , {
      class: "card-face has-status" + ( card.done ? " is-done" : "" ) + ( card.archived ? " is-archived" : "" ),
      attrs: { "data-card-id": card.id , "data-position": card.position , tabindex: "0" , role: "button" },
      on: {
        click: function () { ctx.openCard( card.id ); },
        keydown: function ( event ) { if ( event.key === "Enter" ) ctx.openCard( card.id ); },
      },
    } );
    face.appendChild( card.archived ? CardBits.restoreIcon( card , ctx ) : CardBits.statusIcons( card , ctx ) );
    const labels = CardBits.labels( card , ctx.board );
    if ( labels ) face.appendChild( labels );

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

    const meta = Dom.el( "div" , { class: "card-meta" } );
    [
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
    } ) );
    return column;
  },
};

// "+ Add a card" that turns into an input on click. Enter adds and keeps the
// input open for the next one, which is how people add a list's worth of
// cards in a row.
const QuickAdd = {
  // key identifies this form across re-renders ("card-12", "list"), so after
  // an add -- which re-renders the board -- the new copy can be reopened.
  build( key , buttonKey , placeholderKey , submit ) {
    const wrap = Dom.el( "div" , { class: "quick-add" , attrs: { "data-quick-key": key } } );
    const opener = Dom.el( "button" , { class: "secondary small quick-add-open" , text: T( buttonKey ) , attrs: { type: "button" } } );
    const form = Dom.el( "form" , { class: "quick-add-form" , attrs: { autocomplete: "off" } } );
    form.hidden = true;
    const input = Dom.el( "input" , { attrs: { maxlength: "300" , placeholder: T( placeholderKey ) , "aria-label": T( placeholderKey ) } } );
    const cancel = Ui.button( "board.cancel" , function () { form.hidden = true; opener.hidden = false; } );
    form.appendChild( input );
    form.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Dom.el( "button" , { class: "small" , text: T( "board.add" ) , attrs: { type: "submit" } } ),
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
      if ( !value ) return;
      input.value = "";
      await submit( value );
      const fresh = document.querySelector( "[data-quick-key=\"" + key + "\"] .quick-add-open" );
      if ( fresh && fresh !== opener ) fresh.click();
      else input.focus();
    } );
    input.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Escape" ) cancel.click();
    } );
    wrap.appendChild( opener );
    wrap.appendChild( form );
    return wrap;
  },
};

// A list's menu: copy its card titles as a Markdown list (anyone), archive
// every card on it (editors, a column link's included), share it (the
// owner), and rename, archive or delete it (board editors).
const ListMenu = {
  button( list , ctx ) {
    return Dom.el( "button" , {
      class: "icon-button",
      text: T( "board.menu_symbol" ),
      attrs: { type: "button" , title: T( "board.list_menu" ) , "aria-label": T( "board.list_menu" ) },
      on: { click: function ( event ) {
        event.stopPropagation();
        ListMenu.open( list , ctx );
      } },
    } );
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
      const submit = Dom.el( "button" , { attrs: { type: "submit" } } );
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
        const box = Dom.el( "input" , { attrs: { type: "checkbox" , value: String( card.id ) } } );
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
      ] } ) );
      body.appendChild( copy );
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
      if ( !ctx.canEdit || ctx.listOnly ) return;
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const input = Dom.el( "input" , { attrs: { maxlength: "120" , required: "" } } );
      input.value = list.title;
      form.appendChild( Ui.field( "board.list_title_label" , input ) );
      form.appendChild( Dom.el( "button" , { text: T( "board.save" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async function ( event ) {
        event.preventDefault();
        await ctx.mutate( function () { return ctx.client.updateList( list.id , { title: input.value } ); } );
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
