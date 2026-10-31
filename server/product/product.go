// Package product reads what a shop's page says about the thing it sells --
// name, photo, price, a line of description -- so a card holding a product
// link can show it. It is best effort by design: shops (Amazon above all)
// often answer a server with a robot check instead of the page, and then the
// card simply keeps its link.
//
// Fetching a URL someone typed is a server-side request forgery risk, so
// every connection, redirects included, is refused unless it goes to a
// public address (see publicOnly).
package product

import (
	context "context"
	json "encoding/json"
	errors "errors"
	fmt "fmt"
	io "io"
	net "net"
	http "net/http"
	url "net/url"
	regexp "regexp"
	strconv "strconv"
	strings "strings"
	syscall "syscall"
	time "time"

	html "golang.org/x/net/html"
)

const (
	maxPageBytes  = 4 << 20
	maxImageBytes = 2 << 20
	maxRedirects  = 5
	fetchTimeout  = 10 * time.Second
	// Says what it is. Claiming to be Chrome from a client whose TLS
	// handshake is plainly not Chrome's is what gets a robot check; an
	// honest link-preview fetcher is answered with the page.
	userAgent = "Mozilla/5.0 (compatible; TodoBoard/1.0; link preview)"
)

var (
	ErrBlocked   = errors.New( "the shop answered with a robot check" )
	ErrNoProduct = errors.New( "no product details on the page" )
	ErrBadURL    = errors.New( "not a link that can be fetched" )
)

// Info is what a page said. Image is the photo's absolute URL; Fetch
// downloads it.
type Info struct {
	URL         string
	Title       string
	Description string
	Price       string
	Image       string
}

// ---- finding links ------------------------------------------------------

var linkPattern = regexp.MustCompile( `https?://[^\s<>()\[\]"'` + "`" + `]+` )

// FindURL is the first http(s) link in text, trailing punctuation trimmed,
// or "".
func FindURL( text string ) ( result string ) {
	found := linkPattern.FindString( text )
	result = strings.TrimRight( found , ".,;:!?*_~" )
	return
}

// IsAmazon reports whether a link is on an Amazon storefront or one of its
// short-link hosts.
func IsAmazon( raw string ) ( result bool ) {
	parsed , err := url.Parse( raw )
	if err != nil { return }
	host := strings.ToLower( strings.TrimPrefix( parsed.Hostname() , "www." ) )
	switch host {
	case "a.co" , "amzn.to" , "amzn.eu" , "amzn.asia" , "amzn.com":
		return true
	}
	labels := strings.Split( host , "." )
	for index , label := range labels {
		// amazon.com, amazon.co.uk, smile.amazon.de -- "amazon" followed by
		// a country or generic suffix.
		if label == "amazon" && index < len( labels ) - 1 { return true }
	}
	return
}

// ---- fetching -------------------------------------------------------------

// publicOnly refuses a connection to anything but a public unicast address.
// It runs on the resolved address of every dial, so neither a redirect nor
// a DNS answer that changes between lookups can reach the local network.
func publicOnly( network string , address string , _ syscall.RawConn ) ( err error ) {
	host , _ , split_err := net.SplitHostPort( address )
	if split_err != nil { return split_err }
	ip := net.ParseIP( host )
	if ip == nil || PublicIP( ip ) == false {
		err = fmt.Errorf( "refusing to connect to %s" , host )
	}
	return
}

var carrierNAT = &net.IPNet{ IP: net.IPv4( 100 , 64 , 0 , 0 ) , Mask: net.CIDRMask( 10 , 32 ) }

// PublicIP is false for loopback, private, link-local, multicast,
// unspecified and carrier-grade NAT addresses.
func PublicIP( ip net.IP ) ( result bool ) {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || carrierNAT.Contains( ip ) {
		return
	}
	// IPv4-mapped IPv6 is judged as the IPv4 address it carries.
	if v4 := ip.To4(); v4 != nil && len( ip ) == net.IPv6len && v4.Equal( ip ) == false { return PublicIP( v4 ) }
	result = true
	return
}

var client = &http.Client{
	Timeout: fetchTimeout,
	Transport: &http.Transport{
		Proxy: nil,
		// A custom dialer turns HTTP/2 off unless asked; shops treat an
		// HTTP/1.1-only client as a robot far more often.
		ForceAttemptHTTP2: true,
		DialContext: ( &net.Dialer{ Timeout: 5 * time.Second , Control: publicOnly } ).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
	},
	CheckRedirect: func( request *http.Request , via []*http.Request ) ( err error ) {
		if len( via ) >= maxRedirects { return errors.New( "too many redirects" ) }
		if request.URL.Scheme != "http" && request.URL.Scheme != "https" { return ErrBadURL }
		return
	},
}

func get( ctx context.Context , raw string , accept string , limit int64 ) ( body []byte , final *url.URL , status int , content_type string , err error ) {
	parsed , parse_err := url.Parse( raw )
	if parse_err != nil || ( parsed.Scheme != "http" && parsed.Scheme != "https" ) || parsed.Host == "" {
		err = ErrBadURL
		return
	}
	request , err := http.NewRequestWithContext( ctx , http.MethodGet , parsed.String() , nil )
	if err != nil { return }
	request.Header.Set( "User-Agent" , userAgent )
	request.Header.Set( "Accept" , accept )
	request.Header.Set( "Accept-Language" , "en-US,en;q=0.9" )
	response , err := client.Do( request )
	if err != nil { return }
	defer response.Body.Close()
	status = response.StatusCode
	final = response.Request.URL
	content_type = response.Header.Get( "Content-Type" )
	body , err = io.ReadAll( io.LimitReader( response.Body , limit + 1 ) )
	if err == nil && int64( len( body ) ) > limit {
		if limit == maxImageBytes {
			err = errors.New( "image too large" )
			return
		}
		// A page is read as far as the cap; the details are near the top.
		body = body[ :limit ]
	}
	return
}

// Fetch reads the product page at raw and, if it names a photo, downloads
// that too. image is nil when there is no usable photo.
func Fetch( ctx context.Context , raw string ) ( info *Info , image []byte , image_type string , err error ) {
	body , final , status , _ , err := get( ctx , raw , "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8" , maxPageBytes )
	if err != nil { return }
	if status == http.StatusServiceUnavailable || status == http.StatusTooManyRequests {
		err = ErrBlocked
		return
	}
	if status >= 400 {
		err = fmt.Errorf( "the shop answered %d" , status )
		return
	}
	info , err = Parse( final , body )
	if err != nil { return }
	info.URL = raw
	if info.Image != "" {
		data , _ , image_status , _ , image_err := get( ctx , info.Image , "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.5" , maxImageBytes )
		if image_err == nil && image_status < 400 {
			if kind := ImageType( data ); kind != "" {
				image , image_type = data , kind
			}
		}
	}
	return
}

// ImageType is the sniffed type of an image this package keeps, or "".
func ImageType( data []byte ) ( result string ) {
	switch kind := http.DetectContentType( data ); kind {
	case "image/jpeg" , "image/png" , "image/webp" , "image/gif":
		result = kind
	}
	return
}

// ---- reading the page -----------------------------------------------------

// page is what one walk of the document collects.
type page struct {
	meta        map[string]string
	ldJSON      []string
	title       string
	byID        map[string]*html.Node
	offscreen   string
	blocked     bool
}

// Parse reads a product page. base resolves relative image links. The best
// source wins field by field: structured data (JSON-LD), then Open Graph,
// then Amazon's own markup, then the page title.
func Parse( base *url.URL , body []byte ) ( info *Info , err error ) {
	root , parse_err := html.Parse( strings.NewReader( string( body ) ) )
	if parse_err != nil {
		err = ErrNoProduct
		return
	}
	found := &page{ meta: map[string]string{} , byID: map[string]*html.Node{} }
	found.walk( root , false )
	if found.blocked {
		err = ErrBlocked
		return
	}

	info = &Info{}
	for _ , raw := range found.ldJSON { fromLDJSON( raw , info ) }

	first := func( current string , values ...string ) ( result string ) {
		result = current
		for _ , value := range values {
			if result != "" { return }
			result = clean( value )
		}
		return
	}
	info.Title = first( info.Title , text( found.byID[ "productTitle" ] ) , found.meta[ "og:title" ] , found.meta[ "twitter:title" ] , found.title )
	info.Image = first( info.Image , found.meta[ "og:image" ] , found.meta[ "og:image:url" ] , found.meta[ "twitter:image" ] , amazonImage( found.byID[ "landingImage" ] ) , amazonImage( found.byID[ "imgBlkFront" ] ) )
	info.Description = first( info.Description , bullets( found.byID[ "feature-bullets" ] ) , found.meta[ "og:description" ] , found.meta[ "description" ] , found.meta[ "twitter:description" ] )
	if info.Price == "" {
		info.Price = formatPrice( first( "" , found.meta[ "product:price:amount" ] , found.meta[ "og:price:amount" ] ) , first( "" , found.meta[ "product:price:currency" ] , found.meta[ "og:price:currency" ] ) )
	}
	info.Price = first( info.Price , offscreen( found.byID[ "corePriceDisplay_desktop_feature_div" ] ) , offscreen( found.byID[ "corePrice_feature_div" ] ) ,
		offscreen( found.byID[ "apex_desktop" ] ) , text( found.byID[ "priceblock_ourprice" ] ) , text( found.byID[ "priceblock_dealprice" ] ) , found.offscreen )

	if info.Image != "" && base != nil {
		if resolved , resolve_err := base.Parse( info.Image ); resolve_err == nil && ( resolved.Scheme == "http" || resolved.Scheme == "https" ) {
			info.Image = resolved.String()
		} else {
			info.Image = ""
		}
	}
	info.Title = truncate( info.Title , 300 )
	info.Description = truncate( info.Description , 600 )
	info.Price = truncate( info.Price , 40 )
	if info.Title == "" && info.Image == "" && info.Price == "" { err = ErrNoProduct }
	return
}

func ( found *page ) walk( node *html.Node , in_price bool ) {
	if node.Type == html.ElementNode {
		attrs := map[string]string{}
		for _ , attr := range node.Attr { attrs[ strings.ToLower( attr.Key ) ] = attr.Val }
		switch node.Data {
		case "meta":
			key := strings.ToLower( attrs[ "property" ] )
			if key == "" { key = strings.ToLower( attrs[ "name" ] ) }
			if key == "" { key = strings.ToLower( attrs[ "itemprop" ] ) }
			if _ , seen := found.meta[ key ]; key != "" && seen == false { found.meta[ key ] = attrs[ "content" ] }
		case "script":
			if strings.Contains( strings.ToLower( attrs[ "type" ] ) , "ld+json" ) && node.FirstChild != nil {
				found.ldJSON = append( found.ldJSON , node.FirstChild.Data )
			}
		case "title":
			if found.title == "" { found.title = text( node ) }
			if strings.Contains( found.title , "Robot Check" ) { found.blocked = true }
		case "form":
			// Amazon's robot check.
			if strings.Contains( attrs[ "action" ] , "validateCaptcha" ) { found.blocked = true }
		}
		if id := attrs[ "id" ]; id != "" {
			if _ , seen := found.byID[ id ]; seen == false { found.byID[ id ] = node }
		}
		classes := " " + attrs[ "class" ] + " "
		if strings.Contains( classes , " a-price " ) { in_price = true }
		// The first price Amazon shows in an a-price block, read the way a
		// screen reader is given it.
		if in_price && found.offscreen == "" && strings.Contains( classes , " a-offscreen " ) {
			found.offscreen = clean( text( node ) )
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling { found.walk( child , in_price ) }
}

// fromLDJSON fills empty fields from a JSON-LD block's Product, wherever it
// sits: the top level, a list, or an @graph.
func fromLDJSON( raw string , info *Info ) {
	var data any
	if json.Unmarshal( []byte( strings.TrimSpace( raw ) ) , &data ) != nil { return }
	var visit func( value any , depth int )
	visit = func( value any , depth int ) {
		if depth > 6 { return }
		switch typed := value.( type ) {
		case []any:
			for _ , item := range typed { visit( item , depth + 1 ) }
		case map[string]any:
			if isProduct( typed[ "@type" ] ) {
				if info.Title == "" { info.Title = clean( stringOf( typed[ "name" ] ) ) }
				if info.Description == "" { info.Description = clean( stringOf( typed[ "description" ] ) ) }
				if info.Image == "" { info.Image = imageOf( typed[ "image" ] ) }
				if info.Price == "" { info.Price = priceOf( typed[ "offers" ] , 0 ) }
				return
			}
			if graph , ok := typed[ "@graph" ]; ok { visit( graph , depth + 1 ) }
		}
	}
	visit( data , 0 )
}

func isProduct( value any ) ( result bool ) {
	switch typed := value.( type ) {
	case string:
		result = typed == "Product" || strings.HasSuffix( typed , "/Product" ) || typed == "ProductGroup"
	case []any:
		for _ , item := range typed {
			if isProduct( item ) { return true }
		}
	}
	return
}

func stringOf( value any ) ( result string ) {
	switch typed := value.( type ) {
	case string:
		result = typed
	case float64:
		result = strconv.FormatFloat( typed , 'f' , -1 , 64 )
	}
	return
}

func imageOf( value any ) ( result string ) {
	switch typed := value.( type ) {
	case string:
		result = typed
	case []any:
		for _ , item := range typed {
			if result = imageOf( item ); result != "" { return }
		}
	case map[string]any:
		result = stringOf( typed[ "url" ] )
		if result == "" { result = stringOf( typed[ "contentUrl" ] ) }
	}
	return
}

func priceOf( value any , depth int ) ( result string ) {
	if depth > 4 { return }
	switch typed := value.( type ) {
	case []any:
		for _ , item := range typed {
			if result = priceOf( item , depth + 1 ); result != "" { return }
		}
	case map[string]any:
		amount := stringOf( typed[ "price" ] )
		if amount == "" { amount = stringOf( typed[ "lowPrice" ] ) }
		if amount == "" {
			if spec , ok := typed[ "priceSpecification" ]; ok { return priceOf( spec , depth + 1 ) }
			if offers , ok := typed[ "offers" ]; ok { return priceOf( offers , depth + 1 ) }
		}
		result = formatPrice( amount , stringOf( typed[ "priceCurrency" ] ) )
	}
	return
}

var currencySymbols = map[string]string{
	"USD": "$" , "EUR": "€" , "GBP": "£" , "JPY": "¥" , "INR": "₹" , "CAD": "CA$" , "AUD": "A$" ,
	"NZD": "NZ$" , "MXN": "MX$" , "BRL": "R$" , "CHF": "CHF " , "SEK": "SEK " , "PLN": "zł " ,
}

func formatPrice( amount string , currency string ) ( result string ) {
	amount = strings.TrimSpace( amount )
	if amount == "" { return }
	if number , err := strconv.ParseFloat( amount , 64 ); err == nil {
		amount = strconv.FormatFloat( number , 'f' , 2 , 64 )
		if currency == "JPY" { amount = strconv.FormatFloat( number , 'f' , 0 , 64 ) }
	}
	currency = strings.ToUpper( strings.TrimSpace( currency ) )
	if symbol , ok := currencySymbols[ currency ]; ok {
		result = symbol + amount
	} else if currency != "" {
		result = amount + " " + currency
	} else {
		result = amount
	}
	return
}

// amazonImage is the large photo an Amazon <img> points at: the hi-res
// attribute, else the biggest of its dynamic sizes, else its src.
func amazonImage( node *html.Node ) ( result string ) {
	if node == nil { return }
	attrs := map[string]string{}
	for _ , attr := range node.Attr { attrs[ attr.Key ] = attr.Val }
	if result = attrs[ "data-old-hires" ]; result != "" { return }
	if dynamic := attrs[ "data-a-dynamic-image" ]; dynamic != "" {
		sizes := map[string][]int{}
		if json.Unmarshal( []byte( dynamic ) , &sizes ) == nil {
			best := 0
			for candidate , size := range sizes {
				if len( size ) == 2 && size[ 0 ] * size[ 1 ] > best {
					best = size[ 0 ] * size[ 1 ]
					result = candidate
				}
			}
			if result != "" { return }
		}
	}
	result = attrs[ "src" ]
	if strings.HasPrefix( result , "data:" ) { result = "" }
	return
}

// offscreen is the first a-offscreen price text under node, or "".
func offscreen( node *html.Node ) ( result string ) {
	if node == nil { return }
	for _ , attr := range node.Attr {
		if attr.Key == "class" && strings.Contains( " " + attr.Val + " " , " a-offscreen " ) {
			if result = clean( text( node ) ); result != "" { return }
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result = offscreen( child ); result != "" { return }
	}
	return
}

// bullets is the first few points of Amazon's "About this item".
func bullets( node *html.Node ) ( result string ) {
	if node == nil { return }
	points := []string{}
	var visit func( current *html.Node )
	visit = func( current *html.Node ) {
		if len( points ) >= 3 { return }
		if current.Type == html.ElementNode && current.Data == "li" {
			if point := clean( text( current ) ); point != "" { points = append( points , point ) }
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling { visit( child ) }
	}
	visit( node )
	result = strings.Join( points , " · " )
	return
}

// text is all the text under a node, scripts and styles left out.
func text( node *html.Node ) ( result string ) {
	if node == nil { return }
	var builder strings.Builder
	var visit func( current *html.Node )
	visit = func( current *html.Node ) {
		if current.Type == html.TextNode { builder.WriteString( current.Data + " " ) }
		if current.Type == html.ElementNode && ( current.Data == "script" || current.Data == "style" ) { return }
		for child := current.FirstChild; child != nil; child = child.NextSibling { visit( child ) }
	}
	visit( node )
	result = clean( builder.String() )
	return
}

func clean( value string ) ( result string ) {
	result = strings.Join( strings.Fields( html.UnescapeString( value ) ) , " " )
	return
}

func truncate( value string , max_runes int ) ( result string ) {
	runes := []rune( value )
	if len( runes ) <= max_runes { return value }
	result = strings.TrimSpace( string( runes[ :max_runes - 1 ] ) ) + "…"
	return
}
