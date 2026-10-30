// The theme editor: ready-made themes to start from, light or dark, then
// any of the five colours picked by hand, with a small preview drawn in the
// theme being edited. Used for a person's own theme (Account page, a guest's
// panel) and for a board's (Board settings).
//
//   ThemeEditor.render( theme , save , { inherit } )
//
// save( theme ) is called, debounced, after each change; with "inherit" the
// editor also offers "each person's own theme", which saves null.
const ThemeEditor = {
  fields: [ "accent" , "background" , "surface" , "column" , "card" ],

  presets: [
    { key: "light" , theme: { mode: "light" } },
    { key: "dark" , theme: { mode: "dark" } },
    { key: "ocean" , theme: { mode: "light" , accent: "#0b7285" , background: "#eef6f8" , surface: "#ffffff" , column: "#dcecf0" , card: "#ffffff" } },
    { key: "forest" , theme: { mode: "light" , accent: "#2b8a3e" , background: "#f1f6ef" , surface: "#ffffff" , column: "#e1ebdd" , card: "#ffffff" } },
    { key: "sepia" , theme: { mode: "light" , accent: "#9c5a1c" , background: "#f6f0e4" , surface: "#fbf7ef" , column: "#ece3d1" , card: "#fffaf0" } },
    { key: "grape" , theme: { mode: "light" , accent: "#7048c8" , background: "#f5f2fc" , surface: "#ffffff" , column: "#e9e2f8" , card: "#ffffff" } },
    { key: "pink" , theme: { mode: "light" , accent: "#c2255c" , background: "#fdf0f5" , surface: "#ffffff" , column: "#f7dfe9" , card: "#ffffff" } },
    { key: "midnight" , theme: { mode: "dark" , accent: "#8c7cff" , background: "#0e1020" , surface: "#171a30" , column: "#13162a" , card: "#1c2038" } },
    { key: "slate" , theme: { mode: "dark" , accent: "#4fb3a9" , background: "#15191e" , surface: "#1d232a" , column: "#181d23" , card: "#232a32" } },
    { key: "dark_pink" , theme: { mode: "dark" , accent: "#f06595" , background: "#1f0d16" , surface: "#2a1220" , column: "#24101a" , card: "#331828" } },
  ],

  // Swatches offered for each colour; the background ones depend on the mode.
  swatches: {
    accent: [ "#1f5fd6" , "#0b7285" , "#2b8a3e" , "#9c5a1c" , "#c4302b" , "#b3256f" , "#7048c8" , "#57606a" ],
    light: [ "#ffffff" , "#f7f8fa" , "#eef1f5" , "#eef6f8" , "#f1f6ef" , "#f6f0e4" , "#f5f2fc" , "#fdf0f0" ],
    dark: [ "#0b0d10" , "#101215" , "#15181d" , "#1d2127" , "#0e1020" , "#171a30" , "#15191e" , "#232a32" ],
  },

  // The mode's own text colour, which every picked colour sits under.
  text: { light: "#16191d" , dark: "#e8eaed" },

  render( theme , save , options ) {
    const settings = options || {};
    const root = Dom.el( "div" , { class: "theme-editor" } );
    // null is "no theme of its own": a board then shows each person's, and a
    // person sees plain light.
    let current = theme || !settings.inherit ? Object.assign( { mode: "light" } , theme ) : null;
    let timer = 0;
    const commit = function () {
      window.clearTimeout( timer );
      timer = window.setTimeout( function () { save( current ? Object.assign( {} , current ) : null ); } , 400 );
    };
    const draw = () => {
      Dom.clear( root );
      if ( settings.inherit ) root.appendChild( this.inheritChoice( current , function ( own ) {
        current = own ? Object.assign( { mode: "light" } , settings.start || {} ) : null;
        draw();
        commit();
      } ) );
      if ( !current ) return;
      const preview = this.preview( current );
      const presets = this.presetRow( current , function ( preset ) {
        current = Object.assign( {} , preset );
        draw();
        commit();
      } );
      root.appendChild( presets );
      root.appendChild( this.modeSwitch( current.mode , function ( mode ) {
        // Background colours picked for one mode rarely suit the other.
        current = { mode: mode , accent: current.accent || "" };
        draw();
        commit();
      } ) );
      const rows = Dom.el( "div" , { class: "theme-colors" } );
      this.fields.forEach( ( field ) => {
        const picker = ColorPicker.custom( current[ field ] || "" , function ( hex ) {
          current[ field ] = hex;
          ThemeEditor.paint( preview , current );
          ThemeEditor.markPreset( presets , current );
          commit();
        } , {
          labelKey: "theme.field_" + field,
          presets: field === "accent" ? this.swatches.accent : this.swatches[ current.mode ],
          against: field === "accent" ? null : function () { return ThemeEditor.text[ current.mode ]; },
        } );
        rows.appendChild( Dom.el( "div" , { class: "theme-color-row" , children: [
          Dom.el( "span" , { class: "theme-color-name" , text: T( "theme.field_" + field ) } ),
          picker,
        ] } ) );
      } );
      root.appendChild( rows );
      root.appendChild( preview );
      root.appendChild( Ui.button( "theme.reset" , function () {
        current = { mode: current.mode };
        draw();
        commit();
      } ) );
    };
    draw();
    return root;
  },

  inheritChoice( current , onChange ) {
    const group = Dom.el( "div" , { class: "segmented btn-group" , attrs: { role: "group" , "aria-label": T( "theme.board_choice" ) } } );
    [ [ false , "theme.board_inherit" ] , [ true , "theme.board_own" ] ].forEach( function ( pair ) {
      const pressed = !!current === pair[ 0 ];
      group.appendChild( Dom.el( "button" , {
        class: Ui.buttonClass( "secondary small" + ( pressed ? " active" : "" ) ),
        text: T( pair[ 1 ] ),
        attrs: { type: "button" , "aria-pressed": pressed ? "true" : "false" },
        on: { click: function () { if ( !pressed ) onChange( pair[ 0 ] ); } },
      } ) );
    } );
    return group;
  },

  presetRow( current , onPick ) {
    const row = Dom.el( "div" , { class: "theme-presets" , attrs: { role: "group" , "aria-label": T( "theme.presets" ) } } );
    this.presets.forEach( ( preset ) => {
      const button = Dom.el( "button" , {
        class: "theme-preset",
        attrs: { type: "button" },
        on: { click: function () { onPick( preset.theme ); } },
        children: [ this.paint( Dom.el( "span" , { class: "theme-preset-swatch" , children: [
          Dom.el( "span" , { class: "theme-preset-column" } ),
          Dom.el( "span" , { class: "theme-preset-accent" } ),
        ] } ) , preset.theme ) , Dom.el( "span" , { class: "theme-preset-name" , text: T( "theme.preset_" + preset.key ) } ) ],
      } );
      row.appendChild( button );
    } );
    this.markPreset( row , current );
    return row;
  },

  // Highlight the ready-made theme that matches, if any still does.
  markPreset( row , current ) {
    Array.from( row.children ).forEach( ( button , index ) => {
      const same = this.same( current , this.presets[ index ].theme );
      button.classList.toggle( "active" , same );
      button.setAttribute( "aria-pressed" , same ? "true" : "false" );
    } );
  },

  modeSwitch( mode , onChange ) {
    const group = Dom.el( "div" , { class: "segmented btn-group" , attrs: { role: "group" , "aria-label": T( "theme.mode" ) } } );
    [ "light" , "dark" ].forEach( function ( value ) {
      group.appendChild( Dom.el( "button" , {
        class: Ui.buttonClass( "secondary small" + ( mode === value ? " active" : "" ) ),
        text: T( "theme.mode_" + value ),
        attrs: { type: "button" , "aria-pressed": mode === value ? "true" : "false" },
        on: { click: function () { if ( mode !== value ) onChange( value ); } },
      } ) );
    } );
    return group;
  },

  // A miniature board in the theme: header, a column, two cards, a button.
  preview( theme ) {
    const card = function ( width ) {
      // Through the CSSOM: the page's CSP refuses style="" attributes.
      const line = Dom.el( "span" , { class: "theme-preview-line" } );
      line.style.width = width;
      return Dom.el( "span" , { class: "theme-preview-card" , children: [ line ] } );
    };
    const preview = Dom.el( "div" , { class: "theme-preview" , attrs: { "aria-hidden": "true" } , children: [
      Dom.el( "span" , { class: "theme-preview-header" , children: [ Dom.el( "span" , { class: "theme-preview-button" } ) ] } ),
      Dom.el( "span" , { class: "theme-preview-body" , children: [
        Dom.el( "span" , { class: "theme-preview-column" , children: [ card( "70%" ) , card( "45%" ) ] } ),
        Dom.el( "span" , { class: "theme-preview-column" , children: [ card( "55%" ) ] } ),
      ] } ),
    ] } );
    return this.paint( preview , theme );
  },

  // Draw an element in a theme: the mode by data-theme and data-bs-theme (so
  // the stylesheets' dark colours apply inside it) and the picked colours as
  // its variables.
  paint( element , theme ) {
    element.setAttribute( "data-theme" , theme.mode === "dark" ? "dark" : "light" );
    element.setAttribute( "data-bs-theme" , theme.mode === "dark" ? "dark" : "light" );
    Object.keys( Theme.vars ).forEach( function ( field ) {
      if ( Theme.isColor( theme[ field ] ) ) element.style.setProperty( Theme.vars[ field ] , theme[ field ] );
      else element.style.removeProperty( Theme.vars[ field ] );
    } );
    return element;
  },

  same( a , b ) {
    if ( !a || !b || ( a.mode || "light" ) !== ( b.mode || "light" ) ) return false;
    return this.fields.every( function ( field ) { return ( a[ field ] || "" ) === ( b[ field ] || "" ); } );
  },
};
