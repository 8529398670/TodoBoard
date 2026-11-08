// QR codes: for the authenticator app on the two-factor setup screen.
const Qr = {
  // Black on white whatever the theme: scanners want the contrast, and four
  // modules of margin.
  draw( canvas , text ) {
    const code = qrcode( 0 , "M" );
    code.addData( text );
    code.make();
    const quiet = 4;
    const scale = 5;
    const count = code.getModuleCount();
    canvas.width = canvas.height = ( count + quiet * 2 ) * scale;
    const context = canvas.getContext( "2d" );
    context.fillStyle = "#ffffff";
    context.fillRect( 0 , 0 , canvas.width , canvas.height );
    context.fillStyle = "#000000";
    for ( let row = 0; row < count; row++ ) {
      for ( let column = 0; column < count; column++ ) {
        if ( code.isDark( row , column ) ) {
          context.fillRect( ( column + quiet ) * scale , ( row + quiet ) * scale , scale , scale );
        }
      }
    }
  },
};
