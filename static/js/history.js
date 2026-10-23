// Undo and redo for the board page: Ctrl+Z / Ctrl+Y (⌘Z / ⇧⌘Z on a Mac) and
// the two arrows in the header.
//
// It sits on the board client rather than in the screens: attach() wraps the
// client's write methods once, so every change -- a drag, a tick, the card
// editor, the timeline, board settings -- is recorded without any of them
// knowing. Each recorded call works out its own inverse from the board as it
// was just before the call, and both directions are sent to the server like
// any other edit, so undo is shared truth, not a local illusion.
//
// Putting back something deleted makes it again, and the server gives the
// new copy a new id. Entries keep the ids they were recorded with and look
// them up through `ids` when they run, so older entries still find the card,
// list, label or checklist item they meant.
//
// Not undoable: who is on the board, sharing, subscriptions, deleting or
// leaving the board. History lasts as long as the tab.
const History = {
  undoStack: [],
  redoStack: [],
  limit: 100,
  replaying: false,
  busy: false,
  raw: {},
  ids: { card: {} , list: {} , label: {} , item: {} },
  getData: null,
  reload: null,
  onChange: null,

  methods: [
    "patchCard" , "createCard" , "deleteCard" ,
    "addItem" , "updateItem" , "deleteItem" ,
    "createList" , "updateList" , "deleteList" , "archiveCards" ,
    "createLabel" , "updateLabel" , "deleteLabel" ,
    "updateBoard",
  ],

  attach( client , getData , reload ) {
    this.getData = getData;
    this.reload = reload;
    this.methods.forEach( ( name ) => {
      const original = client[ name ].bind( client );
      this.raw[ name ] = original;
      client[ name ] = ( ...args ) => this.record( name , args );
    } );
  },

  canUndo() { return this.undoStack.length > 0 && !this.busy; },
  canRedo() { return this.redoStack.length > 0 && !this.busy; },

  changed() {
    if ( typeof this.onChange === "function" ) this.onChange();
  },

  // Run the real call; if it worked, remember how to take it back.
  async record( name , args ) {
    if ( this.replaying ) return this.raw[ name ]( ...args );
    const before = this.getData();
    const result = await this.raw[ name ]( ...args );
    let entry = null;
    try {
      entry = before ? this.inverse[ name ].call( this , before , args , result ) : null;
    } catch ( error ) {
      entry = null;
    }
    if ( entry ) {
      this.undoStack.push( entry );
      if ( this.undoStack.length > this.limit ) this.undoStack.shift();
      this.redoStack = [];
      this.changed();
    }
    return result;
  },

  undo() { return this.replay( this.undoStack , this.redoStack , "undo" ); },
  redo() { return this.replay( this.redoStack , this.undoStack , "redo" ); },

  async replay( from , to , direction ) {
    if ( this.busy || !from.length ) return;
    const entry = from.pop();
    this.busy = true;
    this.replaying = true;
    this.changed();
    try {
      await entry[ direction ]();
      to.push( entry );
    } catch ( error ) {
      // Whatever it pointed at is gone or out of reach; the entry goes too.
      Ui.toast( T( "errors.undo_failed" ) , "error" );
    } finally {
      this.replaying = false;
      this.busy = false;
    }
    await this.reload();
    this.changed();
  },

  // Ctrl+Z / ⌘Z undoes; Ctrl+Y, Ctrl+Shift+Z or ⇧⌘Z redoes. Not while typing
  // -- a text field keeps the browser's own undo.
  bindKeys() {
    document.addEventListener( "keydown" , ( event ) => {
      if ( !( event.ctrlKey || event.metaKey ) || event.altKey ) return;
      const target = event.target;
      if ( target && target.closest && target.closest( "input, textarea, select, [contenteditable]:not([contenteditable=\"false\"])" ) ) return;
      const key = String( event.key || "" ).toLowerCase();
      if ( key === "z" && !event.shiftKey ) {
        event.preventDefault();
        this.undo();
      } else if ( ( key === "z" && event.shiftKey ) || ( key === "y" && !event.shiftKey ) ) {
        event.preventDefault();
        this.redo();
      }
    } );
  },

  // ---- ids ------------------------------------------------------------

  // The id something has now, following every time it was made again.
  id( kind , id ) {
    const map = this.ids[ kind ];
    let current = id;
    for ( let hops = 0; map[ current ] !== undefined && hops < 1000; hops += 1 ) current = map[ current ];
    return current;
  },

  remap( kind , oldId , newId ) {
    const current = this.id( kind , oldId );
    if ( current !== newId ) this.ids[ kind ][ current ] = newId;
  },

  // ---- card fields ----------------------------------------------------

  // A card date as a patch sends it: "" for none, a plain day for all-day,
  // RFC 3339 otherwise.
  encodeDate( value , allDay ) {
    if ( !value ) return "";
    if ( allDay ) return Dates.isoDay( new Date( value ) , true );
    return new Date( value ).toISOString();
  },

  cardValue( card , key ) {
    if ( key === "start_at" ) return this.encodeDate( card.start_at , card.start_all_day );
    if ( key === "due_at" ) return this.encodeDate( card.due_at , card.due_all_day );
    const value = card[ key ];
    return Array.isArray( value ) ? value.slice() : value;
  },

  // A patch with today's ids for the lists and labels it names.
  mapCardFields( fields ) {
    const mapped = Object.assign( {} , fields );
    if ( mapped.list_id !== undefined ) mapped.list_id = this.id( "list" , mapped.list_id );
    if ( mapped.label_ids ) mapped.label_ids = mapped.label_ids.map( ( id ) => this.id( "label" , id ) );
    return mapped;
  },

  // The previous values of the keys a patch sends, or null when the patch
  // changed nothing worth undoing.
  previous( source , fields , keys , read ) {
    const back = {};
    const forward = {};
    keys.forEach( ( key ) => {
      if ( fields[ key ] === undefined ) return;
      back[ key ] = read( source , key );
      forward[ key ] = Array.isArray( fields[ key ] ) ? fields[ key ].slice() : fields[ key ];
    } );
    if ( !Object.keys( back ).length || JSON.stringify( back ) === JSON.stringify( forward ) ) return null;
    return { back: back , forward: forward };
  },

  // Make a deleted card again from a copy taken before it went: the card,
  // then everything on it, then its checklist. Returns the new id.
  async restoreCard( copy ) {
    const created = await this.raw.createCard( this.id( "list" , copy.list_id ) , copy.title , copy.position );
    await this.raw.patchCard( created.id , this.mapCardFields( {
      description: copy.description,
      label_ids: copy.label_ids,
      assignees: copy.assignees,
      start_at: this.encodeDate( copy.start_at , copy.start_all_day ),
      due_at: this.encodeDate( copy.due_at , copy.due_all_day ),
      done: copy.done,
      archived: copy.archived,
    } ) );
    for ( const item of copy.checklist ) {
      const newId = await this.addItemBack( created.id , item );
      this.remap( "item" , item.id , newId );
    }
    this.remap( "card" , copy.id , created.id );
    return created.id;
  },

  async addItemBack( cardId , item ) {
    const card = await this.raw.addItem( cardId , item.text );
    const newId = this.newestItem( card );
    if ( item.done ) await this.raw.updateItem( cardId , newId , { done: true } );
    if ( item.position !== undefined ) await this.raw.updateItem( cardId , newId , { position: item.position } );
    return newId;
  },

  newestItem( card ) {
    return card.checklist.reduce( function ( highest , item ) { return Math.max( highest , item.id ); } , 0 );
  },

  copy( value ) {
    return JSON.parse( JSON.stringify( value ) );
  },

  findCard( data , cardId ) {
    return data.cards.find( function ( card ) { return card.id === cardId; } ) || null;
  },

  // ---- inverses -------------------------------------------------------
  // Each takes ( board before the call , the call's arguments , its result )
  // and returns { undo , redo }, or null for nothing to record. `this` is
  // History.
  inverse: {
    patchCard( before , args ) {
      const cardId = args[ 0 ];
      const card = this.findCard( before , cardId );
      if ( !card ) return null;
      const keys = [ "title" , "description" , "list_id" , "position" , "label_ids" , "assignees" , "start_at" , "due_at" , "done" , "archived" ];
      const change = this.previous( card , args[ 1 ] , keys , ( source , key ) => this.cardValue( source , key ) );
      if ( !change ) return null;
      return {
        undo: () => this.raw.patchCard( this.id( "card" , cardId ) , this.mapCardFields( change.back ) ),
        redo: () => this.raw.patchCard( this.id( "card" , cardId ) , this.mapCardFields( change.forward ) ),
      };
    },

    createCard( before , args , result ) {
      const listId = args[ 0 ];
      const cardId = result.id;
      return {
        undo: () => this.raw.deleteCard( this.id( "card" , cardId ) ),
        redo: async () => {
          const created = await this.raw.createCard( this.id( "list" , listId ) , args[ 1 ] , args[ 2 ] );
          this.remap( "card" , cardId , created.id );
        },
      };
    },

    deleteCard( before , args ) {
      const card = this.findCard( before , args[ 0 ] );
      if ( !card ) return null;
      const copy = this.copy( card );
      return {
        undo: () => this.restoreCard( copy ),
        redo: () => this.raw.deleteCard( this.id( "card" , copy.id ) ),
      };
    },

    addItem( before , args , result ) {
      const cardId = args[ 0 ];
      const itemId = this.newestItem( result );
      return {
        undo: () => this.raw.deleteItem( this.id( "card" , cardId ) , this.id( "item" , itemId ) ),
        redo: async () => {
          const card = await this.raw.addItem( this.id( "card" , cardId ) , args[ 1 ] );
          this.remap( "item" , itemId , this.newestItem( card ) );
        },
      };
    },

    updateItem( before , args ) {
      const cardId = args[ 0 ];
      const itemId = args[ 1 ];
      const card = this.findCard( before , cardId );
      const item = card ? card.checklist.find( function ( entry ) { return entry.id === itemId; } ) : null;
      if ( !item ) return null;
      const change = this.previous( item , args[ 2 ] , [ "text" , "done" , "position" ] , function ( source , key ) { return source[ key ]; } );
      if ( !change ) return null;
      return {
        undo: () => this.raw.updateItem( this.id( "card" , cardId ) , this.id( "item" , itemId ) , change.back ),
        redo: () => this.raw.updateItem( this.id( "card" , cardId ) , this.id( "item" , itemId ) , change.forward ),
      };
    },

    deleteItem( before , args ) {
      const cardId = args[ 0 ];
      const card = this.findCard( before , cardId );
      const item = card ? card.checklist.find( function ( entry ) { return entry.id === args[ 1 ]; } ) : null;
      if ( !item ) return null;
      const copy = this.copy( item );
      return {
        undo: async () => {
          const newId = await this.addItemBack( this.id( "card" , cardId ) , copy );
          this.remap( "item" , copy.id , newId );
        },
        redo: () => this.raw.deleteItem( this.id( "card" , cardId ) , this.id( "item" , copy.id ) ),
      };
    },

    createList( before , args , result ) {
      const listId = result.id;
      return {
        undo: () => this.raw.deleteList( this.id( "list" , listId ) ),
        redo: async () => {
          const created = await this.raw.createList( args[ 0 ] , args[ 1 ] );
          this.remap( "list" , listId , created.id );
        },
      };
    },

    updateList( before , args ) {
      const listId = args[ 0 ];
      const list = before.board.lists.find( function ( item ) { return item.id === listId; } );
      if ( !list ) return null;
      const change = this.previous( list , args[ 1 ] , [ "title" , "position" , "archived" ] , function ( source , key ) { return source[ key ]; } );
      if ( !change ) return null;
      return {
        undo: () => this.raw.updateList( this.id( "list" , listId ) , change.back ),
        redo: () => this.raw.updateList( this.id( "list" , listId ) , change.forward ),
      };
    },

    // A list goes with its cards, so putting it back puts them back too.
    deleteList( before , args ) {
      const listId = args[ 0 ];
      const list = before.board.lists.find( function ( item ) { return item.id === listId; } );
      if ( !list ) return null;
      const copy = this.copy( list );
      const cards = this.copy( before.cards.filter( function ( card ) { return card.list_id === listId; } ) )
        .sort( function ( a , b ) { return a.position - b.position; } );
      return {
        undo: async () => {
          const created = await this.raw.createList( copy.title , copy.position );
          this.remap( "list" , listId , created.id );
          if ( copy.archived ) await this.raw.updateList( created.id , { archived: true } );
          for ( const card of cards ) await this.restoreCard( card );
        },
        redo: () => this.raw.deleteList( this.id( "list" , listId ) ),
      };
    },

    // Archiving a column touches exactly these cards; undo and redo touch
    // them again, and no others.
    archiveCards( before , args , result ) {
      const ids = ( result && result.archived ) || [];
      if ( !ids.length ) return null;
      const each = async ( archived ) => {
        for ( const id of ids ) await this.raw.patchCard( this.id( "card" , id ) , { archived: archived } );
      };
      return {
        undo: () => each( false ),
        redo: () => each( true ),
      };
    },

    createLabel( before , args , result ) {
      const labelId = result.id;
      return {
        undo: () => this.raw.deleteLabel( this.id( "label" , labelId ) ),
        redo: async () => {
          const created = await this.raw.createLabel( args[ 0 ] , args[ 1 ] );
          this.remap( "label" , labelId , created.id );
        },
      };
    },

    updateLabel( before , args ) {
      const labelId = args[ 0 ];
      const label = before.board.labels.find( function ( item ) { return item.id === labelId; } );
      if ( !label ) return null;
      const change = this.previous( label , args[ 1 ] , [ "name" , "color" ] , function ( source , key ) { return source[ key ]; } );
      if ( !change ) return null;
      return {
        undo: () => this.raw.updateLabel( this.id( "label" , labelId ) , change.back ),
        redo: () => this.raw.updateLabel( this.id( "label" , labelId ) , change.forward ),
      };
    },

    // Deleting a label takes it off every card; putting it back puts it back
    // on those cards.
    deleteLabel( before , args ) {
      const labelId = args[ 0 ];
      const label = before.board.labels.find( function ( item ) { return item.id === labelId; } );
      if ( !label ) return null;
      const copy = this.copy( label );
      const cardIds = before.cards.filter( function ( card ) { return card.label_ids.indexOf( labelId ) >= 0; } ).map( function ( card ) { return card.id; } );
      return {
        undo: async () => {
          const created = await this.raw.createLabel( copy.name , copy.color );
          this.remap( "label" , labelId , created.id );
          const now = this.getData();
          for ( const cardId of cardIds ) {
            const card = now ? this.findCard( now , this.id( "card" , cardId ) ) : null;
            if ( !card ) continue;
            const labels = card.label_ids.map( ( id ) => this.id( "label" , id ) ).filter( function ( id ) { return id !== created.id; } );
            labels.push( created.id );
            await this.raw.patchCard( card.id , { label_ids: labels } );
          }
        },
        redo: () => this.raw.deleteLabel( this.id( "label" , labelId ) ),
      };
    },

    // The board's own settings. The address is left alone: undoing a rename
    // should not take the board's link away from anyone who just got it.
    updateBoard( before , args ) {
      const fields = args[ 0 ];
      const change = this.previous( before.board , fields , [ "title" , "default_view" , "color" , "layout" , "numbering" ] , function ( source , key ) {
        return source[ key ] || "";
      } );
      if ( !change ) return null;
      if ( fields.list_titles ) change.forward.list_titles = fields.list_titles.slice();
      return {
        undo: () => this.raw.updateBoard( change.back ),
        redo: () => this.raw.updateBoard( change.forward ),
      };
    },
  },
};
