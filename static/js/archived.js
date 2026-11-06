// The board's archive: archived lists, then archived cards, each with
// Restore, and Open for a card. Reached from Board settings. Restoring keeps
// this panel open, so several can go back in a row.
const Archived = {
  lists( ctx ) {
    return ctx.board.lists.filter( function ( list ) { return list.archived; } );
  },

  cards( ctx ) {
    return ctx.cards.filter( function ( card ) { return card.archived; } );
  },

  count( ctx ) {
    return this.lists( ctx ).length + this.cards( ctx ).length;
  },

  open( ctx , back ) {
    Panel.open( "settings.archived_heading" , ( body ) => {
      if ( !this.count( ctx ) ) {
        body.appendChild( Dom.el( "p" , { class: "muted" , text: T( "settings.nothing_archived" ) } ) );
        return;
      }
      const list = Dom.el( "ul" , { class: "link-list trash-list" } );
      this.lists( ctx ).forEach( ( item ) => list.appendChild( this.listRow( ctx , item , back ) ) );
      this.cards( ctx ).forEach( ( card ) => list.appendChild( this.cardRow( ctx , card , back ) ) );
      body.appendChild( list );
    } , back );
  },

  listRow( ctx , item , back ) {
    const title = Dom.el( "span" , { class: "trash-title" } );
    Markdown.renderInline( I18n.fill( "settings.archived_list" , { name: item.title } ) , title );
    return Dom.el( "li" , { class: "link-row trash-row" , children: [
      Dom.el( "div" , { class: "trash-text" , children: [ title ] } ),
      Dom.el( "div" , { class: "row-actions" , children: [
        this.restore( ctx , function () { return ctx.client.updateList( item.id , { archived: false } ); } , back ),
      ] } ),
    ] } );
  },

  cardRow( ctx , card , back ) {
    const title = Dom.el( "span" , { class: "trash-title" } );
    Markdown.renderInline( card.title , title );
    const text = Dom.el( "div" , { class: "trash-text" , children: [ title ] } );
    const list = ctx.board.lists.find( function ( item ) { return item.id === card.list_id; } );
    if ( list ) text.appendChild( Dom.el( "span" , { class: "muted small-print" , text: list.title } ) );
    return Dom.el( "li" , { class: "link-row trash-row" , children: [
      text,
      Dom.el( "div" , { class: "row-actions" , children: [
        this.restore( ctx , function () { return ctx.client.patchCard( card.id , { archived: false } ); } , back ),
        Ui.button( "settings.open_card" , function () {
          Panel.close();
          ctx.openCard( card.id );
        } ),
      ] } ),
    ] } );
  },

  restore( ctx , change , back ) {
    return Ui.button( "settings.restore" , async () => {
      if ( await ctx.mutate( change ) ) Ui.toast( T( "settings.restored" ) , "ok" );
      this.open( BoardPage.context() , back );
    } );
  },
};
