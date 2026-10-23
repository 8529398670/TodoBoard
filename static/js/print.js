// The print layout behind "Export as PDF": every list and card written out
// in full, math rendered, then the browser's print dialog -- where "Save as
// PDF" produces a document with real, selectable text.
//
// /print/b/<id> for a member, /print/s/<credential> for a guest link.
const PrintPage = {
  data: null,

  async init() {
    try {
      await I18n.load();
      I18n.apply();
    } catch ( error ) {
      return;
    }
    Markdown.init();
    let client;
    try {
      const token = BoardApi.tokenFromPath( "/print/s/" );
      if ( token ) {
        client = await BoardApi.guest( token );
      } else {
        const match = /^\/print\/b\/(\d+)/.exec( window.location.pathname );
        if ( !match ) throw new Error( "board_not_found" );
        const me = await Api.me();
        if ( !me.authenticated ) throw new Error( "not authenticated" );
        Dates.configure( me.display );
        client = BoardApi.member( Number( match[ 1 ] ) );
      }
      this.data = await client.board();
    } catch ( error ) {
      Dom.get( "print-root" ).appendChild( Dom.el( "p" , { class: "notice notice-error" , text: T( "errors." + error.message ) || T( "errors.board_not_found" ) } ) );
      return;
    }
    Dom.get( "print-button" ).addEventListener( "click" , function () { window.print(); } );
    Dom.get( "print-descriptions" ).addEventListener( "change" , () => this.render() );
    Dom.get( "print-done" ).addEventListener( "change" , () => this.render() );
    this.render();
    // Fonts first, or KaTeX prints in fallback glyphs.
    if ( document.fonts && document.fonts.ready ) await document.fonts.ready;
    // ?preview shows the layout without opening the print dialog.
    if ( new URLSearchParams( window.location.search ).has( "preview" ) ) return;
    window.setTimeout( function () { window.print(); } , 300 );
  },

  render() {
    const data = this.data;
    const board = data.board;
    const withDescriptions = Dom.get( "print-descriptions" ).checked;
    const withDone = Dom.get( "print-done" ).checked;
    document.title = board.title;
    const root = Dom.get( "print-root" );
    Dom.clear( root );

    const title = Dom.el( "h1" , { class: "print-title" } );
    Markdown.renderInline( board.title , title );
    root.appendChild( title );
    root.appendChild( Dom.el( "p" , { class: "muted" , text: I18n.fill( "export.printed_on" , { when: new Date().toLocaleString() } ) } ) );

    board.lists.filter( function ( list ) { return !list.archived; } ).forEach( function ( list ) {
      const cards = data.cards
        .filter( function ( card ) { return card.list_id === list.id && !card.archived && ( withDone || !card.done ); } )
        .sort( function ( a , b ) { return a.position - b.position; } );
      const section = Dom.el( "section" , { class: "print-list" } );
      const heading = Dom.el( "h2" );
      Markdown.renderInline( list.title , heading );
      section.appendChild( heading );
      if ( !cards.length ) section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "export.md_empty_list" ) } ) );
      cards.forEach( function ( card ) {
        const article = Dom.el( "article" , { class: "print-card" + ( card.done ? " is-done" : "" ) } );
        const cardTitle = Dom.el( "h3" );
        if ( card.done ) cardTitle.appendChild( Dom.el( "span" , { class: "done-tick" , text: T( "card.done_symbol" ) + " " } ) );
        const text = Dom.el( "span" );
        Markdown.renderInline( card.title , text );
        cardTitle.appendChild( text );
        article.appendChild( cardTitle );

        const meta = Dom.el( "div" , { class: "card-meta" } );
        [ CardBits.labels( card , board ) , CardBits.due( card ) , CardBits.checklist( card ) ].filter( Boolean ).forEach( function ( node ) { meta.appendChild( node ); } );
        if ( card.assignees.length ) {
          meta.appendChild( Dom.el( "span" , { class: "muted" , text: card.assignees.map( function ( id ) { return data.people[ id ] || "?"; } ).join( ", " ) } ) );
        }
        if ( meta.childNodes.length ) article.appendChild( meta );

        if ( withDescriptions && CardBits.hasDescription( card ) ) {
          const description = Dom.el( "div" , { class: "description" } );
          Markdown.render( card.description , description );
          article.appendChild( description );
        }
        if ( card.checklist.length ) {
          const items = Dom.el( "ul" , { class: "checklist" } );
          card.checklist.forEach( function ( item ) {
            const row = Dom.el( "li" , { class: "checklist-item" + ( item.done ? " is-done" : "" ) } );
            row.appendChild( Dom.el( "span" , { class: "export-box" , text: item.done ? T( "export.box_checked" ) : T( "export.box_empty" ) } ) );
            const itemText = Dom.el( "span" );
            Markdown.renderInline( item.text , itemText );
            row.appendChild( itemText );
            items.appendChild( row );
          } );
          article.appendChild( items );
        }
        section.appendChild( article );
      } );
      root.appendChild( section );
    } );
  },
};

document.addEventListener( "DOMContentLoaded" , function () {
  PrintPage.init();
} );
