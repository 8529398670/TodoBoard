// A board's Notes and Purpose: two separate panels that work the same way,
// each with its own navbar icon and tile in Board settings. Notes are the
// board's reference shelf: links, documents, an index. The purpose says what
// the board is for and how to work on it, and is what an agent reads first;
// it is a section the board turns on in Board settings. Both are markdown.
// Members can edit them; a guest only reads.
const BoardNotes = {
  // Text being written, by field: { text , rev , conflict }. It lives here
  // rather than in the panel, so closing the panel -- Escape, a click outside
  // -- loses nothing and opening it again carries on. rev is what the text
  // was started from: the server refuses it if the board has moved on.
  drafts: {},

  // field is "notes" or "purpose". back returns to whatever opened it.
  open( field , back ) {
    this.back = back;
    this.show( field );
  },

  show( field ) {
    const ctx = BoardPage.context();
    Panel.open( "notes." + field + "_heading" , ( body ) => body.appendChild( this.section( ctx , field ) ) , this.back );
  },

  section( ctx , field ) {
    const canEdit = ctx.canEdit && !ctx.isGuest;
    const section = Dom.el( "section" , { class: "panel-section board-notes-section" } );
    const head = Dom.el( "div" , { class: "section-head board-notes-head" } );
    if ( field === "purpose" ) head.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "notes.purpose_hint" ) } ) );
    section.appendChild( head );

    const draft = this.drafts[ field ];
    if ( draft && canEdit ) {
      this.editor( field , draft ).forEach( function ( node ) { section.appendChild( node ); } );
      return section;
    }
    if ( canEdit ) head.appendChild( Ui.button( "notes.edit" , () => this.startEditing( field ) ) );
    const rendered = Dom.el( "div" , { class: "description" } );
    if ( ctx.board[ field ] ) {
      Markdown.render( ctx.board[ field ] , rendered );
    } else {
      rendered.appendChild( Dom.el( "p" , { class: "muted" , text: T( "notes." + field + "_empty" ) } ) );
    }
    section.appendChild( rendered );
    return section;
  },

  // From the board as it is now, not as it was when the panel opened: an
  // agent may have written to it in between.
  startEditing( field ) {
    const board = BoardPage.context().board;
    this.drafts[ field ] = { text: board[ field ] , rev: board[ field + "_rev" ] , conflict: false };
    this.show( field );
    const area = Dom.get( "panel-body" ).querySelector( ".description-input" );
    if ( area ) area.focus();
  },

  editor( field , draft ) {
    const nodes = [];
    if ( draft.conflict ) nodes.push( this.conflictBox( field , draft ) );
    const area = Dom.el( "textarea" , { class: "description-input form-control" , attrs: {
      rows: "10",
      "aria-label": T( "notes." + field + "_heading" ),
      placeholder: T( "notes." + field + "_placeholder" ),
    } } );
    area.value = draft.text;
    area.addEventListener( "input" , function () { draft.text = area.value; } );
    nodes.push( area );
    if ( !draft.conflict ) {
      nodes.push( Dom.el( "div" , { class: "row-actions board-notes-actions" , children: [
        Ui.button( "notes.save" , () => this.save( field ) , "small" ),
        Ui.button( "notes.cancel" , () => this.discard( field ) ),
      ] } ) );
    }
    return nodes;
  },

  conflictBox( field , draft ) {
    return Dom.el( "div" , { class: "alert alert-warning conflict" , children: [
      Dom.el( "p" , { text: T( "notes.conflict" ) } ),
      Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "notes.conflict_keep_mine" , () => {
          draft.rev = BoardPage.context().board[ field + "_rev" ];
          draft.conflict = false;
          this.save( field );
        } ),
        Ui.button( "notes.conflict_take_theirs" , () => this.discard( field ) ),
      ] } ),
    ] } );
  },

  discard( field ) {
    delete this.drafts[ field ];
    this.show( field );
  },

  async save( field ) {
    const draft = this.drafts[ field ];
    const ctx = BoardPage.context();
    const fields = {};
    fields[ field ] = draft.text;
    fields[ field + "_rev" ] = draft.rev;
    try {
      await ctx.client.updateBoard( fields );
    } catch ( error ) {
      if ( error.message === "board_text_stale" ) {
        draft.conflict = true;
        await ctx.reload();
        this.show( field );
      } else {
        Ui.error( error );
      }
      return;
    }
    delete this.drafts[ field ];
    await ctx.reload();
    Ui.toast( T( "notes.saved" ) , "ok" );
    this.show( field );
  },
};
