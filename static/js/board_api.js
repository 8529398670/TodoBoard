// Every board and card call, behind one interface with two transports:
//
//   BoardApi.member( boardId )  a signed-in person: the session cookie, plus
//                               the CSRF token in each write body (Api.csrfToken)
//   BoardApi.guest( token )     a share link: the credential in X-Share-Token,
//                               and no cookie at all
//
// The views only ever see a client and call the same methods on it, so the
// board, its views and the card editor work unchanged for guests. What a
// guest may actually do is decided by the server; the UI only avoids offering
// what it would refuse.
class BoardClient {
  constructor( base , boardId , shareToken ) {
    this.base = base;
    this.boardId = boardId;
    this.shareToken = shareToken || null;
    this.guestName = "";
  }

  get isGuest() {
    return this.shareToken !== null;
  }

  async call( method , path , body ) {
    const headers = { "Content-Type": "application/json" };
    let payload = body;
    if ( this.isGuest ) {
      headers[ "X-Share-Token" ] = this.shareToken;
      if ( this.guestName ) headers[ "X-Guest-Name" ] = this.guestName;
    } else if ( method !== "GET" ) {
      payload = Object.assign( {} , body || {} , { csrf_token: Api.csrfToken } );
    }
    const sent = method === "GET" ? undefined : JSON.stringify( payload || {} );
    const response = await fetch( this.base + path , {
      method: method,
      // A small write outlives the page, so text saved as the tab closes
      // still arrives. Browsers cap keepalive bodies at 64 KB in total.
      keepalive: sent !== undefined && sent.length < 60000,
      // A guest sends no cookie, so a share link behaves the same whether or
      // not the browser also happens to be signed in.
      credentials: this.isGuest ? "omit" : "same-origin",
      headers: headers,
      body: sent,
    } );
    let result = {};
    try {
      result = await response.json();
    } catch ( error ) {
      result = {};
    }
    if ( !response.ok ) {
      const error = new Error( result.error || response.statusText );
      error.status = response.status;
      error.payload = result;
      throw error;
    }
    return result;
  }

  get( path )         { return this.call( "GET" , path ); }

  // A file rather than JSON -- the spreadsheet export -- with the same
  // credentials as every other call.
  async file( path ) {
    const headers = {};
    if ( this.isGuest ) headers[ "X-Share-Token" ] = this.shareToken;
    const response = await fetch( this.base + path , { credentials: this.isGuest ? "omit" : "same-origin" , headers: headers } );
    if ( !response.ok ) {
      let result = {};
      try { result = await response.json(); } catch ( error ) { result = {}; }
      const error = new Error( result.error || response.statusText );
      error.status = response.status;
      throw error;
    }
    return response.blob();
  }
  post( path , body ) { return this.call( "POST" , path , body ); }

  board()                          { return this.get( "/" + this.boardId ); }
  stamp()                          { return this.get( "/" + this.boardId + "/stamp" ); }
  updateBoard( fields )            { return this.post( "/" + this.boardId , fields ); }
  // format "xlsx" or "csv"; timed dates are written in zone.
  exportSheet( format , archived , zone ) {
    return this.file( "/" + this.boardId + "/export?format=" + format + "&archived=" + ( archived ? "1" : "0" ) + "&tz=" + encodeURIComponent( zone || "" ) );
  }

  createList( title , position )   { return this.post( "/" + this.boardId + "/lists" , { title: title , position: position } ); }
  updateList( listId , fields )    { return this.post( "/" + this.boardId + "/lists/" + listId , fields ); }
  deleteList( listId )             { return this.post( "/" + this.boardId + "/lists/" + listId + "/delete" ); }
  // Every open card in the list, or only cardIds.
  exportColumn( listId )           { return this.get( "/" + this.boardId + "/lists/" + listId + "/export" ); }
  importColumn( listId , labels , cards ) {
    return this.post( "/" + this.boardId + "/lists/" + listId + "/import" , { labels: labels , cards: cards } );
  }
  archiveCards( listId , cardIds ) { return this.post( "/" + this.boardId + "/lists/" + listId + "/archive-cards" , { card_ids: cardIds } ); }

  createLabel( name , color )      { return this.post( "/" + this.boardId + "/labels" , { name: name , color: color } ); }
  updateLabel( labelId , fields )  { return this.post( "/" + this.boardId + "/labels/" + labelId , fields ); }
  deleteLabel( labelId )           { return this.post( "/" + this.boardId + "/labels/" + labelId + "/delete" ); }

  createCard( listId , title , position ) {
    return this.post( "/" + this.boardId + "/cards" , { list_id: listId , title: title , position: position } );
  }
  patchCard( cardId , fields )     { return this.post( "/" + this.boardId + "/cards/" + cardId , fields ); }
  deleteCard( cardId )             { return this.post( "/" + this.boardId + "/cards/" + cardId + "/delete" ); }
  // The product a card's link sells (see server/routes/products.go).
  refreshProduct( cardId )         { return this.post( "/" + this.boardId + "/cards/" + cardId + "/product/refresh" ); }
  removeProduct( cardId )          { return this.post( "/" + this.boardId + "/cards/" + cardId + "/product/delete" ); }
  productImage( cardId )           { return this.file( "/" + this.boardId + "/cards/" + cardId + "/product-image" ); }
  // A picture pasted onto a card; data is the image's bytes as base64.
  uploadImage( cardId , data )     { return this.post( "/" + this.boardId + "/cards/" + cardId + "/image" , { data: data } ); }
  removeImage( cardId )            { return this.post( "/" + this.boardId + "/cards/" + cardId + "/image/delete" ); }
  cardImage( cardId )              { return this.file( "/" + this.boardId + "/cards/" + cardId + "/image" ); }
  // Deleted cards wait in the board's trash (see trash.js).
  trash()                          { return this.get( "/" + this.boardId + "/trash" ); }
  restoreFromTrash( cardId )       { return this.post( "/" + this.boardId + "/trash/" + cardId + "/restore" ); }
  purgeCard( cardId )              { return this.post( "/" + this.boardId + "/trash/" + cardId + "/delete" ); }
  emptyTrash()                     { return this.post( "/" + this.boardId + "/trash/empty" ); }

  addItem( cardId , text )         { return this.post( "/" + this.boardId + "/cards/" + cardId + "/checklist" , { text: text } ); }
  updateItem( cardId , itemId , fields ) {
    return this.post( "/" + this.boardId + "/cards/" + cardId + "/checklist/" + itemId , fields );
  }
  deleteItem( cardId , itemId )    { return this.post( "/" + this.boardId + "/cards/" + cardId + "/checklist/" + itemId + "/delete" ); }

  // Members only -- the server answers a guest 401/404 for all of these.
  setMember( userId , role )       { return this.post( "/" + this.boardId + "/members" , { user_id: userId , role: role } ); }
  shareCardWithUser( cardId , userId , mode ) {
    return this.post( "/" + this.boardId + "/cards/" + cardId + "/share-user" , { user_id: userId , mode: mode } );
  }
  shareLinks()                     { return this.get( "/" + this.boardId + "/shares" ); }
  createShareLink( fields )        { return this.post( "/" + this.boardId + "/shares" , fields ); }
  revokeShareLink( linkId )        { return this.post( "/" + this.boardId + "/shares/" + encodeURIComponent( linkId ) + "/revoke" ); }
  deleteShareLink( linkId )        { return this.post( "/" + this.boardId + "/shares/" + encodeURIComponent( linkId ) + "/delete" ); }
  boardSubscription()              { return this.get( "/" + this.boardId + "/subscription" ); }
  setBoardSubscription( events )   { return this.post( "/" + this.boardId + "/subscription" , { events: events } ); }
  cardSubscription( cardId )       { return this.get( "/" + this.boardId + "/cards/" + cardId + "/subscription" ); }
  setCardSubscription( cardId , events ) {
    return this.post( "/" + this.boardId + "/cards/" + cardId + "/subscription" , { events: events } );
  }
  // How the board looks to this member alone.
  setMyView( fields )              { return this.post( "/" + this.boardId + "/my-view" , fields ); }
  leave()                          { return this.post( "/" + this.boardId + "/leave" ); }
  deleteBoard()                    { return this.post( "/" + this.boardId + "/delete" ); }
}

const BoardApi = {
  member( boardId ) {
    return new BoardClient( "/api/boards" , boardId , null );
  },

  // A guest client needs the board id, which only the server knows from the
  // token -- hence the info call first.
  async guest( token ) {
    const probe = new BoardClient( "/api/shared" , 0 , token );
    const info = await probe.get( "" );
    const client = new BoardClient( "/api/shared" , info.board_id , token );
    client.info = info;
    return client;
  },

  // The credential in a /s/<credential> or /print/s/<credential> path.
  tokenFromPath( prefix ) {
    const path = window.location.pathname;
    if ( path.indexOf( prefix ) !== 0 ) return "";
    try {
      return decodeURIComponent( path.slice( prefix.length ) ).replace( /\/+$/ , "" );
    } catch ( error ) {
      return "";
    }
  },
};
