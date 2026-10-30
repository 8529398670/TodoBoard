// Small UI pieces the board screens share: buttons, a reusable side panel,
// a transient message, file downloads, and the date and colour helpers card
// faces need. No app logic -- that lives in each screen's own file.
const T = function ( key ) { return I18n.get( key ); };

const Ui = {
  button( key , onClick , className ) {
    return Dom.el( "button" , {
      class: this.buttonClass( className ),
      text: T( key ),
      attrs: { type: "button" },
      on: { click: onClick },
    } );
  },

  // A button that is only a Bootstrap icon (bi-<icon>); label is its
  // accessible name and tooltip, already translated.
  iconButton( icon , label , onClick , className ) {
    return Dom.el( "button" , {
      class: "btn btn-link app-icon-button " + ( className || "" ),
      attrs: { type: "button" , "aria-label": label , title: label },
      children: [ Dom.el( "i" , { class: "bi bi-" + icon , attrs: { "aria-hidden": "true" } } ) ],
      on: { click: onClick },
    } );
  },

  // Bootstrap's x, for closing a dialog or removing one item.
  closeButton( label , onClick , className ) {
    return Dom.el( "button" , {
      class: "btn-close " + ( className || "" ),
      attrs: { type: "button" , "aria-label": label , title: label },
      on: { click: onClick },
    } );
  },

  // Shorthand for a Bootstrap button's classes: "secondary" (outlined),
  // "danger" (outlined red), primary when neither, "small" for btn-sm. Any
  // other name is kept as a class of its own ("active", "panel-back").
  buttonClass( className ) {
    const shorthand = [ "secondary" , "danger" , "small" ];
    const names = ( className || "secondary small" ).split( /\s+/ ).filter( Boolean );
    const has = function ( name ) { return names.indexOf( name ) >= 0; };
    const look = has( "danger" ) ? "btn-outline-danger" : has( "secondary" ) ? "btn-outline-secondary" : "btn-primary";
    const rest = names.filter( function ( name ) { return shorthand.indexOf( name ) < 0; } );
    return [ "btn" , look ].concat( has( "small" ) ? [ "btn-sm" ] : [] , rest ).join( " " );
  },

  // A labelled form control. Labels stay visible at every width.
  field( labelKey , control ) {
    const id = control.id || ( "f-" + Math.random().toString( 36 ).slice( 2 ) );
    control.id = id;
    this.styleControl( control );
    return Dom.el( "div" , { class: "field" , children: [
      Dom.el( "label" , { class: "form-label" , text: T( labelKey ) , attrs: { for: id } } ),
      control,
    ] } );
  },

  // Bootstrap's look for a text input, textarea or select the scripts built.
  styleControl( control ) {
    if ( control.tagName === "SELECT" ) control.classList.add( "form-select" );
    else if ( control.tagName === "TEXTAREA" || ( control.tagName === "INPUT" && [ "checkbox" , "radio" , "color" ].indexOf( control.type ) < 0 ) ) control.classList.add( "form-control" );
    return control;
  },

  select( options , value ) {
    const select = Dom.el( "select" , { class: "form-select" } );
    options.forEach( function ( option ) {
      const node = Dom.el( "option" , { text: option.text , attrs: { value: String( option.value ) } } );
      if ( String( option.value ) === String( value ) ) node.selected = true;
      select.appendChild( node );
    } );
    return select;
  },

  checkbox( labelKey , checked , onChange ) {
    return this.check( T( labelKey ) , checked , onChange );
  },

  // A checkbox with its label as text rather than a language key.
  check( text , checked , onChange ) {
    const input = Dom.el( "input" , { class: "form-check-input" , attrs: { type: "checkbox" } } );
    input.checked = !!checked;
    if ( onChange ) input.addEventListener( "change" , function () { onChange( input.checked ); } );
    return Dom.el( "label" , { class: "check form-check" , children: [ input , Dom.el( "span" , { class: "form-check-label" , text: text } ) ] } );
  },

  toast( message , kind ) {
    Dom.toast( message , kind );
  },

  // Errors from the server are codes ("card_stale", "share_link_invalid");
  // language.yaml's errors: section words them.
  error( error ) {
    const code = error && error.message ? error.message : "";
    const worded = code ? T( "errors." + code ) : "";
    this.toast( worded || code || T( "errors.generic" ) , "error" );
  },

  download( blob , filename ) {
    const url = URL.createObjectURL( blob );
    const link = Dom.el( "a" , { attrs: { href: url , download: filename } } );
    document.body.appendChild( link );
    link.click();
    link.remove();
    window.setTimeout( function () { URL.revokeObjectURL( url ); } , 5000 );
  },

  slug( text ) {
    const slug = String( text || "" ).toLowerCase().replace( /[^a-z0-9]+/g , "-" ).replace( /^-+|-+$/g , "" ).slice( 0 , 60 );
    return slug || "card";
  },

  // A textarea that grows with what is typed instead of scrolling.
  autoGrow( area ) {
    area.style.height = "auto";
    area.style.height = area.scrollHeight + 2 + "px";
  },

  // List titles and checklist items stay on one line: line breaks become
  // single spaces.
  oneLine( text ) {
    return String( text || "" ).replace( /\s*[\r\n]+\s*/g , " " );
  },

  // Asks for one file and resolves with it, or with null if none was picked.
  pickFile( accept ) {
    return new Promise( function ( resolve ) {
      const input = Dom.el( "input" , { attrs: { type: "file" , accept: accept || "" } } );
      input.addEventListener( "change" , function () { resolve( input.files && input.files[ 0 ] ? input.files[ 0 ] : null ); } );
      input.addEventListener( "cancel" , function () { resolve( null ); } );
      input.click();
    } );
  },

  async copyText( text ) {
    try {
      await navigator.clipboard.writeText( text );
      this.toast( T( "export.copied" ) , "ok" );
    } catch ( error ) {
      this.toast( T( "export.copy_failed" ) , "error" );
    }
  },
};

// A right-click menu: a short list of actions at the pointer. Closes on a
// pick, Escape, a click elsewhere, scrolling or resizing.
//
//   items: [ { key , action , danger } ] -- key is the language.yaml text.
const ContextMenu = {
  node: null,

  open( event , items ) {
    event.preventDefault();
    this.close();
    const menu = Dom.el( "div" , { class: "context-menu" , attrs: { role: "menu" } } );
    items.forEach( ( item ) => {
      menu.appendChild( Dom.el( "button" , {
        class: "context-menu-item" + ( item.danger ? " danger" : "" ),
        text: T( item.key ),
        attrs: { type: "button" , role: "menuitem" },
        on: { click: () => {
          this.close();
          item.action();
        } },
      } ) );
    } );
    menu.addEventListener( "keydown" , ( key ) => {
      const buttons = Array.from( menu.querySelectorAll( "button" ) );
      const at = buttons.indexOf( document.activeElement );
      if ( key.key === "ArrowDown" || key.key === "ArrowUp" ) {
        key.preventDefault();
        const step = key.key === "ArrowDown" ? 1 : -1;
        buttons[ ( at + step + buttons.length ) % buttons.length ].focus();
      } else if ( key.key === "Escape" || key.key === "Tab" ) {
        key.preventDefault();
        key.stopPropagation();
        this.close();
      }
    } );
    // Inside an open dialog the menu has to live in the dialog to be on top.
    const host = event.currentTarget && event.currentTarget.closest ? event.currentTarget.closest( "dialog[open]" ) : null;
    ( host || document.body ).appendChild( menu );
    this.node = menu;
    this.returnFocus = event.currentTarget;

    // At the pointer; from the keyboard (menu key) at the element itself.
    let x = event.clientX;
    let y = event.clientY;
    if ( !x && !y && event.currentTarget && event.currentTarget.getBoundingClientRect ) {
      const box = event.currentTarget.getBoundingClientRect();
      x = box.left + 12;
      y = box.top + 12;
    }
    const width = menu.offsetWidth;
    const height = menu.offsetHeight;
    menu.style.left = Math.max( 4 , Math.min( x , window.innerWidth - width - 4 ) ) + "px";
    menu.style.top = Math.max( 4 , Math.min( y , window.innerHeight - height - 4 ) ) + "px";
    const first = menu.querySelector( "button" );
    if ( first ) first.focus();

    this.dismiss = ( other ) => {
      if ( other.type === "pointerdown" && menu.contains( other.target ) ) return;
      this.close();
    };
    window.setTimeout( () => {
      document.addEventListener( "pointerdown" , this.dismiss , true );
      window.addEventListener( "scroll" , this.dismiss , true );
      window.addEventListener( "resize" , this.dismiss );
      window.addEventListener( "blur" , this.dismiss );
    } , 0 );
  },

  close() {
    if ( !this.node ) return;
    const hadFocus = this.node.contains( document.activeElement );
    this.node.remove();
    this.node = null;
    document.removeEventListener( "pointerdown" , this.dismiss , true );
    window.removeEventListener( "scroll" , this.dismiss , true );
    window.removeEventListener( "resize" , this.dismiss );
    window.removeEventListener( "blur" , this.dismiss );
    if ( hadFocus && this.returnFocus && this.returnFocus.isConnected ) this.returnFocus.focus();
  },
};

// One dialog reused for every side panel (sharing, notifications, export,
// board settings). build( body ) fills it. back, when given, is where the
// panel came from -- Board settings, for the panels opened from its tiles --
// and puts a button above the body that returns there.
const Panel = {
  open( titleKey , build , back ) {
    const dialog = Dom.get( "panel-dialog" );
    Dom.text( Dom.get( "panel-title" ) , T( titleKey ) );
    const body = Dom.get( "panel-body" );
    Dom.clear( body );
    if ( back ) {
      body.appendChild( Dom.el( "button" , {
        class: Ui.buttonClass( "secondary small panel-back" ),
        text: T( "ui.back" ),
        attrs: { type: "button" },
        on: { click: back },
      } ) );
    }
    build( body );
    if ( !dialog.open ) dialog.showModal();
  },

  close() {
    const dialog = Dom.get( "panel-dialog" );
    if ( dialog.open ) dialog.close();
  },

  bind() {
    const close = Dom.get( "panel-close" );
    if ( close ) close.addEventListener( "click" , function () { Panel.close(); } );
    closeOnBackdrop( Dom.get( "panel-dialog" ) );
  },
};

// Text that shows rendered -- markdown and math -- and turns into an input
// when clicked, so nothing editable has to show raw TeX. Saves on Enter or
// blur, gives up on Escape.
//
//   options: { onSave( value ) , label , maxlength , className , textarea }
const InlineEdit = {
  build( text , options ) {
    const settings = options || {};
    const wrap = Dom.el( "span" , { class: "inline-edit " + ( settings.className || "" ) } );
    const show = function ( value ) {
      Dom.clear( wrap );
      const view = Dom.el( "span" , {
        class: "inline-edit-view",
        attrs: { tabindex: "0" , role: "button" , "aria-label": settings.label || "" , title: settings.label || "" },
      } );
      Markdown.renderInline( value , view );
      view.addEventListener( "click" , function ( event ) {
        if ( event.target.closest( "a" ) ) return;
        edit( value );
      } );
      view.addEventListener( "keydown" , function ( event ) {
        if ( event.key === "Enter" || event.key === " " ) {
          event.preventDefault();
          edit( value );
        }
      } );
      wrap.appendChild( view );
    };
    const edit = function ( value ) {
      Dom.clear( wrap );
      const input = Dom.el( settings.textarea ? "textarea" : "input" , {
        class: "inline-edit-input form-control",
        attrs: { maxlength: String( settings.maxlength || 300 ) , "aria-label": settings.label || "" , rows: "1" },
      } );
      if ( !settings.textarea ) input.type = "text";
      input.value = value;
      let done = false;
      const finish = function ( save ) {
        if ( done ) return;
        done = true;
        const next = input.value.trim();
        if ( save && next && next !== value ) {
          show( next );
          settings.onSave( next );
          return;
        }
        show( value );
      };
      input.addEventListener( "keydown" , function ( event ) {
        if ( event.key === "Enter" ) {
          event.preventDefault();
          finish( true );
        } else if ( event.key === "Escape" ) {
          event.preventDefault();
          event.stopPropagation();
          finish( false );
        }
      } );
      input.addEventListener( "blur" , function () { finish( true ); } );
      wrap.appendChild( input );
      input.focus();
      input.select();
    };
    show( String( text || "" ) );
    return wrap;
  },
};

function closeOnBackdrop( dialog ) {
  if ( !dialog ) return;
  dialog.addEventListener( "click" , function ( event ) {
    if ( event.target === dialog ) dialog.close();
  } );
}

// Dates. Cards carry RFC 3339 instants; an all-day date is midnight UTC of
// that day and is always shown and edited as a plain date, whatever the
// timezone.
//
// A timed value is shown on the clock of the person's chosen timezone
// (Dates.configure, from their settings), else the browser's. Calendar days
// -- what the views lay out and step through -- are plain Date objects at
// local midnight used only as a year/month/day holder; Dates.zoned and
// Dates.fromZoned convert between those and real instants in the zone.
const Dates = {
  DAY: 24 * 60 * 60 * 1000,
  zone: null,
  hour12: null,
  formatters: {},

  // settings: { timezone: "" | IANA name , hour12: null | bool }
  configure( settings ) {
    const value = settings || {};
    this.zone = value.timezone || null;
    this.hour12 = typeof value.hour12 === "boolean" ? value.hour12 : null;
    this.formatters = {};
    try {
      if ( this.zone ) new Intl.DateTimeFormat( undefined , { timeZone: this.zone } );
    } catch ( error ) {
      this.zone = null;
    }
  },

  browserZone() {
    try { return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"; } catch ( error ) { return "UTC"; }
  },

  timezones() {
    let zones = [];
    try { zones = Intl.supportedValuesOf( "timeZone" ); } catch ( error ) { zones = []; }
    if ( zones.indexOf( "UTC" ) < 0 ) zones.unshift( "UTC" );
    return zones;
  },

  // Intl options with the person's zone and clock applied.
  options( base , utc ) {
    const options = Object.assign( {} , base );
    if ( utc ) options.timeZone = "UTC";
    else if ( this.zone ) options.timeZone = this.zone;
    if ( options.hour !== undefined && options.hourCycle === undefined && this.hour12 !== null ) options.hour12 = this.hour12;
    return options;
  },

  formatter( base , utc ) {
    const key = JSON.stringify( base ) + ( utc ? "|utc" : "" );
    if ( !this.formatters[ key ] ) this.formatters[ key ] = new Intl.DateTimeFormat( undefined , this.options( base , utc ) );
    return this.formatters[ key ];
  },

  // The wall-clock parts of an instant in the configured zone.
  zoned( value ) {
    const date = value instanceof Date ? value : new Date( value );
    const parts = {};
    this.formatter( { year: "numeric" , month: "numeric" , day: "numeric" , hour: "numeric" , minute: "numeric" , hourCycle: "h23" } )
      .formatToParts( date ).forEach( function ( part ) { parts[ part.type ] = Number( part.value ); } );
    return { year: parts.year , month: parts.month , day: parts.day , hour: parts.hour % 24 , minute: parts.minute };
  },

  // The instant a wall-clock time in the configured zone names. Two passes
  // settle the zone offset, DST included.
  fromZoned( year , month , day , hour , minute ) {
    const wall = Date.UTC( year , month - 1 , day , hour || 0 , minute || 0 );
    let guess = wall;
    for ( let pass = 0; pass < 2; pass += 1 ) {
      const seen = this.zoned( new Date( guess ) );
      const offset = Date.UTC( seen.year , seen.month - 1 , seen.day , seen.hour , seen.minute ) - guess;
      guess = wall - offset;
    }
    return new Date( guess );
  },

  // An <input type=date> + <input type=time> pair, read in the zone.
  fromInputs( day , time ) {
    const [ year , month , date ] = day.split( "-" ).map( Number );
    const [ hour , minute ] = ( time || "00:00" ).split( ":" ).map( Number );
    return this.fromZoned( year , month , date , hour , minute );
  },

  // "14:05", for an <input type=time>, in the zone.
  timeInput( value ) {
    const parts = this.zoned( value );
    return String( parts.hour ).padStart( 2 , "0" ) + ":" + String( parts.minute ).padStart( 2 , "0" );
  },

  parse( value ) {
    return value ? new Date( value ) : null;
  },

  // "2026-10-30". With utc, the UTC day of an instant (all-day values);
  // otherwise the day a calendar-day Date holds.
  isoDay( date , utc ) {
    const year = utc ? date.getUTCFullYear() : date.getFullYear();
    const month = ( utc ? date.getUTCMonth() : date.getMonth() ) + 1;
    const day = utc ? date.getUTCDate() : date.getDate();
    return year + "-" + String( month ).padStart( 2 , "0" ) + "-" + String( day ).padStart( 2 , "0" );
  },

  // The calendar day a card value falls on in the zone.
  localDay( value , allDay ) {
    const date = new Date( value );
    if ( allDay ) return new Date( date.getUTCFullYear() , date.getUTCMonth() , date.getUTCDate() );
    const parts = this.zoned( date );
    return new Date( parts.year , parts.month - 1 , parts.day );
  },

  // The instant a calendar day starts in the zone.
  dayStart( day ) {
    return this.fromZoned( day.getFullYear() , day.getMonth() + 1 , day.getDate() , 0 , 0 );
  },

  today() {
    return this.localDay( new Date() , false );
  },

  addDays( date , days ) {
    const copy = new Date( date.getTime() );
    copy.setDate( copy.getDate() + days );
    return copy;
  },

  daysBetween( a , b ) {
    return Math.round( ( Date.UTC( b.getFullYear() , b.getMonth() , b.getDate() ) - Date.UTC( a.getFullYear() , a.getMonth() , a.getDate() ) ) / this.DAY );
  },

  format( value , allDay , withWeekday ) {
    if ( !value ) return "";
    const options = { month: "short" , day: "numeric" };
    if ( withWeekday ) options.weekday = "short";
    if ( allDay ) return this.formatter( options , true ).format( new Date( value ) );
    options.hour = "numeric";
    options.minute = "2-digit";
    return this.formatter( options ).format( new Date( value ) );
  },

  // "14:05" or "2:05 PM", per the person's clock.
  formatTime( value ) {
    return this.formatter( { hour: "numeric" , minute: "2-digit" } ).format( new Date( value ) );
  },

  // A card's dates as one line: "Oct 23, 10:00 – 11:30", "Oct 23 – Oct 25".
  formatSpan( card ) {
    const start = card.start_at;
    const due = card.due_at;
    if ( !start ) return this.format( due , card.due_all_day );
    if ( !due ) return this.format( start , card.start_all_day );
    const sameDay = !card.start_all_day && !card.due_all_day &&
      this.isoDay( this.localDay( start , false ) ) === this.isoDay( this.localDay( due , false ) );
    if ( sameDay ) return this.format( start , false ) + " – " + this.formatTime( due );
    return this.format( start , card.start_all_day ) + " – " + this.format( due , card.due_all_day );
  },

  // The moment a due date counts as passed: the instant itself, or the end
  // of the day for an all-day date.
  deadline( card ) {
    if ( !card.due_at ) return null;
    if ( card.due_all_day ) return Dates.dayStart( Dates.addDays( Dates.localDay( card.due_at , true ) , 1 ) );
    return new Date( card.due_at );
  },

  // overdue | soon (within a day) | later | done
  dueState( card ) {
    if ( !card.due_at ) return "";
    if ( card.done ) return "done";
    const deadline = this.deadline( card );
    const left = deadline.getTime() - Date.now();
    if ( left < 0 ) return "overdue";
    if ( left < this.DAY ) return "soon";
    return "later";
  },
};

// Pieces of a card face, shared by the kanban, todo and timeline views.
const CardBits = {
  labels( card , board ) {
    const wrap = Dom.el( "div" , { class: "card-labels" } );
    card.label_ids.forEach( function ( id ) {
      const label = board.labels.find( function ( item ) { return item.id === id; } );
      if ( !label ) return;
      wrap.appendChild( Dom.el( "span" , { class: "label-chip color-" + label.color , text: label.name , attrs: { title: label.name } } ) );
    } );
    return wrap.childNodes.length ? wrap : null;
  },

  due( card ) {
    if ( !card.due_at ) return null;
    return Dom.el( "span" , {
      class: "due-chip due-" + Dates.dueState( card ),
      text: Dates.format( card.due_at , card.due_all_day ),
      attrs: { title: T( "card.due_label" ) },
    } );
  },

  checklist( card ) {
    if ( !card.checklist.length ) return null;
    const done = card.checklist.filter( function ( item ) { return item.done; } ).length;
    return Dom.el( "span" , {
      class: "checklist-chip" + ( done === card.checklist.length ? " complete" : "" ),
      text: I18n.fill( "card.checklist_progress" , { done: done , total: card.checklist.length } ),
    } );
  },

  // The checklist itself on a card face, under the body; only when it has items.
  checklistItems( card ) {
    if ( !card.checklist.length ) return null;
    const list = Dom.el( "ul" , { class: "card-checklist" } );
    card.checklist.forEach( function ( item ) {
      const row = Dom.el( "li" , { class: item.done ? "is-done" : "" } );
      row.appendChild( Dom.el( "span" , { class: "card-checklist-box" , text: item.done ? "\u2611" : "\u2610" , attrs: { "aria-hidden": "true" } } ) );
      const text = Dom.el( "span" , { class: "card-checklist-text" } );
      Markdown.renderInline( item.text , text );
      row.appendChild( text );
      list.appendChild( row );
    } );
    return list;
  },

  initials( name ) {
    return String( name || "?" ).trim().split( /\s+/ ).slice( 0 , 2 ).map( function ( part ) { return part.charAt( 0 ).toUpperCase(); } ).join( "" );
  },

  assignees( card , people ) {
    if ( !card.assignees.length ) return null;
    const wrap = Dom.el( "div" , { class: "avatars" } );
    card.assignees.forEach( function ( id ) {
      const name = people[ id ] || "?";
      wrap.appendChild( Dom.el( "span" , { class: "avatar" , text: CardBits.initials( name ) , attrs: { title: name } } ) );
    } );
    return wrap;
  },

  hasDescription( card ) {
    return card.description && card.description.trim() !== "";
  },

  // "Sam (guest)" for a guest edit, the name otherwise.
  actorName( stored ) {
    const value = String( stored || "" );
    if ( value.indexOf( "guest:" ) === 0 ) {
      const name = value.slice( 6 ) || T( "share.anonymous" );
      return I18n.fill( "share.guest_name" , { name: name } );
    }
    return value;
  },

  // Midpoint position for a card dropped between two neighbours.
  positionBetween( before , after ) {
    if ( before == null && after == null ) return 1024;
    if ( before == null ) return after / 2;
    if ( after == null ) return before + 1024;
    return ( before + after ) / 2;
  },

  // Keeps a click or key inside a card face's own controls from also
  // opening the card.
  inert( node ) {
    [ "click" , "keydown" ].forEach( function ( name ) {
      node.addEventListener( name , function ( event ) { event.stopPropagation(); } );
    } );
    return node;
  },

  // A card's place in its column, 1-based. Typing another number and
  // pressing Enter moves the card there; Escape puts it back. cards is the
  // whole column in order (ctx.cardsIn), done cards and all.
  numberField( card , cards , ctx ) {
    const index = cards.indexOf( card );
    const label = String( index + 1 );
    const title = I18n.fill( "board.number_label" , { n: label } );
    if ( !ctx.canEdit ) return Dom.el( "span" , { class: "card-number" , text: label , attrs: { title: title } } );
    const input = Dom.el( "input" , { class: "card-number" , attrs: { type: "text" , inputmode: "numeric" , maxlength: "4" , "aria-label": title , title: title } } );
    input.value = label;
    this.inert( input );
    input.addEventListener( "focus" , function () { input.select(); } );
    input.addEventListener( "blur" , function () { input.value = label; } );
    input.addEventListener( "keydown" , function ( event ) {
      if ( event.key === "Escape" ) {
        input.value = label;
        input.blur();
        return;
      }
      if ( event.key !== "Enter" ) return;
      event.preventDefault();
      const wanted = parseInt( input.value , 10 );
      if ( isNaN( wanted ) ) {
        input.value = label;
        return;
      }
      const target = Math.min( Math.max( wanted , 1 ) , cards.length );
      if ( target === index + 1 ) {
        input.value = label;
        input.blur();
        return;
      }
      const others = cards.filter( function ( other ) { return other.id !== card.id; } );
      const before = others[ target - 2 ];
      const after = others[ target - 1 ];
      const position = CardBits.positionBetween( before ? before.position : null , after ? after.position : null );
      ctx.mutate( function () { return ctx.client.patchCard( card.id , { position: position } ); } );
    } );
    return input;
  },

  // Doing / Done icons, top right of a card. On a three-column board they
  // move the card to the second or third column, and pressing the lit one
  // sends it back to the first; they never touch the done flag. On any other
  // board there is just Done, which ticks the card's done flag.
  statusIcons( card , ctx ) {
    const wrap = this.inert( Dom.el( "div" , { class: "status-icons" } ) );
    const icon = function ( kind , pressed , enabled , onClick ) {
      const label = T( "board.status_" + kind );
      const button = Dom.el( "button" , {
        class: "status-icon status-" + kind,
        text: T( "board.status_" + kind + "_symbol" ),
        attrs: { type: "button" , "aria-pressed": pressed ? "true" : "false" , "aria-label": label , title: label },
        on: { click: onClick },
      } );
      button.disabled = !enabled;
      wrap.appendChild( button );
    };
    const lists = ctx.lists();
    if ( ctx.isThree() && lists.length >= 3 ) {
      [ [ "doing" , lists[ 1 ] ] , [ "done" , lists[ 2 ] ] ].forEach( function ( pair ) {
        const column = pair[ 1 ];
        const here = card.list_id === column.id;
        icon( pair[ 0 ] , here , ctx.canEdit , function () {
          const target = here ? lists[ 0 ] : column;
          const there = ctx.cardsIn( target.id ).filter( function ( other ) { return other.id !== card.id; } );
          const last = there[ there.length - 1 ];
          const position = CardBits.positionBetween( last ? last.position : null , null );
          ctx.mutate( function () { return ctx.client.patchCard( card.id , { list_id: target.id , position: position } ); } );
        } );
      } );
    } else {
      icon( "done" , card.done , card.access === "edit" , function () {
        ctx.mutate( function () { return ctx.client.patchCard( card.id , { done: !card.done } ); } );
      } );
    }
    return wrap;
  },

  // An archived card's one control, where the status icons would be.
  restoreIcon( card , ctx ) {
    const wrap = this.inert( Dom.el( "div" , { class: "status-icons" } ) );
    const button = Dom.el( "button" , {
      class: "status-icon status-restore",
      text: T( "card.restore_symbol" ),
      attrs: { type: "button" , "aria-label": T( "card.unarchive" ) , title: T( "card.unarchive" ) },
      on: { click: function () { ctx.mutate( function () { return ctx.client.patchCard( card.id , { archived: false } ); } ); } },
    } );
    button.disabled = !ctx.canEdit;
    wrap.appendChild( button );
    return wrap;
  },
};

// The board colour palette, the same closed set the server accepts.
const Colors = [ "blue" , "green" , "teal" , "yellow" , "orange" , "red" , "pink" , "purple" , "gray" ];
