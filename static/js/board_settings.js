// Board settings: everything about the board in one panel. It opens on
// tiles for the notes, the purpose, sharing, notifications and export (each
// its own panel, with a way back here), the switch for the Purpose section,
// then which view to show, name, default view, colour,
// numbering, and the ways to the theme, the labels, the archive and the
// trash (each its own panel too), and leaving or deleting the board.
const BoardSettings = {
  open( ctx ) {
    Panel.open( "settings.title" , ( body ) => {
      // A guest edits cards, not the board's own setup.
      const member = ctx.canEdit && !ctx.isGuest;
      body.appendChild( this.tiles( ctx ) );
      if ( member ) body.appendChild( this.purposeSwitch( ctx ) );
      body.appendChild( this.views( ctx ) );
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
    if ( ctx.board.notes !== undefined ) tiles.push( [ "notes.notes_heading" , "notes.notes_tile_hint" , function () { BoardNotes.open( "notes" , back ); } ] );
    if ( ctx.board.purpose_enabled ) tiles.push( [ "notes.purpose_heading" , "notes.purpose_tile_hint" , function () { BoardNotes.open( "purpose" , back ); } ] );
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

  // Off by default. Turning it on adds the Purpose icon and tile; turning it
  // off hides them and the text from everyone, agents too, but keeps it.
  purposeSwitch( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Ui.checkbox( "notes.purpose_switch" , ctx.board.purpose_enabled , async function ( checked ) {
      await ctx.mutate( function () { return ctx.client.updateBoard( { purpose_enabled: checked } ); } );
      BoardSettings.open( BoardPage.context() );
    } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: T( "notes.purpose_switch_hint" ) } ) );
    return section;
  },

  // Board, Todo or Timeline, and whether archived cards show, for this
  // browser (BoardPage remembers both per board). Picking a view closes the
  // panel onto it; the checkbox redraws the board behind the panel.
  views( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.view_heading" ) } ) );
    const group = Dom.el( "div" , { class: "btn-group view-picker" , attrs: { role: "group" , "aria-label": T( "views.label" ) } } );
    const calendar = ctx.isCalendar && ctx.isCalendar();
    [ [ "kanban" , calendar ? "calendar3" : "kanban" ] , [ "todo" , "list-check" ] , [ "timeline" , "calendar-range" ] ].forEach( function ( view ) {
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
          Dom.el( "span" , { text: T( view[ 0 ] === "kanban" && calendar ? "views.calendar" : "views." + view[ 0 ] ) } ),
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

  // The label editor in its own panel: from Board settings (with a way
  // back) or from a card's Manage labels.
  openLabels( ctx , back ) {
    Panel.open( "settings.labels_heading" , ( body ) => body.appendChild( this.labelsEditor( ctx ) ) , back );
  },

  labelsSection( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.labels_heading" ) } ) );
    const back = function () { BoardSettings.open( BoardPage.context() ); };
    const button = Ui.button( "settings.labels_button" , function () { BoardSettings.openLabels( ctx , back ); } );
    button.textContent = I18n.fill( "settings.labels_button" , { n: ctx.board.labels.length } );
    section.appendChild( button );
    return section;
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
      { value: "kanban" , text: T( ctx.board.layout === "calendar" ? "views.calendar" : "views.kanban" ) },
      { value: "todo" , text: T( "views.todo" ) },
      { value: "timeline" , text: T( "views.timeline" ) },
    ] , ctx.board.default_view );
    let boardColor = ctx.board.color || "";
    const color = ColorPicker.named( boardColor , function ( name ) {
      boardColor = name;
      save( { color: name } );
    } , { none: true , labelKey: "settings.color_label" } );
    const layout = Ui.select( [
      { value: "" , text: T( "settings.layout_free" ) },
      { value: "three" , text: T( "settings.layout_three" ) },
      { value: "staged" , text: T( "settings.layout_staged" ) },
      { value: "calendar" , text: T( "settings.layout_calendar" ) },
    ] , ctx.board.layout || "" );
    const numbering = Ui.select( [
      { value: "" , text: T( "settings.numbering_default" ) },
      { value: "on" , text: T( "settings.numbering_on" ) },
      { value: "off" , text: T( "settings.numbering_off" ) },
    ] , ctx.board.numbering || "" );
    const placement = function ( current , accountKey ) {
      const mine = !!( ctx.me && ctx.me.display && ctx.me.display[ accountKey ] );
      return Ui.select( [
        { value: "" , text: I18n.fill( "settings.placement_default" , { choice: T( mine ? "settings.placement_top" : "settings.placement_bottom" ) } ) },
        { value: "top" , text: T( "settings.placement_top_all" ) },
        { value: "bottom" , text: T( "settings.placement_bottom_all" ) },
      ] , current || "" );
    };
    const donePlacement = placement( ctx.board.done_placement , "done_on_top" );
    const addPlacement = placement( ctx.board.add_placement , "add_on_top" );

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
    // Only a staged board has a ranking, so the choice follows the layout.
    const showRank = Dom.el( "div" , { children: [
      Ui.checkbox( "settings.show_rank_label" , ctx.board.show_rank , function ( checked ) { save( { show_rank: checked } ); } ),
      Dom.el( "p" , { class: "muted small-print" , text: T( "settings.show_rank_hint" ) } ),
    ] } );
    Dom.show( showRank , layout.value === "staged" );
    form.appendChild( showRank );
    // What this board calls its stages; empty keeps the app's word, shown as
    // the placeholder. Each saves by itself when it is left. Only a staged
    // board has them: on three columns the column titles are the names.
    const stageNames = Dom.el( "div" );
    const stageFields = Dom.el( "div" , { class: "form-row" } );
    Stages.order.forEach( function ( stage ) {
      const key = "stage_" + stage;
      const input = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , placeholder: T( "board." + key ) } } );
      input.value = ctx.board[ key ] || "";
      let saved = input.value;
      input.addEventListener( "change" , function () {
        input.value = input.value.trim();
        if ( input.value === saved ) return;
        saved = input.value;
        save( { [ key ]: saved } );
      } );
      stageFields.appendChild( Ui.field( "settings." + key + "_label" , input ) );
    } );
    stageNames.appendChild( stageFields );
    stageNames.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.stage_names_hint" ) } ) );
    Dom.show( stageNames , layout.value === "staged" || layout.value === "calendar" );
    form.appendChild( stageNames );
    // A calendar board's one setting of its own.
    const carryOver = Dom.el( "div" , { children: [
      Ui.checkbox( "settings.carry_over_label" , ctx.board.carry_over !== false , function ( checked ) { save( { carry_over: checked } ); } ),
      Dom.el( "p" , { class: "muted small-print" , text: T( "settings.carry_over_hint" ) } ),
    ] } );
    Dom.show( carryOver , layout.value === "calendar" );
    form.appendChild( carryOver );
    // A three-column board already fills the page.
    const centerColumns = Dom.el( "div" , { children: [
      Ui.checkbox( "settings.center_columns_label" , ctx.board.center_columns , function ( checked ) { save( { center_columns: checked } ); } ),
      Dom.el( "p" , { class: "muted small-print" , text: T( "settings.center_columns_hint" ) } ),
    ] } );
    Dom.show( centerColumns , layout.value !== "three" );
    form.appendChild( centerColumns );
    form.appendChild( Ui.field( "settings.numbering_label" , numbering ) );
    const placements = Dom.el( "div" , { class: "form-row" } );
    placements.appendChild( Ui.field( "settings.done_placement_label" , donePlacement ) );
    placements.appendChild( Ui.field( "settings.add_placement_label" , addPlacement ) );
    form.appendChild( placements );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.placement_hint" ) } ) );
    form.appendChild( Ui.checkbox( "settings.shopping_label" , ctx.board.shopping , function ( checked ) { save( { shopping: checked } ); } ) );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.shopping_hint" ) } ) );
    // Every field saves by itself: choices as they are picked, the name as
    // it is typed (after a pause), the address when it is left.
    const status = Dom.el( "p" , { class: "muted small-print autosave-status" , attrs: { "aria-live": "polite" } } );
    form.appendChild( status );
    const save = async function ( changes ) {
      Dom.text( status , T( "card.autosave_saving" ) );
      const result = await ctx.mutate( function () { return ctx.client.updateBoard( changes ); } );
      Dom.text( status , result ? T( "card.autosave_saved" ) : "" );
    };
    let titleTimer = null;
    let savedTitle = ctx.board.title;
    const saveTitle = function () {
      window.clearTimeout( titleTimer );
      const value = title.value.trim();
      if ( !value || value === savedTitle ) return;
      savedTitle = value;
      save( { title: value } );
    };
    title.addEventListener( "input" , function () {
      window.clearTimeout( titleTimer );
      Dom.text( status , T( "card.autosave_pending" ) );
      titleTimer = window.setTimeout( saveTitle , 800 );
    } );
    title.addEventListener( "change" , saveTitle );
    let savedSlug = ctx.board.slug || "";
    slug.addEventListener( "change" , function () {
      const value = slug.value.replace( /^-+|-+$/g , "" );
      if ( value === savedSlug || !slug.checkValidity() ) return;
      savedSlug = value;
      save( { slug: value } );
    } );
    view.addEventListener( "change" , function () { save( { default_view: view.value } ); } );
    numbering.addEventListener( "change" , function () { save( { numbering: numbering.value } ); } );
    donePlacement.addEventListener( "change" , function () { save( { done_placement: donePlacement.value } ); } );
    addPlacement.addEventListener( "change" , function () { save( { add_placement: addPlacement.value } ); } );
    layout.addEventListener( "change" , function () {
      Dom.show( showRank , layout.value === "staged" );
      Dom.show( stageNames , layout.value === "staged" || layout.value === "calendar" );
      Dom.show( carryOver , layout.value === "calendar" );
      Dom.show( centerColumns , layout.value !== "three" );
      const changes = { layout: layout.value };
      // Lists to add if the three-column layout finds fewer than three.
      if ( layout.value === "three" ) changes.list_titles = T( "home.default_lists" ).split( "," ).map( function ( name ) { return name.trim(); } );
      save( changes );
    } );
    // Enter in a box saves now rather than submitting anything.
    form.addEventListener( "submit" , function ( event ) {
      event.preventDefault();
      saveTitle();
      slug.dispatchEvent( new Event( "change" ) );
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
  // own (their Account setting). The editor is long, so it opens as its own
  // panel; here is just what the board uses now and the way in.
  theme( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "theme.board_heading" ) } ) );
    section.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( ctx.board.theme ? "theme.board_own" : "theme.board_inherit" ) } ) );
    const back = function () { BoardSettings.open( BoardPage.context() ); };
    section.appendChild( Ui.button( "theme.board_edit" , function () { BoardSettings.openTheme( ctx , back ); } ) );
    return section;
  },

  openTheme( ctx , back ) {
    Panel.open( "theme.board_heading" , ( body ) => {
      body.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "theme.board_hint" ) } ) );
      body.appendChild( ThemeEditor.render( ctx.board.theme , async function ( theme ) {
        const result = await ctx.mutate( function () { return ctx.client.updateBoard( theme ? { theme: theme } : { clear_theme: true } ); } );
        if ( result ) Ui.toast( T( "settings.saved" ) , "ok" );
      } , { inherit: true , start: Theme.user } ) );
    } , back );
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

  labelsEditor( ctx ) {
    const section = Dom.el( "div" );
    const list = Dom.el( "ul" , { class: "label-editor" } );
    const redraw = () => {
      const fresh = BoardPage.context();
      section.replaceWith( this.labelsEditor( fresh ) );
    };
    ctx.board.labels.forEach( function ( label ) {
      const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , "aria-label": T( "settings.label_name" ) } } );
      name.value = label.name;
      let labelColor = label.color;
      const save = async function () {
        await ctx.mutate( function () { return ctx.client.updateLabel( label.id , { name: name.value , color: labelColor } ); } );
        redraw();
      };
      const color = ColorPicker.label( labelColor , function ( value ) {
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
    const color = ColorPicker.label( newColor , function ( value ) { newColor = value; } , { labelKey: "settings.label_color" } );
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

  // The archive and the trash each open as their own panel, so a long
  // archive never stretches this one.
  archived( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.archived_heading" ) } ) );
    const back = function () { BoardSettings.open( BoardPage.context() ); };
    const actions = Dom.el( "div" , { class: "row-actions" } );
    const view = Ui.button( "settings.archived_button" , function () { Archived.open( ctx , back ); } );
    view.textContent = I18n.fill( "settings.archived_button" , { n: Archived.count( ctx ) } );
    actions.appendChild( view );
    if ( ctx.canEdit && !ctx.fixedColumns ) {
      actions.appendChild( Ui.button( "trash.open_button" , function () { Trash.open( ctx , back ); } ) );
    }
    section.appendChild( actions );
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
