// Sharing, for a board's owner: who is on the board, share links for the
// board or a single card (view or edit, for people without an account), and
// sharing one card with an existing account.
//
// A new link's secret is shown once, at creation -- the server only keeps a
// hash, so a lost link is revoked and replaced, never recovered.
const Sharing = {
  team: null,

  async loadTeam() {
    if ( !this.team ) this.team = await Api.listTeam();
    return this.team;
  },

  open( ctx , back ) {
    Panel.open( "share.board_title" , async ( body ) => {
      const members = Dom.el( "section" , { class: "panel-section" } );
      const links = Dom.el( "section" , { class: "panel-section" } );
      body.appendChild( members );
      body.appendChild( links );
      await this.renderMembers( members , ctx );
      await this.renderLinks( links , ctx , {} );
    } , back );
  },

  // Links to one column: whoever holds one sees that column as a board of
  // its own.
  openList( ctx , list ) {
    Panel.open( "share.list_title" , async ( body ) => {
      body.appendChild( Dom.el( "p" , { class: "muted" , text: I18n.fill( "share.list_intro" , { list: list.title } ) } ) );
      const links = Dom.el( "section" , { class: "panel-section" } );
      body.appendChild( links );
      await this.renderLinks( links , ctx , { list_id: list.id } );
    } );
  },

  openCard( ctx , card ) {
    Panel.open( "share.card_title" , async ( body ) => {
      body.appendChild( Dom.el( "p" , { class: "muted" , text: I18n.fill( "share.card_intro" , { card: card.title } ) } ) );
      const people = Dom.el( "section" , { class: "panel-section" } );
      const links = Dom.el( "section" , { class: "panel-section" } );
      body.appendChild( people );
      body.appendChild( links );
      await this.renderCardPeople( people , ctx , card );
      await this.renderLinks( links , ctx , { card_id: card.id } );
    } );
  },

  async renderMembers( section , ctx ) {
    Dom.clear( section );
    section.appendChild( Dom.el( "h3" , { text: T( "share.members_heading" ) } ) );
    const table = Dom.el( "ul" , { class: "member-list" } );
    ctx.board.members.forEach( ( member ) => {
      const row = Dom.el( "li" , { class: "member-row" } );
      row.appendChild( Dom.el( "span" , { class: "avatar" , text: CardBits.initials( member.display_name ) } ) );
      row.appendChild( Dom.el( "span" , { class: "member-name" , text: member.display_name } ) );
      if ( member.role === "owner" ) {
        row.appendChild( Dom.el( "span" , { class: "muted" , text: T( "share.role_owner" ) } ) );
      } else {
        const role = Ui.select( [
          { value: "editor" , text: T( "share.role_editor" ) },
          { value: "viewer" , text: T( "share.role_viewer" ) },
        ] , member.role );
        role.setAttribute( "aria-label" , T( "share.role_label" ) );
        role.addEventListener( "change" , async () => {
          await ctx.mutate( function () { return ctx.client.setMember( member.user_id , role.value ); } );
          this.renderMembers( section , BoardPage.context() );
        } );
        row.appendChild( role );
        row.appendChild( Ui.button( "share.remove" , async () => {
          if ( !window.confirm( I18n.fill( "share.remove_confirm" , { name: member.display_name } ) ) ) return;
          await ctx.mutate( function () { return ctx.client.setMember( member.user_id , "" ); } );
          this.renderMembers( section , BoardPage.context() );
        } ) );
      }
      table.appendChild( row );
    } );
    section.appendChild( table );

    let team = [];
    try { team = await this.loadTeam(); } catch ( error ) { team = []; }
    const existing = {};
    ctx.board.members.forEach( function ( member ) { existing[ member.user_id ] = true; } );
    const candidates = team.filter( function ( person ) { return !existing[ person.id ]; } );
    if ( !candidates.length ) {
      section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "share.no_one_to_add" ) } ) );
      return;
    }
    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const person = Ui.select( candidates.map( function ( item ) { return { value: item.id , text: item.display_name }; } ) , "" );
    const role = Ui.select( [
      { value: "editor" , text: T( "share.role_editor" ) },
      { value: "viewer" , text: T( "share.role_viewer" ) },
    ] , "editor" );
    form.appendChild( Ui.field( "share.person_label" , person ) );
    form.appendChild( Ui.field( "share.role_label" , role ) );
    form.appendChild( Dom.el( "button" , { text: T( "share.add_member" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      await ctx.mutate( function () { return ctx.client.setMember( Number( person.value ) , role.value ); } );
      this.renderMembers( section , BoardPage.context() );
    } );
    section.appendChild( form );
  },

  async renderCardPeople( section , ctx , card ) {
    Dom.clear( section );
    section.appendChild( Dom.el( "h3" , { text: T( "share.card_people_heading" ) } ) );
    const shared = card.shared_with || {};
    const list = Dom.el( "ul" , { class: "member-list" } );
    Object.keys( shared ).forEach( ( id ) => {
      const name = ctx.people[ id ] || id;
      list.appendChild( Dom.el( "li" , { class: "member-row" , children: [
        Dom.el( "span" , { class: "member-name" , text: name } ),
        Dom.el( "span" , { class: "muted" , text: T( "share.mode_" + shared[ id ] ) } ),
        Ui.button( "share.remove" , async () => {
          await ctx.mutate( function () { return ctx.client.shareCardWithUser( card.id , Number( id ) , "" ); } );
          const fresh = BoardPage.context();
          this.renderCardPeople( section , fresh , fresh.card( card.id ) );
        } ),
      ] } ) );
    } );
    if ( !Object.keys( shared ).length ) list.appendChild( Dom.el( "li" , { class: "muted" , text: T( "share.card_people_none" ) } ) );
    section.appendChild( list );

    let team = [];
    try { team = await this.loadTeam(); } catch ( error ) { team = []; }
    const onBoard = {};
    ctx.board.members.forEach( function ( member ) { onBoard[ member.user_id ] = true; } );
    const candidates = team.filter( function ( person ) { return !onBoard[ person.id ] && !shared[ person.id ]; } );
    if ( !candidates.length ) return;
    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const person = Ui.select( candidates.map( function ( item ) { return { value: item.id , text: item.display_name }; } ) , "" );
    const mode = Ui.select( [ { value: "edit" , text: T( "share.mode_edit" ) } , { value: "view" , text: T( "share.mode_view" ) } ] , "edit" );
    form.appendChild( Ui.field( "share.person_label" , person ) );
    form.appendChild( Ui.field( "share.mode_label" , mode ) );
    form.appendChild( Dom.el( "button" , { text: T( "share.share_card_with" ) , attrs: { type: "submit" } } ) );
    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      await ctx.mutate( function () { return ctx.client.shareCardWithUser( card.id , Number( person.value ) , mode.value ); } );
      const fresh = BoardPage.context();
      this.renderCardPeople( section , fresh , fresh.card( card.id ) );
    } );
    section.appendChild( form );
  },

  // Share links for the board (scope {}), one card ({ card_id }) or one
  // column ({ list_id }).
  async renderLinks( section , ctx , scope ) {
    Dom.clear( section );
    const heading = scope.card_id ? "share.card_links_heading" : scope.list_id ? "share.list_links_heading" : "share.links_heading";
    section.appendChild( Dom.el( "h3" , { text: T( heading ) } ) );
    section.appendChild( Dom.el( "p" , { class: "muted" , text: T( "share.links_intro" ) } ) );

    const form = Dom.el( "form" , { attrs: { autocomplete: "off" } } );
    const mode = Ui.select( [ { value: "view" , text: T( "share.mode_view" ) } , { value: "edit" , text: T( "share.mode_edit" ) } ] , "view" );
    const label = Dom.el( "input" , { attrs: { maxlength: "80" , placeholder: T( "share.link_label_placeholder" ) } } );
    const days = Dom.el( "input" , { attrs: { type: "number" , min: "0" , max: "3650" , value: "0" , inputmode: "numeric" } } );
    form.appendChild( Ui.field( "share.mode_label" , mode ) );
    form.appendChild( Ui.field( "share.link_label" , label ) );
    form.appendChild( Ui.field( "share.link_days" , days ) );
    form.appendChild( Dom.el( "button" , { text: T( "share.create_link" ) , attrs: { type: "submit" } } ) );
    section.appendChild( form );

    const created = Dom.el( "div" , { class: "notice notice-link" } );
    created.hidden = true;
    section.appendChild( created );

    form.addEventListener( "submit" , async ( event ) => {
      event.preventDefault();
      try {
        const link = await ctx.client.createShareLink( { card_id: scope.card_id || 0 , list_id: scope.list_id || 0 , mode: mode.value , label: label.value , days: Number( days.value ) || 0 } );
        Dom.clear( created );
        const url = Dom.el( "code" , { text: window.location.origin + link.share_path } );
        const notice = Dom.el( "span" , { class: "muted" , text: T( "share.copied" ) } );
        notice.hidden = true;
        created.appendChild( Dom.el( "p" , { text: T( "share.link_created" ) } ) );
        created.appendChild( url );
        created.appendChild( Ui.button( "share.copy_link" , function () { Dom.copy( url , notice ); } ) );
        created.appendChild( notice );
        created.hidden = false;
        label.value = "";
        await this.renderLinkList( list , ctx , scope );
      } catch ( error ) {
        Ui.error( error );
      }
    } );

    const list = Dom.el( "ul" , { class: "link-list" } );
    section.appendChild( list );
    await this.renderLinkList( list , ctx , scope );
  },

  scopeText( ctx , link ) {
    if ( link.card_id ) {
      const card = ctx.card( link.card_id );
      return I18n.fill( "share.scope_card" , { card: card ? card.title : "#" + link.card_id } );
    }
    if ( link.list_id ) {
      const column = ctx.board.lists.find( function ( item ) { return item.id === link.list_id; } );
      return I18n.fill( "share.scope_list" , { list: column ? column.title : "#" + link.list_id } );
    }
    return T( "share.scope_board" );
  },

  async renderLinkList( list , ctx , scope ) {
    Dom.clear( list );
    let links = [];
    try { links = await ctx.client.shareLinks(); } catch ( error ) { Ui.error( error ); return; }
    links = links.filter( function ( link ) {
      if ( scope.card_id ) return link.card_id === scope.card_id;
      if ( scope.list_id ) return link.list_id === scope.list_id;
      return true;
    } );
    if ( !links.length ) {
      list.appendChild( Dom.el( "li" , { class: "muted" , text: T( "share.no_links" ) } ) );
      return;
    }
    links.forEach( ( link ) => {
      const where = this.scopeText( ctx , link );
      const status = link.live ? ( link.expires_at ? I18n.fill( "share.expires" , { when: Dates.format( link.expires_at ) } ) : T( "share.never_expires" ) ) : T( link.revoked_at ? "share.revoked" : "share.expired" );
      const row = Dom.el( "li" , { class: "link-row" + ( link.live ? "" : " is-dead" ) } );
      row.appendChild( Dom.el( "div" , { children: [
        Dom.el( "strong" , { text: link.label || T( "share.unlabelled" ) } ),
        Dom.el( "div" , { class: "muted" , text: [ where , T( "share.mode_" + link.mode ) , status ].join( " · " ) } ),
        Dom.el( "div" , { class: "muted" , text: link.last_used_at ? I18n.fill( "share.last_used" , { when: Dates.format( link.last_used_at ) } ) : T( "share.never_used" ) } ),
      ] } ) );
      const actions = Dom.el( "div" , { class: "row-actions" } );
      if ( link.live ) actions.appendChild( Ui.button( "share.revoke" , async () => {
        try { await ctx.client.revokeShareLink( link.id ); } catch ( error ) { Ui.error( error ); }
        this.renderLinkList( list , ctx , scope );
      } ) );
      actions.appendChild( Ui.button( "share.delete_link" , async () => {
        try { await ctx.client.deleteShareLink( link.id ); } catch ( error ) { Ui.error( error ); }
        this.renderLinkList( list , ctx , scope );
      } , "secondary small danger" ) );
      row.appendChild( actions );
      list.appendChild( row );
    } );
  },
};
