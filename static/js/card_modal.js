// The card editor: a dialog over the board with everything about one card.
// It re-renders whenever the board is re-read (another person's edit, or our
// own), except while someone is typing in it -- then the refresh waits for
// the field to lose focus, so nobody's half-typed sentence vanishes.
const CardModal = {
  cardId: null,
  ctx: null,
  editingDescription: false,
  draft: "",
  previewing: false,
  conflict: null,
  pendingRefresh: false,

  dialog() {
    return Dom.get( "card-dialog" );
  },

  bind() {
    const dialog = this.dialog();
    if ( dialog.dataset.bound ) return;
    dialog.dataset.bound = "1";
    closeOnBackdrop( dialog );
    dialog.addEventListener( "close" , () => {
      this.cardId = null;
      this.editingDescription = false;
      this.conflict = null;
      const url = new URL( window.location.href );
      url.searchParams.delete( "card" );
      window.history.replaceState( null , "" , url.pathname + url.search );
    } );
    dialog.addEventListener( "focusout" , () => {
      window.setTimeout( () => {
        if ( this.pendingRefresh && !this.typing() ) {
          this.pendingRefresh = false;
          this.render();
        }
      } , 0 );
    } );
  },

  open( ctx , cardId ) {
    this.bind();
    this.ctx = ctx;
    if ( !ctx.card( cardId ) ) {
      Ui.toast( T( "errors.card_not_found" ) );
      return;
    }
    this.cardId = cardId;
    this.editingDescription = false;
    this.conflict = null;
    const url = new URL( window.location.href );
    url.searchParams.set( "card" , String( cardId ) );
    window.history.replaceState( null , "" , url.pathname + url.search );
    this.render();
    if ( !this.dialog().open ) this.dialog().showModal();
  },

  close() {
    if ( this.dialog().open ) this.dialog().close();
  },

  typing() {
    const active = document.activeElement;
    return !!active && this.dialog().contains( active ) && ( active.tagName === "TEXTAREA" || ( active.tagName === "INPUT" && active.type === "text" ) );
  },

  refresh( ctx ) {
    if ( this.cardId === null ) return;
    this.ctx = ctx;
    if ( !ctx.card( this.cardId ) ) {
      this.close();
      Ui.toast( T( "card.removed_notice" ) );
      return;
    }
    if ( this.typing() ) {
      this.pendingRefresh = true;
      return;
    }
    this.render();
  },

  card() {
    return this.ctx.card( this.cardId );
  },

  // Patch the card, then re-read the board (which re-renders this dialog).
  save( fields ) {
    const ctx = this.ctx;
    const cardId = this.cardId;
    return ctx.mutate( function () { return ctx.client.patchCard( cardId , fields ); } );
  },

  render() {
    const card = this.card();
    if ( !card ) return;
    const ctx = this.ctx;
    const canEdit = card.access === "edit";
    const canMove = ctx.canEdit;
    const body = Dom.get( "card-dialog-body" );
    const scrollTop = body.scrollTop;
    Dom.clear( body );

    body.appendChild( this.header( card , canEdit ) );
    body.appendChild( this.placement( card , canEdit , canMove ) );
    const labels = this.labels( card , canEdit );
    if ( labels ) body.appendChild( labels );
    body.appendChild( this.dates( card , canEdit ) );
    const people = this.assignees( card , canEdit );
    if ( people ) body.appendChild( people );
    body.appendChild( this.description( card , canEdit ) );
    body.appendChild( this.checklist( card , canEdit ) );
    body.appendChild( this.actions( card , canEdit , canMove ) );
    body.appendChild( Dom.el( "p" , { class: "muted card-updated" , text: I18n.fill( "card.updated_by" , {
      name: CardBits.actorName( card.updated_by ),
      when: Dates.format( card.updated_at , false , true ),
    } ) } ) );
    body.scrollTop = scrollTop;
  },

  header( card , canEdit ) {
    const header = Dom.el( "header" , { class: "card-dialog-head" } );
    if ( canEdit ) {
      const title = Dom.el( "h2" , { class: "card-dialog-title" } );
      title.appendChild( InlineEdit.build( card.title , {
        label: T( "card.title_label" ),
        maxlength: 300,
        className: "title-inline",
        onSave: ( value ) => this.save( { title: value , rev: card.rev } ),
      } ) );
      header.appendChild( title );
    } else {
      const title = Dom.el( "h2" , { class: "card-dialog-title" } );
      Markdown.renderInline( card.title , title );
      header.appendChild( title );
    }
    header.appendChild( Dom.el( "button" , {
      class: "icon-button",
      text: T( "ui.close_symbol" ),
      attrs: { type: "button" , "aria-label": T( "ui.close" ) , title: T( "ui.close" ) },
      on: { click: () => this.close() },
    } ) );
    return header;
  },

  placement( card , canEdit , canMove ) {
    const row = Dom.el( "div" , { class: "card-placement row-actions" } );
    const lists = this.ctx.lists();
    if ( canMove ) {
      const select = Ui.select( lists.map( function ( list ) { return { value: list.id , text: list.title }; } ) , card.list_id );
      select.addEventListener( "change" , () => {
        const target = Number( select.value );
        const end = this.ctx.cardsIn( target );
        const position = end.length ? end[ end.length - 1 ].position + 1024 : 1024;
        this.save( { list_id: target , position: position } );
      } );
      row.appendChild( Ui.field( "card.list_label" , select ) );
    } else {
      const list = this.ctx.board.lists.find( function ( item ) { return item.id === card.list_id; } );
      if ( list ) {
        const where = Dom.el( "span" , { class: "muted" } );
        Markdown.renderInline( I18n.fill( "card.in_list" , { list: list.title } ) , where );
        row.appendChild( where );
      }
    }
    const done = Ui.checkbox( "card.done_label" , card.done , ( checked ) => this.save( { done: checked } ) );
    done.querySelector( "input" ).disabled = !canEdit;
    row.appendChild( done );
    return row;
  },

  labels( card , canEdit ) {
    const board = this.ctx.board;
    if ( !board.labels.length && !this.ctx.canEdit ) return null;
    const section = Dom.el( "section" , { class: "card-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "card.labels_heading" ) } ) );
    const chips = Dom.el( "div" , { class: "card-labels" } );
    board.labels.forEach( ( label ) => {
      const on = card.label_ids.indexOf( label.id ) >= 0;
      if ( !canEdit && !on ) return;
      const chip = Dom.el( "button" , {
        class: "label-chip color-" + label.color + ( on ? " selected" : " unselected" ),
        text: label.name || T( "card.unnamed_label" ),
        attrs: { type: "button" , "aria-pressed": on ? "true" : "false" },
      } );
      chip.disabled = !canEdit;
      chip.addEventListener( "click" , () => {
        const next = on ? card.label_ids.filter( function ( id ) { return id !== label.id; } ) : card.label_ids.concat( [ label.id ] );
        this.save( { label_ids: next } );
      } );
      chips.appendChild( chip );
    } );
    section.appendChild( chips );
    if ( this.ctx.canEdit ) {
      section.appendChild( Ui.button( "card.manage_labels" , () => BoardSettings.openLabels( this.ctx ) ) );
    }
    return section;
  },

  dates( card , canEdit ) {
    const section = Dom.el( "section" , { class: "card-section card-dates" } );
    section.appendChild( Dom.el( "h3" , { text: T( "card.dates_heading" ) } ) );
    if ( !canEdit ) {
      if ( !card.start_at && !card.due_at ) section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "card.no_dates" ) } ) );
      if ( card.start_at ) section.appendChild( Dom.el( "p" , { text: T( "card.start_label" ) + ": " + Dates.format( card.start_at , card.start_all_day , true ) } ) );
      if ( card.due_at ) {
        const state = Dates.dueState( card );
        section.appendChild( Dom.el( "p" , { class: "due-chip due-" + state , text: T( "card.due_state_" + state ) + " · " + Dates.format( card.due_at , card.due_all_day , true ) } ) );
      }
      return section;
    }
    const row = Dom.el( "div" , { class: "date-row" } );
    const start = this.datePair( card.start_at , card.start_all_day , "card.start_label" , "card.start_time_label" , ( value ) => this.save( { start_at: value } ) );
    const due = this.datePair( card.due_at , card.due_all_day , "card.due_label" , "card.due_time_label" , ( value ) => this.save( { due_at: value } ) );
    row.appendChild( start );
    row.appendChild( due );
    section.appendChild( row );

    if ( card.start_at && card.due_at ) section.appendChild( Dom.el( "p" , { class: "muted" , text: Dates.formatSpan( card ) } ) );
    if ( card.due_at ) {
      const state = Dates.dueState( card );
      section.appendChild( Dom.el( "p" , { class: "due-chip due-" + state , text: T( "card.due_state_" + state ) + " · " + Dates.format( card.due_at , card.due_all_day , true ) } ) );
    }
    if ( card.start_at || card.due_at ) {
      section.appendChild( Ui.button( "card.clear_dates" , () => this.save( { start_at: "" , due_at: "" } ) ) );
    }
    return section;
  },

  // A date input and an optional time input for one card date. No time
  // means an all-day date; with a time it is an instant on the person's
  // clock (see Dates.fromInputs), sent to the server as UTC.
  datePair( value , allDay , dateKey , timeKey , save ) {
    const day = Dom.el( "input" , { attrs: { type: "date" } } );
    const time = Dom.el( "input" , { attrs: { type: "time" , step: "300" } } );
    if ( value ) {
      day.value = allDay ? Dates.isoDay( new Date( value ) , true ) : Dates.isoDay( Dates.localDay( value , false ) );
      if ( !allDay ) time.value = Dates.timeInput( value );
    }
    const changed = function () {
      if ( !day.value ) save( "" );
      else if ( !time.value ) save( day.value );
      else save( Dates.fromInputs( day.value , time.value ).toISOString() );
    };
    day.addEventListener( "change" , changed );
    time.addEventListener( "change" , changed );
    return Dom.el( "div" , { class: "date-pair" , children: [ Ui.field( dateKey , day ) , Ui.field( timeKey , time ) ] } );
  },

  // People who can be assigned: everyone on the board, plus anyone this card
  // is shared with directly.
  assignable() {
    const ids = {};
    ( this.ctx.board.members || [] ).forEach( function ( member ) { ids[ member.user_id ] = member.display_name; } );
    const card = this.card();
    Object.keys( card.shared_with || {} ).forEach( ( id ) => { ids[ id ] = this.ctx.people[ id ] || id; } );
    return ids;
  },

  assignees( card , canEdit ) {
    const assignable = this.assignable();
    const editable = canEdit && Object.keys( assignable ).length > 0;
    if ( !card.assignees.length && !editable ) return null;
    const section = Dom.el( "section" , { class: "card-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "card.assignees_heading" ) } ) );
    const list = Dom.el( "div" , { class: "people" } );
    card.assignees.forEach( ( id ) => {
      const name = this.ctx.people[ id ] || "?";
      const chip = Dom.el( "span" , { class: "person" , children: [
        Dom.el( "span" , { class: "avatar" , text: CardBits.initials( name ) } ),
        Dom.el( "span" , { text: name } ),
      ] } );
      if ( editable ) {
        chip.appendChild( Dom.el( "button" , {
          class: "icon-button small",
          text: T( "ui.close_symbol" ),
          attrs: { type: "button" , "aria-label": I18n.fill( "card.unassign" , { name: name } ) },
          on: { click: () => this.save( { assignees: card.assignees.filter( function ( other ) { return other !== id; } ) } ) },
        } ) );
      }
      list.appendChild( chip );
    } );
    section.appendChild( list );
    if ( editable ) {
      const options = [ { value: "" , text: T( "card.assign_placeholder" ) } ];
      Object.keys( assignable ).forEach( function ( id ) {
        if ( card.assignees.indexOf( Number( id ) ) < 0 ) options.push( { value: id , text: assignable[ id ] } );
      } );
      if ( options.length > 1 ) {
        const select = Ui.select( options , "" );
        select.addEventListener( "change" , () => {
          if ( select.value ) this.save( { assignees: card.assignees.concat( [ Number( select.value ) ] ) } );
        } );
        section.appendChild( Ui.field( "card.assign_label" , select ) );
      }
    }
    return section;
  },

  description( card , canEdit ) {
    const section = Dom.el( "section" , { class: "card-section" } );
    const head = Dom.el( "div" , { class: "section-head" } );
    head.appendChild( Dom.el( "h3" , { text: T( "card.description_label" ) } ) );
    section.appendChild( head );

    if ( this.conflict ) section.appendChild( this.conflictBox( card ) );

    if ( !this.editingDescription ) {
      const startEditing = () => {
        this.editingDescription = true;
        this.previewing = false;
        this.draft = card.description;
        this.baseRev = card.rev;
        this.render();
        const area = this.dialog().querySelector( ".description-input" );
        if ( area ) area.focus();
      };
      if ( canEdit ) head.appendChild( Ui.button( "card.edit_description" , startEditing ) );
      const rendered = Dom.el( "div" , { class: "description" + ( canEdit ? " editable" : "" ) } );
      if ( CardBits.hasDescription( card ) ) {
        Markdown.render( card.description , rendered );
      } else if ( canEdit ) {
        rendered.appendChild( Dom.el( "p" , { class: "muted" , text: T( "card.no_description" ) } ) );
      }
      // Clicking the body edits it, except on a link inside it.
      if ( canEdit ) {
        rendered.setAttribute( "title" , T( "card.edit_description" ) );
        rendered.addEventListener( "click" , ( event ) => {
          if ( event.target.closest( "a" ) ) return;
          startEditing();
        } );
      }
      if ( rendered.childNodes.length ) section.appendChild( rendered );
      return section;
    }

    const tabs = Dom.el( "div" , { class: "tabs" , attrs: { role: "tablist" } } );
    const write = Ui.button( "card.write_tab" , () => { this.previewing = false; this.render(); } , "secondary small" + ( this.previewing ? "" : " active" ) );
    const preview = Ui.button( "card.preview_tab" , () => { this.previewing = true; this.render(); } , "secondary small" + ( this.previewing ? " active" : "" ) );
    tabs.appendChild( write );
    tabs.appendChild( preview );
    section.appendChild( tabs );

    if ( this.previewing ) {
      const rendered = Dom.el( "div" , { class: "description preview" } );
      Markdown.render( this.draft , rendered );
      section.appendChild( rendered );
    } else {
      const area = Dom.el( "textarea" , { class: "description-input" , attrs: { rows: "10" , "aria-label": T( "card.description_label" ) , placeholder: T( "card.description_placeholder" ) } } );
      area.value = this.draft;
      area.addEventListener( "input" , () => { this.draft = area.value; } );
      section.appendChild( area );
      section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "card.markdown_hint" ) } ) );
    }

    section.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Dom.el( "button" , { class: "small" , text: T( "board.save" ) , attrs: { type: "button" } , on: { click: () => this.saveDescription( this.baseRev ) } } ),
      Ui.button( "board.cancel" , () => {
        this.editingDescription = false;
        this.conflict = null;
        this.render();
      } ),
    ] } ) );
    return section;
  },

  // Saving text is checked against the revision the editor started from. If
  // someone else saved in between, the server answers 409 with their copy,
  // and the person chooses -- nothing is overwritten silently.
  async saveDescription( rev ) {
    const ctx = this.ctx;
    try {
      await ctx.client.patchCard( this.cardId , { description: this.draft , rev: rev } );
      this.editingDescription = false;
      this.conflict = null;
      await ctx.reload();
    } catch ( error ) {
      if ( error.status === 409 && error.payload && error.payload.card ) {
        this.conflict = error.payload.card;
        this.render();
        return;
      }
      Ui.error( error );
    }
  },

  conflictBox( card ) {
    const theirs = this.conflict;
    const box = Dom.el( "div" , { class: "notice notice-warn conflict" } );
    box.appendChild( Dom.el( "p" , { text: I18n.fill( "card.conflict" , { name: CardBits.actorName( theirs.updated_by ) } ) } ) );
    box.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Ui.button( "card.conflict_keep_mine" , () => this.saveDescription( theirs.rev ) ),
      Ui.button( "card.conflict_take_theirs" , async () => {
        this.conflict = null;
        this.editingDescription = false;
        await this.ctx.reload();
      } ),
    ] } ) );
    return box;
  },

  checklist( card , canEdit ) {
    const section = Dom.el( "section" , { class: "card-section" } );
    const done = card.checklist.filter( function ( item ) { return item.done; } ).length;
    const head = Dom.el( "div" , { class: "section-head" } );
    head.appendChild( Dom.el( "h3" , { text: T( "card.checklist_heading" ) } ) );
    if ( card.checklist.length ) head.appendChild( Dom.el( "span" , { class: "muted" , text: I18n.fill( "card.checklist_progress" , { done: done , total: card.checklist.length } ) } ) );
    section.appendChild( head );

    if ( card.checklist.length ) {
      const bar = Dom.el( "div" , { class: "progress" } );
      const fill = Dom.el( "div" , { class: "progress-fill" } );
      fill.style.width = Math.round( 100 * done / card.checklist.length ) + "%";
      bar.appendChild( fill );
      section.appendChild( bar );
    }

    const ctx = this.ctx;
    const cardId = this.cardId;
    const list = Dom.el( "ul" , { class: "checklist" } );
    card.checklist.forEach( ( item ) => {
      const row = Dom.el( "li" , { class: "checklist-item" + ( item.done ? " is-done" : "" ) , attrs: { "data-item-id": item.id , "data-position": item.position } } );
      if ( canEdit ) row.appendChild( Dom.el( "span" , { class: "todo-handle" , text: T( "todo.drag_symbol" ) , attrs: { "aria-hidden": "true" } } ) );
      const box = Dom.el( "input" , { attrs: { type: "checkbox" , "aria-label": item.text } } );
      box.checked = item.done;
      box.disabled = !canEdit;
      box.addEventListener( "change" , function () {
        ctx.mutate( function () { return ctx.client.updateItem( cardId , item.id , { done: box.checked } ); } );
      } );
      row.appendChild( box );
      if ( canEdit ) {
        row.appendChild( InlineEdit.build( item.text , {
          label: T( "card.checklist_item_label" ),
          maxlength: 500,
          className: "item-text",
          onSave: function ( value ) { ctx.mutate( function () { return ctx.client.updateItem( cardId , item.id , { text: value } ); } ); },
        } ) );
        row.appendChild( Dom.el( "button" , {
          class: "icon-button small",
          text: T( "ui.close_symbol" ),
          attrs: { type: "button" , "aria-label": T( "card.delete_item" ) , title: T( "card.delete_item" ) },
          on: { click: function () { ctx.mutate( function () { return ctx.client.deleteItem( cardId , item.id ); } ); } },
        } ) );
      } else {
        const text = Dom.el( "span" , { class: "item-text" } );
        Markdown.renderInline( item.text , text );
        row.appendChild( text );
      }
      list.appendChild( row );
    } );
    section.appendChild( list );

    if ( canEdit && typeof Sortable !== "undefined" && card.checklist.length > 1 ) {
      Sortable.create( list , {
        handle: ".todo-handle",
        animation: 150,
        forceFallback: true,
        onStart: function () { ctx.setDragging( true ); },
        onEnd: function ( event ) {
          ctx.setDragging( false );
          if ( event.oldIndex === event.newIndex ) return;
          const node = event.item;
          const neighbour = function ( other ) { return other ? Number( other.getAttribute( "data-position" ) ) : null; };
          const position = CardBits.positionBetween( neighbour( node.previousElementSibling ) , neighbour( node.nextElementSibling ) );
          ctx.mutate( function () { return ctx.client.updateItem( cardId , Number( node.getAttribute( "data-item-id" ) ) , { position: position } ); } );
        },
      } );
    }

    if ( canEdit ) section.appendChild( QuickAdd.build( "item-" + cardId , "card.add_item" , "card.item_placeholder" , function ( text ) {
      return ctx.mutate( function () { return ctx.client.addItem( cardId , text ); } );
    } ) );
    return section;
  },

  actions( card , canEdit , canMove ) {
    const ctx = this.ctx;
    const section = Dom.el( "section" , { class: "card-section card-actions" } );
    const row = Dom.el( "div" , { class: "row-actions" } );
    row.appendChild( Ui.button( "export.card_button" , () => CardExport.open( ctx , card ) ) );
    if ( !ctx.isGuest ) row.appendChild( Ui.button( "card.notifications_button" , () => Subscribe.openCard( ctx , card ) ) );
    if ( ctx.canManage ) row.appendChild( Ui.button( "share.card_button" , () => Sharing.openCard( ctx , card ) ) );
    if ( canMove ) {
      row.appendChild( Ui.button( card.archived ? "card.unarchive" : "card.archive" , () => {
        this.save( { archived: !card.archived } );
        if ( !card.archived ) this.close();
      } ) );
      row.appendChild( Ui.button( "card.delete" , async () => {
        if ( !window.confirm( I18n.fill( "card.delete_confirm" , { name: card.title } ) ) ) return;
        this.close();
        await ctx.mutate( function () { return ctx.client.deleteCard( card.id ); } );
      } , "secondary small danger" ) );
    }
    section.appendChild( row );
    return section;
  },
};
