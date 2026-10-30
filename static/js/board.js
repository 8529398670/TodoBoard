// The board page, for members (/b/<id>) and for guests holding a share link
// (/s/<credential>). It owns the loaded board, which view is showing, and
// the polling that picks up other people's changes. The views, the card
// editor, and the panels are separate files; this one only wires them.
//
// Every change goes to the server first and the board is then re-read. That
// costs one small GET per edit and buys a screen that never disagrees with
// the database -- important when several people edit the same board.
const BoardPage = {
  client: null,
  data: null,
  me: null,
  view: "kanban",
  dragging: false,
  loading: false,
  pollTimer: null,

  views() {
    return { kanban: KanbanView , todo: TodoView , timeline: TimelineView };
  },

  async init() {
    try {
      await I18n.load();
      I18n.apply();
    } catch ( error ) {
      Dom.toast( "Could not load language file." , "error" );
      return;
    }
    Panel.bind();
    Markdown.init();

    try {
      await this.connect();
    } catch ( error ) {
      this.showFatal( error );
      return;
    }

    Dom.get( "settings-button" ).addEventListener( "click" , () => BoardSettings.open( this.context() ) );
    Dom.get( "trash-button" ).addEventListener( "click" , () => Trash.open( this.context() ) );
    UserSettings.bind( () => this.me );

    // Every change made from this tab can be undone; see history.js.
    History.attach( this.client , () => this.data , () => this.load() );
    History.onChange = () => this.showHistory();
    History.bindKeys();
    Dom.get( "undo-button" ).addEventListener( "click" , () => History.undo() );
    Dom.get( "redo-button" ).addEventListener( "click" , () => History.redo() );

    await this.load();
    const params = new URLSearchParams( window.location.search );
    const cardId = Number( params.get( "card" ) );
    if ( cardId ) this.openCard( cardId );
    this.startPolling();
  },

  // Decide who we are: a member by path /b/<id>, or a guest by /s/<token>.
  async connect() {
    const token = BoardApi.tokenFromPath( "/s/" );
    if ( token ) {
      this.client = await BoardApi.guest( token );
      this.me = { authenticated: false };
      Dom.show( Dom.get( "home-link" ) , false );
      if ( this.client.info.mode === "edit" ) await this.askGuestName();
      const info = this.client.info;
      const banner = Dom.get( "guest-banner" );
      Dom.text( banner , I18n.fill( info.mode === "edit" ? "share.banner_edit" : "share.banner_view" , {
        label: info.label || ( info.list_title ? info.board_title + " › " + info.list_title : info.board_title ),
      } ) );
      Dom.show( banner , true );
      return;
    }
    // /b/<id> or /b/<slug>; the server takes either. Once the board is
    // loaded the client switches to the numeric ID, so renaming the slug
    // mid-session cannot strand it.
    const match = /^\/b\/([a-z0-9-]+)\/?$/.exec( window.location.pathname );
    if ( !match ) throw new Error( "board_not_found" );
    this.me = await Api.me();
    if ( !this.me.authenticated ) {
      window.location.replace( "/" );
      throw new Error( "not authenticated" );
    }
    Dates.configure( this.me.display );
    Theme.setUser( this.me.display.theme , this.me.display.own_theme_on_boards );
    this.client = BoardApi.member( /^\d+$/.test( match[ 1 ] ) ? Number( match[ 1 ] ) : match[ 1 ] );
  },

  // A guest with an edit link is asked once (per browser) what to be called,
  // so edits read "Sam (guest)" rather than nobody.
  askGuestName() {
    let saved = "";
    try { saved = window.localStorage.getItem( "guest_name" ) || ""; } catch ( error ) { saved = ""; }
    if ( saved ) {
      this.client.guestName = saved;
      return Promise.resolve();
    }
    const dialog = Dom.get( "guest-name-dialog" );
    const form = Dom.get( "guest-name-form" );
    const input = Dom.get( "guest-name-input" );
    return new Promise( ( resolve ) => {
      const finish = () => {
        const name = input.value.trim().slice( 0 , 40 );
        this.client.guestName = name;
        try { if ( name ) window.localStorage.setItem( "guest_name" , name ); } catch ( error ) { /* not remembered */ }
        if ( dialog.open ) dialog.close();
        resolve();
      };
      form.addEventListener( "submit" , function ( event ) {
        event.preventDefault();
        finish();
      } );
      Dom.get( "guest-name-skip" ).addEventListener( "click" , finish );
      dialog.addEventListener( "cancel" , finish );
      dialog.showModal();
    } );
  },

  async load() {
    if ( this.loading ) return;
    this.loading = true;
    try {
      this.data = await this.client.board();
    } catch ( error ) {
      this.loading = false;
      if ( error.status === 404 ) {
        this.showFatal( error );
        return;
      }
      Ui.error( error );
      return;
    }
    this.loading = false;
    if ( !this.client.isGuest ) {
      this.client.boardId = this.data.board.id;
      this.showSlug();
    }
    if ( !this.viewChosen ) {
      this.viewChosen = true;
      let saved = "";
      try { saved = window.localStorage.getItem( "view:" + this.data.board.id ) || ""; } catch ( error ) { saved = ""; }
      this.view = this.views()[ saved ] ? saved : ( this.data.board.default_view || "kanban" );
    }
    this.render();
    CardModal.refresh( this.context() );
  },

  // Keep the address bar on the board's slug (/b/1 becomes /b/lab-work),
  // query string and all.
  showSlug() {
    const slug = this.data.board.slug;
    if ( !slug ) return;
    const path = "/b/" + slug;
    if ( window.location.pathname !== path ) window.history.replaceState( null , "" , path + window.location.search );
  },

  // What every view, panel, and the card editor is handed.
  context() {
    const data = this.data;
    const level = data ? data.access : "none";
    const page = this;
    return {
      data: data,
      board: data ? data.board : null,
      cards: data ? data.cards : [],
      people: data ? data.people : {},
      client: this.client,
      me: this.me,
      isGuest: this.client ? this.client.isGuest : false,
      canEdit: level === "edit" || level === "manage",
      canManage: level === "manage",
      cardOnly: data ? data.card_only : false,
      // A column share link: the one column is the whole board for them.
      listOnly: data ? !!data.list_only : false,
      view: this.view,
      openCard( cardId ) { page.openCard( cardId ); },
      reload() { return page.load(); },
      setDragging( flag ) { page.dragging = flag; },
      // A three-column board shows its first three open lists and nothing
      // else; the rest are kept, and come back when the layout is changed.
      // Someone shown a single column sees just that column either way.
      isThree() {
        return !!data && data.board.layout === "three" && !data.list_only;
      },
      // Cards are numbered when the board says so, or, left to each person,
      // when their account does. A guest has no account to ask.
      numbered() {
        if ( !data ) return false;
        if ( data.board.numbering ) return data.board.numbering === "on";
        return !!( page.me && page.me.display && page.me.display.number_cards );
      },
      // Grey status icons: this person's choice for this board, else their
      // account default.
      plainIcons() {
        if ( !data ) return false;
        if ( typeof data.me.plain_icons === "boolean" ) return data.me.plain_icons;
        return !!( page.me && page.me.display && page.me.display.plain_icons );
      },
      swipeColumns() {
        return !!( page.me && page.me.display && page.me.display.swipe_columns );
      },
      lists() {
        const open = data.board.lists.filter( function ( list ) { return !list.archived; } );
        return this.isThree() ? open.slice( 0 , 3 ) : open;
      },
      // A list's open cards, in order: what is counted, numbered, copied
      // and moved among.
      cardsIn( listId ) {
        return data.cards
          .filter( function ( card ) { return card.list_id === listId && !card.archived; } )
          .sort( function ( a , b ) { return a.position - b.position; } );
      },
      showArchived() { return page.showArchived(); },
      // What a column shows: its open cards, then -- with "Show archived"
      // on -- its archived ones, faded, after them.
      shownIn( listId ) {
        const open = this.cardsIn( listId );
        if ( !page.showArchived() ) return open;
        return open.concat( data.cards
          .filter( function ( card ) { return card.list_id === listId && card.archived; } )
          .sort( function ( a , b ) { return a.position - b.position; } ) );
      },
      card( cardId ) {
        return data.cards.find( function ( card ) { return card.id === cardId; } ) || null;
      },
      // Run a change, then re-read the board. Errors become a toast.
      async mutate( action ) {
        try {
          const result = await action();
          await page.load();
          return result;
        } catch ( error ) {
          Ui.error( error );
          await page.load();
          return null;
        }
      },
    };
  },

  render() {
    const ctx = this.context();
    const board = ctx.board;
    document.title = board.title + " · " + T( "app.title" );
    Markdown.renderInline( board.title , Dom.get( "board-title" ) );
    Theme.setBoard( board.theme );
    document.body.className = "board-page color-" + ( board.color || "none" ) + ( ctx.plainIcons() ? " plain-status" : "" ) + ( ctx.swipeColumns() ? " swipe-columns" : "" );

    Dom.show( Dom.get( "settings-button" ) , !ctx.cardOnly );
    // The trash is the whole board's, so it is for board editors only.
    Dom.show( Dom.get( "trash-button" ) , ctx.canEdit && !ctx.cardOnly && !ctx.listOnly );
    Dom.show( Dom.get( "account-link" ) , !ctx.isGuest );
    Dom.show( Dom.get( "guest-settings-button" ) , ctx.isGuest );
    this.showHistory();

    const root = Dom.get( "board-root" );
    const scroller = root.querySelector( ".scroll-x" );
    const scrollLeft = scroller ? scroller.scrollLeft : 0;
    const scrollTop = window.scrollY;
    Dom.clear( root );
    root.className = "board-root view-" + ( ctx.cardOnly ? "cards" : this.view );
    if ( ctx.cardOnly ) {
      CardOnlyView.render( root , ctx );
    } else {
      this.views()[ this.view ].render( root , ctx );
    }
    const restored = root.querySelector( ".scroll-x" );
    if ( restored && scrollLeft ) restored.scrollLeft = scrollLeft;
    window.scrollTo( 0 , scrollTop );
  },

  // "Show archived" is remembered per board, in this browser.
  showArchived() {
    if ( !this.data ) return false;
    try { return window.localStorage.getItem( "show_archived:" + this.data.board.id ) === "1"; } catch ( error ) { return false; }
  },

  setShowArchived( on ) {
    try { window.localStorage.setItem( "show_archived:" + this.data.board.id , on ? "1" : "0" ); } catch ( error ) { /* not remembered */ }
  },

  // How many archived cards the visible columns hold: what "Show archived"
  // (Board settings, View) would bring back.
  archivedCount( ctx ) {
    const shown = {};
    ctx.lists().forEach( function ( list ) { shown[ list.id ] = true; } );
    return ctx.cards.filter( function ( card ) { return card.archived && shown[ card.list_id ]; } ).length;
  },

  // The undo and redo buttons: there for anyone who can change something,
  // usable when there is something to undo or redo.
  showHistory() {
    const data = this.data;
    const canEdit = !!data && ( data.access === "edit" || data.access === "manage" ||
      data.cards.some( function ( card ) { return card.access === "edit"; } ) );
    const undo = Dom.get( "undo-button" );
    const redo = Dom.get( "redo-button" );
    Dom.show( undo , canEdit );
    Dom.show( redo , canEdit );
    undo.disabled = !History.canUndo();
    redo.disabled = !History.canRedo();
  },

  setView( view ) {
    if ( !this.views()[ view ] ) return;
    this.view = view;
    try { window.localStorage.setItem( "view:" + this.data.board.id , view ); } catch ( error ) { /* per-tab only */ }
    this.render();
  },

  openCard( cardId ) {
    CardModal.open( this.context() , cardId );
  },

  // Poll the board's revision; re-read only when it moved. Paused while the
  // tab is hidden or a drag is in progress, so a refresh never yanks a card
  // out from under someone's finger.
  startPolling() {
    this.pollTimer = window.setInterval( async () => {
      if ( document.hidden || this.dragging || this.loading || !this.data ) return;
      try {
        const stamp = await this.client.stamp();
        if ( stamp.rev !== this.data.board.rev ) await this.load();
      } catch ( error ) {
        if ( error.status === 404 || error.status === 401 ) {
          window.clearInterval( this.pollTimer );
          this.showFatal( error );
        }
      }
    } , 4000 );
    document.addEventListener( "visibilitychange" , () => {
      if ( !document.hidden && this.data ) this.load();
    } );
  },

  showFatal( error ) {
    const root = Dom.get( "board-root" );
    Dom.clear( root );
    const code = error && error.message ? error.message : "";
    root.appendChild( Dom.el( "section" , { class: "card card-body empty-state" , children: [
      Dom.el( "h2" , { text: T( "errors." + code ) || T( "errors.board_not_found" ) } ),
      Dom.el( "a" , { text: T( "nav.boards" ) , attrs: { href: "/" } } ),
    ] } ) );
  },
};

// Someone who reaches this board only through individual cards sees just
// those cards, each opening in the card editor.
const CardOnlyView = {
  render( root , ctx ) {
    const wrap = Dom.el( "section" , { class: "card-only" } );
    wrap.appendChild( Dom.el( "p" , { class: "muted" , text: T( "share.card_only_note" ) } ) );
    ctx.cards.forEach( function ( card ) {
      const button = Dom.el( "button" , { class: "card-face standalone" , attrs: { type: "button" } , on: { click: function () { ctx.openCard( card.id ); } } } );
      const title = Dom.el( "span" , { class: "card-title" } );
      Markdown.renderInline( card.title , title );
      button.appendChild( title );
      const meta = Dom.el( "div" , { class: "card-meta" } );
      [ CardBits.due( card ) , CardBits.checklist( card ) ].filter( Boolean ).forEach( function ( node ) { meta.appendChild( node ); } );
      if ( meta.childNodes.length ) button.appendChild( meta );
      wrap.appendChild( button );
    } );
    root.appendChild( wrap );
    // A single shared card opens straight away.
    if ( ctx.cards.length === 1 && !CardModal.cardId ) ctx.openCard( ctx.cards[ 0 ].id );
  },
};

document.addEventListener( "DOMContentLoaded" , function () {
  BoardPage.init();
} );
