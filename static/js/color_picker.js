// Colour pickers: a row of swatches rather than a dropdown of names.
//
//   ColorPicker.named( value , onChange , { none } )
//       the board palette (Colors) -- board colour, label colour. The value
//       is a name; "none" adds a "Default" swatch whose value is "".
//   ColorPicker.custom( value , onChange , { presets , against } )
//       any colour, for themes: "Default" (""), preset swatches, a "Custom"
//       swatch that opens the system picker, and a #rrggbb field, all kept
//       in sync. "against" is the text colour it will sit under, to warn
//       when the pick would be hard to read.
//
// Both are a radiogroup: Tab reaches the chosen swatch, arrow keys move.
const ColorPicker = {
  named( value , onChange , options ) {
    const settings = options || {};
    const choices = ( settings.none ? [ "" ] : [] ).concat( Colors );
    const group = this.group( settings.labelKey );
    choices.forEach( ( name ) => {
      const swatch = this.swatch( name ? T( "colors." + name ) : T( "color_picker.default" ) , "swatch-named color-" + ( name || "none" ) + ( name ? "" : " swatch-default" ) );
      swatch.dataset.value = name;
      group.appendChild( swatch );
    } );
    this.wire( group , value || "" , onChange );
    return group;
  },

  custom( value , onChange , options ) {
    const settings = options || {};
    const wrap = Dom.el( "div" , { class: "color-custom" } );
    const group = this.group( settings.labelKey );
    const add = ( hex , label , extra ) => {
      const swatch = this.swatch( label , "swatch-hex" + ( extra || "" ) );
      swatch.dataset.value = hex;
      if ( Theme.isColor( hex ) ) swatch.style.setProperty( "--swatch" , hex );
      group.appendChild( swatch );
      return swatch;
    };
    add( "" , T( "color_picker.default" ) , " swatch-default" );
    ( settings.presets || [] ).forEach( function ( hex ) { add( hex , hex ); } );

    // The "Custom" swatch opens the system picker, kept out of sight next
    // to the swatches (an input cannot sit inside a button).
    const native = Dom.el( "input" , { class: "swatch-native" , attrs: { type: "color" , tabindex: "-1" , "aria-hidden": "true" } } );
    const customSwatch = add( "custom" , T( "color_picker.custom" ) , " swatch-custom" );

    const hexField = Dom.el( "input" , { class: "color-hex form-control" , attrs: {
      maxlength: "7" , spellcheck: "false" , autocapitalize: "off" , placeholder: "#rrggbb" , "aria-label": T( "color_picker.hex_label" ) ,
    } } );
    const warning = Dom.el( "p" , { class: "color-warning small-print" , text: T( "color_picker.low_contrast" ) } );
    warning.hidden = true;

    let current = Theme.isColor( value ) ? value.toLowerCase() : "";
    const show = function () {
      const preset = ( settings.presets || [] ).indexOf( current ) >= 0 || current === "";
      Array.from( group.children ).forEach( function ( swatch ) {
        const chosen = preset ? swatch.dataset.value === current : swatch === customSwatch;
        swatch.setAttribute( "aria-checked" , chosen ? "true" : "false" );
        swatch.tabIndex = chosen ? 0 : -1;
      } );
      if ( current ) {
        native.value = current;
        if ( !preset ) customSwatch.style.setProperty( "--swatch" , current );
      } else {
        customSwatch.style.removeProperty( "--swatch" );
      }
      if ( document.activeElement !== hexField ) hexField.value = current;
      warning.hidden = !( current && settings.against && Theme.contrast( current , settings.against() ) < 3 );
    };
    const choose = function ( hex ) {
      if ( hex === current ) return;
      current = hex;
      show();
      onChange( current );
    };

    this.keys( group , function ( swatch ) {
      if ( swatch === customSwatch ) native.click();
      else choose( swatch.dataset.value );
    } );
    native.addEventListener( "input" , function () { choose( native.value.toLowerCase() ); } );
    hexField.addEventListener( "input" , function () {
      let typed = hexField.value.trim().toLowerCase();
      if ( typed && typed[ 0 ] !== "#" ) typed = "#" + typed;
      hexField.classList.toggle( "invalid" , typed !== "" && !Theme.isColor( typed ) );
      if ( typed === "" || Theme.isColor( typed ) ) choose( typed );
    } );
    hexField.addEventListener( "blur" , function () {
      hexField.classList.remove( "invalid" );
      hexField.value = current;
    } );

    show();
    wrap.appendChild( group );
    wrap.appendChild( native );
    wrap.appendChild( hexField );
    wrap.appendChild( warning );
    wrap.refresh = show;
    return wrap;
  },

  group( labelKey ) {
    const group = Dom.el( "div" , { class: "swatches" , attrs: { role: "radiogroup" } } );
    if ( labelKey ) group.setAttribute( "aria-label" , T( labelKey ) );
    return group;
  },

  swatch( label , className ) {
    return Dom.el( "button" , { class: "swatch " + className , attrs: {
      type: "button" , role: "radio" , "aria-checked": "false" , "aria-label": label , title: label ,
    } } );
  },

  // Named swatches: one is chosen, and choosing another calls onChange.
  wire( group , value , onChange ) {
    let current = value;
    const show = function () {
      Array.from( group.children ).forEach( function ( swatch ) {
        const chosen = swatch.dataset.value === current;
        swatch.setAttribute( "aria-checked" , chosen ? "true" : "false" );
        swatch.tabIndex = chosen ? 0 : -1;
      } );
    };
    this.keys( group , function ( swatch ) {
      if ( swatch.dataset.value === current ) return;
      current = swatch.dataset.value;
      show();
      onChange( current );
    } );
    show();
  },

  // Click picks; arrow keys move focus and pick, like native radios.
  keys( group , pick ) {
    group.addEventListener( "click" , function ( event ) {
      const swatch = event.target.closest( ".swatch" );
      if ( swatch && group.contains( swatch ) ) pick( swatch );
    } );
    group.addEventListener( "keydown" , function ( event ) {
      const step = { ArrowRight: 1 , ArrowDown: 1 , ArrowLeft: -1 , ArrowUp: -1 }[ event.key ];
      if ( !step ) return;
      event.preventDefault();
      const swatches = Array.from( group.children );
      const from = swatches.indexOf( document.activeElement );
      const next = swatches[ ( from + step + swatches.length ) % swatches.length ];
      next.focus();
      // Moving onto "Custom" would open the system picker; leave that to
      // Enter or Space.
      if ( !next.classList.contains( "swatch-custom" ) ) pick( next );
    } );
  },
};
