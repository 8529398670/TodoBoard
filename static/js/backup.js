// Backups as JSON files: every board you are on (the Account page), and one
// column at a time (a column's menu). Importing always adds -- each board in
// a backup comes back as a new board you own, a column file's cards go to
// the end of the column -- so importing never overwrites anything.
//
// A whole backup is sent a piece at a time: each board's lists and labels
// first, then its cards, column by column, in batches small enough for the
// server's request limit. If something fails partway, what got in stays and
// the message says how far it got.
const Backup = {
  // Well under the server's 1 MiB body limit, with room for the wrapper.
  batchBytes: 700 * 1024,
  batchCards: 2000,

  // Cards split into batches by encoded size.
  batches( cards ) {
    const batches = [];
    let current = [];
    let size = 0;
    cards.forEach( ( card ) => {
      const bytes = JSON.stringify( card ).length;
      if ( current.length && ( size + bytes > this.batchBytes || current.length >= this.batchCards ) ) {
        batches.push( current );
        current = [];
        size = 0;
      }
      current.push( card );
      size += bytes;
    } );
    if ( current.length ) batches.push( current );
    return batches;
  },

  async readFile( file , format ) {
    let parsed = null;
    try {
      parsed = JSON.parse( await file.text() );
    } catch ( error ) {
      parsed = null;
    }
    if ( !parsed || parsed.format !== format || parsed.version !== 1 ) throw new Error( "backup_invalid" );
    return parsed;
  },

  save( data , filename ) {
    Ui.download( new Blob( [ JSON.stringify( data , null , 2 ) ] , { type: "application/json" } ) , filename );
  },

  // send( labels , cards ) posts one batch; resolves with how many went in.
  async sendCards( labels , cards , send ) {
    let count = 0;
    for ( const batch of this.batches( cards ) ) {
      const result = await send( labels , batch );
      count += result.count || 0;
    }
    return count;
  },

  // --- Account page ------------------------------------------------------

  initAccount( me ) {
    const panel = Dom.get( "backup-panel" );
    if ( !panel || !me.authenticated ) return;
    Dom.show( panel , true );
    Dom.get( "backup-download" ).addEventListener( "click" , () => this.downloadAll() );
    Dom.get( "backup-import" ).addEventListener( "click" , async () => {
      const file = await Ui.pickFile( ".json,application/json" );
      if ( file ) this.importAll( file );
    } );
  },

  status( text , kind ) {
    const line = Dom.get( "backup-status" );
    line.className = "notice " + ( kind === "error" ? "notice-error" : kind === "ok" ? "notice-ok" : "notice-link" );
    Dom.text( line , text );
    Dom.show( line , true );
  },

  async downloadAll() {
    try {
      const backup = await Api.request( "/api/backup" );
      const day = new Date().toISOString().slice( 0 , 10 );
      this.save( backup , "todoboard-backup-" + day + ".json" );
      this.status( I18n.fill( "backup.downloaded" , { n: backup.boards.length } ) , "ok" );
    } catch ( error ) {
      this.status( this.worded( error ) , "error" );
    }
  },

  async importAll( file ) {
    let backup = null;
    try {
      backup = await this.readFile( file , "todoboard-backup" );
    } catch ( error ) {
      this.status( this.worded( error ) , "error" );
      return;
    }
    const boards = backup.boards || [];
    if ( !boards.length ) {
      this.status( T( "backup.empty" ) , "error" );
      return;
    }
    if ( !window.confirm( I18n.fill( "backup.import_confirm" , { n: boards.length } ) ) ) return;
    let done = 0;
    for ( const board of boards ) {
      this.status( I18n.fill( "backup.importing" , { n: done + 1 , total: boards.length , name: board.title } ) );
      try {
        const lists = board.lists || [];
        const skeleton = Object.assign( {} , board , { lists: lists.map( function ( list ) { return { title: list.title , archived: !!list.archived }; } ) } );
        const made = await Api.post( "/api/backup/board" , { board: skeleton } );
        for ( let index = 0; index < lists.length; index += 1 ) {
          await this.sendCards( board.labels || [] , lists[ index ].cards || [] , function ( labels , cards ) {
            return Api.post( "/api/boards/" + made.id + "/lists/" + made.lists[ index ] + "/import" , { labels: labels , cards: cards } );
          } );
        }
      } catch ( error ) {
        this.status( I18n.fill( "backup.import_failed" , { name: board.title , done: done , total: boards.length , reason: this.worded( error ) } ) , "error" );
        return;
      }
      done += 1;
    }
    this.status( I18n.fill( "backup.imported" , { n: done } ) , "ok" );
  },

  worded( error ) {
    const code = error && error.message ? error.message : "";
    return ( code && T( "errors." + code ) ) || code || T( "errors.generic" );
  },

  // --- One column (board page) --------------------------------------------

  async exportColumn( ctx , list ) {
    try {
      const column = await ctx.client.exportColumn( list.id );
      this.save( column , Ui.slug( ctx.board.title + "-" + list.title ) + ".json" );
    } catch ( error ) {
      Ui.error( error );
    }
  },

  async importColumn( ctx , list ) {
    const file = await Ui.pickFile( ".json,application/json" );
    if ( !file ) return;
    let column = null;
    try {
      column = await this.readFile( file , "todoboard-column" );
    } catch ( error ) {
      Ui.error( error );
      return;
    }
    const cards = ( column.list && column.list.cards ) || [];
    if ( !cards.length ) {
      Ui.toast( T( "backup.column_empty" ) , "error" );
      return;
    }
    if ( !window.confirm( I18n.fill( "backup.column_confirm" , { n: cards.length , list: list.title } ) ) ) return;
    Panel.close();
    let count = 0;
    try {
      count = await this.sendCards( column.labels || [] , cards , function ( labels , batch ) {
        return ctx.client.importColumn( list.id , labels , batch );
      } );
      Ui.toast( I18n.fill( "backup.column_imported" , { n: count , list: list.title } ) , "ok" );
    } catch ( error ) {
      Ui.error( error );
    }
    await ctx.reload();
  },
};
