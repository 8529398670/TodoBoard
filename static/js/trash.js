// The board's trash: cards deleted from this board, newest first, each with
// Restore and Delete forever, and Empty trash for the lot. A deleted card
// waits here for 30 days (server/models/trash.go) before it goes for good.
const Trash = {
  async open( ctx , back ) {
    let cards;
    try {
      cards = ( await ctx.client.trash() ).cards;
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    Panel.open( "trash.title" , ( body ) => {
      body.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "trash.hint" ) } ) );
      if ( !cards.length ) {
        body.appendChild( Dom.el( "p" , { class: "muted" , text: T( "trash.empty" ) } ) );
        return;
      }
      const list = Dom.el( "ul" , { class: "link-list trash-list" } );
      cards.forEach( ( card ) => list.appendChild( this.row( ctx , card , back ) ) );
      body.appendChild( list );
      body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "trash.empty_button" , async () => {
          if ( !window.confirm( I18n.fill( "trash.empty_confirm" , { n: cards.length } ) ) ) return;
          try { await ctx.client.emptyTrash(); } catch ( error ) { Ui.error( error ); }
          this.open( ctx , back );
        } , "secondary small danger" ),
      ] } ) );
    } , back );
  },

  row( ctx , card , back ) {
    const title = Dom.el( "span" , { class: "trash-title" } );
    Markdown.renderInline( card.title , title );
    const list = ctx.board.lists.find( function ( item ) { return item.id === card.list_id; } );
    const meta = Dom.el( "span" , { class: "muted small-print" , text: I18n.fill( "trash.deleted_meta" , {
      list: list ? list.title : T( "trash.list_gone" ),
      name: card.deleted_by || "?",
      when: card.deleted_at ? Dates.format( card.deleted_at , false ) : "",
    } ) } );
    return Dom.el( "li" , { class: "link-row trash-row" , children: [
      Dom.el( "div" , { class: "trash-text" , children: [ title , meta ] } ),
      Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "trash.restore" , async () => {
          await ctx.mutate( function () { return ctx.client.restoreFromTrash( card.id ); } );
          Ui.toast( T( "trash.restored" ) , "ok" );
          this.open( BoardPage.context() , back );
        } ),
        Ui.button( "trash.purge" , async () => {
          if ( !window.confirm( I18n.fill( "trash.purge_confirm" , { name: card.title } ) ) ) return;
          try { await ctx.client.purgeCard( card.id ); } catch ( error ) { Ui.error( error ); }
          this.open( ctx , back );
        } , "secondary small danger" ),
      ] } ),
    ] } );
  },
};
