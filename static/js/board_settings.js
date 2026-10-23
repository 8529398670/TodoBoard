// Board settings: name, default view, colour, labels, archived lists and
// cards, and leaving or deleting the board.
const BoardSettings = {
  open( ctx ) {
    Panel.open( "settings.title" , ( body ) => {
      if ( ctx.canEdit ) body.appendChild( this.general( ctx ) );
      if ( ctx.canEdit ) body.appendChild( this.labelsSection( ctx ) );
      if ( ctx.canEdit ) body.appendChild( this.archived( ctx ) );
      body.appendChild( this.danger( ctx ) );
    } );
  },

  openLabels( ctx ) {
    Panel.open( "settings.labels_heading" , ( body ) => body.appendChild( this.labelsSection( ctx ) ) );
  },

  general( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.general_heading" ) } ) );
    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const title = Dom.el( "input" , { attrs: { maxlength: "120" , required: "" } } );
    title.value = ctx.board.title;
    const slug = Dom.el( "input" , { attrs: { maxlength: "60" , pattern: "[a-z0-9]+(-[a-z0-9]+)*" , spellcheck: "false" , autocapitalize: "off" } } );
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
    const color = Ui.select( [ { value: "" , text: T( "settings.color_none" ) } ].concat( Colors.map( function ( name ) {
      return { value: name , text: T( "colors." + name ) };
    } ) ) , ctx.board.color );
    const layout = Ui.select( [
      { value: "" , text: T( "settings.layout_free" ) },
      { value: "three" , text: T( "settings.layout_three" ) },
    ] , ctx.board.layout || "" );

    const names = Dom.el( "div" , { class: "form-row" } );
    names.appendChild( Ui.field( "settings.board_title_label" , title ) );
    names.appendChild( slugField );
    form.appendChild( names );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.slug_hint" ) } ) );
    const options = Dom.el( "div" , { class: "form-row" } );
    options.appendChild( Ui.field( "settings.default_view_label" , view ) );
    options.appendChild( Ui.field( "settings.color_label" , color ) );
    options.appendChild( Ui.field( "settings.layout_label" , layout ) );
    form.appendChild( options );
    form.appendChild( Dom.el( "p" , { class: "muted small-print" , text: T( "settings.layout_hint" ) } ) );
    form.appendChild( Dom.el( "button" , { text: T( "board.save" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async function ( event ) {
      event.preventDefault();
      const changes = { title: title.value , default_view: view.value , color: color.value , layout: layout.value };
      if ( slug.value !== ( ctx.board.slug || "" ) ) changes.slug = slug.value.replace( /^-+|-+$/g , "" );
      // Lists to add if the three-column layout finds fewer than three.
      if ( layout.value === "three" ) changes.list_titles = T( "home.default_lists" ).split( "," ).map( function ( name ) { return name.trim(); } );
      const result = await ctx.mutate( function () { return ctx.client.updateBoard( changes ); } );
      if ( result ) Ui.toast( T( "settings.saved" ) , "ok" );
    } );
    section.appendChild( form );
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
      const name = Dom.el( "input" , { attrs: { maxlength: "40" , "aria-label": T( "settings.label_name" ) } } );
      name.value = label.name;
      const color = Ui.select( Colors.map( function ( item ) { return { value: item , text: T( "colors." + item ) }; } ) , label.color );
      color.setAttribute( "aria-label" , T( "settings.label_color" ) );
      const swatch = Dom.el( "span" , { class: "label-chip swatch color-" + label.color , text: T( "settings.swatch_symbol" ) } );
      const save = async function () {
        await ctx.mutate( function () { return ctx.client.updateLabel( label.id , { name: name.value , color: color.value } ); } );
        redraw();
      };
      name.addEventListener( "change" , save );
      color.addEventListener( "change" , save );
      list.appendChild( Dom.el( "li" , { class: "label-row" , children: [
        swatch , name , color ,
        Ui.button( "settings.delete_label" , async function () {
          if ( !window.confirm( I18n.fill( "settings.delete_label_confirm" , { name: label.name || label.color } ) ) ) return;
          await ctx.mutate( function () { return ctx.client.deleteLabel( label.id ); } );
          redraw();
        } , "secondary small danger" ),
      ] } ) );
    } );
    section.appendChild( list );

    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const name = Dom.el( "input" , { attrs: { maxlength: "40" , placeholder: T( "settings.label_name_placeholder" ) } } );
    const color = Ui.select( Colors.map( function ( item ) { return { value: item , text: T( "colors." + item ) }; } ) , Colors[ ctx.board.labels.length % Colors.length ] );
    const fields = Dom.el( "div" , { class: "form-row" } );
    fields.appendChild( Ui.field( "settings.label_name" , name ) );
    fields.appendChild( Ui.field( "settings.label_color" , color ) );
    form.appendChild( fields );
    form.appendChild( Dom.el( "button" , { text: T( "settings.add_label" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async function ( event ) {
      event.preventDefault();
      await ctx.mutate( function () { return ctx.client.createLabel( name.value , color.value ); } );
      redraw();
    } );
    section.appendChild( form );
    return section;
  },

  archived( ctx ) {
    const section = Dom.el( "section" , { class: "panel-section" } );
    section.appendChild( Dom.el( "h3" , { text: T( "settings.archived_heading" ) } ) );
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
