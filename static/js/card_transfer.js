// Send a card to another board, or copy it to any board, from its right-click
// menu or its editor. A copy can stay linked to the original: one way (the
// copy follows) or two way (both follow each other). The server does the
// work (models/card_link.go); this is the form, and the card editor's list
// of a card's links.
const CardTransfer = {
  // The boards a card can go to: ones this person edits. Fetched once a
  // page; a new board shows up on the next load.
  boards: null,

  async editableBoards() {
    if ( !this.boards ) {
      const result = await Api.listBoards();
      this.boards = ( result.boards || [] ).filter( function ( board ) { return board.role === "owner" || board.role === "editor"; } );
    }
    return this.boards;
  },

  // A board's open columns, in order: for a three-column board its first
  // three, which are all it shows.
  async columns( boardId ) {
    const data = await Api.request( "/api/boards/" + boardId );
    const lists = data.board.lists.filter( function ( list ) { return !list.archived; } )
      .sort( function ( a , b ) { return a.position - b.position; } );
    return data.board.layout === "three" ? lists.slice( 0 , 3 ) : lists;
  },

  // send: true moves the card, false copies it.
  async open( ctx , card , send ) {
    let boards;
    try {
      boards = await this.editableBoards();
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    const here = ctx.board.id;
    const choices = boards.filter( function ( board ) { return !send || board.id !== here; } );
    Panel.open( send ? "transfer.send_heading" : "transfer.copy_heading" , ( body ) => {
      const intro = Dom.el( "p" , { class: "muted small-print" } );
      Markdown.renderInline( I18n.fill( send ? "transfer.send_intro" : "transfer.copy_intro" , { card: card.title } ) , intro );
      body.appendChild( intro );
      if ( !choices.length ) {
        body.appendChild( Dom.el( "p" , { class: "muted" , text: T( "transfer.no_boards" ) } ) );
        return;
      }
      const form = Dom.el( "form" , { class: "d-flex flex-column gap-3" , attrs: { autocomplete: "off" } } );
      const boardSelect = Ui.select( choices.map( function ( board ) {
        return { value: board.id , text: board.id === here ? I18n.fill( "transfer.this_board" , { board: board.title } ) : board.title };
      } ) , send ? choices[ 0 ].id : here );
      const listSelect = Ui.select( [] , "" );
      const status = Dom.el( "p" , { class: "muted small-print mb-0" } );
      form.appendChild( Ui.field( "transfer.board_label" , boardSelect ) );
      form.appendChild( Ui.field( "transfer.list_label" , listSelect ) );

      const placeSelect = Ui.select( [
        { value: "bottom" , text: T( "transfer.place_bottom" ) },
        { value: "top" , text: T( "transfer.place_top" ) },
      ] , ctx.addOnTop() ? "top" : "bottom" );
      form.appendChild( Ui.field( "transfer.place_label" , placeSelect ) );

      let mirror = null;
      if ( !send ) {
        mirror = Dom.el( "fieldset" , { class: "d-flex flex-column gap-1" } );
        mirror.appendChild( Dom.el( "legend" , { class: "form-label fs-6 mb-1" , text: T( "transfer.mirror_label" ) } ) );
        [ [ "" , "transfer.mirror_none" ] , [ "one_way" , "transfer.mirror_one_way" ] , [ "two_way" , "transfer.mirror_two_way" ] ].forEach( function ( pair , index ) {
          const id = "transfer-mirror-" + index;
          const input = Dom.el( "input" , { class: "form-check-input" , attrs: { type: "radio" , name: "mirror" , value: pair[ 0 ] , id: id } } );
          input.checked = index === 0;
          mirror.appendChild( Dom.el( "div" , { class: "form-check" , children: [
            input,
            Dom.el( "label" , { class: "form-check-label" , text: T( pair[ 1 ] ) , attrs: { for: id } } ),
            Dom.el( "div" , { class: "muted small-print" , text: T( pair[ 1 ] + "_hint" ) } ),
          ] } ) );
        } );
        form.appendChild( mirror );
      }

      const submit = Dom.el( "button" , { class: Ui.buttonClass( "primary" ) , text: T( send ? "transfer.send_button" : "transfer.copy_button" ) , attrs: { type: "submit" } } );
      form.appendChild( Dom.el( "div" , { class: "d-flex flex-wrap gap-2 align-items-center" , children: [ submit , status ] } ) );
      body.appendChild( form );

      const loadColumns = async () => {
        submit.disabled = true;
        Dom.clear( listSelect );
        Dom.text( status , T( "transfer.loading" ) );
        try {
          const lists = await this.columns( Number( boardSelect.value ) );
          lists.forEach( function ( list ) { listSelect.appendChild( Dom.el( "option" , { text: list.title , attrs: { value: String( list.id ) } } ) ); } );
          if ( Number( boardSelect.value ) === here ) listSelect.value = String( card.list_id );
          Dom.text( status , lists.length ? "" : T( "transfer.no_columns" ) );
          submit.disabled = !lists.length;
        } catch ( error ) {
          Dom.text( status , "" );
          Ui.error( error );
        }
      };
      boardSelect.addEventListener( "change" , loadColumns );
      loadColumns();

      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        const target = choices.find( function ( board ) { return board.id === Number( boardSelect.value ); } );
        const fields = { board: String( target.id ) , list_id: Number( listSelect.value ) , top: placeSelect.value === "top" };
        if ( mirror ) fields.mirror = mirror.querySelector( "input:checked" ).value;
        submit.disabled = true;
        const result = await ctx.mutate( function () {
          return send ? ctx.client.sendCard( card.id , fields ) : ctx.client.copyCard( card.id , fields );
        } );
        if ( !result ) {
          submit.disabled = false;
          return;
        }
        Panel.close();
        if ( send ) CardModal.close();
        Ui.toast( I18n.fill( send ? "transfer.sent" : "transfer.copied" , { board: target.title } ) , "ok" );
      } );
    } );
  },

  // The card editor's list of the card's links, filled in once they load.
  section( ctx , card , canEdit ) {
    if ( !card.mirror || ctx.isGuest ) return null;
    const section = Dom.el( "section" , { class: "card-section card-mirrors" } );
    const head = Dom.el( "div" , { class: "section-head" } );
    head.appendChild( Dom.el( "h3" , { children: [
      Dom.el( "i" , { class: "bi bi-link-45deg" , attrs: { "aria-hidden": "true" } } ),
      Dom.el( "span" , { text: " " + T( "transfer.mirrors_heading" ) } ),
    ] } ) );
    section.appendChild( head );
    if ( card.mirror === "copy" ) section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "transfer.copy_locked" ) } ) );
    const list = Dom.el( "ul" , { class: "link-list" } );
    section.appendChild( list );
    ctx.client.cardLinks( card.id ).then( ( result ) => {
      ( result.links || [] ).forEach( ( link ) => list.appendChild( this.linkRow( ctx , card , link , canEdit ) ) );
    } ).catch( function ( error ) { Ui.error( error ); } );
    return section;
  },

  linkRow( ctx , card , link , canEdit ) {
    const text = Dom.el( "div" , { class: "trash-text" } );
    const where = link.board_title ? I18n.fill( "transfer.link_where" , { board: link.board_title , list: link.list_title } ) : T( "transfer.link_hidden" );
    text.appendChild( Dom.el( "span" , { text: where } ) );
    text.appendChild( Dom.el( "span" , { class: "muted small-print" , text: T( "transfer.direction_" + link.direction ) } ) );
    const actions = Dom.el( "div" , { class: "row-actions" } );
    if ( link.card_path ) actions.appendChild( Dom.el( "a" , { class: Ui.buttonClass( "secondary small" ) , text: T( "transfer.open_link" ) , attrs: { href: link.card_path } } ) );
    if ( canEdit ) {
      actions.appendChild( Ui.button( "transfer.unlink" , async function () {
        if ( await ctx.mutate( function () { return ctx.client.unlinkCard( card.id , link.id ); } ) ) Ui.toast( T( "transfer.unlinked" ) , "ok" );
      } ) );
    }
    return Dom.el( "li" , { class: "link-row trash-row" , children: [ text , actions ] } );
  },
};
