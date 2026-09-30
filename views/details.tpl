{{template "partials/header.tpl" .}}

<div class="details-page">
  <div class="container">

    <nav class="breadcrumb" aria-label="Breadcrumb">
      <a href="/">Discover</a>
      {{if .Event.City}}
        <span aria-hidden="true">/</span>
        <a href="{{.BackURL}}">{{.Event.City}}</a>
      {{end}}
      <span aria-hidden="true">/</span>
      <span aria-current="page">Event details</span>
    </nav>

    <a href="{{.BackURL}}" class="back-link">&larr; Back to events</a>

    <div class="details-grid">

      <article class="details-main">
        <div class="details-hero">
          {{if .Event.HeroURL}}
            <img src="{{.Event.HeroURL}}" alt="{{.Event.Name}}">
          {{else}}
            <div class="card-media-fallback" aria-hidden="true"></div>
          {{end}}
        </div>

        <p class="details-category">{{if .Event.Category}}{{.Event.Category}}{{else}}Event{{end}}</p>
        <h1 class="details-title">{{.Event.Name}}</h1>

        <hr class="details-rule">

        <h2 class="details-about">About this event</h2>
        {{if .Event.Description}}
          <p class="details-desc">{{.Event.Description}}</p>
        {{else}}
          <p class="details-desc muted">No description is available for this event yet.</p>
        {{end}}
      </article>

      <aside class="details-card" aria-label="Event details">
        <p class="section-eyebrow">Make a plan</p>
        <h2 class="details-card-title">The details</h2>

        <div class="detail-item">
          <p class="detail-label">When</p>
          <p class="detail-value">{{if .Event.Date}}{{.Event.Date}}{{else}}Date to be announced{{end}}</p>
          {{if .Event.Time}}
            <p class="detail-sub">{{.Event.Time}}{{if .Event.Timezone}} ({{.Event.Timezone}}){{end}}</p>
          {{end}}
        </div>

        <div class="detail-item">
          <p class="detail-label">Where</p>
          <p class="detail-value">{{if .Event.Venue}}{{.Event.Venue}}{{else}}Venue to be announced{{end}}</p>
          {{if .Event.City}}
            <p class="detail-sub">{{.Event.City}}{{if .Event.CountryCode}}, {{.Event.CountryCode}}{{end}}</p>
          {{end}}
        </div>

        {{if .Event.Category}}
        <div class="detail-item">
          <p class="detail-label">Category</p>
          <p class="detail-value">{{.Event.Category}}</p>
        </div>
        {{end}}

        <a href="{{.Event.TicketURL}}" class="btn-primary btn-block" rel="nofollow">
  View tickets
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
       stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path d="M7 17L17 7M8 7h9v9"></path>
  </svg>
</a>
<p class="detail-note">You will continue on Ticketmaster to buy tickets.</p>
      </aside>

    </div>
  </div>
</div>

{{template "partials/footer.tpl" .}}