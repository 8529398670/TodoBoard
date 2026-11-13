// The boards home, in two parts: "Your boards" -- the ones I own, sorted
// into my own folders -- and "Shared with you" -- other people's boards I am
// on, and single cards shared with me from boards I am not on. A search box
// over both, and a form to start a board. A new board gets the lists named in language.yaml
// (home.default_lists), so they come out in whatever language the app is set
// to.
//
// Folders are private to each person (see server/models/folder.go): filing a
// shared board here moves it on my home page and nobody else's. They nest, as
// an indented tree on this one page, and each level reads A to Z, or in the
// person's own order (an account setting); dragging a folder by its grip
// switches to their own order. A folder moves to another level from its menu.
//
// Above it all, "Recently edited": the few boards changed last, mine and
// shared, for jumping straight back in.
const Home = {
  me: null,
  boards: [],
  shared: [],
  folders: [],
  recentCount: 6,
  // Fewer boards than this and the whole list is already a glance away.
  recentMinBoards: 4,
  // server/models/folder.go's maxFolderDepth: what is offered here, not what
  // is enforced.
  folderMaxDepth: 3,

  async init( me ) {
    this.me = me;
    const sort = Dom.get( "folder-sort" );
    sort.value = this.manual() ? "manual" : "";
    sort.addEventListener( "change" , () => this.setSort( sort.value ) );
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
    this.boards = result.boards.filter( function ( board ) { return board.role === "owner"; } );
    this.shared = result.boards.filter( function ( board ) { return board.role !== "owner"; } );
    this.folders = result.folders || [];
    this.render();
    this.renderShared();
    this.renderRecent( result.boards );

    const shared = Dom.get( "shared-cards" );
    Dom.clear( shared );
    result.shared_cards.forEach( function ( item ) {
      const link = Dom.el( "a" , { attrs: { href: item.card_path || ( "/b/" + ( item.board_slug || item.board_id ) + "?card=" + item.card_id ) } } );
      Markdown.renderInline( item.card_title , link );
      shared.appendChild( Dom.el( "li" , { class: "link-row list-group-item" , children: [
        link,
        Dom.el( "span" , { class: "muted" , text: item.board_title } ),
      ] } ) );
    } );
    Dom.show( Dom.get( "shared-cards-section" ) , result.shared_cards.length > 0 );
    Dom.show( Dom.get( "shared-section" ) , this.shared.length > 0 || result.shared_cards.length > 0 );
    this.filter();
  },

  manual() {
    return !!( this.me && this.me.display && this.me.display.folder_sort === "manual" );
  },

  async setSort( value ) {
    if ( !this.me ) return;
    this.me.display = this.me.display || {};
    await UserSettings.save( this.me , { folder_sort: value } );
    Dom.get( "folder-sort" ).value = this.manual() ? "manual" : "";
    this.render();
  },

  // Folders in the order chosen: the server's order is the person's own,
  // A to Z is sorted here.
  orderedFolders() {
    if ( this.manual() ) return this.folders.slice();
    return this.folders.slice().sort( function ( a , b ) {
      return a.name.localeCompare( b.name , undefined , { sensitivity: "base" } );
    } );
  },

  // The folders directly inside parentId, in the chosen order; 0 is the top.
  // A folder whose parent cannot be found shows at the top, not nowhere.
  childFolders( parentId ) {
    const known = {};
    this.folders.forEach( function ( folder ) { known[ folder.id ] = true; } );
    return this.orderedFolders().filter( function ( folder ) {
      return ( known[ folder.parent_id ] ? folder.parent_id : 0 ) === parentId;
    } );
  },

  boardsIn( folderId ) {
    return this.boards.filter( function ( board ) { return board.folder_id === folderId; } );
  },

  // Boards in a folder and in every folder inside it.
  boardsUnder( folderId ) {
    return this.childFolders( folderId ).reduce( ( total , child ) => total + this.boardsUnder( child.id ) , this.boardsIn( folderId ).length );
  },

  // Every folder as a tree read top to bottom, each with how deep it sits,
  // 1 at the top; leaving out skipId and everything inside it.
  folderTree( parentId , depth , skipId ) {
    const rows = [];
    this.childFolders( parentId ).forEach( ( folder ) => {
      if ( folder.id === skipId ) return;
      rows.push( { folder: folder , depth: depth } );
      this.folderTree( folder.id , depth + 1 , skipId ).forEach( function ( row ) { rows.push( row ); } );
    } );
    return rows;
  },

  // How many levels a folder and what is inside it take up.
  folderHeight( folderId ) {
    return 1 + this.childFolders( folderId ).reduce( ( most , child ) => Math.max( most , this.folderHeight( child.id ) ) , 0 );
  },

  // A folder's name for a dropdown, indented to its level. Spaces that a
  // select will not collapse.
  treeLabel( row ) {
    return "\u00a0\u00a0\u00a0".repeat( row.depth - 1 ) + row.folder.name;
  },

  renderRecent( all ) {
    const section = Dom.get( "recent-section" );
    const grid = Dom.get( "recent-boards" );
    Dom.clear( grid );
    if ( all.length < this.recentMinBoards ) {
      Dom.show( section , false );
      return;
    }
    all.slice()
      .sort( function ( a , b ) { return new Date( b.updated_at ) - new Date( a.updated_at ); } )
      .slice( 0 , this.recentCount )
      .forEach( ( board ) => {
        const wrap = this.tile( board );
        wrap.querySelector( ".board-tile" ).appendChild( Dom.el( "span" , {
          class: "board-tile-when muted",
          text: I18n.fill( "home.edited_ago" , { when: this.ago( board.updated_at ) } ),
          attrs: { title: new Date( board.updated_at ).toLocaleString() },
        } ) );
        grid.appendChild( wrap );
      } );
    Dom.show( section , true );
  },

  // "5 minutes ago", "yesterday", in the browser's language.
  ago( value ) {
    const seconds = ( new Date( value ).getTime() - Date.now() ) / 1000;
    const steps = [ [ 60 , "second" ] , [ 60 , "minute" ] , [ 24 , "hour" ] , [ 7 , "day" ] , [ 4.35 , "week" ] , [ 12 , "month" ] , [ Infinity , "year" ] ];
    let amount = seconds;
    for ( const [ size , unit ] of steps ) {
      if ( Math.abs( amount ) < size ) {
        try {
          return new Intl.RelativeTimeFormat( undefined , { numeric: "auto" } ).format( Math.round( amount ) , unit );
        } catch ( error ) {
          return new Date( value ).toLocaleDateString();
        }
      }
      amount /= size;
    }
    return new Date( value ).toLocaleDateString();
  },

  // Boards other people own and I am on: one plain grid. Folders are for
  // sorting your own boards.
  renderShared() {
    const grid = Dom.get( "shared-boards" );
    Dom.clear( grid );
    this.shared.forEach( ( board ) => grid.appendChild( this.tile( board , false ) ) );
    Dom.show( grid , this.shared.length > 0 );
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

  // One section per top-level folder, in the chosen order, each holding the
  // folders inside it, then the boards in none. With no folders at all, the
  // unfiled boards are just the grid, with no heading.
  render() {
    const root = Dom.get( "board-folders" );
    Dom.clear( root );
    const known = {};
    this.folders.forEach( function ( folder ) { known[ folder.id ] = true; } );
    Dom.show( Dom.get( "folder-sort" ) , this.folders.length > 1 );
    this.childFolders( 0 ).forEach( ( folder ) => root.appendChild( this.section( folder , this.boardsIn( folder.id ) ) ) );
    const unfiled = this.boards.filter( function ( board ) { return !known[ board.folder_id ]; } );
    if ( this.folders.length || unfiled.length ) root.appendChild( this.section( null , unfiled ) );
    Dom.show( Dom.get( "no-boards" ) , this.boards.length === 0 );
    this.bindDragging( root );
    this.filter();
  },

  section( folder , boards ) {
    const folderId = folder ? folder.id : 0;
    const section = Dom.el( "section" , { class: "folder" + ( boards.length ? "" : " is-empty" ) , attrs: { "data-folder-id": folderId } } );
    // What collapsing the folder hides: its boards and the folders inside it.
    const body = Dom.el( "div" , { class: "folder-body" } );
    const grid = Dom.el( "div" , { class: "boards-grid" , attrs: { "data-folder-id": folderId } } );
    boards.forEach( ( board ) => grid.appendChild( this.tile( board ) ) );

    if ( folder || this.folders.length ) {
      const closed = this.collapsed().indexOf( folderId ) >= 0;
      const head = Dom.el( "div" , { class: "folder-head" } );
      if ( folder && this.siblingCount( folder ) > 1 ) {
        head.appendChild( Dom.el( "span" , {
          class: "folder-grip",
          text: T( "home.folder_drag_symbol" ),
          attrs: { title: T( "home.folder_drag" ) , "aria-hidden": "true" },
        } ) );
      }
      const toggle = Dom.el( "button" , {
        class: "folder-toggle",
        attrs: { type: "button" , "aria-expanded": closed ? "false" : "true" },
        children: [
          Dom.el( "span" , { class: "folder-symbol" , text: T( closed ? "home.closed_symbol" : "home.open_symbol" ) , attrs: { "aria-hidden": "true" } } ),
          Dom.el( "span" , { class: "folder-name" , text: folder ? folder.name : T( "home.unfiled" ) } ),
          Dom.el( "span" , { class: "folder-count" , text: I18n.fill( "home.folder_count" , { n: folder ? this.boardsUnder( folder.id ) : boards.length } ) } ),
        ],
      } );
      toggle.addEventListener( "click" , () => {
        const open = toggle.getAttribute( "aria-expanded" ) !== "true";
        this.setCollapsed( folderId , !open );
        this.showFolder( section , open );
      } );
      head.appendChild( toggle );
      if ( folder ) {
        head.appendChild( Ui.iconButton( "three-dots" , T( "home.folder_menu" ) , () => this.editFolder( folder ) ) );
      }
      section.appendChild( head );
      body.hidden = closed;
    }
    body.appendChild( grid );
    if ( folder ) {
      const children = Dom.el( "div" , { class: "folder-children" } );
      this.childFolders( folder.id ).forEach( ( child ) => children.appendChild( this.section( child , this.boardsIn( child.id ) ) ) );
      body.appendChild( children );
    }
    section.appendChild( body );
    return section;
  },

  // How many folders sit at this one's level, itself included.
  siblingCount( folder ) {
    return this.folders.filter( function ( other ) { return ( other.parent_id || 0 ) === ( folder.parent_id || 0 ); } ).length;
  },

  showFolder( section , open ) {
    const toggle = section.querySelector( ":scope > .folder-head .folder-toggle" );
    if ( !toggle ) return;
    toggle.setAttribute( "aria-expanded" , open ? "true" : "false" );
    Dom.text( toggle.querySelector( ".folder-symbol" ) , T( open ? "home.open_symbol" : "home.closed_symbol" ) );
    section.querySelector( ":scope > .folder-body" ).hidden = !open;
  },

  // menu false: no options button (boards shared with me, in their own grid).
  tile( board , menu ) {
    const wrap = Dom.el( "div" , { class: "board-tile-wrap" , attrs: { "data-board-id": board.id } } );
    const tile = Dom.el( "a" , { class: "board-tile color-" + ( board.color || "none" ) , attrs: { href: "/b/" + ( board.slug || board.id ) } } );
    const title = Dom.el( "span" , { class: "board-tile-title" } );
    Markdown.renderInline( board.title , title );
    tile.appendChild( title );
    wrap.appendChild( tile );
    if ( menu === false ) return wrap;
    wrap.appendChild( Ui.iconButton( "three-dots" , T( "home.board_menu" ) , ( event ) => ContextMenu.open( event , this.boardMenu( board ) ) , "tile-menu" ) );
    return wrap;
  },

  // The tile's options. Filing and deleting are for my own boards; any board
  // I can open, I can copy into one of my own.
  boardMenu( board ) {
    const mine = board.role === "owner";
    const items = [];
    if ( mine ) items.push( { key: "home.move_menu" , action: () => this.moveBoard( board ) } );
    items.push( { key: "home.duplicate" , action: () => this.duplicateBoard( board ) } );
    if ( mine ) items.push( { key: "home.delete_board" , danger: true , action: () => this.deleteBoard( board ) } );
    return items;
  },

  // The copy stays here on the home page, beside the original.
  async duplicateBoard( board ) {
    try {
      await Api.post( "/api/boards/" + board.id + "/duplicate" , { title: I18n.fill( "home.duplicate_title" , { name: board.title.slice( 0 , 110 ) } ) } );
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    Ui.toast( T( "home.duplicated" ) , "ok" );
    await this.refresh();
  },

  // For good, for everyone on it: asked first.
  async deleteBoard( board ) {
    if ( !window.confirm( I18n.fill( "settings.delete_board_confirm" , { name: board.title } ) ) ) return;
    try {
      await Api.post( "/api/boards/" + board.id + "/delete" );
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    Ui.toast( T( "home.deleted" ) , "ok" );
    await this.refresh();
  },

  // Tiles drag between folders. Order inside a folder is most recently
  // changed first, so a drop only ever changes the folder. Folders drag by
  // their grip, among the others at their level; Unfiled stays last. Each
  // level is its own list, and a list only picks up its direct children, so
  // the lists nest without getting in each other's way.
  bindDragging( root ) {
    if ( typeof Sortable === "undefined" ) return;
    // root outlives each render, so its last Sortable has to go first.
    const previous = Sortable.get( root );
    if ( previous ) previous.destroy();
    [ root ].concat( Dom.all( ".folder-children" , root ) ).forEach( ( level ) => {
      Sortable.create( level , {
        animation: 150,
        forceFallback: true,
        handle: ".folder-grip",
        draggable: ".folder:not([data-folder-id=\"0\"])",
        ghostClass: "drag-ghost",
        onMove: function ( event ) {
          // Nothing goes below Unfiled.
          return event.related.getAttribute( "data-folder-id" ) !== "0" || !event.willInsertAfter;
        },
        onEnd: ( event ) => {
          if ( event.oldIndex === event.newIndex ) return;
          // Every level as it reads top to bottom: sorting A to Z, the first
          // drag makes the whole order the person's own, so nothing else moves.
          const ids = Dom.all( ".folder" , root )
            .map( function ( section ) { return Number( section.getAttribute( "data-folder-id" ) ); } )
            .filter( Boolean );
          this.setOrder( ids );
        },
      } );
    } );
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

  async setOrder( ids ) {
    try {
      await Api.post( "/api/folders/order" , { ids: ids } );
      if ( !this.manual() ) await this.setSort( "manual" );
    } catch ( error ) {
      App.showError( error.message );
    }
    await this.refresh();
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
    let sharedShown = 0;
    Dom.all( ".board-tile-wrap" , Dom.get( "shared-boards" ) ).forEach( ( wrap ) => {
      const board = this.shared.find( function ( item ) { return item.id === Number( wrap.getAttribute( "data-board-id" ) ); } );
      const hit = !query || ( board && board.title.toLowerCase().indexOf( query ) >= 0 );
      wrap.hidden = !hit;
      if ( hit ) sharedShown += 1;
    } );
    Dom.all( "li" , Dom.get( "shared-cards" ) ).forEach( function ( row ) {
      const hit = !query || row.textContent.toLowerCase().indexOf( query ) >= 0;
      row.hidden = !hit;
      if ( hit ) sharedShown += 1;
    } );
    const closed = this.collapsed();
    let shown = 0;
    Dom.all( ":scope > .folder" , Dom.get( "board-folders" ) ).forEach( ( section ) => {
      shown += this.filterFolder( section , query , closed , false ).boards;
    } );
    Dom.show( Dom.get( "no-matches" ) , !!query && shown + sharedShown === 0 );
    // Search covers every board below; the recent strip would repeat hits.
    Dom.get( "recent-section" ).classList.toggle( "is-searching" , !!query );
  },

  // One folder and everything inside it, under a search. A folder whose name
  // matches shows all it holds, folders included, so inherited says one above
  // already did. A folder stays in view if anything inside it matched, which
  // is what opens the way down to a match. Answers how many boards it is
  // showing, and whether it is showing at all.
  filterFolder( section , query , closed , inherited ) {
    const folderId = Number( section.getAttribute( "data-folder-id" ) );
    const folder = this.folders.find( function ( item ) { return item.id === folderId; } );
    const named = !!query && !!folder && folder.name.toLowerCase().indexOf( query ) >= 0;
    const showAll = inherited || named;
    let boards = 0;
    Dom.all( ":scope > .folder-body > .boards-grid > .board-tile-wrap" , section ).forEach( ( wrap ) => {
      const board = this.boards.find( function ( item ) { return item.id === Number( wrap.getAttribute( "data-board-id" ) ); } );
      const hit = !query || showAll || ( board && board.title.toLowerCase().indexOf( query ) >= 0 );
      wrap.hidden = !hit;
      if ( hit ) boards += 1;
    } );
    let visible = boards > 0 || showAll;
    Dom.all( ":scope > .folder-body > .folder-children > .folder" , section ).forEach( ( child ) => {
      const found = this.filterFolder( child , query , closed , showAll );
      boards += found.boards;
      visible = visible || found.visible;
    } );
    section.hidden = !!query && !visible;
    this.showFolder( section , query ? true : closed.indexOf( folderId ) < 0 );
    return { boards: boards , visible: visible };
  },

  // The folders this one could go inside, as dropdown choices: not itself or
  // anything in it, and none that would leave it too deep. folder null is a
  // new one, with nothing inside it.
  parentChoices( folder ) {
    const height = folder ? this.folderHeight( folder.id ) : 1;
    const rows = this.folderTree( 0 , 1 , folder ? folder.id : null ).filter( ( row ) => row.depth + height <= this.folderMaxDepth );
    return [ { value: 0 , text: T( "home.folder_parent_none" ) } ].concat( rows.map( ( row ) => {
      return { value: row.folder.id , text: this.treeLabel( row ) };
    } ) );
  },

  // New folder (folder null), or an existing one's options: start a board
  // in it, rename / move it, delete it.
  editFolder( folder ) {
    Panel.open( "home.folder_title" , ( body ) => {
      if ( folder ) body.appendChild( this.newBoardSection( folder ) );
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "80" , required: "" , placeholder: T( "home.folder_name_placeholder" ) } } );
      name.value = folder ? folder.name : "";
      form.appendChild( Ui.field( "home.folder_name_label" , name ) );
      const parent = Ui.select( this.parentChoices( folder ) , folder ? folder.parent_id || 0 : 0 );
      form.appendChild( Ui.field( "home.folder_parent_label" , parent ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( folder ? "home.rename_folder" : "home.create_folder" ) , attrs: { type: "submit" } } ) );
      form.addEventListener( "submit" , async ( event ) => {
        event.preventDefault();
        try {
          const parentId = Number( parent.value );
          if ( folder ) {
            await Api.post( "/api/folders/" + folder.id , { name: name.value } );
            if ( parentId !== ( folder.parent_id || 0 ) ) await Api.post( "/api/folders/" + folder.id + "/parent" , { parent_id: parentId } );
          } else {
            await Api.post( "/api/folders" , { name: name.value , parent_id: parentId } );
          }
          Panel.close();
        } catch ( error ) {
          Ui.error( error );
          return;
        }
        await this.refresh();
      } );
      if ( folder ) {
        const section = Dom.el( "section" , { class: "panel-section" } );
        section.appendChild( Dom.el( "h3" , { text: T( "home.folder_edit_heading" ) } ) );
        section.appendChild( form );
        body.appendChild( section );
      } else {
        body.appendChild( form );
      }
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
      if ( !folder ) window.setTimeout( function () { name.focus(); } , 0 );
    } );
  },

  // The folder menu's quick start: a board that opens already filed here.
  newBoardSection( folder ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "home.folder_new_board_heading" ) } ) );
    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const title = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "120" , required: "" } } );
    form.appendChild( Ui.field( "home.board_title_label" , title ) );
    form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "home.create_button" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      try {
        await this.createBoard( title.value , folder.id );
      } catch ( error ) {
        Ui.error( error );
      }
    } );
    section.appendChild( form );
    window.setTimeout( function () { title.focus(); } , 0 );
    return section;
  },

  // The keyboard and touch way to file a board.
  moveBoard( board ) {
    Panel.open( "home.move_title" , ( body ) => {
      const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
      const select = Ui.select( [ { value: 0 , text: T( "home.move_none" ) } ].concat( this.folderTree( 0 , 1 , null ).map( ( row ) => {
        return { value: row.folder.id , text: this.treeLabel( row ) };
      } ) ) , board.folder_id || 0 );
      form.appendChild( Ui.field( "home.move_label" , select ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "home.move_button" ) , attrs: { type: "submit" } } ) );
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
    try {
      await this.createBoard( Dom.get( "new-board-title" ).value , 0 );
    } catch ( error ) {
      App.showError( error.message );
    }
  },

  // Makes a board with the default lists, files it in folderId (0 for none)
  // and opens it.
  async createBoard( title , folderId ) {
    const lists = T( "home.default_lists" ).split( "," ).map( function ( name ) { return name.trim(); } ).filter( Boolean );
    const created = await Api.post( "/api/boards" , { title: title.trim() , lists: lists } );
    if ( folderId ) await Api.post( "/api/boards/" + created.id + "/folder" , { folder_id: folderId } );
    window.location.href = "/b/" + ( created.slug || created.id );
  },
};
