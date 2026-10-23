// Exporting one card: as a PNG (copied to the clipboard or downloaded) or as
// Markdown (copied or downloaded). The Markdown keeps the description exactly
// as written, so $math$ round-trips into any other Markdown tool.
//
// The PNG is drawn from an off-screen "export card" that always uses the
// light palette, so an image pasted into a document looks the same whichever
// theme the person exporting it had on.
const CardExport = {
  open( ctx , card ) {
    Panel.open( "export.card_title" , ( body ) => {
      body.appendChild( Dom.el( "h3" , { text: T( "export.image_heading" ) } ) );
      body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Dom.el( "button" , { class: "small" , text: T( "export.copy_image" ) , attrs: { type: "button" } , on: { click: () => this.copyImage( ctx , card ) } } ),
        Ui.button( "export.download_image" , () => this.downloadImage( ctx , card ) ),
      ] } ) );
      body.appendChild( Dom.el( "h3" , { text: T( "export.markdown_heading" ) } ) );
      body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Dom.el( "button" , { class: "small" , text: T( "export.copy_markdown" ) , attrs: { type: "button" } , on: { click: () => Ui.copyText( this.markdown( ctx , card ) ) } } ),
        Ui.button( "export.download_markdown" , () => {
          Ui.download( new Blob( [ this.markdown( ctx , card ) ] , { type: "text/markdown;charset=utf-8" } ) , Ui.slug( card.title ) + ".md" );
        } ),
      ] } ) );
      const preview = Dom.el( "pre" , { class: "markdown-source" , text: this.markdown( ctx , card ) } );
      body.appendChild( preview );
    } );
  },

  listTitle( ctx , card ) {
    const list = ctx.board.lists.find( function ( item ) { return item.id === card.list_id; } );
    return list ? list.title : "";
  },

  labelNames( ctx , card ) {
    return card.label_ids.map( function ( id ) {
      const label = ctx.board.labels.find( function ( item ) { return item.id === id; } );
      return label ? ( label.name || label.color ) : "";
    } ).filter( Boolean );
  },

  dateText( value , allDay ) {
    if ( !value ) return "";
    if ( allDay ) return Dates.isoDay( new Date( value ) , true );
    return Dates.isoDay( Dates.localDay( value , false ) ) + " " + Dates.timeInput( value );
  },

  // headingLevel lets the board export nest cards under list headings.
  markdown( ctx , card , headingLevel ) {
    const level = headingLevel || 1;
    const hashes = "#".repeat( level );
    const lines = [ hashes + " " + ( card.done ? T( "export.md_done_marker" ) + " " : "" ) + card.title , "" ];
    const meta = [];
    if ( level === 1 ) {
      meta.push( "**" + T( "export.md_board" ) + ":** " + ctx.board.title );
      meta.push( "**" + T( "export.md_list" ) + ":** " + this.listTitle( ctx , card ) );
    }
    const labels = this.labelNames( ctx , card );
    if ( labels.length ) meta.push( "**" + T( "export.md_labels" ) + ":** " + labels.join( ", " ) );
    if ( card.start_at ) meta.push( "**" + T( "export.md_start" ) + ":** " + this.dateText( card.start_at , card.start_all_day ) );
    if ( card.due_at ) meta.push( "**" + T( "export.md_due" ) + ":** " + this.dateText( card.due_at , card.due_all_day ) );
    if ( card.assignees.length ) {
      meta.push( "**" + T( "export.md_assignees" ) + ":** " + card.assignees.map( function ( id ) { return ctx.people[ id ] || "?"; } ).join( ", " ) );
    }
    if ( card.done ) meta.push( "**" + T( "export.md_status" ) + ":** " + T( "card.done_label" ) );
    if ( meta.length ) {
      lines.push( meta.join( "  \n" ) );
      lines.push( "" );
    }
    if ( CardBits.hasDescription( card ) ) {
      lines.push( card.description.trim() );
      lines.push( "" );
    }
    if ( card.checklist.length ) {
      lines.push( hashes + "# " + T( "card.checklist_heading" ) );
      lines.push( "" );
      card.checklist.forEach( function ( item ) {
        lines.push( "- [" + ( item.done ? "x" : " " ) + "] " + item.text );
      } );
      lines.push( "" );
    }
    return lines.join( "\n" );
  },

  // The element the PNG is drawn from.
  buildNode( ctx , card ) {
    const node = Dom.el( "article" , { class: "export-card" } );
    const crumbs = Dom.el( "p" , { class: "export-crumbs" , text: ctx.board.title + " › " + this.listTitle( ctx , card ) } );
    node.appendChild( crumbs );
    const labels = CardBits.labels( card , ctx.board );
    if ( labels ) node.appendChild( labels );
    const title = Dom.el( "h1" , { class: "export-title" + ( card.done ? " is-done" : "" ) } );
    Markdown.renderInline( card.title , title );
    node.appendChild( title );

    const meta = Dom.el( "div" , { class: "card-meta" } );
    if ( card.start_at ) meta.appendChild( Dom.el( "span" , { class: "due-chip" , text: T( "card.start_label" ) + " " + Dates.format( card.start_at , card.start_all_day ) } ) );
    const due = CardBits.due( card );
    if ( due ) meta.appendChild( due );
    if ( card.done ) meta.appendChild( Dom.el( "span" , { class: "due-chip due-done" , text: T( "card.done_label" ) } ) );
    const people = CardBits.assignees( card , ctx.people );
    if ( people ) meta.appendChild( people );
    if ( meta.childNodes.length ) node.appendChild( meta );

    if ( CardBits.hasDescription( card ) ) {
      const description = Dom.el( "div" , { class: "description" } );
      Markdown.render( card.description , description );
      node.appendChild( description );
    }
    if ( card.checklist.length ) {
      const list = Dom.el( "ul" , { class: "checklist" } );
      card.checklist.forEach( function ( item ) {
        const row = Dom.el( "li" , { class: "checklist-item" + ( item.done ? " is-done" : "" ) } );
        row.appendChild( Dom.el( "span" , { class: "export-box" , text: item.done ? T( "export.box_checked" ) : T( "export.box_empty" ) } ) );
        const text = Dom.el( "span" , { class: "item-text" } );
        Markdown.renderInline( item.text , text );
        row.appendChild( text );
        list.appendChild( row );
      } );
      node.appendChild( list );
    }
    return node;
  },

  async toBlob( ctx , card ) {
    const stage = Dom.get( "export-stage" );
    const node = this.buildNode( ctx , card );
    stage.appendChild( node );
    try {
      if ( document.fonts && document.fonts.ready ) await document.fonts.ready;
      await this.settleImages( node );
      return await htmlToImage.toBlob( node , {
        pixelRatio: 2,
        backgroundColor: "#ffffff",
        // KaTeX's stylesheet lists each font three times; only the woff2
        // copies are vendored, and they are what every browser uses.
        preferredFontFormat: "woff2",
      } );
    } finally {
      node.remove();
    }
  },

  // One broken image in a description would make html-to-image reject the
  // whole export, so wait for each to load and swap the ones that do not
  // for their alt text.
  async settleImages( node ) {
    const images = Dom.all( "img" , node );
    await Promise.all( images.map( function ( image ) {
      return new Promise( function ( resolve ) {
        const done = function () {
          if ( !image.complete || image.naturalWidth === 0 ) {
            image.replaceWith( document.createTextNode( image.getAttribute( "alt" ) || "" ) );
          }
          resolve();
        };
        if ( image.complete ) {
          done();
          return;
        }
        image.addEventListener( "load" , done , { once: true } );
        image.addEventListener( "error" , done , { once: true } );
        window.setTimeout( done , 3000 );
      } );
    } ) );
  },

  async downloadImage( ctx , card ) {
    try {
      const blob = await this.toBlob( ctx , card );
      Ui.download( blob , Ui.slug( card.title ) + ".png" );
    } catch ( error ) {
      Ui.toast( T( "export.image_failed" ) );
    }
  },

  // ClipboardItem is handed the *promise* of the image: Safari only allows a
  // clipboard write that starts inside the click, before any await.
  async copyImage( ctx , card ) {
    if ( typeof ClipboardItem === "undefined" || !navigator.clipboard || !navigator.clipboard.write ) {
      Ui.toast( T( "export.clipboard_unsupported" ) );
      return;
    }
    try {
      await navigator.clipboard.write( [ new ClipboardItem( { "image/png": this.toBlob( ctx , card ) } ) ] );
      Ui.toast( T( "export.copied_image" ) , "ok" );
    } catch ( error ) {
      Ui.toast( T( "export.clipboard_unsupported" ) );
    }
  },
};

// Exporting a whole board: Markdown (copy or download), or the print layout,
// which the browser saves as a PDF with real, selectable text and math.
const BoardExport = {
  open( ctx ) {
    Panel.open( "export.board_title" , ( body ) => {
      body.appendChild( Dom.el( "h3" , { text: T( "export.markdown_heading" ) } ) );
      body.appendChild( Dom.el( "div" , { class: "row-actions" , children: [
        Dom.el( "button" , { class: "small" , text: T( "export.copy_markdown" ) , attrs: { type: "button" } , on: { click: () => Ui.copyText( this.markdown( ctx ) ) } } ),
        Ui.button( "export.download_markdown" , () => {
          Ui.download( new Blob( [ this.markdown( ctx ) ] , { type: "text/markdown;charset=utf-8" } ) , Ui.slug( ctx.board.title ) + ".md" );
        } ),
      ] } ) );
      body.appendChild( Dom.el( "h3" , { text: T( "export.pdf_heading" ) } ) );
      body.appendChild( Dom.el( "p" , { class: "muted" , text: T( "export.pdf_hint" ) } ) );
      body.appendChild( Dom.el( "a" , {
        class: "button-link",
        text: T( "export.open_print" ),
        attrs: { href: this.printPath( ctx ) , target: "_blank" , rel: "noopener" },
      } ) );
    } );
  },

  printPath( ctx ) {
    if ( ctx.isGuest ) return "/print/s/" + encodeURIComponent( ctx.client.shareToken );
    return "/print/b/" + ctx.board.id;
  },

  markdown( ctx ) {
    const lines = [ "# " + ctx.board.title , "" ];
    ctx.lists().forEach( function ( list ) {
      lines.push( "## " + list.title );
      lines.push( "" );
      const cards = ctx.cardsIn( list.id );
      if ( !cards.length ) {
        lines.push( "_" + T( "export.md_empty_list" ) + "_" );
        lines.push( "" );
      }
      cards.forEach( function ( card ) {
        lines.push( CardExport.markdown( ctx , card , 3 ) );
      } );
    } );
    return lines.join( "\n" ).replace( /\n{3,}/g , "\n\n" );
  },
};
