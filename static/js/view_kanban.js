// Kanban: one column per list, cards dragged within and between columns.
// SortableJS does the dragging (touch included); on drop the card gets the
// midpoint of its new neighbours' positions, so a move writes one card.
const KanbanView = {
  render( root , ctx ) {
    const three = ctx.isThree();
    const scroller = Dom.el( "div" , { class: "kanban scroll-x" + ( three ? " three" : "" ) } );
    const lists = ctx.lists();
    lists.forEach( ( list ) => scroller.appendChild( this.column( list , ctx ) ) );
    if ( ctx.canEdit && !three ) scroller.appendChild( this.addListColumn( ctx ) );
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
        onStart: function () { ctx.setDragging( true ); },
        onEnd: function ( event ) {
          ctx.setDragging( false );
          KanbanView.dropped( event , ctx );
        },
      } );
    } );

    // A three-column board's columns are fixed in place.
    if ( three ) return;
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
    const neighbour = function ( node ) {
      return node && node.hasAttribute( "data-card-id" ) ? Number( node.getAttribute( "data-position" ) ) : null;
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
    if ( ctx.canEdit ) header.appendChild( ListMenu.button( list , ctx ) );
    column.appendChild( header );

    const container = Dom.el( "div" , { class: "kanban-cards" , attrs: { "data-list-id": list.id } } );
    cards.forEach( ( card ) => container.appendChild( this.face( card , ctx ) ) );
    column.appendChild( container );

    if ( ctx.canEdit ) column.appendChild( QuickAdd.build( "card-" + list.id , "board.add_card" , "board.card_title_placeholder" , function ( title ) {
      return ctx.mutate( function () { return ctx.client.createCard( list.id , title ); } );
    } ) );
    return column;
  },

  face( card , ctx ) {
    const face = Dom.el( "article" , {
      class: "card-face" + ( card.done ? " is-done" : "" ),
      attrs: { "data-card-id": card.id , "data-position": card.position , tabindex: "0" , role: "button" },
      on: {
        click: function () { ctx.openCard( card.id ); },
        keydown: function ( event ) { if ( event.key === "Enter" ) ctx.openCard( card.id ); },
      },
    } );
    const labels = CardBits.labels( card , ctx.board );
    if ( labels ) face.appendChild( labels );

    const title = Dom.el( "div" , { class: "card-title" } );
    if ( card.done ) title.appendChild( Dom.el( "span" , { class: "done-tick" , text: T( "card.done_symbol" ) , attrs: { title: T( "card.done_label" ) } } ) );
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

// A list's menu: rename, archive, delete.
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

  open( list , ctx ) {
    Panel.open( "board.list_menu" , function ( body ) {
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
