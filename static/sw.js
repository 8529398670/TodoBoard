// The service worker: shows a notification when a push arrives while the app
// is closed, and opens the right page when one is clicked. It sits at the
// site root because a worker only reaches pages under its own path. It has no
// fetch handler on purpose: nothing here caches, and no request passes
// through it, share links included.
self.addEventListener( "install" , function () {
  self.skipWaiting();
} );

self.addEventListener( "activate" , function ( event ) {
  event.waitUntil( self.clients.claim() );
} );

self.addEventListener( "push" , function ( event ) {
  let message = {};
  try {
    message = event.data ? event.data.json() : {};
  } catch ( error ) {
    message = {};
  }
  const options = { body: message.body || "" , icon: "/apple-touch-icon.png" , data: { path: message.path || "/" } };
  // Later news of the same board replaces what is on screen.
  if ( message.tag ) options.tag = message.tag;
  event.waitUntil( self.registration.showNotification( message.title || "TodoBoards" , options ) );
} );

self.addEventListener( "notificationclick" , function ( event ) {
  event.notification.close();
  const target = new URL( event.notification.data.path , self.location.origin );
  if ( target.origin !== self.location.origin ) return;
  event.waitUntil( self.clients.matchAll( { type: "window" , includeUncontrolled: true } ).then( function ( windows ) {
    const open = windows.find( function ( item ) { return item.url === target.href; } );
    if ( open ) return open.focus();
    // A new window first: moving one already open could lose what someone is
    // typing there. Only where none can be opened is one moved instead.
    return self.clients.openWindow( target.href ).then( function ( opened ) {
      if ( !opened ) throw new Error( "no window" );
    } ).catch( function () {
      if ( !windows.length ) return null;
      return windows[ 0 ].navigate( target.href ).then( function ( moved ) {
        return moved ? moved.focus() : null;
      } );
    } );
  } ) );
} );
