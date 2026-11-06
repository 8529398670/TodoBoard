package models

import (
	errors "errors"
	strconv "strconv"
	testing "testing"
	time "time"

	db "todoboard/server/db"
)

const (
	testP256dh = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	testAuth   = "BTBZMqHH6r4Tts7J_aSIgg"
)

func pushCount( t *testing.T , store *db.Store , user_id uint64 ) ( count int ) {
	t.Helper()
	found , err := ListPushSubscriptions( store , user_id )
	if err != nil { t.Fatal( err ) }
	count = len( found )
	return
}

func TestEnsureVAPIDKeysIsStable( t *testing.T ) {
	store := testStore( t )
	first , err := EnsureVAPIDKeys( store )
	if err != nil { t.Fatal( err ) }
	second , err := EnsureVAPIDKeys( store )
	if err != nil { t.Fatal( err ) }
	if first.Public == "" || *first != *second { t.Errorf( "the keys changed: %+v then %+v" , first , second ) }
}

func TestPushSubscriptionUpsert( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	first , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/a" , testP256dh , testAuth , "Laptop" )
	if err != nil { t.Fatal( err ) }
	if err = MarkPushDelivered( store , first , time.Now() ); err != nil { t.Fatal( err ) }
	second , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/a" , testP256dh , testAuth , "Renamed" )
	if err != nil { t.Fatal( err ) }
	if second.ID != first.ID || second.Label != "Renamed" || second.LastSuccess == nil { t.Errorf( "not the same browser: %+v" , second ) }
	if count := pushCount( t , store , owner.ID ); count != 1 { t.Errorf( "%d browsers" , count ) }
}

func TestPushSubscriptionRefusesBadKeys( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	if _ , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/a" , "AAAA" , testAuth , "" ); errors.Is( err , ErrBadInput ) == false { t.Errorf( "bad key: %v" , err ) }
}

func TestPushSubscriptionMovesBetweenUsers( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	if _ , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/shared" , testP256dh , testAuth , "" ); err != nil { t.Fatal( err ) }
	if _ , err := SavePushSubscription( store , bob.ID , "https://fcm.googleapis.com/shared" , testP256dh , testAuth , "" ); err != nil { t.Fatal( err ) }
	if count := pushCount( t , store , owner.ID ); count != 0 { t.Errorf( "the last person still has the browser" ) }
	if count := pushCount( t , store , bob.ID ); count != 1 { t.Errorf( "the new person has %d" , count ) }
}

func TestPushSubscriptionCap( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	for count := 0 ; count < maxPushSubscriptions ; count += 1 {
		if _ , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/"+strconv.Itoa( count ) , testP256dh , testAuth , "" ); err != nil { t.Fatal( err ) }
	}
	if _ , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/more" , testP256dh , testAuth , "" ); errors.Is( err , ErrTooManyDevices ) == false { t.Errorf( "over the cap: %v" , err ) }
	// One already here can still check in.
	if _ , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/0" , testP256dh , testAuth , "" ); err != nil { t.Errorf( "refresh at the cap: %v" , err ) }
}

func TestDeletePushSubscriptionIsOwnersOnly( t *testing.T ) {
	store := testStore( t )
	owner , _ := CreateUser( store , "Owner" , RoleAdmin )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	saved , err := SavePushSubscription( store , owner.ID , "https://fcm.googleapis.com/a" , testP256dh , testAuth , "" )
	if err != nil { t.Fatal( err ) }
	DeletePushSubscription( store , bob.ID , saved.ID )
	if count := pushCount( t , store , owner.ID ); count != 1 { t.Errorf( "someone else removed it" ) }
	if err = DeletePushSubscription( store , owner.ID , saved.ID ); err != nil { t.Fatal( err ) }
	if count := pushCount( t , store , owner.ID ); count != 0 { t.Errorf( "still there" ) }
}

func TestDeleteUserRemovesPushSubscriptions( t *testing.T ) {
	store := testStore( t )
	CreateUser( store , "Owner" , RoleAdmin )
	EnsureOwner( store )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	if _ , err := SavePushSubscription( store , bob.ID , "https://fcm.googleapis.com/a" , testP256dh , testAuth , "" ); err != nil { t.Fatal( err ) }
	if _ , err := DeleteUser( store , bob.ID , 0 ); err != nil { t.Fatal( err ) }
	if count := pushCount( t , store , bob.ID ); count != 0 { t.Errorf( "%d left" , count ) }
}

func TestRevokeRemovesPushSubscriptions( t *testing.T ) {
	store := testStore( t )
	CreateUser( store , "Owner" , RoleAdmin )
	EnsureOwner( store )
	bob , _ := CreateUser( store , "Bob" , RoleUser )
	if _ , err := SavePushSubscription( store , bob.ID , "https://fcm.googleapis.com/a" , testP256dh , testAuth , "" ); err != nil { t.Fatal( err ) }
	if _ , err := RevokeAllCredentials( store , bob.ID , false ); err != nil { t.Fatal( err ) }
	if count := pushCount( t , store , bob.ID ); count != 0 { t.Errorf( "%d left" , count ) }
}
