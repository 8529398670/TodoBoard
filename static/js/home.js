// The boards home: every board I own or am on, sorted into my own folders,
// a search box over them, cards shared with me from boards I am not on, and a
// form to start a board. A new board gets the lists named in language.yaml
// (home.default_lists), so they come out in whatever language the app is set
// to.
//
// Folders are private to each person (see server/models/folder.go): filing a
// shared board here moves it on my home page and nobody else's.
const Home = {
  boards: [],
  folders: [],

  async init( me ) {
    Dom.show( Dom.get( "home" ) , true );
    Dom.get( "create-board-form" ).addEventListener( "submit" , this.onCreate.bind( this ) );
    Dom.get( "board-search" ).addEventListener( "input" , () => this.filter() );
    Dom.get( "new-folder-button" ).addEventListener( "click" , () => this.editFolder( null ) );
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
    this.boards = result.boards;
    this.folders = result.folders || [];
    this.render();

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

  // Collapsed folders, remembered in this browser: folder ids.
  collapsed() {
    try { return JSON.parse( window.localStorage.getItem( "home_collapsed" ) || "[]" ); } catch ( error ) { return []; }
  },

  setCollapsed( folderId , closed ) {
    const ids = this.collapsed().filter( function ( id ) { return id !== folderId; } );
    if ( closed ) ids.push( folderId );
    try { window.localStorage.setItem( "home_collapsed" , JSON.stringify( ids ) ); } catch ( error ) { /* not remembered */ }
  },

  // One section per folder, A to Z, then the boards in none. With no folders
  // at all, the unfiled boards are just the grid, with no heading.
  render() {
    const root = Dom.get( "board-folders" );
    Dom.clear( root );
    const known = {};
    this.folders.forEach( function ( folder ) { known[ folder.id ] = true; } );
    this.folders.forEach( ( folder ) => {
      root.appendChild( this.section( folder , this.boards.filter( function ( board ) { return board.folder_id === folder.id; } ) ) );
    } );
    const unfiled = this.boards.filter( function ( board ) { return !known[ board.folder_id ]; } );
    if ( this.folders.length || unfiled.length ) root.appendChild( this.section( null , unfiled ) );
    Dom.show( Dom.get( "no-boards" ) , this.boards.length === 0 );
    this.bindDragging( root );
    this.filter();
  },

  section( folder , boards ) {
    const folderId = folder ? folder.id : 0;
    const section = Dom.el( "section" , { class: "folder" + ( boards.length ? "" : " is-empty" ) , attrs: { "data-folder-id": folderId } } );
    const grid = Dom.el( "div" , { class: "boards-grid" , attrs: { "data-folder-id": folderId } } );
    boards.forEach( ( board ) => grid.appendChild( this.tile( board ) ) );

    if ( folder || this.folders.length ) {
      const closed = this.collapsed().indexOf( folderId ) >= 0;
      const head = Dom.el( "div" , { class: "folder-head" } );
      const toggle = Dom.el( "button" , {
        class: "folder-toggle",
        attrs: { type: "button" , "aria-expanded": closed ? "false" : "true" },
        children: [
          Dom.el( "span" , { class: "folder-symbol" , text: T( closed ? "home.closed_symbol" : "home.open_symbol" ) , attrs: { "aria-hidden": "true" } } ),
          Dom.el( "span" , { class: "folder-name" , text: folder ? folder.name : T( "home.unfiled" ) } ),
          Dom.el( "span" , { class: "folder-count" , text: I18n.fill( "home.folder_count" , { n: boards.length } ) } ),
        ],
      } );
      toggle.addEventListener( "click" , () => {
        const open = toggle.getAttribute( "aria-expanded" ) !== "true";
        this.setCollapsed( folderId , !open );
        this.showFolder( section , open );
      } );
      head.appendChild( toggle );
      if ( folder ) {
        head.appendChild( Dom.el( "button" , {
          class: "icon-button small",
          text: T( "board.menu_symbol" ),
          attrs: { type: "button" , title: T( "home.folder_menu" ) , "aria-label": T( "home.folder_menu" ) },
          on: { click: () => this.editFolder( folder ) },
        } ) );
      }
      section.appendChild( head );
      grid.hidden = closed;
    }
    section.appendChild( grid );
    return section;
  },

  showFolder( section , open ) {
    const toggle = section.querySelector( ".folder-toggle" );
    if ( !toggle ) return;
    toggle.setAttribute( "aria-expanded" , open ? "true" : "false" );
    Dom.text( toggle.querySelector( ".folder-symbol" ) , T( open ? "home.open_symbol" : "home.closed_symbol" ) );
    section.querySelector( ".boards-grid" ).hidden = !open;
  },

  tile( board ) {
    const wrap = Dom.el( "div" , { class: "board-tile-wrap" , attrs: { "data-board-id": board.id } } );
    const tile = Dom.el( "a" , { class: "board-tile color-" + ( board.color || "none" ) , attrs: { href: "/b/" + ( board.slug || board.id ) } } );
    const title = Dom.el( "span" , { class: "board-tile-title" } );
    Markdown.renderInline( board.title , title );
    tile.appendChild( title );
    tile.appendChild( Dom.el( "span" , { class: "board-tile-role" , text: T( "share.role_" + board.role ) } ) );
    wrap.appendChild( tile );
    wrap.appendChild( Dom.el( "button" , {
      class: "icon-button small tile-menu",
      text: T( "board.menu_symbol" ),
      attrs: { type: "button" , title: T( "home.board_menu" ) , "aria-label": T( "home.board_menu" ) },
      on: { click: () => this.moveBoard( board ) },
    } ) );
    return wrap;
  },

  // Tiles drag between folders. Order inside a folder is most recently
  // changed first, so a drop only ever changes the folder.
  bindDragging( root ) {
    if ( typeof Sortable === "undefined" ) return;
    Dom.all( ".boards-grid" , root ).forEach( ( grid ) => {
      Sortable.create( grid , {
        group: "boards",
        sort: false,
        animation: 150,
        forceFallback: true,
        delay: 150,
        delayOnTouchOnly: true,
        draggable: ".board-tile-wrap",
        filter: ".tile-menu",
        preventOnFilter: false,
        ghostClass: "drag-ghost",
        onEnd: ( event ) => {
          if ( event.from === event.to ) return;
          this.setFolder( Number( event.item.getAttribute( "data-board-id" ) ) , Number( event.to.getAttribute( "data-folder-id" ) ) );
        },
      } );
    } );
  },

  async setFolder( boardId , folderId ) {
    try {
      await Api.post( "/api/boards/" + boardId + "/folder" , { folder_id: folderId } );
    } catch ( error ) {
      App.showError( error.message );
    }
    await this.refresh();
  },

  // Live search over board titles and folder names. While searching, every
  // folder with a match opens and the rest step aside.
  filter() {
    const query = Dom.get( "board-search" ).value.trim().toLowerCase();
    const closed = this.collapsed();
    let shown = 0;
    Dom.all( ".folder" , Dom.get( "board-folders" ) ).forEach( ( section ) => {
      const folderId = Number( section.getAttribute( "data-folder-id" ) );
      const folder = this.folders.find( function ( item ) { return item.id === folderId; } );
      const folderMatches = !!query && !!folder && folder.name.toLowerCase().indexOf( query ) >= 0;
      let matches = 0;
      Dom.all( ".board-tile-wrap" , section ).forEach( ( wrap ) => {
        const board = this.boards.find( function ( item ) { return item.id === Number( wrap.getAttribute( "data-board-id" ) ); } );
        const hit = !query || folderMatches || ( board && board.title.toLowerCase().indexOf( query ) >= 0 );
        wrap.hidden = !hit;
        if ( hit ) matches += 1;
      } );
      shown += matches;
      section.hidden = !!query && matches === 0 && !folderMatches;
      this.showFolder( section , query ? true : closed.indexOf( folderId ) < 0 );
    } );
    Dom.show( Dom.get( "no-matches" ) , !!query && shown === 0 && this.boards.length > 0 );
  },

  // New folder (folder null), or rename / delete one.
  editFolder( folder ) {
    Panel.open( "home.folder_title" , ( body ) => {
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const name = Dom.el( "input" , { attrs: { maxlength: "80" , required: "" , placeholder: T( "home.folder_name_placeholder" ) } } );
      name.value = folder ? folder.name : "";
      form.appendChild( Ui.field( "home.folder_name_label" , name ) );
      form.appendChild( Dom.el( "button" , { text: T( folder ? "home.rename_folder" : "home.create_folder" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        try {
          if ( folder ) await Api.post( "/api/folders/" + folder.id , { name: name.value } );
          else await Api.post( "/api/folders" , { name: name.value } );
          Panel.close();
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        await this.refresh();
      } );
      body.appendChild( form );
      if ( folder ) {
        body.appendChild( Ui.button( "home.delete_folder" , async () => {
          if ( !window.confirm( I18n.fill( "home.delete_folder_confirm" , { name: folder.name } ) ) ) return;
          try {
            await Api.post( "/api/folders/" + folder.id + "/delete" );
            Panel.close();
          } catch ( error ) {
            Ui.error( error );
            return;
          }
          await this.refresh();
        } , "secondary small danger" ) );
      }
      window.setTimeout( function () { name.focus(); } , 0 );
    } );
  },

  // The keyboard and touch way to file a board.
  moveBoard( board ) {
    Panel.open( "home.move_title" , ( body ) => {
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const select = Ui.select( [ { value: 0 , text: T( "home.move_none" ) } ].concat( this.folders.map( function ( folder ) {
        return { value: folder.id , text: folder.name };
      } ) ) , board.folder_id || 0 );
      form.appendChild( Ui.field( "home.move_label" , select ) );
      form.appendChild( Dom.el( "button" , { text: T( "home.move_button" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        Panel.close();
        await this.setFolder( board.id , Number( select.value ) );
      } );
      body.appendChild( form );
    } );
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
