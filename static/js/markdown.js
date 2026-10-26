// Markdown with math: `$H_2O$` inline, and
//
//   $$
//   \ce{H2O}
//   $$
//
// as a display block. mhchem's \ce{} and \pu{} work in both.
//
// The pipeline, and why each step is where it is:
//
//   1. marked parses the text. Two extensions recognise math *as tokens*, so
//      `$x$` inside a code span or a fenced block stays code -- a regex over
//      the raw text could not tell. Math becomes an empty placeholder element
//      carrying the TeX in a data attribute.
//   2. DOMPurify sanitises marked's HTML straight into a DOM fragment, so
//      nothing a user typed is ever assigned through innerHTML.
//   3. katex.render() fills each placeholder. It builds nodes and sets styles
//      through the CSSOM, which the page's style-src 'self' policy allows.
const Markdown = {
  ready: false,

  init() {
    if ( this.ready || typeof marked === "undefined" ) return;
    this.ready = true;

    const escapeAttr = function ( value ) {
      return String( value )
        .replace( /&/g , "&amp;" ).replace( /"/g , "&quot;" ).replace( /'/g , "&#39;" )
        .replace( /</g , "&lt;" ).replace( />/g , "&gt;" );
    };
    const placeholder = function ( tag , tex , display ) {
      return "<" + tag + " class=\"math " + ( display ? "math-display" : "math-inline" ) +
        "\" data-tex=\"" + escapeAttr( tex ) + "\" data-display=\"" + ( display ? "1" : "0" ) + "\"></" + tag + ">";
    };

    // $$ on its own lines (or $$...$$ alone on one line) is a display block.
    const blockMath = {
      name: "blockMath",
      level: "block",
      // Only a $$ at the start of a line can open a block. marked cuts a
      // paragraph wherever start() says a block might begin and, when none
      // does, glues the pieces back with a newline -- so claiming a $$ in
      // the middle of a line would break inline $$...$$ apart.
      start( src ) {
        const match = /(^|\n) {0,3}\$\$/.exec( src );
        if ( !match ) return undefined;
        return match.index + match[ 1 ].length;
      },
      // Either $$ alone on a line, the TeX, and $$ alone on a line; or
      // $$TeX$$ as the whole line. Anything after the closing $$ on the same
      // line makes it inline math in a paragraph instead.
      tokenizer( src ) {
        const match = /^ {0,3}\$\$[ \t]*\n([\s\S]+?)\n {0,3}\$\$[ \t]*(?:\n+|$)/.exec( src ) ||
          /^ {0,3}\$\$((?:\\.|[^\\$])+?)\$\$[ \t]*(?:\n+|$)/.exec( src );
        if ( match ) return { type: "blockMath" , raw: match[ 0 ] , text: match[ 1 ].trim() };
        return undefined;
      },
      renderer( token ) {
        return placeholder( "div" , token.text , true ) + "\n";
      },
    };

    // $...$ inline. Not a math delimiter: a $ followed by a space, a closing
    // $ preceded by a space or followed by a digit -- which is what keeps
    // "costs $5 and $10" as plain text. \$ is marked's own escape.
    const inlineMath = {
      name: "inlineMath",
      level: "inline",
      start( src ) {
        const index = src.indexOf( "$" );
        return index < 0 ? undefined : index;
      },
      tokenizer( src ) {
        let match = /^\$\$((?:\\.|[^\\$])+?)\$\$/.exec( src );
        if ( match ) return { type: "inlineMath" , raw: match[ 0 ] , text: match[ 1 ].trim() , display: true };
        match = /^\$(?![\s$])((?:\\.|[^\\$\n])+?)(?<!\s)\$(?!\d)/.exec( src );
        if ( match ) return { type: "inlineMath" , raw: match[ 0 ] , text: match[ 1 ] , display: false };
        return undefined;
      },
      renderer( token ) {
        return placeholder( "span" , token.text , token.display );
      },
    };

    marked.use( { gfm: true , breaks: true , extensions: [ blockMath , inlineMath ] } );

    // Links open in a new tab and never hand this page to the target.
    DOMPurify.addHook( "afterSanitizeAttributes" , function ( node ) {
      if ( node.tagName === "A" && node.hasAttribute( "href" ) ) {
        node.setAttribute( "target" , "_blank" );
        node.setAttribute( "rel" , "noopener noreferrer" );
      }
      if ( node.tagName === "INPUT" ) node.setAttribute( "disabled" , "" );
    } );
  },

  // Replace container's contents with text rendered.
  render( text , container ) {
    this.init();
    Dom.clear( container );
    container.classList.add( "markdown" );
    if ( !text ) return;
    const html = marked.parse( String( text ) );
    const fragment = DOMPurify.sanitize( html , {
      RETURN_DOM_FRAGMENT: true,
      FORBID_TAGS: [ "style" , "form" ],
      FORBID_ATTR: [ "style" ],
    } );
    container.appendChild( fragment );
    this.renderMath( container );
  },

  renderMath( root ) {
    Dom.all( ".math[data-tex]" , root ).forEach( function ( element ) {
      const tex = element.getAttribute( "data-tex" );
      const display = element.getAttribute( "data-display" ) === "1";
      try {
        katex.render( tex , element , {
          displayMode: display,
          throwOnError: false,
          trust: false,
          strict: "ignore",
          output: "htmlAndMathml",
        } );
      } catch ( error ) {
        element.textContent = tex;
        element.classList.add( "math-error" );
      }
    } );
  },

  // One-line text -- titles, list names, checklist items: inline markdown
  // (bold, italics, code, links, strikethrough), line breaks and inline math,
  // but no blocks, so a title cannot grow headings or lists on a card face.
  inlineTags: [ "br" , "b" , "strong" , "i" , "em" , "code" , "a" , "del" , "s" , "span" ],

  renderInline( text , container ) {
    this.init();
    Dom.clear( container );
    const value = String( text || "" );
    if ( !this.ready ) {
      this.renderMathOnly( value , container );
      return;
    }
    const fragment = DOMPurify.sanitize( marked.parseInline( value ) , {
      RETURN_DOM_FRAGMENT: true,
      ALLOWED_TAGS: this.inlineTags,
      ALLOWED_ATTR: [ "href" , "title" , "class" , "data-tex" , "data-display" , "target" , "rel" ],
    } );
    // A link inside a clickable row opens the link, not the row as well.
    Dom.all( "a" , fragment ).forEach( function ( link ) {
      link.addEventListener( "click" , function ( event ) { event.stopPropagation(); } );
    } );
    container.appendChild( fragment );
    this.renderMath( container );
  },

  // Without marked (a page that does not load it): math only.
  renderMathOnly( value , container ) {
    const pattern = /\$(?![\s$])((?:\\.|[^\\$\n])+?)(?<!\s)\$(?!\d)/g;
    let last = 0;
    let match;
    while ( ( match = pattern.exec( value ) ) !== null ) {
      if ( match.index > last ) container.appendChild( document.createTextNode( value.slice( last , match.index ) ) );
      const span = Dom.el( "span" , { class: "math math-inline" , attrs: { "data-tex": match[ 1 ] , "data-display": "0" } } );
      container.appendChild( span );
      last = match.index + match[ 0 ].length;
    }
    if ( last < value.length ) container.appendChild( document.createTextNode( value.slice( last ) ) );
    this.renderMath( container );
  },
};
