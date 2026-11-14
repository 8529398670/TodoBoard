package routes

import (
	errors "errors"
	strconv "strconv"

	fiber "github.com/gofiber/fiber/v3"

	access "todoboard/server/access"
	mcp "todoboard/server/mcp"
	models "todoboard/server/models"
)

// What the card tools add to their routes: get_card, a card's own route,
// and the words create_card and update_card answer with where the page
// reads the status and body instead.

// rankedCardPayload is cardPayload with the card's place in the board's
// ranking, which takes the rest of the board to work out.
func ( handlers *Handlers ) rankedCardPayload( principal *access.Principal , board *models.Board , card *models.Card ) ( payload fiber.Map ) {
	payload = cardPayload( card , access.Card( principal , board , card ) , access.Board( principal , board ) )
	payload[ "rank_place" ] = 0
	if card.Rank == 0 { return }
	if cards , err := models.ListCards( handlers.Store , board.ID ); err == nil {
		payload[ "rank_place" ] = models.RankPlaces( board , cards )[ card.ID ]
	}
	return
}

// getCard is one card, for an agent: the board's GET answers with every
// card on it.
//
// @route("GET /api/boards/{board_id}/cards/{card_id}", summary="Read a card",
//     tag="Cards", errors=(404,),
//     description="One card in full: title, description, column, labels, assignees, dates, checklist, its "
//         "place in the board's ranking (rank_place, 0 for none), and rev (pass rev back to update_card when "
//         "changing the title or description).",
//     mcp="get_card", read_only=True,
// )
func ( handlers *Handlers ) getCard( c fiber.Ctx , in *cardPath ) ( reply *fiber.Map , err error ) {
	principal , board , card , err := handlers.requireCardAt( c , in , access.View )
	if err != nil { return }
	payload := handlers.rankedCardPayload( principal , board , card )
	reply = &payload
	return
}

// withRankPlace adds rank_place to a card reply, as get_card has it.
func ( handlers *Handlers ) withRankPlace( payload fiber.Map ) ( result fiber.Map ) {
	result = payload
	result[ "rank_place" ] = 0
	rank , _ := payload[ "rank" ].( float64 )
	if rank == 0 { return }
	board_id , _ := payload[ "board_id" ].( uint64 )
	card_id , _ := payload[ "id" ].( uint64 )
	board , get_err := models.GetBoard( handlers.Store , board_id )
	if get_err != nil { return }
	if cards , err := models.ListCards( handlers.Store , board_id ); err == nil {
		result[ "rank_place" ] = models.RankPlaces( board , cards )[ card_id ]
	}
	return
}

// mcpCreateCard is create_card's answer: the card with its rank_place, or,
// when the card was made but its other fields were refused, words that say
// so and name it.
func ( handlers *Handlers ) mcpCreateCard( call *mcp.Call , in *createCardRequest , serve func( fiber.Ctx , *createCardRequest ) ( *fiber.Map , error ) ) ( result any , err error ) {
	reply , serve_err := serve( call.Ctx , in )
	var refused *refusal
	if errors.As( serve_err , &refused ) && refused.status < 500 {
		if card_id , made := refused.extra[ "card_id" ].( uint64 ); made {
			err = mcp.Refuse( "card " + strconv.FormatUint( card_id , 10 ) + " was created, but its other fields were not set: " + refused.message )
			return
		}
	}
	if result , err = mcp.Answer( reply , serve_err ); err != nil { return }
	result = handlers.withRankPlace( *reply )
	return
}

// mcpUpdateCard is update_card's answer: the card with its rank_place, or
// for a stale rev, what to do about it.
func ( handlers *Handlers ) mcpUpdateCard( call *mcp.Call , in *patchCardRequest , serve func( fiber.Ctx , *patchCardRequest ) ( *fiber.Map , error ) ) ( result any , err error ) {
	reply , serve_err := serve( call.Ctx , in )
	var refused *refusal
	if errors.As( serve_err , &refused ) && refused.message == "card_stale" {
		rev := ""
		if current , ok := refused.extra[ "card" ].( fiber.Map ); ok {
			if number , ok := current[ "rev" ].( uint64 ); ok { rev = " (it is now rev " + strconv.FormatUint( number , 10 ) + ")" }
		}
		err = mcp.Refuse( "card_stale: someone changed this card since that rev" + rev + ". Read it again with get_card before changing the title or description." )
		return
	}
	if result , err = mcp.Answer( reply , serve_err ); err != nil { return }
	result = handlers.withRankPlace( *reply )
	return
}
