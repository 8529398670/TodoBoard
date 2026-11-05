// The API keys screen: what has standing, non-browser access to this app.
//
// Its own file with its own init(), per the convention in the frontend
// reference -- app.js stays the account/admin screen rather than growing a
// second one. app.js calls Keys.init( me ) once it knows who is signed in.
//
// Two sections, and the difference matters: "my keys" is for every signed-in
// user, and the all-keys table is the admin's answer to "who and what can get
// in". A key created here carries a role from the same set users have, and can
// never be given more than the account that owns it.
const Keys = {
  me: null,
  // The boards a key can be limited to: [ { id , title } ], by title.
  boards: [],
  createPicker: null,

  async init( me ) {
    this.me = me;
    // The server can switch the whole feature off (auth.api_keys: false in config.yaml). When it
    // has, the endpoints are gone, so showing the panel would only produce a
    // form that 404s.
    if ( me.api_keys_enabled === false ) return;

    Dom.show( Dom.get( "api-keys-panel" ) , true );
    // Only an admin is offered the role choice. A non-admin has exactly one
    // option, and an "admin" entry the server would refuse is worse than no
    // choice at all.
    Dom.show( Dom.get( "key-role-field" ) , me.role === "admin" );
    Dom.get( "key-role" ).addEventListener( "change" , this.syncBoardsField.bind( this ) );
    Dom.get( "create-key-form" ).addEventListener( "submit" , this.onCreate.bind( this ) );
    Dom.get( "copy-key-button" ).addEventListener( "click" , this.onCopy.bind( this ) );
    await this.refresh();
    this.resetCreatePicker();

    if ( me.role !== "admin" ) return;
    Dom.show( Dom.get( "admin-keys-panel" ) , true );
    await this.refreshAll();
  },

  // Boards reached through one shared card count too: a key can be kept off
  // those, or held to them, the same as any other.
  async loadBoards() {
    const found = await Api.listBoards();
    const titles = new Map();
    found.boards.forEach( function ( board ) { titles.set( board.id , board.title ); } );
    found.shared_cards.forEach( function ( card ) {
      if ( !titles.has( card.board_id ) ) titles.set( card.board_id , card.board_title );
    } );
    this.boards = Array.from( titles , function ( entry ) {
      return { id: entry[ 0 ] , title: entry[ 1 ] };
    } ).sort( function ( a , b ) { return a.title.localeCompare( b.title ); } );
  },

  // One picker for creating a key and for changing one. value() is what the
  // server wants; ids a key holds for a board that has since gone are dropped
  // there, since nothing on screen shows them.
  picker( access , ids ) {
    const chosen = new Set( ids );
    const select = Dom.el( "select" , {
      class: "form-select key-boards-mode",
      attrs: { "aria-label": I18n.get( "api_keys.boards_label" ) },
      children: [
        Dom.el( "option" , { text: I18n.get( "api_keys.boards_all" ) , attrs: { value: "" } } ),
        Dom.el( "option" , { text: I18n.get( "api_keys.boards_only" ) , attrs: { value: "only" } } ),
        Dom.el( "option" , { text: I18n.get( "api_keys.boards_except" ) , attrs: { value: "except" } } ),
      ],
    } );
    select.value = access;
    const list = Dom.el( "div" , { class: "key-board-list" } );

    const render = function () {
      Dom.clear( list );
      Dom.show( list , select.value !== "" );
      if ( Keys.boards.length === 0 ) {
        list.appendChild( Dom.el( "p" , { class: "text-body-secondary mb-0" , text: I18n.get( "api_keys.boards_none" ) } ) );
        return;
      }
      Keys.boards.forEach( function ( board ) {
        const box = Dom.el( "input" , {
          class: "form-check-input",
          attrs: { type: "checkbox" },
          on: { change: function () {
            if ( box.checked ) chosen.add( board.id );
            else chosen.delete( board.id );
          } },
        } );
        box.checked = chosen.has( board.id );
        list.appendChild( Dom.el( "label" , { class: "form-check" , children: [
          box,
          Dom.el( "span" , { class: "form-check-label" , text: board.title } ),
        ] } ) );
      } );
    };

    // Reloaded on opening so a board made since this page loaded is on offer.
    select.addEventListener( "change" , async function () {
      if ( select.value !== "" ) {
        try { await Keys.loadBoards(); } catch ( error ) { App.showError( error.message ); }
      }
      render();
    } );
    render();

    return {
      element: Dom.el( "div" , { class: "key-boards" , children: [ select , list ] } ),
      value() {
        if ( select.value === "" ) return { access: "" , ids: [] };
        const ids = Keys.boards.map( function ( board ) { return board.id; } ).filter( function ( id ) { return chosen.has( id ); } );
        return { access: select.value , ids: ids };
      },
    };
  },

  resetCreatePicker() {
    this.createPicker = this.picker( "" , [] );
    Dom.get( "key-boards-picker" ).replaceChildren( this.createPicker.element );
    this.syncBoardsField();
  },

  // An admin key cannot be limited to boards, so it is not offered.
  syncBoardsField() {
    Dom.show( Dom.get( "key-boards-field" ) , this.chosenRole() === "user" );
  },

  chosenRole() {
    return this.me.role === "admin" ? Dom.get( "key-role" ).value : "user";
  },

  // A key is shown once, on creation, and is unrecoverable afterwards -- the
  // server stores only a hash. Saying so next to the value is the whole reason
  // this box exists rather than a toast that disappears.
  showSecret( credential ) {
    Dom.text( Dom.get( "key-secret-value" ) , credential );
    Dom.show( Dom.get( "key-secret-box" ) , true );
  },

  async onCopy() {
    await Dom.copy( Dom.get( "key-secret-value" ) );
  },

  async onCreate( event ) {
    event.preventDefault();
    const name = Dom.get( "key-name" ).value.trim();
    const role = this.chosenRole();

    // An empty field means "no opinion", which the server reads as its
    // configured default. Sending 0 instead would ask for a key that never
    // expires, which is a different thing and has to be typed on purpose.
    const rawDays = Dom.get( "key-days" ).value.trim();
    const expiresInDays = rawDays === "" ? null : Number( rawDays );

    try {
      const boards = role === "user" ? this.createPicker.value() : { access: "" , ids: [] };
      const created = await Api.createKey( name , role , expiresInDays , boards.access , boards.ids );
      this.showSecret( created.credential );
      Dom.get( "key-name" ).value = "";
      await this.refresh();
      this.resetCreatePicker();
      if ( this.me.role === "admin" ) await this.refreshAll();
    } catch ( error ) {
      App.showError( error.message );
    }
  },

  // Dates are rendered in the viewer's own timezone, and a key with no expiry
  // says so rather than showing a blank cell.
  formatWhen( value ) {
    if ( !value ) return I18n.get( "api_keys.never" );
    const parsed = new Date( value );
    if ( isNaN( parsed.getTime() ) ) return I18n.get( "api_keys.never" );
    return parsed.toLocaleString();
  },

  statusText( key ) {
    if ( key.revoked ) return I18n.get( "api_keys.status_revoked" );
    if ( key.expired ) return I18n.get( "api_keys.status_expired" );
    return I18n.get( "api_keys.status_active" );
  },

  statusBadge( key ) {
    const tone = key.revoked ? "secondary" : key.expired ? "warning" : "success";
    return Dom.badge( Keys.statusText( key ) , tone );
  },

  // The owner's table names the boards. The admin table is built from
  // boardCountText instead: a key's boards are its owner's business.
  boardsText( key ) {
    if ( !key.board_access ) return I18n.get( "api_keys.boards_summary_all" );
    const titles = [];
    let others = 0;
    ( key.board_ids || [] ).forEach( function ( id ) {
      const board = Keys.boards.find( function ( item ) { return item.id === id; } );
      if ( board ) titles.push( board.title );
      else others += 1;
    } );
    let names = titles.join( ", " );
    if ( others > 0 ) names = I18n.fill( "api_keys.boards_summary_more" , { names: names , count: others } );
    return I18n.fill( key.board_access === "only" ? "api_keys.boards_summary_only" : "api_keys.boards_summary_except" , { names: names } );
  },

  boardCountText( key ) {
    if ( !key.board_access ) return I18n.get( "api_keys.boards_summary_all" );
    return I18n.fill( key.board_access === "only" ? "api_keys.boards_count_only" : "api_keys.boards_count_except" , { count: key.board_count } );
  },

  // Opens an editor in a row of its own under the key. Dead keys have nothing
  // to change, and an admin key cannot be limited.
  boardsButton( key , row ) {
    if ( !key.live || key.role !== "user" ) return null;
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-secondary",
      text: I18n.get( "api_keys.boards_button" ),
      attrs: { type: "button" },
      on: { click: function () {
        const next = row.nextElementSibling;
        if ( next && next.classList.contains( "key-boards-editor" ) ) {
          next.remove();
          return;
        }
        row.after( Keys.boardsEditor( key ) );
      } },
    } );
  },

  boardsEditor( key ) {
    const picker = this.picker( key.board_access , key.board_ids );
    const row = Dom.el( "tr" , { class: "key-boards-editor" } );
    row.appendChild( Dom.el( "td" , { attrs: { colspan: "7" } , children: [
      picker.element,
      Dom.el( "div" , { class: "row-actions mt-2" , children: [
        Dom.el( "button" , {
          class: "btn btn-sm btn-primary",
          text: I18n.get( "api_keys.boards_save" ),
          attrs: { type: "button" },
          on: { click: async function () {
            const boards = picker.value();
            try {
              await Api.setKeyBoards( key.id , boards.access , boards.ids );
              await Keys.refresh();
              if ( Keys.me.role === "admin" ) await Keys.refreshAll();
            } catch ( error ) { App.showError( error.message ); }
          } },
        } ),
        Dom.el( "button" , {
          class: "btn btn-sm btn-outline-secondary",
          text: I18n.get( "api_keys.boards_cancel" ),
          attrs: { type: "button" },
          on: { click: function () { row.remove(); } },
        } ),
      ] } ),
    ] } ) );
    return row;
  },

  // revokeButton and deleteButton are shared by both tables: the only
  // difference is which endpoint acts on the key, so that is the parameter.
  // Revoking keeps the row, marked dead; deleting removes it. Revoke is only
  // offered on a live key, delete on every key.
  revokeButton( key , revoke ) {
    if ( !key.live ) return null;
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-danger",
      text: I18n.get( "api_keys.revoke_button" ),
      on: { click: async function () {
        // An empty confirm message means "do not ask", following the same
        // language.yaml rule as every other string here.
        const question = I18n.get( "api_keys.revoke_confirm" );
        if ( question !== "" && !window.confirm( question ) ) return;
        try {
          await revoke( key.id );
          await Keys.refresh();
          if ( Keys.me.role === "admin" ) await Keys.refreshAll();
        } catch ( error ) { App.showError( error.message ); }
      } },
    } );
  },

  deleteButton( key , remove ) {
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-danger",
      text: I18n.get( "api_keys.delete_button" ),
      on: { click: async function () {
        const question = I18n.get( "api_keys.delete_confirm" );
        if ( question !== "" && !window.confirm( question ) ) return;
        try {
          await remove( key.id );
          await Keys.refresh();
          if ( Keys.me.role === "admin" ) await Keys.refreshAll();
        } catch ( error ) { App.showError( error.message ); }
      } },
    } );
  },

  async refresh() {
    const body = Dom.get( "keys-table" ).querySelector( "tbody" );
    let keys = [];
    try {
      await this.loadBoards();
      keys = await Api.listKeys();
    } catch ( error ) {
      App.showError( error.message );
      return;
    }

    Dom.clear( body );
    keys.forEach( function ( key ) {
      // createElement + textContent throughout: a key's name is free text its
      // owner typed. See the note at the top of dom.js.
      const row = Dom.el( "tr" );
      [
        Dom.el( "td" , { text: key.name } ),
        Dom.el( "td" , { text: key.role } ),
        Dom.el( "td" , { text: Keys.boardsText( key ) } ),
        Dom.el( "td" , { children: [ Keys.statusBadge( key ) ] } ),
        Dom.el( "td" , { text: Keys.formatWhen( key.last_used_at ) } ),
        Dom.el( "td" , { text: Keys.formatWhen( key.expires_at ) } ),
        Dom.el( "td" , { children: [
          Dom.el( "div" , { class: "row-actions" , children: [
            Keys.boardsButton( key , row ),
            Keys.revokeButton( key , function ( id ) { return Api.revokeKey( id ); } ),
            Keys.deleteButton( key , function ( id ) { return Api.deleteKey( id ); } ),
          ].filter( Boolean ) } ),
        ] } ),
      ].forEach( function ( cell ) { row.appendChild( cell ); } );
      body.appendChild( row );
    } );
  },

  async refreshAll() {
    const body = Dom.get( "admin-keys-table" ).querySelector( "tbody" );
    let keys = [];
    try {
      keys = await Api.listAllKeys();
    } catch ( error ) {
      App.showError( error.message );
      return;
    }

    Dom.clear( body );
    keys.forEach( function ( key ) {
      body.appendChild( Dom.el( "tr" , { children: [
        Dom.el( "td" , { text: key.display_name || ( "#" + key.user_id ) } ),
        Dom.el( "td" , { text: key.name } ),
        Dom.el( "td" , { text: key.role } ),
        Dom.el( "td" , { text: Keys.boardCountText( key ) } ),
        Dom.el( "td" , { children: [ Keys.statusBadge( key ) ] } ),
        Dom.el( "td" , { text: Keys.formatWhen( key.last_used_at ) } ),
        Dom.el( "td" , { children: [
          Dom.el( "div" , { class: "row-actions" , children: [
            Keys.revokeButton( key , function ( id ) { return Api.revokeAnyKey( id ); } ),
            Keys.deleteButton( key , function ( id ) { return Api.deleteAnyKey( id ); } ),
          ].filter( Boolean ) } ),
        ] } ),
      ] } ) );
    } );
  },
};
