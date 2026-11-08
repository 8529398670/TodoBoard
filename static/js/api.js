// Every call to the server goes through here, so the things that are easy to
// forget -- sending the cookie, attaching the CSRF token, turning a non-2xx
// response into a real Error -- happen once instead of at each call site.
const Api = {
  csrfToken: null,

  async request( path , options ) {
    const settings = options || {};
    const response = await fetch( path , {
      method: settings.method || "GET",
      // The session cookie is the only credential; without this an
      // authenticated request silently arrives as an anonymous one.
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: settings.body ? JSON.stringify( settings.body ) : undefined,
    } );

    let payload = {};
    try {
      payload = await response.json();
    } catch ( parseError ) {
      payload = {};
    }

    if ( !response.ok ) {
      const error = new Error( payload.error || response.statusText );
      error.status = response.status;
      throw error;
    }
    return payload;
  },

  // Any state-changing call carries the per-session CSRF token in the body.
  // Wrapping it here means a new endpoint cannot forget it.
  async post( path , body ) {
    return this.request( path , {
      method: "POST",
      body: Object.assign( {} , body || {} , { csrf_token: this.csrfToken } ),
    } );
  },

  async me() {
    try {
      const payload = await this.request( "/api/me" );
      this.csrfToken = payload.csrf_token;
      return payload;
    } catch ( error ) {
      if ( error.status === 401 ) return { authenticated: false };
      throw error;
    }
  },

  // The one POST with no CSRF token: there is no session yet to hold one.
  // The server guards it by requiring this JSON body instead -- see the
  // login route on the server side.
  login( credential ) {
    return this.request( "/api/login" , { method: "POST" , body: { credential: credential } } );
  },

  // What the signed-out page may offer, and the salt a password is hashed
  // with in the browser (see password.js).
  loginOptions() { return this.request( "/api/login/options" ); },

  // Signing in with a password, and the code step any sign-in goes on to when
  // the account has two-factor on. No session yet, so the same rule as login().
  passwordLogin( username , digest ) {
    return this.request( "/api/login/password" , { method: "POST" , body: { username: username , password_digest: digest } } );
  },
  finishSignIn( challenge , code ) {
    return this.request( "/api/login/two-factor" , { method: "POST" , body: { challenge: challenge , code: code } } );
  },

  // Invite links: no session yet either, so the same JSON-body rule as
  // login() applies. The credential always goes in the body.
  joinInfo( credential )                { return this.request( "/api/join/info" , { method: "POST" , body: { credential: credential } } ); },
  joinMatch( credential , name )        { return this.request( "/api/join/match" , { method: "POST" , body: { credential: credential , display_name: name } } ); },
  joinCreate( credential , name )       { return this.request( "/api/join/create" , { method: "POST" , body: { credential: credential , display_name: name } } ); },
  joinClaim( credential , userId , pin ) {
    return this.request( "/api/join/claim" , { method: "POST" , body: { credential: credential , user_id: userId , pin: pin || "" } } );
  },

  rename( displayName )        { return this.post( "/api/account/rename" , { display_name: displayName } ); },
  setPin( pin )                { return this.post( "/api/account/pin" , { pin: pin } ); },
  createMyLoginLink( options ) { return this.post( "/api/account/login-link" , options ); },
  cancelMyLoginLink()          { return this.post( "/api/account/login-link/cancel" ); },
  setPassword( username , digest , code ) {
    return this.post( "/api/account/password" , { username: username , password_digest: digest , code: code } );
  },
  startTwoFactor()             { return this.post( "/api/account/two-factor/start" ); },
  confirmTwoFactor( code )     { return this.post( "/api/account/two-factor/confirm" , { code: code } ); },
  disableTwoFactor( code )     { return this.post( "/api/account/two-factor/disable" , { code: code } ); },
  sessions()                   { return this.request( "/api/account/sessions" ); },
  labelSession( sessionId , label , expect ) {
    return this.post( "/api/account/sessions/" + encodeURIComponent( sessionId ) + "/label" , { label: label , expect: expect } );
  },
  revokeSession( sessionId )   { return this.post( "/api/account/sessions/" + encodeURIComponent( sessionId ) + "/revoke" ); },
  revokeOtherSessions()        { return this.post( "/api/account/sessions/revoke-others" ); },
  logout()                     { return this.post( "/api/logout" ); },
  listTeam()                   { return this.request( "/api/team" ); },
  listUsers()                  { return this.request( "/api/admin/users" ); },
  createUser( name , role , options ) {
    return this.post( "/api/admin/users" , Object.assign( { display_name: name , role: role } , options || {} ) );
  },
  reissueLogin( userId , options ) { return this.post( "/api/admin/users/" + userId + "/reissue-login" , options ); },
  cancelLoginLink( userId )    { return this.post( "/api/admin/users/" + userId + "/cancel-login-link" ); },
  renameUser( userId , name , expect ) { return this.post( "/api/admin/users/" + userId + "/rename" , { display_name: name , expect: expect } ); },
  setRole( userId , role )     { return this.post( "/api/admin/users/" + userId + "/role" , { role: role } ); },
  resetTwoFactor( userId )     { return this.post( "/api/admin/users/" + userId + "/reset-two-factor" ); },
  setPasswordAccess( userId , allowed ) {
    return this.post( "/api/owner/users/" + userId + "/password-access" , { allowed: allowed } );
  },
  userSessions( userId )       { return this.request( "/api/admin/users/" + userId + "/sessions" ); },
  revokeUserSession( userId , sessionId ) {
    return this.post( "/api/admin/users/" + userId + "/sessions/" + encodeURIComponent( sessionId ) + "/revoke" );
  },
  setDisabled( userId , flag ) { return this.post( "/api/admin/users/" + userId + "/disabled" , { disabled: flag } ); },
  revokeAccess( userId )       { return this.post( "/api/admin/users/" + userId + "/revoke-access" ); },
  deleteUser( userId )         { return this.post( "/api/admin/users/" + userId + "/delete" ); },
  listInvites()                { return this.request( "/api/admin/invite-links" ); },
  createInvite( label , maxUses , days ) {
    return this.post( "/api/admin/invite-links" , { label: label , max_uses: maxUses , days: days } );
  },
  revokeInvite( linkId )       { return this.post( "/api/admin/invite-links/" + encodeURIComponent( linkId ) + "/revoke" ); },
  deleteInvite( linkId )       { return this.post( "/api/admin/invite-links/" + encodeURIComponent( linkId ) + "/delete" ); },

  // API keys. createKey's expiresInDays is passed through as-is including null,
  // because the server reads absent as "use the default" and 0 as "never
  // expires" -- collapsing them here would lose the distinction.
  listKeys()                   { return this.request( "/api/keys" ); },
  createKey( name , role , expiresInDays , boardAccess , boardIds ) {
    return this.post( "/api/keys" , { name: name , role: role , expires_in_days: expiresInDays , board_access: boardAccess , board_ids: boardIds } );
  },
  setKeyBoards( keyId , boardAccess , boardIds ) {
    return this.post( "/api/keys/" + encodeURIComponent( keyId ) + "/boards" , { board_access: boardAccess , board_ids: boardIds } );
  },
  listBoards()                 { return this.request( "/api/boards" ); },
  revokeKey( keyId )           { return this.post( "/api/keys/" + encodeURIComponent( keyId ) + "/revoke" ); },
  deleteKey( keyId )           { return this.post( "/api/keys/" + encodeURIComponent( keyId ) + "/delete" ); },
  listAllKeys()                { return this.request( "/api/admin/keys" ); },
  createKeyFor( userId , name , role , expiresInDays ) {
    return this.post( "/api/admin/keys" , { user_id: userId , name: name , role: role , expires_in_days: expiresInDays } );
  },
  revokeAnyKey( keyId )        { return this.post( "/api/admin/keys/" + encodeURIComponent( keyId ) + "/revoke" ); },
  deleteAnyKey( keyId )        { return this.post( "/api/admin/keys/" + encodeURIComponent( keyId ) + "/delete" ); },
};
