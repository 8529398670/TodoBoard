// API keys: what has standing, non-browser access to this app. Your own are a
// card on the Account screen; an admin's view of every key is its own screen.
//
// Its own file with its own init(), per the convention in the frontend
// reference -- app.js stays the account/admin screen rather than growing a
// second one. app.js calls Keys.init( me ) once it knows who is signed in.
//
// Two places, and the difference matters: "my keys" is for every signed-in
// user, and the all-keys table is the admin's answer to "who and what can get
// in". A key created here carries a role from the same set users have, and can
// never be given more than the account that owns it.
//
// An admin can also make a key that acts as someone else, from that person's
// row on the Users screen (App.newKeyFor). The key says who made it in both
// tables, so the person it acts as can see that.
//
// A user's key can be limited to some boards, or kept off some: the picker
// under the create form, and Boards on a key's row.
const Keys = {
  me: null,
  // The accounts an admin could make a key for, by id (see mayExtend).
  targets: new Map(),
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
    // What a key can call: openapi.json in app.repo, else the server's own.
    if ( me.openapi_url ) {
      Dom.get( "key-spec-link" ).setAttribute( "href" , me.openapi_url );
      Dom.show( Dom.get( "key-spec-line" ) , true );
    }
    // Only an admin is offered the role choice. A non-admin has exactly one
    // option, and an "admin" entry the server would refuse is worse than no
    // choice at all.
    Dom.show( Dom.get( "key-role-field" ) , me.role === "admin" );
    Dom.get( "key-role" ).addEventListener( "change" , this.syncBoardsField.bind( this ) );
    Dom.onSubmit( Dom.get( "create-key-form" ) , this.onCreate.bind( this ) );
    Dom.get( "copy-key-button" ).addEventListener( "click" , this.onCopy.bind( this ) );
    Pager.sortable( "keys-table" , () => this.refresh() );
    Pager.search( "keys-table" , "keys-search" , () => this.refresh() );
    Pager.filter( "keys-table" , "keys-live-only" , { live: true } , () => this.refresh() );
    await this.refresh();
    this.resetCreatePicker();

    if ( me.role !== "admin" ) return;
    Dom.show( Dom.get( "admin-keys-panel" ) , true );
    Pager.sortable( "admin-keys-table" , () => this.refreshAll() );
    Pager.search( "admin-keys-table" , "admin-keys-search" , () => this.refreshAll() );
    Pager.filter( "admin-keys-table" , "admin-keys-live-only" , { live: true } , () => this.refreshAll() );
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
      Dom.get( "key-days" ).value = "";
      await this.refresh();
      this.resetCreatePicker();
      if ( this.me.role === "admin" ) await this.refreshAll();
    } catch ( error ) {
      App.showError( error.message );
    }
  },

  // Who an admin may make a key for: the server's rules, so nobody is offered
  // a choice it would refuse. Admin accounts are the owner's to choose, nobody
  // picks the owner, and a disabled account cannot hold a new key.
  mayMintFor( user ) {
    if ( user.id === this.me.id || user.owner || user.disabled ) return false;
    return user.role !== "admin" || this.me.is_owner;
  },

  // Every page of accounts, since an account missing here would lose its
  // Expiry button for no reason the admin could see.
  async loadTargets() {
    const users = [];
    let offset = 0;
    while ( offset !== null && offset !== undefined ) {
      const page = await Api.listUsers( offset );
      users.push( ...page.users );
      offset = page.next;
    }
    this.targets = new Map();
    for ( const user of users ) {
      if ( this.mayMintFor( user ) ) this.targets.set( String( user.id ) , user );
    }
  },

  // The name cell: the name edits in place, and a key someone else made says so
  // under it. rename is the endpoint, as in revokeButton; refreshOther is the
  // table this one is not. This table keeps its row, so the name being edited
  // does not vanish from under the keyboard; the other shows the same key.
  nameCell( key , rename , refreshOther ) {
    const cell = Dom.el( "td" , { children: [ Dom.editable( key.name , {
      label: I18n.get( "api_keys.table_name" ),
      maxLength: 80,
      save: async function ( name ) {
        const previous = key.name;
        await rename( key.id , name , previous );
        key.name = name;
        Undo.record( {
          label: I18n.get( "undo.rename_key" ),
          toast: true,
          undo: async () => { await rename( key.id , previous , name ); await Keys.refreshBoth(); },
          redo: async () => { await rename( key.id , name , previous ); await Keys.refreshBoth(); },
        } );
        refreshOther().catch( ( error ) => App.showError( error.message ) );
      },
    } ) ] } );
    cell.appendChild( Dom.el( "div" , {
      class: "small text-body-secondary",
      children: [ Dom.el( "span" , { text: I18n.get( "api_keys.created" ) + " " } ) , Dom.when( key.created_at ) ],
    } ) );
    if ( key.created_by ) {
      cell.appendChild( Dom.el( "div" , {
        class: "small text-body-secondary",
        text: I18n.fill( "api_keys.made_by" , { name: key.created_by_name || I18n.get( "api_keys.made_by_admin" ) } ),
      } ) );
    }
    return cell;
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
  // Revoking keeps the row, marked dead; deleting takes it off the list.
  // Revoke is only offered on a live key, delete on every key.
  //
  // Delete is undoable and revoke is not. The server only marks a deleted
  // key, and can put it back for a few minutes, so delete asks nothing and
  // offers Undo instead. Revoking says "stop trusting this" -- the leaked-key
  // button -- and an undo would quietly trust it again, so it asks first and
  // stays done.
  revokeButton( key , revoke ) {
    if ( !key.live ) return null;
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-danger",
      text: I18n.get( "api_keys.revoke_button" ),
      on: { click: async function () {
        // An empty confirm message means "do not ask", following the same
        // language.yaml rule as every other string here.
        const question = I18n.get( "api_keys.revoke_confirm" );
        if ( !( await Dom.confirm( question ) ) ) return;
        try {
          await revoke( key.id );
          await Keys.refresh();
          if ( Keys.me.role === "admin" ) await Keys.refreshAll();
        } catch ( error ) { App.showError( error.message ); }
      } },
    } );
  },

  // Changing when a key expires. Only on a live key, since the server will not
  // revive a dead one, and with no undo: it counts from today, so there is no
  // exact old date to go back to, and the dialog already asked.
  expiryButton( key , setExpiry ) {
    const text = I18n.get( "api_keys.expiry_button" );
    if ( !key.live || text === "" ) return null;
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-secondary",
      text: text,
      on: { click: async function () {
        const saved = await Dom.form( {
          title: I18n.get( "api_keys.expiry_heading" ).replace( "{name}" , () => key.name ),
          fields: [ {
            name: "days",
            label: I18n.get( "api_keys.days_label" ),
            type: "number",
            min: 0,
            max: 3650,
            required: true,
            help: I18n.get( "api_keys.expiry_help" ),
          } ],
          save: ( values ) => setExpiry( key.id , values.days ),
        } );
        if ( saved === null ) return;
        try {
          await Keys.refreshBoth();
        } catch ( error ) { App.showError( error.message ); }
        Dom.toast( I18n.get( "api_keys.expiry_changed" ) , "ok" );
      } },
    } );
  },

  // Whether the server would let this admin change a key's expiry: their own, or
  // one they could have minted, which is the same rule. refreshAll loads the
  // accounts this looks at first.
  mayExtend( key ) {
    return key.user_id === this.me.id || this.targets.has( String( key.user_id ) );
  },

  deleteButton( key , remove , restore ) {
    return Dom.el( "button" , {
      class: "btn btn-sm btn-outline-danger",
      text: I18n.get( "api_keys.delete_button" ),
      on: { click: async function () {
        try {
          await remove( key.id );
          await Keys.refreshBoth();
        } catch ( error ) {
          App.showError( error.message );
          return;
        }
        Undo.record( {
          label: I18n.get( "undo.deleted_key" ),
          toast: true,
          undo: async () => { await restore( key.id ); await Keys.refreshBoth(); },
          redo: async () => { await remove( key.id ); await Keys.refreshBoth(); },
        } );
      } },
    } );
  },

  async refreshBoth() {
    await Keys.refresh();
    if ( Keys.me.role === "admin" ) await Keys.refreshAll();
  },

  async refresh() {
    const body = Dom.get( "keys-table" ).querySelector( "tbody" );
    try {
      await this.loadBoards();
    } catch ( error ) {
      App.showError( error.message );
    }
    await Pager.load( "keys-table" , "keys" , ( offset ) => Api.listKeys( offset , Pager.params( "keys-table" ) ) , function ( key ) {
      // createElement + textContent throughout: a key's name is free text its
      // owner typed. See the note at the top of dom.js.
      const row = Dom.el( "tr" );
      [
        Keys.nameCell( key , ( id , name , expect ) => Api.renameKey( id , name , expect ) ,
          async () => { if ( Keys.me.role === "admin" ) await Keys.refreshAll(); } ),
        Dom.el( "td" , { text: key.role } ),
        Dom.el( "td" , { text: Keys.boardsText( key ) } ),
        Dom.el( "td" , { children: [ Keys.statusBadge( key ) ] } ),
        Dom.el( "td" , { children: [ Dom.when( key.last_used_at , I18n.get( "api_keys.never" ) ) ] } ),
        Dom.el( "td" , { children: [ Dom.when( key.expires_at , I18n.get( "api_keys.never" ) ) ] } ),
        Dom.el( "td" , { children: [
          Dom.rowActions( key.name , [
            Keys.boardsButton( key , row ),
            Keys.expiryButton( key , function ( id , days ) { return Api.setKeyExpiry( id , days ); } ),
            Keys.revokeButton( key , function ( id ) { return Api.revokeKey( id ); } ),
            Keys.deleteButton( key , function ( id ) { return Api.deleteKey( id ); } , function ( id ) { return Api.restoreKey( id ); } ),
          ] ),
        ] } ),
      ].forEach( function ( cell ) { row.appendChild( cell ); } );
      body.appendChild( row );
    } );
  },

  async refreshAll() {
    await this.loadTargets();
    const body = Dom.get( "admin-keys-table" ).querySelector( "tbody" );
    await Pager.load( "admin-keys-table" , "keys" , ( offset ) => Api.listAllKeys( offset , Pager.params( "admin-keys-table" ) ) , function ( key ) {
      body.appendChild( Dom.el( "tr" , { children: [
        Dom.el( "td" , { text: key.display_name || ( "#" + key.user_id ) } ),
        Keys.nameCell( key , ( id , name , expect ) => Api.renameAnyKey( id , name , expect ) , () => Keys.refresh() ),
        Dom.el( "td" , { text: key.role } ),
        Dom.el( "td" , { text: Keys.boardCountText( key ) } ),
        Dom.el( "td" , { children: [ Keys.statusBadge( key ) ] } ),
        Dom.el( "td" , { children: [ Dom.when( key.last_used_at , I18n.get( "api_keys.never" ) ) ] } ),
        Dom.el( "td" , { children: [ Dom.when( key.expires_at , I18n.get( "api_keys.never" ) ) ] } ),
        Dom.el( "td" , { children: [
          Dom.rowActions( key.name , [
            Keys.mayExtend( key ) ? Keys.expiryButton( key , function ( id , days ) { return Api.setAnyKeyExpiry( id , days ); } ) : null,
            Keys.revokeButton( key , function ( id ) { return Api.revokeAnyKey( id ); } ),
            Keys.deleteButton( key , function ( id ) { return Api.deleteAnyKey( id ); } , function ( id ) { return Api.restoreAnyKey( id ); } ),
          ] ),
        ] } ),
      ] } ) );
    } );
  },
};
