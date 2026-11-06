package encryption

import (
	ecdh "crypto/ecdh"
	ecdsa "crypto/ecdsa"
	elliptic "crypto/elliptic"
	aes "crypto/aes"
	cipher "crypto/cipher"
	hkdf "crypto/hkdf"
	sha256 "crypto/sha256"
	base64 "encoding/base64"
	json "encoding/json"
	big "math/big"
	strings "strings"
	testing "testing"
	time "time"
)

func decoded( t *testing.T , value string ) ( result []byte ) {
	t.Helper()
	result , err := base64.RawURLEncoding.DecodeString( value )
	if err != nil { t.Fatal( err ) }
	return
}

// The worked example in RFC 8291, Appendix A.
func TestWebPushRFC8291Vector( t *testing.T ) {
	key , err := ecdh.P256().NewPrivateKey( decoded( t , "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw" ) )
	if err != nil { t.Fatal( err ) }
	body , err := encryptWebPush(
		"BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4" ,
		"BTBZMqHH6r4Tts7J_aSIgg" ,
		[]byte( "When I grow up, I want to be a watermelon" ) ,
		key ,
		decoded( t , "DGv6ra1nlYgDCS1FRnbzlw" ) ,
	)
	if err != nil { t.Fatal( err ) }
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if got := base64.RawURLEncoding.EncodeToString( body ); got != want { t.Errorf( "got  %s\nwant %s" , got , want ) }
}

// A browser's side of it, written from the standard again: the sealed body
// has to open with the browser's private key.
func TestWebPushRoundTrip( t *testing.T ) {
	browser , err := ecdh.P256().GenerateKey( nil )
	if err != nil { t.Fatal( err ) }
	secret := GenerateRandomBytes( 16 )
	encode := base64.RawURLEncoding.EncodeToString
	message := []byte( `{"title":"Hello"}` )
	body , err := EncryptWebPush( encode( browser.PublicKey().Bytes() ) , encode( secret ) , message )
	if err != nil { t.Fatal( err ) }

	salt , server_bytes , sealed := body[ :16 ] , body[ 21:86 ] , body[ 86: ]
	if body[ 20 ] != 65 { t.Fatalf( "key id length %d" , body[ 20 ] ) }
	server , err := ecdh.P256().NewPublicKey( server_bytes )
	if err != nil { t.Fatal( err ) }
	shared , err := browser.ECDH( server )
	if err != nil { t.Fatal( err ) }
	info := "WebPush: info\x00" + string( browser.PublicKey().Bytes() ) + string( server_bytes )
	input , _ := hkdf.Key( sha256.New , shared , secret , info , 32 )
	content_key , _ := hkdf.Key( sha256.New , input , salt , "Content-Encoding: aes128gcm\x00" , 16 )
	nonce , _ := hkdf.Key( sha256.New , input , salt , "Content-Encoding: nonce\x00" , 12 )
	block , _ := aes.NewCipher( content_key )
	opener , _ := cipher.NewGCM( block )
	opened , err := opener.Open( nil , nonce , sealed , nil )
	if err != nil { t.Fatal( err ) }
	if string( opened ) != string( message ) + "\x02" { t.Errorf( "opened: %q" , opened ) }
}

func TestWebPushRefusesBadKeys( t *testing.T ) {
	if ValidWebPushKeys( "AAAA" , "BTBZMqHH6r4Tts7J_aSIgg" ) { t.Errorf( "a short key passed" ) }
	good := "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	if ValidWebPushKeys( good , "AAAA" ) { t.Errorf( "a short secret passed" ) }
	if ValidWebPushKeys( good , "BTBZMqHH6r4Tts7J_aSIgg" ) == false { t.Errorf( "good keys refused" ) }
	if _ , err := EncryptWebPush( good , "BTBZMqHH6r4Tts7J_aSIgg" , make( []byte , 5000 ) ) ; err == nil { t.Errorf( "an oversized message was sealed" ) }
}

func TestVAPIDTokenVerifies( t *testing.T ) {
	private_key , public_key , err := GenerateVAPIDKeys()
	if err != nil { t.Fatal( err ) }
	now := time.Unix( 1700000000 , 0 )
	header , err := VAPIDAuthorization( private_key , public_key , "https://fcm.googleapis.com" , "mailto:ops@example.com" , now )
	if err != nil { t.Fatal( err ) }
	token , rest , ok := strings.Cut( strings.TrimPrefix( header , "vapid t=" ) , ", k=" )
	if ok == false || rest != public_key { t.Fatalf( "header: %s" , header ) }
	parts := strings.Split( token , "." )
	if len( parts ) != 3 { t.Fatalf( "token: %s" , token ) }

	var claims struct {
		Aud string `json:"aud"`
		Exp int64  `json:"exp"`
		Sub string `json:"sub"`
	}
	if err = json.Unmarshal( decoded( t , parts[ 1 ] ) , &claims ); err != nil { t.Fatal( err ) }
	if claims.Aud != "https://fcm.googleapis.com" || claims.Sub != "mailto:ops@example.com" || claims.Exp != now.Add( 12 * time.Hour ).Unix() { t.Errorf( "claims: %+v" , claims ) }

	public_bytes := decoded( t , public_key )
	public := &ecdsa.PublicKey{ Curve: elliptic.P256() , X: new( big.Int ).SetBytes( public_bytes[ 1:33 ] ) , Y: new( big.Int ).SetBytes( public_bytes[ 33: ] ) }
	signature := decoded( t , parts[ 2 ] )
	if len( signature ) != 64 { t.Fatalf( "signature is %d bytes" , len( signature ) ) }
	digest := sha256.Sum256( []byte( parts[ 0 ] + "." + parts[ 1 ] ) )
	if ecdsa.Verify( public , digest[ : ] , new( big.Int ).SetBytes( signature[ :32 ] ) , new( big.Int ).SetBytes( signature[ 32: ] ) ) == false {
		t.Errorf( "the signature does not verify" )
	}
}
