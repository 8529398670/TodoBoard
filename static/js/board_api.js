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
    const response = await fetch( this.base + path , {
      method: method,
      // A guest sends no cookie, so a share link behaves the same whether or
      // not the browser also happens to be signed in.
      credentials: this.isGuest ? "omit" : "same-origin",
      headers: headers,
      body: method === "GET" ? undefined : JSON.stringify( payload || {} ),
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
  post( path , body ) { return this.call( "POST" , path , body ); }

  board()                          { return this.get( "/" + this.boardId ); }
  stamp()                          { return this.get( "/" + this.boardId + "/stamp" ); }
  updateBoard( fields )            { return this.post( "/" + this.boardId , fields ); }

  createList( title , position )   { return this.post( "/" + this.boardId + "/lists" , { title: title , position: position } ); }
  updateList( listId , fields )    { return this.post( "/" + this.boardId + "/lists/" + listId , fields ); }
  deleteList( listId )             { return this.post( "/" + this.boardId + "/lists/" + listId + "/delete" ); }

  createLabel( name , color )      { return this.post( "/" + this.boardId + "/labels" , { name: name , color: color } ); }
  updateLabel( labelId , fields )  { return this.post( "/" + this.boardId + "/labels/" + labelId , fields ); }
  deleteLabel( labelId )           { return this.post( "/" + this.boardId + "/labels/" + labelId + "/delete" ); }

  createCard( listId , title , position ) {
    return this.post( "/" + this.boardId + "/cards" , { list_id: listId , title: title , position: position } );
  }
  patchCard( cardId , fields )     { return this.post( "/" + this.boardId + "/cards/" + cardId , fields ); }
  deleteCard( cardId )             { return this.post( "/" + this.boardId + "/cards/" + cardId + "/delete" ); }

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
