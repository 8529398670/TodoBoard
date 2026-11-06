// This browser's side of web push: the permission, the service worker, and
// the subscription the server sends to. Only the signed-in pages load this,
// so a share link never registers anything.
//
// The server does not tell a browser which device it is, so the id it gives
// back is kept here. That is also how a browser learns it was removed from
// another one's list, rather than quietly adding itself back.
const PushClient = {
  storageKey: "push_device_id",

  supported() {
    return "serviceWorker" in navigator && "PushManager" in window && "Notification" in window;
  },

  blocked() {
    return this.supported() && Notification.permission === "denied";
  },

  apple() {
    return /iPhone|iPad|iPod/.test( navigator.userAgent ) || ( navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1 );
  },

  // Opened from the Home Screen rather than in a browser tab.
  homeScreenApp() {
    return navigator.standalone === true || window.matchMedia( "(display-mode: standalone)" ).matches;
  },

  // Web push on an iPhone or iPad works only from the Home Screen.
  needsHomeScreen() {
    return this.apple() && !this.homeScreenApp();
  },

  // Web push came to iPhones and iPads in iOS 16.4. An iPad asking for desktop
  // sites does not say its version, so it gets the benefit of the doubt.
  iosTooOld() {
    const found = /\bOS (\d+)_(\d+)/.exec( navigator.userAgent );
    if ( !this.apple() || !found ) return false;
    const major = Number( found[ 1 ] );
    return major < 16 || ( major === 16 && Number( found[ 2 ] ) < 4 );
  },

  savedId() {
    try { return window.localStorage.getItem( this.storageKey ) || ""; } catch ( error ) { return ""; }
  },

  saveId( id ) {
    try {
      if ( id ) window.localStorage.setItem( this.storageKey , id );
      else window.localStorage.removeItem( this.storageKey );
    } catch ( error ) {
      // The id is a convenience: without it this browser just checks in again.
    }
  },

  keyBytes( key ) {
    const padded = key.replace( /-/g , "+" ).replace( /_/g , "/" ) + "=".repeat( ( 4 - key.length % 4 ) % 4 );
    return Uint8Array.from( window.atob( padded ) , function ( character ) { return character.charCodeAt( 0 ); } );
  },

  sameKey( subscription , key ) {
    const held = subscription.options && subscription.options.applicationServerKey;
    if ( !held ) return true;
    const mine = new Uint8Array( held );
    const theirs = this.keyBytes( key );
    return mine.length === theirs.length && mine.every( function ( value , index ) { return value === theirs[ index ]; } );
  },

  async subscription() {
    if ( !this.supported() ) return null;
    const registration = await navigator.serviceWorker.getRegistration( "/" );
    return registration ? registration.pushManager.getSubscription() : null;
  },

  // A name for the device list: enough to tell two browsers apart.
  label() {
    const agent = navigator.userAgent;
    const browser = /Edg\//.test( agent ) ? "Edge" : /Firefox\//.test( agent ) ? "Firefox" : /Chrome\//.test( agent ) ? "Chrome" : /Safari\//.test( agent ) ? "Safari" : "";
    const system = /iPhone|iPad|iPod/.test( agent ) ? "iOS" : /Android/.test( agent ) ? "Android" : /Mac OS X/.test( agent ) ? "macOS" : /Windows/.test( agent ) ? "Windows" : /Linux/.test( agent ) ? "Linux" : "";
    return [ browser , system ].filter( Boolean ).join( " · " );
  },

  async send( subscription ) {
    const fields = subscription.toJSON();
    const device = await Api.post( "/api/notify/devices" , { endpoint: fields.endpoint , p256dh: fields.keys.p256dh , auth: fields.keys.auth , label: this.label() } );
    this.saveId( device.id );
    return device;
  },

  async subscribe( registration , key ) {
    const subscription = await registration.pushManager.subscribe( { userVisibleOnly: true , applicationServerKey: this.keyBytes( key ) } );
    return this.send( subscription );
  },

  // From a click, and the permission first: a browser only asks during one.
  async enable( key ) {
    const permission = await Notification.requestPermission();
    if ( permission !== "granted" ) throw new Error( "push_denied" );
    await navigator.serviceWorker.register( "/sw.js" );
    const registration = await navigator.serviceWorker.ready;
    return this.subscribe( registration , key );
  },

  async disable() {
    const subscription = await this.subscription();
    if ( subscription ) await subscription.unsubscribe();
    const id = this.savedId();
    this.saveId( "" );
    if ( id ) await Api.post( "/api/notify/devices/" + encodeURIComponent( id ) + "/delete" );
  },

  // Each time the settings load. The browser may have replaced its
  // subscription, or the server its keys; devices is the server's list.
  async checkIn( key , devices ) {
    const subscription = await this.subscription();
    if ( !subscription ) return;
    const id = this.savedId();
    if ( id && !devices.some( function ( device ) { return device.id === id; } ) ) {
      this.saveId( "" );
      await subscription.unsubscribe();
      return;
    }
    if ( this.sameKey( subscription , key ) ) {
      await this.send( subscription );
      return;
    }
    await subscription.unsubscribe();
    await this.subscribe( await navigator.serviceWorker.ready , key );
  },

  // Signing out: a shared computer must not keep showing this person's cards.
  // Never in the way of the sign-out itself.
  async leave() {
    try {
      await this.disable();
    } catch ( error ) {
      this.saveId( "" );
    }
  },
};
