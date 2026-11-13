// The Appearance screen: a list of themes beside the one being edited.
//
// Everyone gets "My themes": pick one to use, add, duplicate, rename, delete,
// and change any color. Until someone changes something they follow the
// app's defaults; the first change saves the whole list to their account and
// from then on it is theirs ("Go back to the defaults" undoes that).
//
// Admins also get "Defaults for everyone": the same editor over the list that
// everyone without their own sees, and that new accounts start from.
//
// Each theme has a mode, four base colors and a button style (shape, weight,
// depth, case). Every Bootstrap button color has its own fill, label and
// outline, worked out from the theme and shown under "Buttons" with the rest.
//
// Each theme has a mode and four base colors, and keeps a set for each mode:
// switching it to dark and back gets its light colors back, and the WSU
// built-in switches to its own dark colors. Everything else is worked out
// by theme.js at readable contrast, and shown under "Fine-tune": any of those
// can be set by hand, and Reset puts the worked-out color back. A color that
// leaves text hard to read is flagged where it is, and in a box above.
//
// Every change goes through Undo, and saves the whole list: an undo is the
// same save with the list from before. Dragging in a color picker paints as
// it moves and saves once, when the picker closes.
//
// TodoBoard: Fine-tune also has a board's columns and card faces, and the
// preview shows a column. The same editor, over one theme and without the
// list (single), is a board's own theme in Board settings (board()); it saves
// to the board and has no Undo, as the board page has none. A guest on a share
// link picks from the built-in themes, kept in their browser (guestPicks()).
const ThemeEditor = {
  editors: [],
  // The admin's saved defaults; null while the built-in ones apply.
  defaults: null,
  me: null,

  // The base colors, then the worked-out ones. Labels: theme.color_<key>.
  baseFields: [ "accent" , "second" , "background" , "surface" ],
  fineFields: [ "text" , "background" , "surface" , "nested" , "chrome" , "border" , "link" , "navLink" ,
    "accentFg" , "secondText" , "secondFg" , "info" , "infoBorder" , "column" , "face" ],
  labelKeys: {
    accent: "accent" , second: "second" , background: "background" , surface: "surface" , text: "text" ,
    nested: "nested" , chrome: "chrome" , border: "border" , link: "link" , navLink: "nav_link" ,
    accentFg: "accent_fg" , secondText: "second_text" , secondFg: "second_fg" , info: "info" , infoBorder: "info_border" ,
    column: "column" , face: "face" ,
  },

  // A label for a color of theme.js: its own key, or for a button variant's
  // fill, label or outline, one sentence with the variant's name in it.
  colorLabel( field ) {
    const variant = Theme.variants.find( ( name ) => field.indexOf( Theme.buttonField( name ) ) === 0 );
    if ( !variant ) return I18n.get( "theme.color_" + this.labelKeys[ field ] );
    const part = field.slice( Theme.buttonField( variant ).length ) || "Fill";
    return I18n.get( "theme.color_btn_" + part.toLowerCase() ).replace( "{variant}" , () => I18n.get( "theme.variant_" + variant ) );
  },

  // A line of the hard-to-read box.
  checkLabel( name ) {
    const match = /^(button|outline)_(.+)$/.exec( name );
    if ( !match ) return I18n.get( "theme.check_" + name );
    return I18n.get( "theme.check_" + match[ 1 ] + "_variant" ).replace( "{variant}" , () => I18n.get( "theme.variant_" + match[ 2 ] ) );
  },

  init( me ) {
    this.me = me;
    Theme.load( me );
    const mine = this.create( Dom.get( "theme-editor-mine" ) , {
      kind: "mine",
      themes: () => Theme.list,
      inUse: () => Theme.current,
      use: ( id ) => Theme.pick( id ),
      save: ( next , id , action ) => this.saveMine( next , id , action ),
      restore: () => ( Theme.custom ? { label: "theme.back_to_defaults" , run: () => this.saveMine( null , Theme.current , { label: "undo.theme_defaults_mine" } ) } : null ),
      // Paint the page as a color is dragged, when it is the theme in use.
      live: ( theme ) => { if ( theme.id === Theme.current ) Theme.apply( theme ); },
      settle: () => Theme.apply( Theme.find( Theme.current ) ),
    } );
    Theme.onChange.push( () => mine.render() );
    mine.render();
    this.ownOnBoards( me );
    if ( me.role !== "admin" ) return;
    Dom.show( Dom.get( "theme-tabs" ) , true );
    const defaults = this.create( Dom.get( "theme-editor-defaults" ) , {
      kind: "defaults",
      themes: () => ( this.defaults || Theme.builtIn.map( ( theme ) => Theme.named( theme ) ) ),
      inUse: () => null,
      save: ( next , id , action ) => this.saveDefaults( next , action ),
      restore: () => ( this.defaults ? { label: "theme.restore_built_in" , run: () => this.saveDefaults( null , { label: "undo.theme_defaults" } ) } : null ),
      live: () => {},
      settle: () => {},
    } );
    return Api.themeDefaults().then( ( reply ) => {
      this.defaults = reply.themes || null;
      defaults.render();
    } ).catch( ( error ) => Dom.toast( Dom.worded( error ) , "error" ) );
  },

  // TodoBoard: whether boards with a theme of their own show yours anyway. It
  // is a display setting on the account (UserSettings.save sends them all).
  ownOnBoards( me ) {
    const root = Dom.get( "theme-own-on-boards" );
    if ( !root ) return;
    Dom.clear( root );
    root.appendChild( Ui.checkbox( "user_settings.own_theme_on_boards" , me.display && me.display.own_theme_on_boards , async function ( checked ) {
      await UserSettings.save( me , { own_theme_on_boards: checked } );
    } ) );
    root.appendChild( Dom.el( "p" , { class: "muted small-print mb-0" , text: I18n.get( "user_settings.own_theme_hint" ) } ) );
  },

  // A board's own theme, in its Board settings panel. context() answers the
  // board page's context as it is now (BoardPage.context()); a save goes to the
  // board, and the board page puts it on the page through Theme.setBoard.
  board( root , context ) {
    const self = this;
    const theme = () => { const board = context().board; return board.theme ? Theme.fromDisplay( board.theme , "board" , "" ) : null; };
    // labels: a theme's own, when one is picked for the board; they are added
    // to the board's labels (ThemeEditor.mergeLabels).
    const store = async ( display , labels ) => {
      const ctx = context();
      let counts = null;
      const result = await ctx.mutate( async function () {
        const reply = await ctx.client.updateBoard( display ? { theme: display } : { clear_theme: true } );
        if ( labels && labels.length ) counts = await self.mergeLabels( ctx , labels );
        return reply;
      } );
      if ( result && counts && ( counts.added || counts.updated ) ) {
        Ui.toast( I18n.get( "theme.board_labels_added" ).replace( "{added}" , counts.added ).replace( "{updated}" , counts.updated ) , "ok" );
      } else if ( result ) Ui.toast( I18n.get( "settings.saved" ) , "ok" );
      editor.render();
    };
    // Whose theme the board shows, and a theme to start from: any in the
    // person's own list (the one in use first).
    const header = () => {
      const own = Boolean( context().board.theme );
      const group = Dom.el( "div" , { class: "btn-group btn-group-sm flex-wrap" , attrs: { role: "group" , "aria-label": I18n.get( "theme.board_choice" ) } } );
      [ [ false , "theme.board_inherit" ] , [ true , "theme.board_own" ] ].forEach( function ( pair ) {
        const id = root.id + "-own-" + pair[ 0 ];
        const input = Dom.el( "input" , { class: "btn-check" , attrs: { type: "radio" , name: root.id + "-own" , id: id , "data-focus": "own:" + pair[ 0 ] } } );
        input.checked = own === pair[ 0 ];
        input.addEventListener( "change" , function () {
          const picked = Theme.find( Theme.current );
          store( pair[ 0 ] ? Theme.toDisplay( picked ) : null , pair[ 0 ] ? picked.labels : null );
        } );
        group.appendChild( input );
        group.appendChild( Dom.el( "label" , { class: "btn btn-outline-secondary" , text: I18n.get( pair[ 1 ] ) , attrs: { for: id } } ) );
      } );
      const children = [ group ];
      if ( own ) {
        const select = Dom.el( "select" , { class: "form-select" , attrs: { id: root.id + "-start" , "data-focus": "start" } } );
        select.appendChild( Dom.el( "option" , { text: I18n.get( "theme.board_start_pick" ) , attrs: { value: "" } } ) );
        Theme.list.forEach( function ( item ) {
          select.appendChild( Dom.el( "option" , { text: item.name , attrs: { value: item.id } } ) );
        } );
        select.addEventListener( "change" , function () {
          if ( !select.value ) return;
          const picked = Theme.find( select.value );
          store( Theme.toDisplay( picked ) , picked.labels );
        } );
        children.push( Dom.el( "div" , { children: [
          Dom.el( "label" , { class: "form-label" , text: I18n.get( "theme.board_start" ) , attrs: { for: root.id + "-start" } } ),
          select,
        ] } ) );
      }
      return Dom.el( "div" , { class: "board-theme-head" , children: children } );
    };
    const editor = self.create( root , {
      kind: "board",
      single: true,
      header: header,
      themes: () => { const one = theme(); return one ? [ one ] : []; },
      inUse: () => "board",
      save: ( next ) => store( next && next[ 0 ] ? Theme.toDisplay( next[ 0 ] ) : null ),
      restore: () => null,
      // Paint the page while a color is dragged, when the board's theme is the one showing.
      live: ( edited ) => { if ( Theme.board && !Theme.ownOnBoards ) Theme.apply( edited ); },
      settle: () => Theme.apply( Theme.resolved() ),
    } );
    editor.render();
    return editor;
  },

  // Adds a theme's labels to a board: a board label with the same name (case
  // aside) takes the theme's color, any other is made, up to the board's 50.
  // Labels the theme does not have are left alone, so no card loses one.
  // Each is an ordinary label call, so the board's Undo takes them back.
  async mergeLabels( ctx , labels ) {
    const counts = { added: 0 , updated: 0 };
    const key = ( label ) => String( label.name || "" ).trim().toLowerCase() || "color:" + label.color;
    const have = new Map( ctx.board.labels.map( ( label ) => [ key( label ) , label ] ) );
    let total = ctx.board.labels.length;
    for ( const label of labels ) {
      const found = have.get( key( label ) );
      if ( found ) {
        if ( found.color !== label.color ) {
          await ctx.client.updateLabel( found.id , { color: label.color } );
          counts.updated += 1;
        }
        continue;
      }
      if ( total >= 50 ) break;
      const made = await ctx.client.createLabel( label.name || "" , label.color );
      have.set( key( label ) , made || label );
      total += 1;
      counts.added += 1;
    }
    return counts;
  },

  // Admins: a copy of one of their own themes into the defaults for everyone.
  // A default with the same id (one they started from) is replaced in place;
  // otherwise it goes on the end. Undoable like any change to the defaults.
  async toDefaults( theme ) {
    const list = ( this.defaults || Theme.builtIn.map( ( item ) => Theme.named( item ) ) ).map( ( item ) => Theme.copy( item ) );
    const copy = Theme.copy( theme );
    const at = list.findIndex( ( item ) => item.id === copy.id );
    if ( at < 0 && list.length >= 40 ) {
      Dom.toast( I18n.get( "theme.to_defaults_full" ) , "error" );
      return;
    }
    if ( at >= 0 ) list[ at ] = copy;
    else list.push( copy );
    try {
      await this.saveDefaults( list , { label: "undo.theme_to_defaults" } );
      Dom.toast( I18n.get( at >= 0 ? "theme.to_defaults_replaced" : "theme.to_defaults_added" ).replace( "{name}" , () => theme.name ) , "success" );
    } catch ( error ) {
      Dom.toast( Dom.worded( error ) , "error" );
    }
  },

  // A guest's themes on a share link: the built-in ones, as buttons, the one
  // in use pressed. Kept in this browser.
  guestPicks() {
    const row = Dom.el( "div" , { class: "theme-picks" , attrs: { role: "group" , "aria-label": I18n.get( "theme.heading" ) } } );
    const draw = () => {
      Dom.clear( row );
      Theme.list.forEach( ( theme ) => {
        const pressed = theme.id === Theme.current;
        row.appendChild( Dom.el( "button" , {
          class: "btn btn-sm " + ( pressed ? "btn-primary" : "btn-outline-secondary" ),
          attrs: { type: "button" , "aria-pressed": pressed ? "true" : "false" },
          children: [ this.swatch( theme ) , Dom.el( "span" , { text: theme.name } ) ],
          on: { click: () => { Theme.pick( theme.id ); draw(); } },
        } ) );
      } );
    };
    draw();
    return row;
  },

  saveMine( next , id , action ) {
    const before = { list: Theme.custom ? Theme.list.map( ( theme ) => Theme.copy( theme ) ) : null , id: Theme.current };
    return Undo.run( Object.assign( {} , action , {
      label: I18n.get( action.label ),
      do: () => Theme.store( next , id ),
      undo: () => Theme.store( before.list , before.id ),
    } ) );
  },

  saveDefaults( next , action ) {
    const before = this.defaults ? this.defaults.map( ( theme ) => Theme.copy( theme ) ) : null;
    const store = async ( list ) => {
      const reply = await Api.setThemeDefaults( list );
      this.defaults = reply.themes || null;
      // Someone following the defaults sees the new ones at once, admin included.
      if ( !Theme.custom ) Theme.load( { themes: this.defaults , themes_custom: false , theme_id: Theme.current } );
      this.editors.forEach( ( editor ) => editor.render() );
    };
    return Undo.run( Object.assign( {} , action , {
      label: I18n.get( action.label ),
      do: () => store( next ),
      undo: () => store( before ),
    } ) );
  },

  newID() {
    return "t" + Math.random().toString( 36 ).slice( 2 , 10 );
  },

  // "#abc", "abc" or "#AABBCC" as #aabbcc; "" for anything else.
  hex( value ) {
    let text = String( value || "" ).trim().toLowerCase();
    if ( text.charAt( 0 ) !== "#" ) text = "#" + text;
    if ( /^#[0-9a-f]{3}$/.test( text ) ) text = "#" + text.slice( 1 ).split( "" ).map( ( c ) => c + c ).join( "" );
    return /^#[0-9a-f]{6}$/.test( text ) ? text : "";
  },

  // One editor over one list. source says where the list comes from and goes.
  create( root , source ) {
    const editor = { root: root , source: source , selected: null };
    const self = this;

    const themes = () => source.themes();
    const selectedTheme = () => {
      const list = themes();
      return list.find( ( theme ) => theme.id === editor.selected ) || list.find( ( theme ) => theme.id === source.inUse() ) || list[ 0 ];
    };
    // A save, with any refusal said and the screen put back as it was.
    const attempt = async ( work ) => {
      try {
        await work();
      } catch ( error ) {
        Dom.toast( Dom.worded( error ) , "error" );
        source.settle();
        editor.render();
      }
    };
    // The whole list with one theme replaced by change( copy ).
    const edited = ( theme , change ) => themes().map( ( item ) => {
      if ( item.id !== theme.id ) return Theme.copy( item );
      const copy = Theme.copy( item );
      change( copy );
      return copy;
    } );
    const save = ( next , id , action ) => attempt( () => source.save( next , id === undefined ? source.inUse() : id , action ) );

    const actions = {
      add() {
        const theme = Theme.named( Theme.builtIn[ 0 ] );
        theme.id = self.newID();
        theme.name = I18n.get( "theme.new_name" );
        editor.selected = theme.id;
        return save( themes().map( ( item ) => Theme.copy( item ) ).concat( theme ) , undefined , { label: "undo.theme_new" } );
      },
      duplicate( theme ) {
        const copy = Theme.copy( theme );
        copy.id = self.newID();
        copy.name = I18n.get( "theme.copy_name" ).replace( "{name}" , () => theme.name ).slice( 0 , 60 );
        const next = [];
        themes().forEach( ( item ) => {
          next.push( Theme.copy( item ) );
          if ( item.id === theme.id ) next.push( copy );
        } );
        editor.selected = copy.id;
        return save( next , undefined , { label: "undo.theme_copy" } );
      },
      remove( theme ) {
        const next = themes().filter( ( item ) => item.id !== theme.id ).map( ( item ) => Theme.copy( item ) );
        if ( next.length === 0 ) return Promise.resolve();
        const inUse = source.inUse();
        if ( editor.selected === theme.id ) editor.selected = null;
        return save( next , inUse === theme.id ? next[ 0 ].id : inUse , { label: "undo.theme_delete" , toast: true } );
      },
      rename( theme , name ) {
        return source.save( edited( theme , ( copy ) => { copy.name = name; } ) , source.inUse() , { label: "undo.theme_rename" } );
      },
      set( theme , field , value ) {
        return save( edited( theme , ( copy ) => {
          if ( value ) copy[ field ] = value;
          else delete copy[ field ];
        } ) , undefined , { label: "undo.theme_color" , merge: "theme:" + theme.id + ":" + field } );
      },
      // Light or dark: each keeps its own colors (Theme.setMode), so switching
      // back finds the colors from before.
      mode( theme , value ) {
        return save( edited( theme , ( copy ) => { Theme.setMode( copy , value ); } ) , undefined , { label: "undo.theme_color" } );
      },
      override( theme , field , value ) {
        return save( edited( theme , ( copy ) => {
          if ( value ) copy.overrides[ field ] = value;
          else delete copy.overrides[ field ];
        } ) , undefined , { label: "undo.theme_color" , merge: "theme:" + theme.id + ":" + field } );
      },
      button( theme , key , value ) {
        return save( edited( theme , ( copy ) => {
          copy.button = Object.assign( Theme.button( copy ) , { [ key ]: value } );
        } ) , undefined , { label: "undo.theme_button" , merge: "theme:" + theme.id + ":button:" + key } );
      },
      // TodoBoard: the theme's whole list of labels, replaced.
      labels( theme , labels , merge ) {
        return save( edited( theme , ( copy ) => {
          if ( labels.length ) copy.labels = labels;
          else delete copy.labels;
        } ) , undefined , { label: "undo.theme_labels" , merge: merge ? "theme:" + theme.id + ":labels:" + merge : undefined } );
      },
      resetFine( theme ) {
        return save( edited( theme , ( copy ) => { copy.overrides = {}; } ) , undefined , { label: "undo.theme_reset" } );
      },
    };

    // The row for one theme in the list.
    const listItem = ( theme , selected , inUse ) => {
      const swatch = self.swatch( theme );
      const name = Dom.el( "span" , { class: "theme-item-name" , text: theme.name } );
      const children = [ swatch , name ];
      if ( inUse ) children.push( Dom.badge( I18n.get( "theme.in_use" ) , "success" ) );
      const pick = Dom.el( "button" , {
        class: "theme-item-pick",
        attrs: { type: "button" , "aria-pressed": selected ? "true" : "false" , "data-focus" : "pick:" + theme.id ,
          title: I18n.get( "theme.edit" ).replace( "{name}" , () => theme.name ) },
        children: children,
        on: { click: () => {
          editor.selected = theme.id;
          editor.render();
          const focus = root.querySelector( "[data-focus='pick:" + theme.id + "']" );
          if ( focus ) focus.focus( { preventScroll: true } );
        } },
      } );
      const row = [ pick ];
      if ( source.use && !inUse ) {
        row.push( Dom.el( "button" , { class: "btn btn-sm btn-outline-primary" , text: I18n.get( "theme.use" ) ,
          attrs: { type: "button" , "data-focus": "use:" + theme.id } , on: { click: () => attempt( () => source.use( theme.id ) ) } } ) );
      }
      // TodoBoard has no row menu, so these are icons, to leave the name room.
      const icon = ( kind , key , name , run ) => Dom.el( "button" , { class: "btn btn-sm btn-outline-" + kind ,
        attrs: { type: "button" , title: I18n.get( key ) , "aria-label": I18n.get( key ) } ,
        children: [ Dom.el( "i" , { class: "bi bi-" + name , attrs: { "aria-hidden": "true" } } ) ] , on: { click: run } } );
      const more = [ icon( "secondary" , "theme.duplicate" , "copy" , () => actions.duplicate( theme ) ) ];
      if ( themes().length > 1 ) more.push( icon( "danger" , "theme.delete" , "trash" , () => actions.remove( theme ) ) );
      row.push( Dom.rowActions( theme.name , more ) );
      return Dom.el( "div" , {
        class: "theme-item" + ( selected ? " selected" : "" ),
        attrs: { "data-key": theme.id },
        children: row,
      } );
    };

    // A color: its label over the picker, its hex code, and Reset where there
    // is something to go back to. auto: nothing chosen, so the value shown is
    // worked out. One unit, so the colors can flow into columns on a wide screen.
    const colorRow = ( options ) => {
      const id = root.id + "-" + options.key;
      const picker = Dom.el( "input" , { class: "form-control form-control-color" ,
        attrs: { type: "color" , id: id , "data-focus": "color:" + options.key } } );
      picker.value = options.value;
      const hex = Dom.el( "input" , { class: "form-control form-control-sm theme-hex" ,
        attrs: { type: "text" , maxlength: "7" , spellcheck: "false" , "aria-label": options.label + " " + I18n.get( "theme.hex" ) ,
          "data-focus": "hex:" + options.key } } );
      hex.value = options.value;
      picker.addEventListener( "input" , () => {
        hex.value = picker.value;
        options.preview( picker.value );
      } );
      picker.addEventListener( "change" , () => options.save( picker.value ) );
      hex.addEventListener( "change" , () => {
        const value = self.hex( hex.value );
        if ( !value ) {
          hex.classList.add( "is-invalid" );
          return;
        }
        hex.classList.remove( "is-invalid" );
        if ( value !== options.value ) options.save( value );
        else hex.value = value;
      } );
      const controls = [ picker , hex ];
      if ( options.reset ) {
        controls.push( Dom.el( "button" , { class: "btn btn-sm btn-outline-secondary" , text: I18n.get( "theme.reset" ) ,
          attrs: { type: "button" , title: I18n.get( "theme.reset_title" ) , "data-focus": "reset:" + options.key } ,
          on: { click: () => options.reset() } } ) );
      } else if ( options.auto ) {
        controls.push( Dom.el( "span" , { class: "small text-body-secondary" , text: I18n.get( "theme.auto" ) } ) );
      }
      const label = Dom.el( "label" , { class: "form-label mb-0" , text: options.label , attrs: { for: id } } );
      const row = Dom.el( "div" , { class: "theme-color-input" , children: controls } );
      const kind = "theme-color-row" + ( options.auto ? " auto" : "" ) + ( options.warn ? " warn" : "" );
      return Dom.el( "div" , { class: kind , children: [ label , row ] } );
    };

    // A row of toggle buttons for one choice, like the light/dark one.
    const choiceGroup = ( key , label , values , current , change ) => {
      const group = Dom.el( "div" , { class: "btn-group btn-group-sm flex-wrap" , attrs: { role: "group" , "aria-label": label } } );
      values.forEach( ( value ) => {
        const id = root.id + "-" + key + "-" + value;
        const input = Dom.el( "input" , { class: "btn-check" , attrs: { type: "radio" , name: root.id + "-" + key , id: id , "data-focus": key + ":" + value } } );
        input.checked = current === value;
        input.addEventListener( "change" , () => change( value ) );
        group.appendChild( input );
        group.appendChild( Dom.el( "label" , { class: "btn btn-outline-secondary" , text: I18n.get( "theme." + key + "_" + value ) , attrs: { for: id } } ) );
      } );
      return Dom.el( "div" , { class: "theme-choice" , children: [ Dom.el( "span" , { class: "form-label mb-0" , text: label } ) , group ] } );
    };

    // A theme's labels, edited as a board's are in Board settings: a name and
    // a color for each, and a form to add one. Every change saves the list.
    const labelsPane = ( theme ) => {
      const current = () => ( theme.labels || [] ).map( ( label ) => Object.assign( {} , label ) );
      const wrap = Dom.el( "div" , { class: "theme-labels" } );
      const list = Dom.el( "ul" , { class: "label-editor" } );
      ( theme.labels || [] ).forEach( ( label , index ) => {
        const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , "aria-label": I18n.get( "settings.label_name" ) ,
          "data-focus": "label-name:" + index } } );
        name.value = label.name || "";
        name.addEventListener( "change" , () => {
          const next = current();
          next[ index ].name = name.value.trim();
          actions.labels( theme , next );
        } );
        const color = ColorPicker.label( label.color , ( value ) => {
          const next = current();
          next[ index ].color = value;
          actions.labels( theme , next , "color:" + index );
        } , { labelKey: "settings.label_color" } );
        const remove = Dom.el( "button" , { class: "btn btn-sm btn-outline-danger" , attrs: { type: "button" ,
          title: I18n.get( "theme.label_delete" ) , "aria-label": I18n.get( "theme.label_delete" ) } ,
          children: [ Dom.el( "i" , { class: "bi bi-trash" , attrs: { "aria-hidden": "true" } } ) ] ,
          on: { click: () => actions.labels( theme , current().filter( ( item , at ) => at !== index ) ) } } );
        const chip = Tone.paint( Dom.el( "span" , { class: "label-chip " + Tone.className( label.color ) , text: label.name || "\u00a0" } ) , label.color );
        list.appendChild( Dom.el( "li" , { class: "label-row" , children: [ chip , name , color , remove ] } ) );
      } );
      if ( !( theme.labels || [] ).length ) wrap.appendChild( Dom.el( "p" , { class: "small text-body-secondary mb-2" , text: I18n.get( "theme.labels_none" ) } ) );
      wrap.appendChild( list );
      if ( ( theme.labels || [] ).length >= 50 ) return wrap;
      const form = Dom.el( "form" , { class: "theme-label-add" , attrs: { autocomplete: "off" } } );
      const name = Dom.el( "input" , { class: "form-control" , attrs: { maxlength: "40" , id: root.id + "-label-new" ,
        placeholder: I18n.get( "settings.label_name_placeholder" ) , "data-focus": "label-new" } } );
      let newColor = Colors[ ( theme.labels || [] ).length % Colors.length ];
      const color = ColorPicker.label( newColor , ( value ) => { newColor = value; } , { labelKey: "settings.label_color" } );
      form.appendChild( Dom.el( "div" , { class: "mb-2" , children: [
        Dom.el( "label" , { class: "form-label" , text: I18n.get( "settings.label_name" ) , attrs: { for: root.id + "-label-new" } } ) , name ] } ) );
      form.appendChild( Dom.el( "div" , { class: "mb-2" , children: [
        Dom.el( "span" , { class: "form-label d-block" , text: I18n.get( "settings.label_color" ) } ) , color ] } ) );
      form.appendChild( Dom.el( "button" , { class: "btn btn-sm btn-primary" , attrs: { type: "submit" , "data-focus": "label-add" } ,
        children: [ Dom.el( "i" , { class: "bi bi-plus-lg" , attrs: { "aria-hidden": "true" } } ) , Dom.el( "span" , { text: " " + I18n.get( "settings.add_label" ) } ) ] } ) );
      form.addEventListener( "submit" , ( event ) => {
        event.preventDefault();
        actions.labels( theme , current().concat( [ { name: name.value.trim() , color: newColor } ] ) );
      } );
      wrap.appendChild( form );
      return wrap;
    };

    const editPane = ( theme ) => {
      const colors = Theme.colors( theme );
      const derived = Theme.derived( theme );
      const mode = Theme.modes[ theme.mode ];
      const problems = Theme.check( theme );
      const flagged = new Set();
      problems.forEach( ( problem ) => { flagged.add( problem.front ); flagged.add( problem.back ); } );
      const preview = self.preview( theme );
      // Repaints the preview (and the page, if it is the theme in use) while a
      // picker is dragged; nothing is saved until it closes.
      const previewWith = ( change ) => {
        const copy = Theme.copy( theme );
        change( copy );
        preview.replaceWith( self.preview( copy ) );
        source.live( copy );
      };

      const name = source.single ? null : Dom.editable( theme.name , { label: I18n.get( "theme.name" ) , maxLength: 60 ,
        save: ( value ) => actions.rename( theme , value ) } );

      const modeGroup = Dom.el( "div" , { class: "btn-group" , attrs: { role: "group" , "aria-label": I18n.get( "theme.mode" ) } } );
      [ "light" , "dark" ].forEach( ( value ) => {
        const id = root.id + "-mode-" + value;
        const input = Dom.el( "input" , { class: "btn-check" , attrs: { type: "radio" , name: root.id + "-mode" , id: id , "data-focus": "mode:" + value } } );
        input.checked = theme.mode === value;
        input.addEventListener( "change" , () => actions.mode( theme , value ) );
        modeGroup.appendChild( input );
        modeGroup.appendChild( Dom.el( "label" , { class: "btn btn-outline-secondary btn-sm" , text: I18n.get( "theme.mode_" + value ) , attrs: { for: id } } ) );
      } );

      const base = Dom.el( "div" , { class: "theme-colors" } );
      self.baseFields.forEach( ( field ) => {
        const optional = field === "background" || field === "surface";
        const chosen = theme[ field ] || "";
        base.appendChild( colorRow( {
          key: "base-" + field,
          label: I18n.get( "theme.color_base_" + field ),
          value: chosen || ( optional ? mode[ field ] : "#000000" ),
          auto: optional && !chosen,
          warn: false,
          preview: ( value ) => previewWith( ( copy ) => { copy[ field ] = value; } ),
          save: ( value ) => actions.set( theme , field , value ),
          reset: optional && chosen ? () => actions.set( theme , field , null ) : null,
        } ) );
      } );

      const fine = Dom.el( "div" , { class: "theme-colors" } );
      self.fineFields.forEach( ( field ) => {
        const overridden = Boolean( theme.overrides && theme.overrides[ field ] );
        fine.appendChild( colorRow( {
          key: "fine-" + field,
          label: self.colorLabel( field ),
          value: colors[ field ],
          auto: !overridden,
          warn: flagged.has( field ),
          preview: ( value ) => previewWith( ( copy ) => { copy.overrides[ field ] = value; } ),
          save: ( value ) => ( value === derived[ field ] && !overridden ? null : actions.override( theme , field , value ) ),
          reset: overridden ? () => actions.override( theme , field , null ) : null,
        } ) );
      } );

      // The button's style, then each button color: fill, label, outline.
      const button = Theme.button( theme );
      const choices = Dom.el( "div" , { class: "theme-choices" } );
      Object.keys( Theme.buttonChoices ).forEach( ( key ) => {
        choices.appendChild( choiceGroup( "button_" + key , I18n.get( "theme.button_" + key ) , Object.keys( Theme.buttonChoices[ key ] ) ,
          button[ key ] , ( value ) => {
            previewWith( ( copy ) => { copy.button = Object.assign( Theme.button( copy ) , { [ key ]: value } ); } );
            actions.button( theme , key , value );
          } ) );
      } );
      const buttonColors = Dom.el( "div" , { class: "theme-colors mt-3" } );
      Theme.variants.forEach( ( variant ) => {
        const name = Theme.buttonField( variant );
        [ name , name + "Fg" , name + "Text" ].forEach( ( field ) => {
          const overridden = Boolean( theme.overrides && theme.overrides[ field ] );
          buttonColors.appendChild( colorRow( {
            key: "button-" + field,
            label: self.colorLabel( field ),
            value: colors[ field ],
            auto: !overridden,
            warn: flagged.has( field ),
            preview: ( value ) => previewWith( ( copy ) => { copy.overrides[ field ] = value; } ),
            save: ( value ) => ( value === derived[ field ] && !overridden ? null : actions.override( theme , field , value ) ),
            reset: overridden ? () => actions.override( theme , field , null ) : null,
          } ) );
        } );
      } );

      const aside = [ preview ];
      if ( problems.length > 0 ) {
        const items = problems.map( ( problem ) => Dom.el( "li" , {
          text: self.checkLabel( problem.name ) + " — " + I18n.get( "theme.ratio" ).replace( "{ratio}" , problem.ratio.toFixed( 1 ) ),
        } ) );
        const box = [ Dom.el( "p" , { class: "mb-1" , text: I18n.get( "theme.hard_to_read" ) } ) , Dom.el( "ul" , { class: "mb-2" , children: items } ) ];
        if ( Object.keys( theme.overrides || {} ).length > 0 ) {
          box.push( Dom.el( "button" , { class: "btn btn-sm btn-outline-secondary" , text: I18n.get( "theme.reset_all" ) ,
            attrs: { type: "button" , "data-focus": "reset-all" } , on: { click: () => actions.resetFine( theme ) } } ) );
        }
        aside.push( Dom.el( "div" , { class: "alert alert-warning mt-3 mb-0" , attrs: { role: "status" } , children: box } ) );
      }
      const main = [
        Dom.el( "h4" , { class: "h6 mt-0" , text: I18n.get( "theme.base_heading" ) } ),
        base,
        Dom.el( "h4" , { class: "h6 mt-4 mb-1" , text: I18n.get( "theme.buttons_heading" ) } ),
        Dom.el( "p" , { class: "small text-body-secondary" , text: I18n.get( "theme.buttons_help" ) } ),
        choices,
        buttonColors,
      ];
      const fineHeader = [ Dom.el( "h4" , { class: "h6 mb-0" , text: I18n.get( "theme.fine_heading" ) } ) ];
      if ( Object.keys( theme.overrides || {} ).length > 0 && problems.length === 0 ) {
        fineHeader.push( Dom.el( "button" , { class: "btn btn-sm btn-link p-0" , text: I18n.get( "theme.reset_all" ) ,
          attrs: { type: "button" , "data-focus": "reset-all" } , on: { click: () => actions.resetFine( theme ) } } ) );
      }
      main.push( Dom.el( "div" , { class: "d-flex flex-wrap align-items-baseline gap-3 mt-4 mb-1" , children: fineHeader } ) );
      main.push( Dom.el( "p" , { class: "small text-body-secondary" , text: I18n.get( "theme.fine_help" ) } ) );
      main.push( fine );
      // TodoBoard: the labels that come with the theme. A board's own theme
      // has none here: its labels are the board's, in Board settings.
      if ( !source.single ) {
        main.push( Dom.el( "h4" , { class: "h6 mt-4 mb-1" , text: I18n.get( "theme.labels_heading" ) } ) );
        main.push( Dom.el( "p" , { class: "small text-body-secondary" , text: I18n.get( "theme.labels_help" ) } ) );
        main.push( labelsPane( theme ) );
      }
      // The preview and anything hard to read first, then the colors.
      const top = [ modeGroup ];
      // Admins: this theme, labels and all, into the defaults for everyone.
      if ( source.kind === "mine" && self.me && self.me.role === "admin" ) {
        top.push( Dom.el( "button" , { class: "btn btn-sm btn-outline-secondary ms-sm-auto" , attrs: { type: "button" , "data-focus": "to-defaults" } ,
          children: [ Dom.el( "i" , { class: "bi bi-cloud-upload" , attrs: { "aria-hidden": "true" } } ) , Dom.el( "span" , { text: " " + I18n.get( "theme.to_defaults" ) } ) ] ,
          on: { click: () => self.toDefaults( theme ) } } ) );
      }
      if ( name ) top.unshift( Dom.el( "h3" , { class: "h5 mb-0" , children: [ name ] } ) );
      const children = [ Dom.el( "div" , { class: "d-flex flex-wrap align-items-center gap-2 mb-3" , children: top } ) ]
        .concat( aside , [ Dom.el( "div" , { class: "mt-3" , children: main } ) ] );
      if ( source.single ) return Dom.el( "div" , { children: children } );
      return Dom.el( "div" , { class: "card" , children: [ Dom.el( "div" , { class: "card-body" , children: children } ) ] } );
    };

    // Redraws from the list, keeping the focus on the same control.
    editor.render = () => {
      const active = document.activeElement;
      const focus = active && root.contains( active ) ? active.getAttribute( "data-focus" ) : null;
      if ( source.single ) {
        const one = themes()[ 0 ];
        editor.selected = one ? one.id : null;
        Dom.clear( root );
        root.appendChild( source.header() );
        if ( one ) root.appendChild( editPane( one ) );
        const again = focus ? root.querySelector( "[data-focus='" + focus + "']" ) : null;
        if ( again ) again.focus( { preventScroll: true } );
        return;
      }
      const list = themes();
      const theme = selectedTheme();
      editor.selected = theme ? theme.id : null;
      const items = list.map( ( item ) => listItem( item , item.id === editor.selected , item.id === source.inUse() ) );
      const footer = [ Dom.el( "button" , { class: "btn btn-sm btn-primary" , attrs: { type: "button" , "data-focus": "new" } ,
        children: [ Dom.el( "i" , { class: "bi bi-plus-lg" , attrs: { "aria-hidden": "true" } } ) , Dom.el( "span" , { text: " " + I18n.get( "theme.new" ) } ) ] ,
        on: { click: () => actions.add() } } ) ];
      const restore = source.restore();
      if ( restore ) {
        footer.push( Dom.el( "button" , { class: "btn btn-sm btn-outline-secondary" , text: I18n.get( restore.label ) ,
          attrs: { type: "button" } , on: { click: () => attempt( restore.run ) } } ) );
      }
      const side = Dom.el( "div" , { class: "card" , children: [ Dom.el( "div" , { class: "card-body" , children: [
        Dom.el( "div" , { class: "theme-list" , attrs: { role: "list" } , children: items.map( ( item ) => {
          item.setAttribute( "role" , "listitem" );
          return item;
        } ) } ),
        Dom.el( "div" , { class: "d-flex flex-wrap gap-2 mt-3" , children: footer } ),
      ] } ) ] } );
      Dom.clear( root );
      root.appendChild( Dom.el( "div" , { class: "theme-layout" , children: [ side , theme ? editPane( theme ) : Dom.el( "div" ) ] } ) );
      if ( focus ) {
        const again = root.querySelector( "[data-focus='" + focus + "']" );
        if ( again ) again.focus( { preventScroll: true } );
      }
    };

    this.editors.push( editor );
    return editor;
  },

  swatch( theme ) {
    const colors = Theme.colors( theme );
    const swatch = Dom.el( "span" , { class: "theme-swatch" , attrs: { "aria-hidden": "true" } } );
    swatch.style.setProperty( "--swatch-bg" , colors.background );
    swatch.style.setProperty( "--swatch-accent" , colors.accent );
    swatch.style.setProperty( "--swatch-second" , colors.second );
    return swatch;
  },

  // The page in miniature, in this theme's colors: sidebar, card, nested card,
  // heading, link, info box and every kind of button.
  preview( theme ) {
    const c = Theme.colors( theme );
    const paint = ( element , styles ) => {
      Object.keys( styles ).forEach( ( key ) => { element.style[ key ] = styles[ key ]; } );
      return element;
    };
    const span = ( text , styles ) => paint( Dom.el( "span" , { text: text } ) , styles || {} );
    const side = paint( Dom.el( "div" , { class: "theme-preview-side" , children: [
      span( I18n.get( "theme.preview_nav" ) , { background: c.second , color: c.secondFg } ),
      span( I18n.get( "theme.preview_nav_other" ) , { color: c.navLink } ),
    ] } ) , { background: c.chrome , borderRight: "1px solid " + c.border } );
    const nested = paint( Dom.el( "div" , { class: "theme-preview-nested" , children: [
      span( I18n.get( "theme.preview_heading" ) , { color: c.secondText , fontWeight: "600" } ),
      Dom.el( "br" ),
      span( I18n.get( "theme.preview_text" ) + " " ),
      span( I18n.get( "theme.preview_link" ) , { color: c.link , textDecoration: "underline" } ),
    ] } ) , { background: c.nested , borderColor: c.border } );
    const info = paint( Dom.el( "div" , { class: "theme-preview-info" , text: I18n.get( "theme.preview_info" ) } ) ,
      { background: c.info , borderColor: c.infoBorder } );
    // Real Bootstrap buttons, solid then outline: the preview carries the
    // theme's own custom properties, so style.css draws them as the page will.
    const buttons = ( outline ) => Dom.el( "div" , { class: "theme-preview-buttons" , children:
      [ "primary" ].concat( Theme.variants ).map( ( variant ) => Dom.el( "button" , {
        class: "btn btn-sm btn-" + ( outline ? "outline-" : "" ) + variant,
        text: variant === "primary" ? I18n.get( "theme.preview_button" ) : I18n.get( "theme.variant_" + variant ),
        attrs: { type: "button" , tabindex: "-1" },
      } ) ) } );
    const card = paint( Dom.el( "div" , { class: "theme-preview-card" , children: [
      span( I18n.get( "theme.preview_card" ) ),
      nested,
      info,
      buttons( false ),
      buttons( true ),
    ] } ) , { background: c.surface , borderColor: c.border } );
    // TodoBoard: a board column with two cards on it.
    const face = ( text ) => paint( Dom.el( "span" , { class: "theme-preview-face" , text: text } ) , { background: c.face } );
    const board = paint( Dom.el( "div" , { class: "theme-preview-board" , children: [
      face( I18n.get( "theme.preview_face" ) ), face( I18n.get( "theme.preview_face_other" ) ),
    ] } ) , { background: c.column } );
    card.appendChild( board );
    const page = paint( Dom.el( "div" , { class: "theme-preview-page" , children: [ card ] } ) , { background: c.background } );
    const preview = paint( Dom.el( "div" , { class: "theme-preview" , attrs: { "aria-hidden": "true" , inert: "" } , children: [ side , page ] } ) ,
      { color: c.text , borderColor: c.border } );
    // Its own mode, so a light theme previews light on a dark page too.
    preview.setAttribute( "data-bs-theme" , theme.mode === "dark" ? "dark" : "light" );
    Object.keys( Theme.vars ).forEach( ( field ) => preview.style.setProperty( Theme.vars[ field ] , c[ field ] ) );
    const style = Theme.buttonStyle( theme );
    Object.keys( style ).forEach( ( name ) => preview.style.setProperty( name , style[ name ] ) );
    return preview;
  },
};
