package encryption

import (
	ecdh "crypto/ecdh"
	ecdsa "crypto/ecdsa"
	elliptic "crypto/elliptic"
	aes "crypto/aes"
	cipher "crypto/cipher"
	hkdf "crypto/hkdf"
	random "crypto/rand"
	sha256 "crypto/sha256"
	base64 "encoding/base64"
	binary "encoding/binary"
	json "encoding/json"
	errors "errors"
	fmt "fmt"
	time "time"
)

// Web push, the two standards a browser's push service asks for: RFC 8292
// (VAPID) says who is sending, and RFC 8291 (aes128gcm) keeps the message
// from the push service itself. Written here rather than pulled in because
// it is a few dozen lines on the standard library, and this package is where
// the project's cryptography is meant to be readable in one place.

const webPushRecordSize = 4096

// VAPID tokens last 12 hours; the standard allows up to 24.
const vapidTokenLife = 12 * time.Hour

var ErrWebPushKeys = errors.New( "invalid web push keys" )

func webPushDecode( value string ) ( result []byte , err error ) {
	result , err = base64.RawURLEncoding.DecodeString( value )
	return
}

// GenerateVAPIDKeys makes a server's P-256 key pair: the private scalar, and
// the uncompressed public point browsers subscribe against.
func GenerateVAPIDKeys() ( private_key string , public_key string , err error ) {
	key , err := ecdh.P256().GenerateKey( random.Reader )
	if err != nil { return }
	private_key = base64.RawURLEncoding.EncodeToString( key.Bytes() )
	public_key = base64.RawURLEncoding.EncodeToString( key.PublicKey().Bytes() )
	return
}

// VAPIDAuthorization is the Authorization header for a push to audience, the
// origin of the endpoint. subject is how the push service reaches the sender.
func VAPIDAuthorization( private_key string , public_key string , audience string , subject string , now time.Time ) ( header string , err error ) {
	raw , err := webPushDecode( private_key )
	if err != nil { return }
	key , err := ecdsa.ParseRawPrivateKey( elliptic.P256() , raw )
	if err != nil { return }
	claims , err := json.Marshal( map[string]any{ "aud": audience , "exp": now.Add( vapidTokenLife ).Unix() , "sub": subject } )
	if err != nil { return }
	encode := base64.RawURLEncoding.EncodeToString
	signing := encode( []byte( `{"typ":"JWT","alg":"ES256"}` ) ) + "." + encode( claims )
	digest := sha256.Sum256( []byte( signing ) )
	r , s , err := ecdsa.Sign( random.Reader , key , digest[ : ] )
	if err != nil { return }
	signature := make( []byte , 64 )
	r.FillBytes( signature[ :32 ] )
	s.FillBytes( signature[ 32: ] )
	header = fmt.Sprintf( "vapid t=%s.%s, k=%s" , signing , encode( signature ) , public_key )
	return
}

// ValidWebPushKeys checks what a browser sent when it subscribed: a point on
// the curve, and a 16-byte secret.
func ValidWebPushKeys( p256dh string , auth string ) ( result bool ) {
	public , err := webPushDecode( p256dh )
	if err != nil { return }
	if _ , err = ecdh.P256().NewPublicKey( public ); err != nil { return }
	secret , err := webPushDecode( auth )
	result = err == nil && len( secret ) == 16
	return
}

// EncryptWebPush seals plaintext for one browser, as a single aes128gcm
// record, with a fresh key and salt each time.
func EncryptWebPush( p256dh string , auth string , plaintext []byte ) ( body []byte , err error ) {
	key , err := ecdh.P256().GenerateKey( random.Reader )
	if err != nil { return }
	salt := GenerateRandomBytes( 16 )
	body , err = encryptWebPush( p256dh , auth , plaintext , key , salt )
	return
}

// encryptWebPush takes its key and salt, so that the RFC's worked example can
// be run through it as a test.
func encryptWebPush( p256dh string , auth string , plaintext []byte , key *ecdh.PrivateKey , salt []byte ) ( body []byte , err error ) {
	browser_bytes , err := webPushDecode( p256dh )
	if err != nil { return nil , ErrWebPushKeys }
	secret , err := webPushDecode( auth )
	if err != nil || len( secret ) != 16 { return nil , ErrWebPushKeys }
	browser , err := ecdh.P256().NewPublicKey( browser_bytes )
	if err != nil { return nil , ErrWebPushKeys }
	if len( plaintext ) + 1 + 16 > webPushRecordSize { return nil , errors.New( "web push message too long" ) }

	shared , err := key.ECDH( browser )
	if err != nil { return }
	server_bytes := key.PublicKey().Bytes()
	info := "WebPush: info\x00" + string( browser_bytes ) + string( server_bytes )
	input , err := hkdf.Key( sha256.New , shared , secret , info , 32 )
	if err != nil { return }
	content_key , err := hkdf.Key( sha256.New , input , salt , "Content-Encoding: aes128gcm\x00" , 16 )
	if err != nil { return }
	nonce , err := hkdf.Key( sha256.New , input , salt , "Content-Encoding: nonce\x00" , 12 )
	if err != nil { return }

	block , err := aes.NewCipher( content_key )
	if err != nil { return }
	sealer , err := cipher.NewGCM( block )
	if err != nil { return }
	// 0x02 marks the last record, and there is only ever the one.
	record := append( append( []byte( nil ) , plaintext... ) , 2 )
	sealed := sealer.Seal( nil , nonce , record , nil )

	body = make( []byte , 0 , 21 + len( server_bytes ) + len( sealed ) )
	body = append( body , salt... )
	body = binary.BigEndian.AppendUint32( body , webPushRecordSize )
	body = append( body , byte( len( server_bytes ) ) )
	body = append( body , server_bytes... )
	body = append( body , sealed... )
	return
}
