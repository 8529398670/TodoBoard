package product

import (
	net "net"
	url "net/url"
	testing "testing"
)

func mustURL( raw string ) ( result *url.URL ) {
	result , _ = url.Parse( raw )
	return
}

func TestParseJSONLD( t *testing.T ) {
	page := `<html><head><title>Shop</title>
<script type="application/ld+json">{"@context":"https://schema.org","@graph":[{"@type":"WebPage"},{"@type":"Product","name":"Blue Kettle","description":"Boils water.","image":["/img/kettle.jpg"],"offers":{"@type":"Offer","price":"24.5","priceCurrency":"USD"}}]}</script>
<meta property="og:title" content="Not this one"></head><body></body></html>`
	info , err := Parse( mustURL( "https://shop.example/p/1" ) , []byte( page ) )
	if err != nil { t.Fatal( err ) }
	if info.Title != "Blue Kettle" || info.Price != "$24.50" || info.Description != "Boils water." || info.Image != "https://shop.example/img/kettle.jpg" {
		t.Errorf( "got %+v" , info )
	}
}

func TestParseOpenGraph( t *testing.T ) {
	page := `<html><head><meta property="og:title" content="Lamp &amp; Shade"><meta property="og:image" content="https://cdn.example/lamp.png">
<meta property="product:price:amount" content="10"><meta property="product:price:currency" content="EUR"><meta name="description" content="A lamp."></head></html>`
	info , err := Parse( mustURL( "https://shop.example/lamp" ) , []byte( page ) )
	if err != nil { t.Fatal( err ) }
	if info.Title != "Lamp & Shade" || info.Price != "€10.00" || info.Image != "https://cdn.example/lamp.png" || info.Description != "A lamp." {
		t.Errorf( "got %+v" , info )
	}
}

func TestParseAmazon( t *testing.T ) {
	page := `<html><head><title>Amazon.com: Widget</title></head><body>
<span id="productTitle">   Super   Widget, 2-Pack  </span>
<div id="corePriceDisplay_desktop_feature_div"><span class="a-price"><span class="a-offscreen">$19.99</span><span aria-hidden="true">$19<sup>99</sup></span></span></div>
<img id="landingImage" src="https://m.media-amazon.com/small.jpg" data-a-dynamic-image='{"https://m.media-amazon.com/a.jpg":[300,300],"https://m.media-amazon.com/big.jpg":[1500,1500]}'>
<div id="feature-bullets"><ul><li><span>Sturdy.</span></li><li><span>Shiny.</span></li></ul></div>
</body></html>`
	info , err := Parse( mustURL( "https://www.amazon.com/dp/B000" ) , []byte( page ) )
	if err != nil { t.Fatal( err ) }
	if info.Title != "Super Widget, 2-Pack" || info.Price != "$19.99" || info.Image != "https://m.media-amazon.com/big.jpg" || info.Description != "Sturdy. · Shiny." {
		t.Errorf( "got %+v" , info )
	}
}

func TestParseRobotCheck( t *testing.T ) {
	page := `<html><head><title>Robot Check</title></head><body><form action="/errors/validateCaptcha"></form></body></html>`
	if _ , err := Parse( mustURL( "https://www.amazon.com/dp/B000" ) , []byte( page ) ); err != ErrBlocked {
		t.Errorf( "want ErrBlocked, got %v" , err )
	}
	if _ , err := Parse( mustURL( "https://x.example/" ) , []byte( `<html><body>hi</body></html>` ) ); err != ErrNoProduct {
		t.Errorf( "want ErrNoProduct, got %v" , err )
	}
}

func TestFindURLAndAmazon( t *testing.T ) {
	if got := FindURL( "Buy this: https://www.amazon.co.uk/dp/B0X?th=1." ); got != "https://www.amazon.co.uk/dp/B0X?th=1" {
		t.Errorf( "FindURL got %q" , got )
	}
	if FindURL( "no link here" ) != "" { t.Errorf( "found a link in plain text" ) }
	for _ , raw := range []string{ "https://www.amazon.com/dp/1" , "https://amazon.de/x" , "https://smile.amazon.co.jp/x" , "https://a.co/d/abc" , "https://amzn.to/xyz" } {
		if IsAmazon( raw ) == false { t.Errorf( "%s should be Amazon" , raw ) }
	}
	for _ , raw := range []string{ "https://amazonaws.com/x" , "https://notamazon.com/" , "https://example.com/amazon" } {
		if IsAmazon( raw ) { t.Errorf( "%s should not be Amazon" , raw ) }
	}
}

func TestPublicIP( t *testing.T ) {
	for _ , raw := range []string{ "127.0.0.1" , "10.1.2.3" , "192.168.0.1" , "172.16.5.5" , "169.254.169.254" , "100.64.0.1" , "::1" , "fe80::1" , "fd00::1" , "0.0.0.0" , "::ffff:127.0.0.1" } {
		if PublicIP( net.ParseIP( raw ) ) { t.Errorf( "%s should be refused" , raw ) }
	}
	for _ , raw := range []string{ "8.8.8.8" , "2606:4700:4700::1111" } {
		if PublicIP( net.ParseIP( raw ) ) == false { t.Errorf( "%s should be allowed" , raw ) }
	}
	if publicOnly( "tcp" , "127.0.0.1:80" , nil ) == nil { t.Errorf( "dialing loopback must be refused" ) }
}
