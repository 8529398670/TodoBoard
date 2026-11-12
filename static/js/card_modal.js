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
  // The title has the same write/preview editor as the description.
  editingTitle: false,
  titleDraft: "",
  titlePreviewing: false,
  titleRev: 0,
  conflict: null,
  titleConflict: null,
  pendingRefresh: false,
  // Title and description save themselves while they are typed (see
  // scheduleText): per field, the revision the next save is checked
  // against, what was last saved, and the save in flight, if any.
  text: { title: null , description: null },

  dialog() {
    return Dom.get( "card-dialog" );
  },

  bind() {
    const dialog = this.dialog();
    if ( dialog.dataset.bound ) return;
    dialog.dataset.bound = "1";
    closeOnBackdrop( dialog );
    // A picture pasted anywhere in the editor goes onto the card, unless
    // the add-item box or a text field is taking text from the same paste.
    dialog.addEventListener( "paste" , ( event ) => {
      const card = this.card();
      if ( !card || card.access !== "edit" || event.defaultPrevented ) return;
      const file = CardBits.pastedImage( event );
      if ( !file || CardBits.pasteHasText( event ) ) return;
      event.preventDefault();
      CardBits.attachImage( this.ctx , card.id , file );
    } );
    dialog.addEventListener( "close" , () => {
      this.flushText( "title" );
      this.flushText( "description" );
      this.cardId = null;
      this.editingDescription = false;
      this.editingTitle = false;
      this.conflict = null;
      this.titleConflict = null;
      if ( this.ctx && this.ctx.showPath && !this.ctx.isGuest ) {
        this.ctx.showPath();
        return;
      }
      const url = new URL( window.location.href );
      url.searchParams.delete( "card" );
      window.history.replaceState( null , "" , url.pathname + url.search );
    } );
    // There is no Done button: the title or description editor saves as it
    // is typed, and closes once someone clicks anywhere outside it. Seen on
    // the way down (capture), so the click still reaches what was clicked;
    // a press that began inside the editor (selecting text) does not count.
    let pressed = null;
    dialog.addEventListener( "pointerdown" , ( event ) => { pressed = event.target; } , true );
    dialog.addEventListener( "click" , ( event ) => {
      const outside = ( selector ) => !event.target.closest( selector ) && !( pressed && pressed.closest && pressed.closest( selector ) );
      if ( this.editingTitle && outside( ".card-dialog-title-wrap" ) ) this.finishTitle( false );
      if ( this.editingDescription && outside( ".card-description" ) ) this.finishDescription();
    } , true );
    dialog.addEventListener( "focusout" , () => {
      window.setTimeout( () => {
        if ( this.pendingRefresh && !this.typing() ) {
          this.pendingRefresh = false;
          this.render();
        }
      } , 0 );
    } );
    // Closing or leaving the tab sends what is still waiting (the client
    // sends small writes with keepalive, so they outlive the page).
    window.addEventListener( "pagehide" , () => {
      this.flushText( "title" );
      this.flushText( "description" );
    } );
    document.addEventListener( "visibilitychange" , () => {
      if ( !document.hidden ) return;
      this.flushText( "title" );
      this.flushText( "description" );
    } );
  },

  open( ctx , cardId ) {
    this.bind();
    if ( this.cardId !== null && this.cardId !== cardId ) {
      this.flushText( "title" );
      this.flushText( "description" );
    }
    this.ctx = ctx;
    if ( !ctx.card( cardId ) ) {
      Ui.toast( T( "errors.card_not_found" ) );
      return;
    }
    this.cardId = cardId;
    this.editingDescription = false;
    this.editingTitle = false;
    this.conflict = null;
    this.titleConflict = null;
    // A member's address bar becomes the card's own path; a guest's share
    // link has no room for one, so it carries ?card= as before.
    if ( ctx.showPath && !ctx.isGuest ) ctx.showPath();
    else {
      const url = new URL( window.location.href );
      url.searchParams.set( "card" , String( cardId ) );
      window.history.replaceState( null , "" , url.pathname + url.search );
    }
    this.render();
    if ( !this.dialog().open ) this.dialog().showModal();
    this.loadComments( cardId );
  },

  close() {
    if ( this.dialog().open ) this.dialog().close();
  },

  // An empty add-item box has nothing to lose, so it does not hold a refresh
  // back: after Enter adds an item the list redraws and the box reopens.
  typing() {
    const active = document.activeElement;
    if ( !active || !this.dialog().contains( active ) ) return false;
    if ( active.classList.contains( "quick-add-input" ) && active.value === "" ) return false;
    return active.tagName === "TEXTAREA" || ( active.tagName === "INPUT" && active.type === "text" );
  },

  refresh( ctx ) {
    if ( this.cardId === null ) return;
    this.ctx = ctx;
    if ( !ctx.card( this.cardId ) ) {
      this.close();
      Ui.toast( T( "card.removed_notice" ) );
      return;
    }
    // Someone else may have commented: the board moved, so ask again.
    this.loadComments( this.cardId );
    if ( this.typing() ) {
      this.pendingRefresh = true;
      return;
    }
    this.render();
  },

  // ---------------------------------------------------------------------
  // Comments. The thread is fetched when the card opens and whenever the
  // board moves, kept in commentState, and drawn in its own section, which
  // is swapped on its own so an answer arriving never wipes what someone
  // is typing elsewhere in the card. The box being typed in keeps its
  // words in commentDraft across redraws.

  async loadComments( cardId ) {
    const ctx = this.ctx;
    if ( !ctx || ctx.quick ) return;
    let answer;
    try {
      answer = await ctx.client.comments( cardId );
    } catch ( error ) {
      return;
    }
    if ( this.cardId !== cardId ) return;
    this.commentState = { cardId: cardId , list: answer.comments || [] , canComment: !!answer.can_comment };
    this.redrawComments();
  },

  redrawComments() {
    const old = Dom.get( "card-dialog-body" ).querySelector( ".card-comments" );
    const card = this.card();
    if ( !old || !card ) return;
    const focused = old.contains( document.activeElement ) && document.activeElement.classList.contains( "comment-input" );
    const fresh = this.commentsSection( card );
    old.replaceWith( fresh );
    if ( focused ) {
      const input = fresh.querySelector( ".comment-input" );
      if ( input ) {
        input.focus();
        input.setSelectionRange( input.value.length , input.value.length );
      }
    }
  },

  commentAuthor( by ) {
    return by === "guest:" ? T( "comments.anonymous" ) : CardBits.actorName( by );
  },

  commentsSection( card ) {
    const ctx = this.ctx;
    if ( ctx.quick ) return null;
    const state = this.commentState && this.commentState.cardId === card.id ? this.commentState : null;
    const section = Dom.el( "section" , { class: "card-section card-comments" } );
    section.appendChild( Dom.el( "h3" , { text: T( "comments.heading" ) } ) );
    if ( !state ) {
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "comments.loading" ) } ) );
      return section;
    }
    const list = Dom.el( "ol" , { class: "comment-list" } );
    state.list.forEach( ( comment ) => list.appendChild( this.commentItem( card , comment ) ) );
    if ( state.list.length ) section.appendChild( list );
    else if ( !state.canComment ) section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "comments.none" ) } ) );
    if ( state.canComment ) section.appendChild( this.commentForm( card ) );
    return section;
  },

  commentItem( card , comment ) {
    const ctx = this.ctx;
    const item = Dom.el( "li" , { class: "comment" } );
    const when = Dates.format( comment.created_at , false , true ) + ( comment.edited_at ? " · " + T( "comments.edited" ) : "" );
    item.appendChild( Dom.el( "div" , { class: "comment-head" , children: [
      Dom.el( "strong" , { text: this.commentAuthor( comment.by ) } ),
      Dom.el( "span" , { class: "muted small-print" , text: when } ),
    ] } ) );
    const text = Dom.el( "div" , { class: "comment-text markdown-body" } );
    Markdown.render( comment.text , text );
    item.appendChild( text );
    if ( !comment.can_edit && !comment.can_remove ) return item;
    const actions = Dom.el( "div" , { class: "comment-actions" } );
    if ( comment.can_edit ) actions.appendChild( Ui.button( "comments.edit" , () => {
      const area = Dom.el( "textarea" , { class: "form-control comment-input comment-edit" , attrs: { rows: "3" , maxlength: "4000" , "aria-label": T( "comments.edit" ) } } );
      area.value = comment.text;
      Ui.autoGrow( area );
      const save = Ui.button( "comments.save" , async () => {
        try {
          await ctx.client.editComment( card.id , comment.id , area.value );
          await this.loadComments( card.id );
        } catch ( error ) {
          Ui.error( error );
        }
      } , "small" );
      const cancel = Ui.button( "comments.cancel" , () => this.redrawComments() , "secondary small" );
      text.replaceWith( area );
      actions.replaceWith( Dom.el( "div" , { class: "comment-actions" , children: [ save , cancel ] } ) );
      area.focus();
    } , "secondary small" ) );
    if ( comment.can_remove ) actions.appendChild( Ui.button( "comments.remove" , async () => {
      if ( !window.confirm( T( "comments.remove_confirm" ) ) ) return;
      try {
        await ctx.client.removeComment( card.id , comment.id );
        await this.loadComments( card.id );
        ctx.reload();
      } catch ( error ) {
        Ui.error( error );
      }
    } , "secondary small danger" ) );
    item.appendChild( actions );
    return item;
  },

  // The box to post in. A guest with no name, on a board that does not take
  // anonymous comments, gives one here first; it is kept like the name an
  // edit link asks for.
  commentForm( card ) {
    const ctx = this.ctx;
    const form = Dom.el( "form" , { class: "comment-form" , attrs: { autocomplete: "off" } } );
    const needsName = ctx.isGuest && !ctx.client.guestName && !ctx.board.allow_anon_comments;
    let name = null;
    if ( needsName ) {
      name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , required: "required" , placeholder: T( "share.guest_name_placeholder" ) } } );
      form.appendChild( Ui.field( "comments.name_label" , name ) );
    }
    const area = Dom.el( "textarea" , { class: "form-control comment-input" , attrs: { rows: "2" , maxlength: "4000" , placeholder: T( "comments.placeholder" ) , "aria-label": T( "comments.placeholder" ) } } );
    area.value = this.commentDraft && this.commentDraft.cardId === card.id ? this.commentDraft.text : "";
    area.addEventListener( "input" , () => { this.commentDraft = { cardId: card.id , text: area.value }; } );
    Ui.autoGrow( area );
    // Enter with Ctrl or ⌘ posts, as in most chat boxes; Enter alone is a new line.
    area.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Enter" && ( event.ctrlKey || event.metaKey ) ) {
        event.preventDefault();
        form.requestSubmit();
      }
    } );
    form.appendChild( area );
    const post = Dom.el( "button" , { class: "btn btn-primary btn-sm" , text: T( "comments.post" ) , attrs: { type: "submit" } } );
    form.appendChild( Dom.el( "div" , { class: "comment-form-row" , children: [
      Dom.el( "span" , { class: "muted small-print" , text: T( "card.markdown_hint" ) } ),
      post,
    ] } ) );
    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      const text = area.value.trim();
      if ( !text ) return;
      if ( name ) {
        const given = name.value.trim().slice( 0 , 40 );
        if ( !given ) return;
        ctx.client.guestName = given;
        try { window.localStorage.setItem( "guest_name" , given ); } catch ( error ) { /* not remembered */ }
      }
      post.disabled = true;
      try {
        await ctx.client.addComment( card.id , text );
        this.commentDraft = null;
        area.value = "";
        area.blur();
        await this.loadComments( card.id );
        ctx.reload();
      } catch ( error ) {
        post.disabled = false;
        Ui.error( error );
      }
    } );
    return form;
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
    // A one-way copy follows its original in what a mirror syncs; the rest
    // (column, day, labels, people) is its own.
    const synced = canEdit && card.mirror !== "copy";
    const canMove = ctx.canEdit;
    const body = Dom.get( "card-dialog-body" );
    const scrollTop = body.scrollTop;
    QuickAdd.remember( body );
    Dom.clear( body );

    body.appendChild( this.header( card , synced ) );
    const product = this.productSection( card , synced );
    if ( product ) body.appendChild( product );
    // A card's picture sits at the top; with none, the offer to add one
    // waits below the checklist.
    const picture = this.imageSection( card , synced );
    if ( picture && card.image_at ) body.appendChild( picture );
    const price = this.priceSection( card , synced );
    if ( price ) body.appendChild( price );
    body.appendChild( this.description( card , synced ) );
    body.appendChild( this.checklist( card , synced ) );
    if ( picture && !card.image_at ) body.appendChild( picture );
    const day = this.daySection( card , canEdit );
    if ( day ) body.appendChild( day );
    body.appendChild( this.dates( card , synced ) );
    const people = this.assignees( card , canEdit );
    if ( people ) body.appendChild( people );
    const rank = this.rankSection( card );
    if ( rank ) body.appendChild( rank );
    body.appendChild( this.placement( card , canEdit , canMove ) );
    const labels = this.labels( card , canEdit );
    if ( labels ) body.appendChild( labels );
    const comments = this.commentsSection( card );
    if ( comments ) body.appendChild( comments );
    const mirrors = CardTransfer.section( ctx , card , canEdit );
    if ( mirrors ) body.appendChild( mirrors );
    body.appendChild( this.actions( card , canEdit , canMove ) );
    // A public link is not told who has been working on the card.
    if ( card.updated_by !== undefined ) body.appendChild( Dom.el( "p" , { class: "muted card-updated" , text: I18n.fill( "card.updated_by" , {
      name: CardBits.actorName( card.updated_by ),
      when: Dates.format( card.updated_at , false , true ),
    } ) } ) );
    body.scrollTop = scrollTop;
  },

  header( card , canEdit ) {
    const header = Dom.el( "header" , { class: "card-dialog-head" } );
    if ( canEdit ) {
      header.appendChild( this.titleEditor( card ) );
    } else {
      const title = Dom.el( "h2" , { class: "card-dialog-title" } );
      Markdown.renderInline( card.title , title );
      header.appendChild( title );
    }
    header.appendChild( Ui.closeButton( T( "ui.close" ) , () => this.close() , "mt-2" ) );
    return header;
  },

  // Where the card is: its list, and the same Doing / Done icons as its
  // face on the board.
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
    if ( !card.archived ) row.appendChild( CardBits.statusIcons( card , this.ctx ) );
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
      const chip = Tone.paint( Dom.el( "button" , {
        class: "label-chip " + Tone.className( label.color ) + ( on ? " selected" : " unselected" ),
        text: label.name || T( "card.unnamed_label" ),
        attrs: { type: "button" , "aria-pressed": on ? "true" : "false" },
      } ) , label.color );
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

  // On a calendar board, the day the card is filed under.
  daySection( card , canEdit ) {
    if ( !this.ctx.isCalendar || !this.ctx.isCalendar() ) return null;
    const section = Dom.el( "section" , { class: "card-section card-day" } );
    section.appendChild( Dom.el( "h3" , { text: T( "calendar.day_heading" ) } ) );
    if ( !canEdit ) {
      section.appendChild( Dom.el( "p" , { class: card.day ? "" : "muted" , text: card.day ? CalendarView.format( CalendarView.parse( card.day ) , { weekday: "long" , month: "long" , day: "numeric" , year: "numeric" } ) : T( "calendar.no_day" ) } ) );
      return section;
    }
    const input = Dom.el( "input" , { class: "form-control" , attrs: { type: "date" , "aria-label": T( "calendar.day_heading" ) } } );
    input.value = card.day || "";
    input.addEventListener( "change" , () => this.save( { day: input.value } ) );
    const row = Dom.el( "div" , { class: "row-actions" , children: [ input ] } );
    const today = CalendarView.today();
    if ( card.day !== today ) row.appendChild( Ui.button( "calendar.move_today" , () => this.save( { day: today } ) , "secondary small" ) );
    if ( card.day ) row.appendChild( Ui.button( "calendar.clear_day" , () => this.save( { day: "" } ) , "secondary small" ) );
    section.appendChild( row );
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
    const day = Dom.el( "input" , { class: "form-control" , attrs: { type: "date" } } );
    const time = Dom.el( "input" , { class: "form-control" , attrs: { type: "time" , step: "300" } } );
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
        chip.appendChild( Ui.closeButton( I18n.fill( "card.unassign" , { name: name } ) , () => this.save( { assignees: card.assignees.filter( function ( other ) { return other !== id; } ) } ) , "btn-close-small" ) );
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

  // The title: rendered until clicked, then a write/preview editor like the
  // description's. Enter saves; Shift+Enter adds a line, which the title
  // keeps wherever it is shown.
  titleEditor( card ) {
    const wrap = Dom.el( "div" , { class: "card-dialog-title-wrap" } );
    if ( this.titleConflict ) wrap.appendChild( this.titleConflictBox() );
    if ( !this.editingTitle ) {
      const title = Dom.el( "h2" , { class: "card-dialog-title editable" , attrs: { tabindex: "0" , role: "button" , "aria-label": T( "card.title_label" ) , title: T( "card.title_label" ) } } );
      Markdown.renderInline( card.title , title );
      const startEditing = () => {
        this.editingTitle = true;
        this.titlePreviewing = false;
        this.titleDraft = card.title;
        this.titleRev = card.rev;
        this.startText( "title" , card );
        this.render();
        const area = this.dialog().querySelector( ".title-input" );
        if ( area ) {
          area.focus();
          area.select();
        }
      };
      title.addEventListener( "click" , ( event ) => {
        if ( event.target.closest( "a" ) ) return;
        startEditing();
      } );
      title.addEventListener( "keydown" , ( event ) => {
        if ( event.key === "Enter" || event.key === " " ) {
          event.preventDefault();
          startEditing();
        }
      } );
      wrap.appendChild( title );
      return wrap;
    }

    const tabs = Dom.el( "div" , { class: "btn-group mb-2" , attrs: { role: "group" } } );
    tabs.appendChild( Ui.button( "card.write_tab" , () => { this.titlePreviewing = false; this.render(); } , "secondary small" + ( this.titlePreviewing ? "" : " active" ) ) );
    tabs.appendChild( Ui.button( "card.preview_tab" , () => { this.titlePreviewing = true; this.render(); } , "secondary small" + ( this.titlePreviewing ? " active" : "" ) ) );
    wrap.appendChild( tabs );
    if ( this.titlePreviewing ) {
      const rendered = Dom.el( "h2" , { class: "card-dialog-title preview" } );
      Markdown.renderInline( this.titleDraft , rendered );
      wrap.appendChild( rendered );
    } else {
      const area = Dom.el( "textarea" , { class: "title-input form-control" , attrs: { rows: "1" , "aria-label": T( "card.title_label" ) } } );
      area.value = this.titleDraft;
      area.addEventListener( "input" , () => {
        this.titleDraft = area.value;
        Ui.autoGrow( area );
        this.scheduleText( "title" );
      } );
      area.addEventListener( "blur" , () => { this.flushText( "title" ); } );
      area.addEventListener( "keydown" , ( event ) => {
        if ( event.key === "Enter" && !event.shiftKey && !event.isComposing ) {
          event.preventDefault();
          this.finishTitle( false );
        } else if ( event.key === "Escape" ) {
          event.preventDefault();
          event.stopPropagation();
          this.finishTitle( true );
        }
      } );
      wrap.appendChild( area );
      window.requestAnimationFrame( function () { Ui.autoGrow( area ); } );
      wrap.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "card.markdown_hint" ) } ) );
    }
    wrap.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ this.textStatus( "title" ) ] } ) );
    return wrap;
  },

  // Escape puts back the title editing started from; Enter, or clicking
  // anywhere else, keeps what is there. Either way it is already saved, or
  // about to be.
  async finishTitle( revert ) {
    if ( revert && this.text.title ) this.titleDraft = this.text.title.original;
    await this.flushText( "title" );
    if ( this.titleConflict || !this.editingTitle ) return;
    this.editingTitle = false;
    this.render();
  },

  async finishDescription() {
    await this.flushText( "description" );
    if ( this.conflict || !this.editingDescription ) return;
    this.editingDescription = false;
    this.render();
  },

  titleConflictBox() {
    const theirs = this.titleConflict;
    const box = Dom.el( "div" , { class: "alert alert-warning conflict" } );
    box.appendChild( Dom.el( "p" , { text: I18n.fill( "card.title_conflict" , { name: CardBits.actorName( theirs.updated_by ) } ) } ) );
    box.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Ui.button( "card.conflict_keep_mine" , () => this.keepMine( "title" , theirs ) ),
      Ui.button( "card.conflict_take_theirs" , async () => {
        this.titleConflict = null;
        this.editingTitle = false;
        this.text.title = null;
        await this.ctx.reload();
      } ),
    ] } ) );
    return box;
  },

  // What the card's shop link sells: photo, name, price, a line about it and
  // a Buy button. Filled in by the server a few seconds after the link
  // appears (Amazon anywhere, any shop on a shopping board or column); an
  // editor can look it up again or take it off. A card with a link but no
  // product offers to look it up by hand, on shopping boards and in
  // shopping columns only.
  productSection( card , canEdit ) {
    const product = card.product;
    const section = Dom.el( "section" , { class: "card-section card-product" } );
    const refresh = async ( button ) => {
      button.disabled = true;
      Dom.text( button , T( "product.looking_up" ) );
      try {
        const updated = await this.ctx.client.refreshProduct( card.id );
        Object.assign( card , updated );
        if ( updated.product && updated.product.status !== "ok" ) Ui.toast( T( "product.failed" ) , "error" );
      } catch ( error ) {
        Ui.error( error );
      }
      await this.ctx.reload();
    };
    if ( !product ) {
      if ( !canEdit || !this.ctx.isShopping( card ) || !/https?:\/\//i.test( card.title + " " + card.description ) ) return null;
      const button = Ui.button( "product.look_up" , () => refresh( button ) );
      section.appendChild( button );
      return section;
    }
    const link = CardBits.safeLink( product.url );
    if ( product.status === "ok" ) {
      const photo = CardBits.productImage( card , this.ctx , "card-product-photo" );
      if ( photo ) section.appendChild( photo );
      const info = Dom.el( "div" , { class: "card-product-info" } );
      if ( product.title ) info.appendChild( Dom.el( "p" , { class: "card-product-title" , text: product.title } ) );
      const shownPrice = CardBits.priceText( card );
      if ( shownPrice ) info.appendChild( Dom.el( "p" , { class: "card-product-price" , text: shownPrice } ) );
      if ( product.description ) info.appendChild( Dom.el( "p" , { class: "card-product-description muted" , text: product.description } ) );
      section.appendChild( info );
    } else {
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "product.failed" ) } ) );
    }
    const actions = Dom.el( "div" , { class: "row-actions" } );
    if ( link ) {
      actions.appendChild( Dom.el( "a" , {
        class: "btn btn-sm btn-primary",
        text: T( "product.buy" ),
        attrs: { href: link , target: "_blank" , rel: "noopener noreferrer nofollow" },
      } ) );
    }
    if ( canEdit ) {
      const again = Ui.button( "product.refresh" , () => refresh( again ) );
      actions.appendChild( again );
      actions.appendChild( Ui.button( "product.remove" , async () => {
        try { await this.ctx.client.removeProduct( card.id ); } catch ( error ) { Ui.error( error ); }
        await this.ctx.reload();
      } ) );
    }
    section.appendChild( actions );
    return section;
  },

  // The picture pasted onto the card: shown with Remove, or, for an editor,
  // a hint to paste one and a file picker for when there is no clipboard.
  imageSection( card , canEdit ) {
    const ctx = this.ctx;
    if ( !card.image_at && !canEdit ) return null;
    const section = Dom.el( "section" , { class: "card-section card-image" } );
    const photo = CardBits.cardImage( card , ctx , "card-image-photo" );
    if ( photo ) section.appendChild( photo );
    if ( !canEdit ) return section;
    const picker = Dom.el( "input" , { attrs: { type: "file" , accept: "image/*" , hidden: "" } } );
    picker.addEventListener( "change" , function () {
      if ( picker.files && picker.files[ 0 ] ) CardBits.attachImage( ctx , card.id , picker.files[ 0 ] );
    } );
    const actions = Dom.el( "div" , { class: "row-actions" } );
    if ( !card.image_at ) actions.appendChild( Dom.el( "span" , { class: "muted small-print" , text: T( "card.image_hint" ) } ) );
    actions.appendChild( Ui.button( card.image_at ? "card.image_replace" : "card.image_choose" , function () { picker.click(); } ) );
    if ( card.image_at ) actions.appendChild( Ui.button( "card.image_remove" , function () { ctx.mutate( function () { return ctx.client.removeImage( card.id ); } ); } ) );
    section.appendChild( picker );
    section.appendChild( actions );
    return section;
  },

  // A price typed by hand: one the shop page did not give, or a correction
  // to one it got wrong. Empty falls back to the page's. Offered on shopping
  // boards and columns; elsewhere only a card already holding one shows it,
  // so that price can still be changed or cleared.
  priceSection( card , canEdit ) {
    if ( !canEdit ) return null;
    if ( !card.price && !this.ctx.isShopping( card ) ) return null;
    const product = CardBits.product( card );
    const fetched = product && product.price ? product.price : "";
    const section = Dom.el( "section" , { class: "card-section card-price" } );
    const input = Dom.el( "input" , { class: "form-control form-control-sm card-price-input" , attrs: {
      type: "text" , maxlength: "40" , autocomplete: "off",
      placeholder: fetched ? I18n.fill( "card.price_placeholder_fetched" , { price: fetched } ) : T( "card.price_placeholder" ),
      "aria-label": T( "card.price_label" ),
    } } );
    input.value = card.price || "";
    const save = () => {
      const value = input.value.trim();
      if ( value === ( card.price || "" ) ) return;
      this.save( { price: value } );
    };
    input.addEventListener( "change" , save );
    input.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Enter" ) { event.preventDefault(); input.blur(); }
    } );
    const row = Dom.el( "label" , { class: "card-price-row" } );
    row.appendChild( Dom.el( "span" , { class: "card-price-label" , text: T( "card.price_label" ) } ) );
    row.appendChild( input );
    section.appendChild( row );
    if ( card.price && fetched && fetched !== card.price ) section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: I18n.fill( "card.price_overrides" , { price: fetched } ) } ) );
    return section;
  },

  // The card's place in the board's ranking, on a staged board: type a
  // number and the others make room, or clear it. A text box, not a number
  // one, so a refresh holds off while it is being typed in (see typing).
  // A finished or archived card holds no place until it is back.
  rankSection( card ) {
    const ctx = this.ctx;
    if ( !ctx.isStaged() ) return null;
    const place = card.rank_place || 0;
    const section = Dom.el( "section" , { class: "card-section card-price" } );
    if ( !ctx.canRank || card.done || card.archived ) {
      if ( !place ) return null;
      section.appendChild( Dom.el( "p" , { class: "muted" , text: I18n.fill( "board.rank_title" , { n: place } ) } ) );
      return section;
    }
    const input = Dom.el( "input" , { class: "form-control form-control-sm card-price-input" , attrs: {
      type: "text" , inputmode: "numeric" , maxlength: "5" , autocomplete: "off",
      placeholder: T( "card.rank_placeholder" ), "aria-label": T( "card.rank_label" ),
    } } );
    input.value = place ? String( place ) : "";
    input.addEventListener( "change" , () => {
      const text = input.value.trim();
      const wanted = text === "" ? 0 : parseInt( text , 10 );
      if ( isNaN( wanted ) || wanted < 0 ) {
        input.value = place ? String( place ) : "";
        return;
      }
      if ( wanted !== place ) this.save( { rank_place: wanted } );
    } );
    input.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Enter" ) { event.preventDefault(); input.blur(); }
    } );
    const row = Dom.el( "label" , { class: "card-price-row" } );
    row.appendChild( Dom.el( "span" , { class: "card-price-label" , text: T( "card.rank_label" ) } ) );
    row.appendChild( input );
    section.appendChild( row );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "card.rank_hint" ) } ) );
    return section;
  },

  description( card , canEdit ) {
    const section = Dom.el( "section" , { class: "card-section card-description" } );
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
        this.startText( "description" , card );
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

    const tabs = Dom.el( "div" , { class: "btn-group mb-2" , attrs: { role: "group" } } );
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
      const area = Dom.el( "textarea" , { class: "description-input form-control" , attrs: { rows: "10" , "aria-label": T( "card.description_label" ) , placeholder: T( "card.description_placeholder" ) } } );
      area.value = this.draft;
      area.addEventListener( "input" , () => {
        this.draft = area.value;
        this.scheduleText( "description" );
      } );
      area.addEventListener( "blur" , () => { this.flushText( "description" ); } );
      Ui.indentKeys( area );
      section.appendChild( area );
      section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "card.description_hint" ) } ) );
    }

    section.appendChild( Dom.el( "div" , { class: "row-actions" , children: [ this.textStatus( "description" ) ] } ) );
    return section;
  },

  // ---- saving text as it is typed ---------------------------------------
  // Each save is checked against the revision the last one left (or editing
  // started from). If the card changed in between, the server answers 409
  // with its copy; when that copy's text for this field is not what we last
  // saved, someone else rewrote it, and saving stops until the person
  // chooses -- nothing is overwritten silently.

  startText( field , card ) {
    const value = field === "title" ? card.title : card.description;
    this.text[ field ] = { cardId: card.id , ctx: this.ctx , rev: card.rev , original: value , saved: value , server: value , timer: null , inflight: null , again: false };
  },

  scheduleText( field ) {
    const state = this.text[ field ];
    if ( !state ) return;
    window.clearTimeout( state.timer );
    state.timer = window.setTimeout( () => { this.flushText( field ); } , 800 );
    this.showStatus( field , "card.autosave_pending" );
  },

  draftOf( field ) {
    return field === "title" ? this.titleDraft.trim() : this.draft;
  },

  conflictOf( field ) {
    return field === "title" ? this.titleConflict : this.conflict;
  },

  // Send what is typed now, if it differs from what was saved. One save at
  // a time: anything typed meanwhile goes in the next.
  async flushText( field ) {
    const state = this.text[ field ];
    if ( !state ) return;
    window.clearTimeout( state.timer );
    state.timer = null;
    if ( state.inflight ) {
      state.again = true;
      return state.inflight;
    }
    // Only while this card's editor holds the draft.
    if ( this.cardId !== state.cardId || this.conflictOf( field ) ) return;
    const value = this.draftOf( field );
    if ( value === state.saved || ( field === "title" && !value ) ) {
      this.showStatus( field , value === state.saved && value !== state.original ? "card.autosave_saved" : "" );
      return;
    }
    this.showStatus( field , "card.autosave_saving" );
    state.inflight = ( async () => {
      try {
        // The card's rev moves with any change to it -- our own save of the
        // other field, a label, a checklist tick. That is only a conflict
        // when this field's text itself changed since we last saved it;
        // otherwise take the newer rev and save again.
        let updated = null;
        for ( let tries = 0 ; !updated ; tries++ ) {
          const fields = { rev: state.rev };
          fields[ field ] = value;
          try {
            updated = await state.ctx.client.patchCard( state.cardId , fields );
          } catch ( error ) {
            const theirs = error.status === 409 && error.payload && error.payload.card;
            if ( !theirs || tries >= 3 || theirs[ field ] !== state.server ) throw error;
            state.rev = theirs.rev;
          }
        }
        const before = state.rev;
        state.saved = value;
        state.server = updated[ field ];
        state.rev = updated.rev;
        // The other field, if it is being edited from the same rev, is not
        // stale because of our own save.
        const other = this.text[ field === "title" ? "description" : "title" ];
        if ( other && other.cardId === state.cardId && other.rev === before ) other.rev = updated.rev;
        // Keep the board's copy current without re-reading it, which would
        // redraw the editor under the cursor. The next poll catches up.
        const card = state.ctx.card( state.cardId );
        if ( card ) Object.assign( card , updated );
        this.showStatus( field , "card.autosave_saved" );
      } catch ( error ) {
        if ( error.status === 409 && error.payload && error.payload.card ) {
          if ( field === "title" ) this.titleConflict = error.payload.card;
          else this.conflict = error.payload.card;
          if ( this.cardId === state.cardId ) this.render();
        } else {
          this.showStatus( field , "card.autosave_failed" );
          Ui.error( error );
        }
      } finally {
        state.inflight = null;
      }
      if ( state.again ) {
        state.again = false;
        await this.flushText( field );
      }
    } )();
    return state.inflight;
  },

  // "Keep mine" after a conflict: save over their revision.
  keepMine( field , theirs ) {
    const state = this.text[ field ];
    if ( field === "title" ) this.titleConflict = null;
    else this.conflict = null;
    if ( state ) {
      state.rev = theirs.rev;
      state.server = theirs[ field ];
      state.saved = null;
    }
    this.render();
    return this.flushText( field );
  },

  textStatus( field ) {
    return Dom.el( "span" , { class: "muted small-print autosave-status" , attrs: { "data-field": field , "aria-live": "polite" } } );
  },

  showStatus( field , key ) {
    const node = this.dialog().querySelector( ".autosave-status[data-field=\"" + field + "\"]" );
    if ( node ) Dom.text( node , key ? T( key ) : "" );
  },

  conflictBox( card ) {
    const theirs = this.conflict;
    const box = Dom.el( "div" , { class: "alert alert-warning conflict" } );
    box.appendChild( Dom.el( "p" , { text: I18n.fill( "card.conflict" , { name: CardBits.actorName( theirs.updated_by ) } ) } ) );
    box.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
      Ui.button( "card.conflict_keep_mine" , () => this.keepMine( "description" , theirs ) ),
      Ui.button( "card.conflict_take_theirs" , async () => {
        this.conflict = null;
        this.editingDescription = false;
        this.text.description = null;
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
      const bar = Dom.el( "div" , { class: "progress progress-thin" } );
      const fill = Dom.el( "div" , { class: "progress-bar bg-success" } );
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
      const box = Dom.el( "input" , { class: "form-check-input" , attrs: { type: "checkbox" , "aria-label": item.text } } );
      box.checked = item.done;
      box.disabled = !canEdit;
      box.addEventListener( "change" , function () {
        const fields = { done: box.checked };
        if ( box.checked && ctx.doneOnTop() ) {
          fields.position = CardBits.edgePosition( card.checklist.filter( function ( other ) { return other.id !== item.id; } ) , true );
        }
        ctx.mutate( function () { return ctx.client.updateItem( cardId , item.id , fields ); } );
      } );
      row.appendChild( box );
      if ( canEdit ) {
        row.appendChild( InlineEdit.build( item.text , {
          label: T( "card.checklist_item_label" ),
          maxlength: 500,
          className: "item-text",
          onSave: function ( value ) { ctx.mutate( function () { return ctx.client.updateItem( cardId , item.id , { text: value } ); } ); },
        } ) );
        row.appendChild( Ui.closeButton( T( "card.delete_item" ) , function () { ctx.mutate( function () { return ctx.client.deleteItem( cardId , item.id ); } ); } , "btn-close-small" ) );
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
    } , 500 ) );
    return section;
  },

  actions( card , canEdit , canMove ) {
    const ctx = this.ctx;
    const section = Dom.el( "section" , { class: "card-section card-actions" } );
    const row = Dom.el( "div" , { class: "row-actions" } );
    const link = ctx.cardLink ? ctx.cardLink( card ) : "";
    if ( link ) row.appendChild( Ui.button( "card.copy_link" , () => Ui.copyText( link ) ) );
    row.appendChild( Ui.button( "export.copy_card_markdown" , () => Ui.copyText( CardExport.markdown( ctx , card ) ) ) );
    row.appendChild( Ui.button( "export.card_button" , () => CardExport.open( ctx , card ) ) );
    if ( !ctx.isGuest ) row.appendChild( Ui.button( "card.notifications_button" , () => Subscribe.openCard( ctx , card ) ) );
    if ( ctx.canManage ) row.appendChild( Ui.button( "share.card_button" , () => Sharing.openCard( ctx , card ) ) );
    if ( !ctx.isGuest && !card.archived ) {
      if ( canMove ) row.appendChild( Ui.button( "transfer.send_menu" , () => CardTransfer.open( ctx , card , true ) ) );
      row.appendChild( Ui.button( "transfer.copy_menu" , () => CardTransfer.open( ctx , card , false ) ) );
    }
    if ( canMove ) {
      row.appendChild( Ui.button( card.archived ? "card.unarchive" : "card.archive" , () => {
        this.save( { archived: !card.archived } );
        if ( !card.archived ) this.close();
      } ) );
      row.appendChild( Ui.button( "card.delete" , async () => {
        this.close();
        await ctx.mutate( function () { return ctx.client.deleteCard( card.id ); } );
        Ui.toast( T( "trash.moved" ) , "ok" );
      } , "secondary small danger" ) );
    }
    section.appendChild( row );
    return section;
  },
};
