<article class="event-card">
  <div class="card-media">
    {{if .ImageURL}}
      <img src="{{.ImageURL}}" alt="" loading="lazy">
    {{else}}
      <div class="card-media-fallback" aria-hidden="true"></div>
    {{end}}
  </div>
  <div class="card-body">
    <p class="card-date">{{if .Date}}{{.Date}}{{else}}Date to be announced{{end}}</p>
    <h3 class="card-title">{{.Name}}</h3>
    <p class="card-venue">{{if .Venue}}{{.Venue}}{{else}}Venue to be announced{{end}}</p>
  </div>
  <div class="card-foot">
    <span class="card-city">{{.City}}</span>
    <a href="/events/{{.ID}}" class="card-link" aria-label="View details for {{.Name}}">
      View details
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor"
           stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M7 17L17 7M8 7h9v9"></path>
      </svg>
    </a>
  </div>
</article>