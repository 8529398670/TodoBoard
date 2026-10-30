// Board settings: everything about the board in one panel. It opens on
// tiles for sharing, notifications and export (each its own panel, with a
// way back here), then which view to show, name, default view, colour,
// numbering, theme, labels, archived lists and cards, and leaving or
// deleting the board.
const BoardSettings = {
  open( ctx ) {
    Panel.open( "settings.title" , ( body ) => {
      body.appendChild( this.tiles( ctx ) );
      body.appendChild( this.views( ctx ) );
      // A guest edits cards, not the board's own setup.
      const member = ctx.canEdit && !ctx.isGuest;
      if ( member ) body.appendChild( this.general( ctx ) );
      if ( member ) body.appendChild( this.theme( ctx ) );
      if ( member ) body.appendChild( this.labelsSection( ctx ) );
      if ( member ) body.appendChild( this.archived( ctx ) );
      if ( !ctx.isGuest && ctx.data.me.role ) body.appendChild( this.personal( ctx ) );
      body.appendChild( this.danger( ctx ) );
    } );
  },

  tiles( ctx ) {
    const back = function () { BoardSettings.open( BoardPage.context() ); };
    const tiles = [];
    if ( ctx.canManage ) tiles.push( [ "board.share_button" , "board.share_hint" , function () { Sharing.open( ctx , back ); } ] );
    if ( !ctx.isGuest ) tiles.push( [ "board.notify_button" , "board.notify_hint" , function () { Subscribe.openBoard( ctx , back ); } ] );
    tiles.push( [ "board.export_button" , "board.export_hint" , function () { BoardExport.open( ctx , back ); } ] );
    const section = Dom.el( "section" , { class: "panel-section settings-tiles" } );
    tiles.forEach( function ( tile ) {
      section.appendChild( Dom.el( "button" , {
        class: "settings-tile",
        attrs: { type: "button" },
        on: { click: tile[ 2 ] },
        children: [
          Dom.el( "span" , { class: "settings-tile-title" , text: T( tile[ 0 ] ) } ),
          Dom.el( "span" , { class: "settings-tile-hint" , text: T( tile[ 1 ] ) } ),
        ],
      } ) );
    } );
    return section;
  },

  // Board, Todo or Timeline, and whether archived cards show, for this
  // browser (BoardPage remembers both per board). Picking a view closes the
  // panel onto it; the checkbox redraws the board behind the panel.
  views( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.view_heading" ) } ) );
    const group = Dom.el( "div" , { class: "btn-group view-picker" , attrs: { role: "group" , "aria-label": T( "views.label" ) } } );
    [ [ "kanban" , "kanban" ] , [ "todo" , "list-check" ] , [ "timeline" , "calendar-range" ] ].forEach( function ( view ) {
      const active = BoardPage.view === view[ 0 ];
      group.appendChild( Dom.el( "button" , {
        class: Ui.buttonClass( "secondary" + ( active ? " active" : "" ) ),
        attrs: { type: "button" , "aria-pressed": active ? "true" : "false" },
        on: { click: function () {
          Panel.close();
          BoardPage.setView( view[ 0 ] );
        } },
        children: [
          Dom.el( "i" , { class: "bi bi-" + view[ 1 ] , attrs: { "aria-hidden": "true" } } ),
          Dom.el( "span" , { text: T( "views." + view[ 0 ] ) } ),
        ],
      } ) );
    } );
    section.appendChild( group );
    section.appendChild( Dom.el( "p" , { class: "muted small-print mt-2" , text: T( ctx.canEdit && !ctx.isGuest ? "settings.view_hint" : "settings.view_hint_guest" ) } ) );
    const archived = Ui.check( I18n.fill( "board.show_archived" , { n: BoardPage.archivedCount( ctx ) } ) , BoardPage.showArchived() , function ( checked ) {
      BoardPage.setShowArchived( checked );
      BoardPage.render();
    } );
    section.appendChild( archived );
    section.appendChild( Dom.el( "p" , { class: "muted small-print mt-1 mb-0" , text: T( "settings.show_archived_hint" ) } ) );
    return section;
  },

  openLabels( ctx ) {
    Panel.open( "settings.labels_heading" , ( body ) => body.appendChild( this.labelsSection( ctx ) ) );
  },

  general( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.general_heading" ) } ) );
    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const title = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "120" , required: "" } } );
    title.value = ctx.board.title;
    const slug = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "60" , pattern: "[a-z0-9]+(-[a-z0-9]+)*" , spellcheck: "false" , autocapitalize: "off" } } );
    slug.value = ctx.board.slug || "";
    // Typing a space or capital letter is the natural thing to do; make it
    // what the server will accept as they type.
    slug.addEventListener( "input" , function () {
      const cleaned = slug.value.toLowerCase().replace( /[^a-z0-9-]+/g , "-" ).replace( /-{2,}/g , "-" );
      if ( cleaned !== slug.value ) slug.value = cleaned;
    } );
    const slugField = Ui.field( "settings.slug_label" , slug );
    const slugWrap = Dom.el( "div" , { class: "slug-input" , children: [ Dom.el( "span" , { class: "slug-prefix" , text: T( "settings.slug_prefix" ) } ) ] } );
    slugField.replaceChild( slugWrap , slug );
    slugWrap.appendChild( slug );

    const view = Ui.select( [
      { value: "kanban" , text: T( "views.kanban" ) },
      { value: "todo" , text: T( "views.todo" ) },
      { value: "timeline" , text: T( "views.timeline" ) },
    ] , ctx.board.default_view );
    let boardColor = ctx.board.color || "";
    const color = ColorPicker.named( boardColor , function ( name ) { boardColor = name; } , { none: true , labelKey: "settings.color_label" } );
    const layout = Ui.select( [
      { value: "" , text: T( "settings.layout_free" ) },
      { value: "three" , text: T( "settings.layout_three" ) },
    ] , ctx.board.layout || "" );
    const numbering = Ui.select( [
      { value: "" , text: T( "settings.numbering_default" ) },
      { value: "on" , text: T( "settings.numbering_on" ) },
      { value: "off" , text: T( "settings.numbering_off" ) },
    ] , ctx.board.numbering || "" );

    const names = Dom.el( "div" , { class: "form-row" } );
    names.appendChild( Ui.field( "settings.board_title_label" , title ) );
    names.appendChild( slugField );
    form.appendChild( names );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.slug_hint" ) } ) );
    const options = Dom.el( "div" , { class: "form-row" } );
    options.appendChild( Ui.field( "settings.default_view_label" , view ) );
    options.appendChild( Ui.field( "settings.layout_label" , layout ) );
    form.appendChild( options );
    form.appendChild( BoardSettings.swatchField( "settings.color_label" , color ) );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.layout_hint" ) } ) );
    form.appendChild( Ui.field( "settings.numbering_label" , numbering ) );
    form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "board.save" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async function ( event ) {
      event.preventDefault();
      const changes = { title: title.value , default_view: view.value , color: boardColor , layout: layout.value , numbering: numbering.value };
      if ( slug.value !== ( ctx.board.slug || "" ) ) changes.slug = slug.value.replace( /^-+|-+$/g , "" );
      // Lists to add if the three-column layout finds fewer than three.
      if ( layout.value === "three" ) changes.list_titles = T( "home.default_lists" ).split( "," ).map( function ( name ) { return name.trim(); } );
      const result = await ctx.mutate( function () { return ctx.client.updateBoard( changes ); } );
      if ( result ) Ui.toast( T( "settings.saved" ) , "ok" );
    } );
    section.appendChild( form );
    return section;
  },

  // A labelled swatch row; a radiogroup has no single control to point a
  // <label for> at, so it is named by aria-label and shown with a span.
  swatchField( labelKey , picker ) {
    return Dom.el( "div" , { class: "field" , children: [ Dom.el( "span" , { class: "form-label" , text: T( labelKey ) } ) , picker ] } );
  },

  // The board's theme, which everyone on it sees unless they keep their
  // own (their Account setting).
  theme( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "theme.board_heading" ) } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "theme.board_hint" ) } ) );
    section.appendChild( ThemeEditor.render( ctx.board.theme , async function ( theme ) {
      const result = await ctx.mutate( function () { return ctx.client.updateBoard( theme ? { theme: theme } : { clear_theme: true } ); } );
      if ( result ) Ui.toast( T( "settings.saved" ) , "ok" );
    } , { inherit: true , start: Theme.user } ) );
    return section;
  },

  // How this board looks to you alone; nobody else on it is affected.
  personal( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.personal_heading" ) } ) );
    const mine = ctx.data.me.plain_icons;
    const fallback = !!( ctx.me && ctx.me.display && ctx.me.display.plain_icons );
    const icons = Ui.select( [
      { value: "" , text: I18n.fill( "settings.plain_icons_default" , { choice: T( fallback ? "settings.plain_icons_grey" : "settings.plain_icons_colour" ) } ) },
      { value: "grey" , text: T( "settings.plain_icons_grey" ) },
      { value: "colour" , text: T( "settings.plain_icons_colour" ) },
    ] , typeof mine === "boolean" ? ( mine ? "grey" : "colour" ) : "" );
    icons.addEventListener( "change" , async function () {
      const value = icons.value === "" ? null : icons.value === "grey";
      const result = await ctx.mutate( function () { return ctx.client.setMyView( { plain_icons: value } ); } );
      if ( result ) Ui.toast( T( "settings.saved" ) , "ok" );
    } );
    section.appendChild( Ui.field( "settings.plain_icons" , icons ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.plain_icons_hint" ) } ) );
    return section;
  },

  labelsSection( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.labels_heading" ) } ) );
    const list = Dom.el( "ul" , { class: "label-editor" } );
    const redraw = () => {
      const fresh = BoardPage.context();
      section.replaceWith( this.labelsSection( fresh ) );
    };
    ctx.board.labels.forEach( function ( label ) {
      const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , "aria-label": T( "settings.label_name" ) } } );
      name.value = label.name;
      let labelColor = label.color;
      const save = async function () {
        await ctx.mutate( function () { return ctx.client.updateLabel( label.id , { name: name.value , color: labelColor } ); } );
        redraw();
      };
      const color = ColorPicker.named( labelColor , function ( value ) {
        labelColor = value;
        save();
      } , { labelKey: "settings.label_color" } );
      name.addEventListener( "change" , save );
      list.appendChild( Dom.el( "li" , { class: "label-row" , children: [
        name , color ,
        Ui.button( "settings.delete_label" , async function () {
          if ( !window.confirm( I18n.fill( "settings.delete_label_confirm" , { name: label.name || label.color } ) ) ) return;
          await ctx.mutate( function () { return ctx.client.deleteLabel( label.id ); } );
          redraw();
        } , "secondary small danger" ),
      ] } ) );
    } );
    section.appendChild( list );

    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , placeholder: T( "settings.label_name_placeholder" ) } } );
    let newColor = Colors[ ctx.board.labels.length % Colors.length ];
    const color = ColorPicker.named( newColor , function ( value ) { newColor = value; } , { labelKey: "settings.label_color" } );
    form.appendChild( Ui.field( "settings.label_name" , name ) );
    form.appendChild( BoardSettings.swatchField( "settings.label_color" , color ) );
    form.appendChild( Dom.el( "button" , { class: "btn btn-primary" , text: T( "settings.add_label" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async function ( event ) {
      event.preventDefault();
      await ctx.mutate( function () { return ctx.client.createLabel( name.value , newColor ); } );
      redraw();
    } );
    section.appendChild( form );
    return section;
  },

  archived( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.archived_heading" ) } ) );
    if ( ctx.canEdit && !ctx.listOnly ) {
      section.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Ui.button( "trash.open_button" , function () { Trash.open( ctx , function () { BoardSettings.open( BoardPage.context() ); } ); } ),
      ] } ) );
    }
    const lists = ctx.board.lists.filter( function ( list ) { return list.archived; } );
    const cards = ctx.cards.filter( function ( card ) { return card.archived; } );
    if ( !lists.length && !cards.length ) {
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "settings.nothing_archived" ) } ) );
      return section;
    }
    const list = Dom.el( "ul" , { class: "link-list" } );
    const rendered = function ( text ) {
      const span = Dom.el( "span" );
      Markdown.renderInline( text , span );
      return span;
    };
    lists.forEach( function ( item ) {
      list.appendChild( Dom.el( "li" , { class: "link-row" , children: [
        rendered( I18n.fill( "settings.archived_list" , { name: item.title } ) ),
        Ui.button( "settings.restore" , async function () {
          await ctx.mutate( function () { return ctx.client.updateList( item.id , { archived: false } ); } );
          Panel.close();
        } ),
      ] } ) );
    } );
    cards.forEach( function ( card ) {
      list.appendChild( Dom.el( "li" , { class: "link-row" , children: [
        rendered( card.title ),
        Dom.el( "div" , { class: "row-actions" , children: [
          Ui.button( "settings.restore" , async function () {
            await ctx.mutate( function () { return ctx.client.patchCard( card.id , { archived: false } ); } );
            Panel.close();
          } ),
          Ui.button( "settings.open_card" , function () {
            Panel.close();
            ctx.openCard( card.id );
          } ),
        ] } ),
      ] } ) );
    } );
    section.appendChild( list );
    return section;
  },

  danger( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    const role = ctx.me && ctx.data.me ? ctx.data.me.role : "";
    if ( role ) section.appendChild( Dom.el( "h3" , { text: T( "settings.danger_heading" ) } ) );
    if ( role === "owner" ) {
      section.appendChild( Dom.el( "p" , { children: [ Dom.el( "strong" , { text: T( "settings.delete_heading" ) } ) ] } ) );
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "settings.delete_hint" ) } ) );
      section.appendChild( Ui.button( "settings.delete_board" , async function () {
        if ( !window.confirm( I18n.fill( "settings.delete_board_confirm" , { name: ctx.board.title } ) ) ) return;
        try {
          await ctx.client.deleteBoard();
          window.location.href = "/";
        } catch ( error ) { Ui.error( error ); }
      } , "secondary small danger" ) );
    } else if ( role ) {
      section.appendChild( Ui.button( "settings.leave_board" , async function () {
        if ( !window.confirm( I18n.fill( "settings.leave_confirm" , { name: ctx.board.title } ) ) ) return;
        try {
          await ctx.client.leave();
          window.location.href = "/";
        } catch ( error ) { Ui.error( error ); }
      } , "secondary small danger" ) );
    }
    return section;
  },
};
