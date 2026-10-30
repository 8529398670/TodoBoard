// Timeline: cards with dates as bars on a calendar, grouped by list. A card
// with a start and a due date spans them; one with only a due date is a
// one-day marker. Drag a bar to move it, drag either end to change that
// date, or drag an unscheduled card from the tray onto a day.
//
// Two kinds of scale. Days, Weeks and Months lay out whole calendar days.
// Hours lays out real time -- 24 columns a day -- so a card due at 14:30
// sits at 14:30, and drags snap to 15 minutes. Both read days and times on
// the person's clock (see Dates).
//
// Everything is plain DOM positioned through element.style (the CSSOM),
// which the page's style-src 'self' policy permits.
const TimelineView = {
  // Pixels per day; for "hour", pixels per hour.
  scales: { hour: 40 , day: 44 , week: 16 , month: 5 },
  SNAP_MINUTES: 15,
  MINUTE: 60 * 1000,

  zoom() {
    let saved = "";
    try { saved = window.localStorage.getItem( "timeline_zoom" ) || ""; } catch ( error ) { saved = ""; }
    return this.scales[ saved ] ? saved : "week";
  },

  // A card's span as calendar days (day zooms), or null when it has no dates.
  span( card ) {
    const start = card.start_at ? Dates.localDay( card.start_at , card.start_all_day ) : null;
    const due = card.due_at ? Dates.localDay( card.due_at , card.due_all_day ) : null;
    if ( !start && !due ) return null;
    let first = start || due;
    let last = due || start;
    if ( last < first ) {
      const swap = first;
      first = last;
      last = swap;
    }
    return { first: first , last: last , milestone: !start };
  },

  // A card's span as instants (hour zoom). All-day values cover their whole
  // day; a lone timed start or due is drawn an hour or half an hour long.
  instants( card ) {
    const edge = function ( value , allDay , end ) {
      if ( !allDay ) return new Date( value );
      const day = Dates.localDay( value , true );
      return Dates.dayStart( end ? Dates.addDays( day , 1 ) : day );
    };
    let from = card.start_at ? edge( card.start_at , card.start_all_day , false ) : null;
    let to = card.due_at ? edge( card.due_at , card.due_all_day , true ) : null;
    if ( !from && !to ) return null;
    if ( !from ) from = card.due_all_day ? edge( card.due_at , true , false ) : new Date( to.getTime() - 30 * this.MINUTE );
    if ( !to ) to = card.start_all_day ? edge( card.start_at , true , true ) : new Date( from.getTime() + 60 * this.MINUTE );
    if ( to < from ) {
      const swap = from;
      from = to;
      to = swap;
    }
    return { from: from , to: to , milestone: !card.start_at };
  },

  range( cards , zoom ) {
    const today = Dates.today();
    // Hours is for the days around now; a month of it would be 30,000px.
    if ( zoom === "hour" ) return { start: Dates.addDays( today , -2 ) , days: 15 };
    let min = today;
    let max = today;
    cards.forEach( ( card ) => {
      const span = this.span( card );
      if ( !span ) return;
      if ( span.first < min ) min = span.first;
      if ( span.last > max ) max = span.last;
    } );
    let start = Dates.addDays( min , -7 );
    let end = Dates.addDays( max , 21 );
    const minimum = { day: 45 , week: 120 , month: 400 }[ zoom ];
    if ( Dates.daysBetween( start , end ) < minimum ) end = Dates.addDays( start , minimum );
    if ( zoom !== "day" ) {
      // Start weeks on Monday, so the scale's labels line up.
      while ( start.getDay() !== 1 ) start = Dates.addDays( start , -1 );
    }
    return { start: start , days: Dates.daysBetween( start , end ) + 1 };
  },

  // The geometry every piece of the view shares: where a day or an instant
  // falls, in pixels from the left of the track.
  layout( range , zoom ) {
    const px = this.scales[ zoom ];
    if ( zoom !== "hour" ) {
      return {
        zoom: zoom,
        px: px,
        width: range.days * px,
        range: range,
        dayX: function ( day ) { return Dates.daysBetween( range.start , day ) * px; },
        nowX: function () { return Dates.daysBetween( range.start , Dates.today() ) * px + px / 2; },
        gridSize: px,
      };
    }
    const origin = Dates.dayStart( range.start ).getTime();
    const end = Dates.dayStart( Dates.addDays( range.start , range.days ) ).getTime();
    const perMinute = px / 60;
    const view = this;
    return {
      zoom: zoom,
      px: px,
      perMinute: perMinute,
      origin: origin,
      width: Math.round( ( end - origin ) / view.MINUTE * perMinute ),
      range: range,
      timeX: function ( instant ) { return ( instant.getTime() - origin ) / view.MINUTE * perMinute; },
      dayX: function ( day ) { return ( Dates.dayStart( day ).getTime() - origin ) / view.MINUTE * perMinute; },
      nowX: function () { return ( Date.now() - origin ) / view.MINUTE * perMinute; },
      gridSize: px,
    };
  },

  render( root , ctx ) {
    const zoom = this.zoom();
    const cards = ctx.cards.filter( function ( card ) { return !card.archived; } );
    const geo = this.layout( this.range( cards , zoom ) , zoom );
    const nowX = geo.nowX();

    const wrap = Dom.el( "div" , { class: "timeline" } );
    wrap.appendChild( this.toolbar( zoom , ctx ) );

    const scroller = Dom.el( "div" , { class: "timeline-scroll scroll-x" } );
    const grid = Dom.el( "div" , { class: "timeline-grid" } );

    const head = Dom.el( "div" , { class: "timeline-row timeline-head" + ( zoom === "hour" ? " hours" : "" ) } );
    head.appendChild( Dom.el( "div" , { class: "timeline-label timeline-corner" } ) );
    head.appendChild( this.scale( geo , nowX ) );
    grid.appendChild( head );

    const unscheduled = [];
    ctx.lists().forEach( ( list ) => {
      const listCards = ctx.cardsIn( list.id );
      const dated = listCards.filter( ( card ) => this.span( card ) );
      listCards.forEach( ( card ) => { if ( !this.span( card ) ) unscheduled.push( card ); } );

      const group = Dom.el( "div" , { class: "timeline-row timeline-group" } );
      const groupLabel = Dom.el( "div" , { class: "timeline-label" } );
      Markdown.renderInline( list.title , groupLabel );
      group.appendChild( groupLabel );
      group.appendChild( this.track( geo , nowX , list.id ) );
      grid.appendChild( group );

      dated.forEach( ( card ) => {
        const row = Dom.el( "div" , { class: "timeline-row" } );
        const label = Dom.el( "button" , { class: "timeline-label timeline-card-label" , attrs: { type: "button" } , on: { click: function () { ctx.openCard( card.id ); } } } );
        Markdown.renderInline( card.title , label );
        row.appendChild( label );
        const track = this.track( geo , nowX , list.id );
        const bar = this.bar( card , geo , ctx );
        if ( bar ) track.appendChild( bar );
        row.appendChild( track );
        grid.appendChild( row );
      } );
    } );

    scroller.appendChild( grid );
    wrap.appendChild( scroller );
    wrap.appendChild( this.tray( unscheduled , geo , ctx ) );
    root.appendChild( wrap );

    // Open scrolled to just before now, not at the start of the range.
    if ( !this.scrolledOnce ) {
      this.scrolledOnce = true;
      const lead = zoom === "hour" ? geo.px * 2 : geo.px * 7;
      window.requestAnimationFrame( function () { scroller.scrollLeft = Math.max( 0 , nowX - lead ); } );
    }
    this.lastScroller = scroller;
  },

  toolbar( zoom , ctx ) {
    const bar = Dom.el( "div" , { class: "timeline-toolbar row-actions" } );
    const select = Ui.select( [
      { value: "hour" , text: T( "timeline.zoom_hour" ) },
      { value: "day" , text: T( "timeline.zoom_day" ) },
      { value: "week" , text: T( "timeline.zoom_week" ) },
      { value: "month" , text: T( "timeline.zoom_month" ) },
    ] , zoom );
    select.addEventListener( "change" , () => {
      try { window.localStorage.setItem( "timeline_zoom" , select.value ); } catch ( error ) { /* per-tab only */ }
      this.scrolledOnce = false;
      BoardPage.render();
    } );
    bar.appendChild( Ui.field( "timeline.zoom_label" , select ) );
    bar.appendChild( Ui.button( zoom === "hour" ? "timeline.now" : "timeline.today" , () => {
      this.scrolledOnce = false;
      BoardPage.render();
    } ) );
    if ( ctx.canEdit ) bar.appendChild( Dom.el( "p" , { class: "muted" , text: T( zoom === "hour" ? "timeline.hour_hint" : "timeline.hint" ) } ) );
    return bar;
  },

  scale( geo , nowX ) {
    const scale = Dom.el( "div" , { class: "timeline-scale" + ( geo.zoom === "hour" ? " hours" : "" ) } );
    scale.style.width = geo.width + "px";
    const range = geo.range;
    const tick = function ( x , text , classes , width ) {
      const node = Dom.el( "span" , { class: "tick" + classes , text: text } );
      node.style.left = x + "px";
      if ( width ) node.style.width = width + "px";
      scale.appendChild( node );
    };

    if ( geo.zoom === "hour" ) {
      const dayFormat = Dates.formatter( { weekday: "short" , day: "numeric" , month: "short" } );
      const hourFormat = Dates.formatter( { hour: "numeric" } );
      for ( let index = 0; index < range.days; index += 1 ) {
        const day = Dates.addDays( range.start , index );
        const dayStart = Dates.dayStart( day );
        const next = Dates.dayStart( Dates.addDays( day , 1 ) );
        const weekend = day.getDay() === 0 || day.getDay() === 6 ? " weekend" : "";
        tick( geo.timeX( dayStart ) , dayFormat.format( dayStart ) , " major" + weekend );
        // Every hour gets a line, every other one a label. Stepping real
        // instants keeps a DST day at its true 23 or 25 hours.
        let hour = 1;
        for ( let at = dayStart.getTime() + 60 * this.MINUTE; at < next.getTime(); at += 60 * this.MINUTE , hour += 1 ) {
          tick( geo.timeX( new Date( at ) ) , hour % 2 === 0 ? hourFormat.format( new Date( at ) ) : "" , " hour" + weekend );
        }
      }
    } else {
      const monthFormat = Dates.formatter( { month: "short" , year: "numeric" } );
      const dayFormat = Dates.formatter( { weekday: "narrow" , day: "numeric" } );
      const weekFormat = Dates.formatter( { day: "numeric" , month: "short" } );
      for ( let index = 0; index < range.days; index += 1 ) {
        const day = Dates.addDays( range.start , index );
        let text = "";
        let major = false;
        if ( geo.zoom === "day" ) {
          text = dayFormat.format( Dates.dayStart( day ) );
          major = day.getDate() === 1;
        } else if ( geo.zoom === "week" ) {
          if ( day.getDay() === 1 ) text = weekFormat.format( Dates.dayStart( day ) );
          major = day.getDate() === 1;
        } else if ( day.getDate() === 1 ) {
          text = monthFormat.format( Dates.dayStart( day ) );
          major = true;
        }
        if ( !text && !major ) continue;
        tick( index * geo.px , text , ( major ? " major" : "" ) + ( day.getDay() === 0 || day.getDay() === 6 ? " weekend" : "" ) , geo.zoom === "day" ? geo.px : 0 );
      }
    }
    if ( nowX >= 0 && nowX <= geo.width ) {
      const marker = Dom.el( "span" , { class: "today-line" , attrs: { title: T( geo.zoom === "hour" ? "timeline.now" : "timeline.today" ) } } );
      marker.style.left = nowX + "px";
      scale.appendChild( marker );
    }
    return scale;
  },

  track( geo , nowX , listId ) {
    const track = Dom.el( "div" , { class: "timeline-track" , attrs: { "data-list-id": listId } } );
    track.style.width = geo.width + "px";
    track.style.backgroundSize = geo.gridSize + "px 100%";
    if ( nowX >= 0 && nowX <= geo.width ) {
      const marker = Dom.el( "span" , { class: "today-line" } );
      marker.style.left = nowX + "px";
      track.appendChild( marker );
    }
    return track;
  },

  bar( card , geo , ctx ) {
    let left;
    let width;
    let milestone;
    if ( geo.zoom === "hour" ) {
      const span = this.instants( card );
      left = geo.timeX( span.from );
      width = geo.timeX( span.to ) - left;
      milestone = span.milestone;
      // Off the visible stretch entirely: nothing to draw.
      if ( left + width < 0 || left > geo.width ) return null;
    } else {
      const span = this.span( card );
      left = geo.dayX( span.first );
      width = ( Dates.daysBetween( span.first , span.last ) + 1 ) * geo.px;
      milestone = span.milestone;
    }
    const label = card.label_ids.length ? ctx.board.labels.find( function ( item ) { return item.id === card.label_ids[ 0 ]; } ) : null;
    const bar = Dom.el( "div" , {
      class: "timeline-bar color-" + ( label ? label.color : "none" ) + ( milestone ? " milestone" : "" ) + ( card.done ? " is-done" : "" ) + " due-" + Dates.dueState( card ),
      attrs: { tabindex: "0" , role: "button" , title: card.title + " · " + Dates.formatSpan( card ) },
    } );
    bar.style.left = left + "px";
    bar.style.width = Math.max( width , geo.zoom === "hour" ? geo.px / 4 : geo.px ) + "px";
    if ( width < 72 ) bar.classList.add( "narrow" );
    const text = Dom.el( "span" , { class: "bar-text" } );
    Markdown.renderInline( card.title , text );
    bar.appendChild( text );
    bar.addEventListener( "keydown" , function ( event ) { if ( event.key === "Enter" ) ctx.openCard( card.id ); } );

    if ( card.access !== "edit" ) {
      bar.addEventListener( "click" , function () { ctx.openCard( card.id ); } );
      return bar;
    }
    if ( !milestone ) bar.appendChild( Dom.el( "span" , { class: "bar-handle start" , attrs: { "data-edge": "start" } } ) );
    bar.appendChild( Dom.el( "span" , { class: "bar-handle end" , attrs: { "data-edge": "end" } } ) );
    this.makeDraggable( bar , card , geo , ctx );
    return bar;
  },

  // How far one drag step is, in pixels, for this card's edge. Day zooms
  // step a day. Hours steps 15 minutes, except an all-day edge, which moves
  // by whole days and stays all-day -- unless it is being resized, which
  // turns that edge into a time.
  step( card , geo , edge ) {
    if ( geo.zoom !== "hour" ) return { px: geo.px , minutes: 1440 };
    const allDay = ( edge !== "end" && card.start_at && card.start_all_day ) || ( edge !== "start" && card.due_at && card.due_all_day );
    if ( edge === "move" && allDay ) return { px: geo.perMinute * 1440 , minutes: 1440 };
    return { px: geo.perMinute * this.SNAP_MINUTES , minutes: this.SNAP_MINUTES };
  },

  makeDraggable( bar , card , geo , ctx ) {
    bar.addEventListener( "pointerdown" , ( event ) => {
      if ( event.button !== 0 ) return;
      const edge = event.target.getAttribute( "data-edge" ) || "move";
      const step = this.step( card , geo , edge );
      const originX = event.clientX;
      const left = parseFloat( bar.style.left );
      const width = parseFloat( bar.style.width );
      const minWidth = Math.min( width , step.px );
      let steps = 0;
      let moved = false;
      try { bar.setPointerCapture( event.pointerId ); } catch ( error ) { /* synthetic or already-released pointer */ }
      ctx.setDragging( true );

      const onMove = function ( moveEvent ) {
        const dx = moveEvent.clientX - originX;
        if ( Math.abs( dx ) > 4 ) moved = true;
        steps = Math.round( dx / step.px );
        const shift = steps * step.px;
        if ( edge === "move" ) {
          bar.style.left = ( left + shift ) + "px";
        } else if ( edge === "start" ) {
          const clamped = Math.min( shift , width - minWidth );
          bar.style.left = ( left + clamped ) + "px";
          bar.style.width = ( width - clamped ) + "px";
        } else {
          bar.style.width = Math.max( minWidth , width + shift ) + "px";
        }
      };
      const onUp = () => {
        bar.removeEventListener( "pointermove" , onMove );
        bar.removeEventListener( "pointerup" , onUp );
        bar.removeEventListener( "pointercancel" , onUp );
        ctx.setDragging( false );
        if ( !moved ) {
          ctx.openCard( card.id );
          return;
        }
        if ( steps === 0 ) return;
        const patch = geo.zoom === "hour" ? this.shiftedTime( card , edge , steps * step.minutes ) : this.shifted( card , this.span( card ) , edge , steps );
        if ( patch ) ctx.mutate( function () { return ctx.client.patchCard( card.id , patch ); } );
      };
      bar.addEventListener( "pointermove" , onMove );
      bar.addEventListener( "pointerup" , onUp );
      bar.addEventListener( "pointercancel" , onUp );
    } );
  },

  // The patch for dragging a card's span by delta days. A timed date keeps
  // its time of day; all-day dates stay all-day.
  shifted( card , span , edge , delta ) {
    const patch = {};
    const move = function ( value , allDay , day ) {
      if ( value && !allDay ) return new Date( new Date( value ).getTime() + delta * Dates.DAY ).toISOString();
      return Dates.isoDay( day );
    };
    let first = span.first;
    let last = span.last;
    if ( edge === "move" || edge === "start" ) first = Dates.addDays( first , delta );
    if ( edge === "move" || edge === "end" ) last = Dates.addDays( last , delta );
    if ( last < first ) last = first;

    if ( edge !== "end" && ( card.start_at || edge === "start" ) ) patch.start_at = move( card.start_at , card.start_all_day , first );
    if ( edge !== "start" && ( card.due_at || edge === "end" ) ) patch.due_at = move( card.due_at , card.due_all_day , last );
    return patch;
  },

  // The patch for dragging by a number of minutes on the Hours scale. A
  // whole-day move of an all-day date keeps it all-day; anything else lands
  // on an exact time.
  shiftedTime( card , edge , minutes ) {
    const span = this.instants( card );
    const delta = minutes * this.MINUTE;
    const patch = {};
    const wholeDays = minutes % 1440 === 0;
    const moveValue = function ( value , allDay , instant ) {
      if ( allDay && wholeDays ) return Dates.isoDay( Dates.addDays( Dates.localDay( value , true ) , minutes / 1440 ) );
      return new Date( instant.getTime() + delta ).toISOString();
    };
    if ( edge === "move" ) {
      if ( card.start_at ) patch.start_at = moveValue( card.start_at , card.start_all_day , span.from );
      if ( card.due_at ) patch.due_at = moveValue( card.due_at , card.due_all_day , span.to );
      return patch;
    }
    if ( edge === "start" ) {
      const from = new Date( span.from.getTime() + delta );
      if ( from >= span.to ) return null;
      patch.start_at = from.toISOString();
      return patch;
    }
    const to = new Date( span.to.getTime() + delta );
    if ( to <= span.from ) return null;
    patch.due_at = to.toISOString();
    return patch;
  },

  tray( cards , geo , ctx ) {
    const tray = Dom.el( "section" , { class: "timeline-tray card card-body" } );
    tray.appendChild( Dom.el( "h2" , { text: T( "timeline.unscheduled" ) } ) );
    if ( !cards.length ) {
      tray.appendChild( Dom.el( "p" , { class: "muted" , text: T( "timeline.none_unscheduled" ) } ) );
      return tray;
    }
    if ( ctx.canEdit ) tray.appendChild( Dom.el( "p" , { class: "muted" , text: T( "timeline.tray_hint" ) } ) );
    const items = Dom.el( "div" , { class: "tray-items" } );
    cards.forEach( ( card ) => {
      const chip = Dom.el( "button" , { class: "tray-chip" , attrs: { type: "button" } } );
      Markdown.renderInline( card.title , chip );
      if ( card.access === "edit" ) this.makeSchedulable( chip , card , geo , ctx );
      else chip.addEventListener( "click" , function () { ctx.openCard( card.id ); } );
      items.appendChild( chip );
    } );
    tray.appendChild( items );
    return tray;
  },

  // Drag a tray chip onto any track: on a day scale it becomes due (all
  // day) on that day; on Hours, due at the 15-minute slot it was dropped
  // on. A plain tap opens the card instead.
  makeSchedulable( chip , card , geo , ctx ) {
    chip.addEventListener( "pointerdown" , ( event ) => {
      if ( event.button !== 0 ) return;
      const originX = event.clientX;
      const originY = event.clientY;
      let ghost = null;
      try { chip.setPointerCapture( event.pointerId ); } catch ( error ) { /* synthetic or already-released pointer */ }
      ctx.setDragging( true );

      const onMove = function ( moveEvent ) {
        if ( !ghost && Math.hypot( moveEvent.clientX - originX , moveEvent.clientY - originY ) > 6 ) {
          ghost = Dom.el( "div" , { class: "drag-float" } );
          Markdown.renderInline( card.title , ghost );
          document.body.appendChild( ghost );
        }
        if ( ghost ) {
          ghost.style.left = ( moveEvent.clientX + 8 ) + "px";
          ghost.style.top = ( moveEvent.clientY + 8 ) + "px";
        }
      };
      const onUp = ( upEvent ) => {
        chip.removeEventListener( "pointermove" , onMove );
        chip.removeEventListener( "pointerup" , onUp );
        chip.removeEventListener( "pointercancel" , onUp );
        ctx.setDragging( false );
        if ( !ghost ) {
          ctx.openCard( card.id );
          return;
        }
        ghost.remove();
        const target = document.elementFromPoint( upEvent.clientX , upEvent.clientY );
        const track = target ? target.closest( ".timeline-track" ) : null;
        if ( !track ) return;
        const x = upEvent.clientX - track.getBoundingClientRect().left;
        let due;
        if ( geo.zoom === "hour" ) {
          const minutes = Math.round( x / geo.perMinute / this.SNAP_MINUTES ) * this.SNAP_MINUTES;
          due = new Date( geo.origin + minutes * this.MINUTE ).toISOString();
        } else {
          due = Dates.isoDay( Dates.addDays( geo.range.start , Math.floor( x / geo.px ) ) );
        }
        ctx.mutate( function () { return ctx.client.patchCard( card.id , { due_at: due } ); } );
      };
      chip.addEventListener( "pointermove" , onMove );
      chip.addEventListener( "pointerup" , onUp );
      chip.addEventListener( "pointercancel" , onUp );
    } );
  },
};
