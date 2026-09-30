{{template "partials/header.tpl" .}}

<section class="listing-head">
  <div class="container">
    <nav class="breadcrumb" aria-label="Breadcrumb">
      <a href="/">Discover</a>
      <span aria-hidden="true">/</span>
      <span aria-current="page">{{.City}}</span>
    </nav>

    <div class="listing-title-row">
      <div>
        <p class="eyebrow">Your city. Your next plan.</p>
        <h1 class="listing-title">What is on in <span>{{.City}}.</span></h1>
        <p class="lead">Music and sports, loaded together. Find your next reason to go out.</p>
      </div>
      <a href="/" class="btn-outline">
        Change city
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="M7 17L17 7M8 7h9v9"></path>
        </svg>
      </a>
    </div>

    <dl class="meta-row">
      <div><dt>Location</dt><dd>{{.City}}, {{.Country}}</dd></div>
    </dl>
  </div>
</section>

<section class="event-section" aria-labelledby="music-heading">
  <div class="container">
    <p class="section-eyebrow">Turn up the evening</p>
    <div class="section-head">
      <h2 id="music-heading">Music</h2>
      {{if not .MusicErr}}<span class="count-badge">{{len .Music}}</span>{{end}}
    </div>

    {{if .MusicErr}}
      <p class="notice notice-error" role="alert">{{.MusicErr}}</p>
    {{else if not .Music}}
      <p class="notice">No music events found in {{.City}} right now. Try another city.</p>
    {{else}}
      <div class="event-grid">
        {{range .Music}}{{template "partials/event_card.tpl" .}}{{end}}
      </div>
    {{end}}
  </div>
</section>

<section class="event-section" aria-labelledby="sports-heading">
  <div class="container">
    <p class="section-eyebrow">Catch the action</p>
    <div class="section-head">
      <h2 id="sports-heading">Sports</h2>
      {{if not .SportsErr}}<span class="count-badge">{{len .Sports}}</span>{{end}}
    </div>

    {{if .SportsErr}}
      <p class="notice notice-error" role="alert">{{.SportsErr}}</p>
    {{else if not .Sports}}
      <p class="notice">No sports events found in {{.City}} right now. Try another city.</p>
    {{else}}
      <div class="event-grid">
        {{range .Sports}}{{template "partials/event_card.tpl" .}}{{end}}
      </div>
    {{end}}
  </div>
</section>

{{template "partials/footer.tpl" .}}