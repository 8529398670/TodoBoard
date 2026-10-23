// The boards home: every board I own or am on, cards shared with me from
// boards I am not on, and a form to start a board. A new board gets the
// lists named in language.yaml (home.default_lists), so they come out in
// whatever language the app is set to.
const Home = {
  async init( me ) {
    Dom.show( Dom.get( "home" ) , true );
    Dom.get( "create-board-form" ).addEventListener( "submit" , this.onCreate.bind( this ) );
    await this.refresh();
  },

  async refresh() {
    let result;
    try {
      result = await Api.request( "/api/boards" );
    } catch ( error ) {
      App.showError( error.message );
      return;
    }
    const grid = Dom.get( "boards-grid" );
    Dom.clear( grid );
    result.boards.forEach( function ( board ) {
      const tile = Dom.el( "a" , { class: "board-tile color-" + ( board.color || "none" ) , attrs: { href: "/b/" + ( board.slug || board.id ) } } );
      const title = Dom.el( "span" , { class: "board-tile-title" } );
      Markdown.renderInline( board.title , title );
      tile.appendChild( title );
      tile.appendChild( Dom.el( "span" , { class: "board-tile-role" , text: T( "share.role_" + board.role ) } ) );
      grid.appendChild( tile );
    } );
    Dom.show( Dom.get( "no-boards" ) , result.boards.length === 0 );

    const shared = Dom.get( "shared-cards" );
    Dom.clear( shared );
    result.shared_cards.forEach( function ( item ) {
      const link = Dom.el( "a" , { attrs: { href: "/b/" + ( item.board_slug || item.board_id ) + "?card=" + item.card_id } } );
      Markdown.renderInline( item.card_title , link );
      shared.appendChild( Dom.el( "li" , { class: "link-row" , children: [
        link,
        Dom.el( "span" , { class: "muted" , text: item.board_title + " · " + T( "share.mode_" + item.mode ) } ),
      ] } ) );
    } );
    Dom.show( Dom.get( "shared-cards-section" ) , result.shared_cards.length > 0 );
  },

  async onCreate( event ) {
    event.preventDefault();
    const input = Dom.get( "new-board-title" );
    const lists = T( "home.default_lists" ).split( "," ).map( function ( name ) { return name.trim(); } ).filter( Boolean );
    try {
      const created = await Api.post( "/api/boards" , { title: input.value.trim() , lists: lists } );
      window.location.href = "/b/" + ( created.slug || created.id );
    } catch ( error ) {
      App.showError( error.message );
    }
  },
};
