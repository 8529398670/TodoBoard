// Stages: on a staged board every list holds To do, Doing and Done and shows
// one at a time. A card's stage comes from its done and doing flags. Which
// stage a column is showing is BoardPage's; this file is the controls that
// change it -- the bar in the navbar, swiping and the arrow keys, which all
// move every column at once -- and the words for the flags. A column that
// wants a stage of its own has that in its settings (List.stage_view). The bar
// has one more stop than a column: All, one centred list of every list's
// cards, which flips through the stages with its own tabs.
const Stages = {
  order: [ "todo" , "doing" , "done" ],

  // One full-width column at a time. The query is board.css's for
  // .swipe-columns, word for word. Every column follows the bar here, even
  // one with a stage of its own: see BoardPage.columnView.
  query: window.matchMedia( "(max-width: 39.99rem) and (orientation: portrait)" ),

  singleColumn() {
    return document.body.classList.contains( "swipe-columns" ) && this.query.matches;
  },

  of( card ) {
    if ( card.done ) return "done";
    return card.doing ? "doing" : "todo";
  },

  // A patch or new card's fields for a stage. Both flags always, so undoing
  // comes back to the stage the card left, not just the flag it lost.
  fields( stage ) {
    return { doing: stage === "doing" , done: stage === "done" };
  },

  // A stage's name: the board's own if it has one, the app's otherwise. All
  // is no stage, so it always has the app's word; a link that is not told the
  // board's settings falls back the same way.
  label( ctx , stage ) {
    return ( stage !== "all" && ctx.board[ "stage_" + stage ] ) || T( "board.stage_" + stage );
  },

  // Where the bar and the arrow keys can go.
  boardStages( ctx ) {
    return ctx.showAll() ? this.order.concat( "all" ) : this.order;
  },

  // Buttons for stages; pick is called with the one pressed. counts, by
  // stage, are shown beside the names when given.
  buttons( ctx , className , label , stages , current , counts , pick ) {
    const group = Dom.el( "div" , { class: className , attrs: { role: "group" , "aria-label": T( label ) } } );
    stages.forEach( ( stage ) => {
      const name = this.label( ctx , stage );
      const button = Dom.el( "button" , {
        class: "stage-button" + ( stage === current ? " active" : "" ),
        attrs: { type: "button" , "aria-pressed": stage === current ? "true" : "false" , "data-stage": stage , title: name },
        on: { click: function () { pick( stage ); } },
      } );
      button.appendChild( Dom.el( "span" , { class: "stage-name" , text: name } ) );
      if ( counts ) button.appendChild( Dom.el( "span" , { class: "stage-count" , text: String( counts[ stage ] ) } ) );
      group.appendChild( button );
    } );
    return group;
  },

  // Flips every column at once, or goes to the All list.
  bar( ctx ) {
    return this.buttons( ctx , "stage-bar" , "board.stage_bar" , this.boardStages( ctx ) , ctx.boardStage() , null , function ( stage ) { BoardPage.setStage( stage ); } );
  },

  // Flips the All list, which shows its stage apart from the bar.
  tabs( cards , ctx ) {
    const counts = { todo: 0 , doing: 0 , done: 0 };
    cards.forEach( ( card ) => { counts[ this.of( card ) ] += 1; } );
    return this.buttons( ctx , "stage-tabs" , "board.stage_tabs" , this.order , ctx.stageIn( "all" ) , counts , function ( stage ) { BoardPage.setAllStage( stage ); } );
  },

  // Both are bound once, on the document: a render replaces every column,
  // so anything bound to one would not outlive the next poll.
  bind() {
    this.bindSwipe();
    this.bindKeys();
    this.bindRotate();
  },

  // Turning a phone either way changes whether columns keep a stage of their
  // own or stay collapsed, so they have to be drawn again. Only then, and
  // never mid-drag or mid-typing: a keyboard opening can flip the orientation too, and a
  // render would take a half-typed card with it.
  bindRotate() {
    this.query.addEventListener( "change" , function () {
      if ( !BoardPage.data || BoardPage.dragging ) return;
      if ( !document.body.classList.contains( "swipe-columns" ) ) return;
      const active = document.activeElement;
      if ( active && active.closest && active.closest( "input, textarea, select, [contenteditable]:not([contenteditable=\"false\"])" ) ) return;
      if ( BoardPage.collapsed.length || BoardPage.data.board.lists.some( function ( list ) { return list.stage_view; } ) ) BoardPage.render();
    } );
  },

  // A sideways swipe on a column's cards steps the whole board, as the bar
  // shows; CSS leaves sideways touches on the cards to us
  // (touch-action: pan-y). Touch only:
  // a mouse drags cards, and a trackpad pans the board. Holding still is
  // left to Sortable, which starts a drag; moving first cancels that.
  bindSwipe() {
    let start = null;
    const stale = function ( key ) {
      return !BoardPage.data || BoardPage.dragging || key === null;
    };
    document.addEventListener( "pointerdown" , function ( event ) {
      start = null;
      if ( event.pointerType !== "touch" || !event.isPrimary || !event.target.closest ) return;
      const area = event.target.closest( ".kanban.staged .kanban-cards, .all-view .kanban-cards" );
      if ( !area ) return;
      const column = area.closest( "[data-stage-key]" );
      if ( column ) start = { x: event.clientX , y: event.clientY , key: column.getAttribute( "data-stage-key" ) };
    } , { passive: true } );
    document.addEventListener( "pointercancel" , function () { start = null; } , { passive: true } );
    document.addEventListener( "pointerup" , ( event ) => {
      const from = start;
      start = null;
      if ( !from || event.pointerType !== "touch" ) return;
      const dx = event.clientX - from.x;
      const dy = event.clientY - from.y;
      if ( Math.abs( dx ) < 50 || Math.abs( dx ) < 1.5 * Math.abs( dy ) || stale( from.key ) ) return;
      this.swallowClick();
      BoardPage.stepStage( dx < 0 ? 1 : -1 );
    } , { passive: true } );
  },

  // The tap a browser may still report at the end of a swipe, which would
  // otherwise open the card under the finger.
  swallowClick() {
    const swallow = function ( event ) {
      event.preventDefault();
      event.stopPropagation();
    };
    document.addEventListener( "click" , swallow , { capture: true , once: true } );
    window.setTimeout( function () { document.removeEventListener( "click" , swallow , { capture: true } ); } , 400 );
  },

  // Left and Right flip the whole board. Not while typing, in a dialog or a
  // menu, or mid-drag, and only on the board view, where stages show.
  bindKeys() {
    document.addEventListener( "keydown" , function ( event ) {
      if ( event.key !== "ArrowLeft" && event.key !== "ArrowRight" ) return;
      if ( event.ctrlKey || event.metaKey || event.altKey || event.shiftKey ) return;
      const target = event.target;
      if ( target && target.closest && target.closest( "input, textarea, select, [contenteditable]:not([contenteditable=\"false\"])" ) ) return;
      if ( !BoardPage.data || BoardPage.view !== "kanban" || BoardPage.dragging ) return;
      const ctx = BoardPage.context();
      if ( !ctx.isStaged() || ctx.cardOnly ) return;
      if ( document.querySelector( "dialog[open], .context-menu" ) ) return;
      event.preventDefault();
      BoardPage.stepStage( event.key === "ArrowRight" ? 1 : -1 );
    } );
  },
};
