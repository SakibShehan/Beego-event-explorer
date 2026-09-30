{{template "partials/header.tpl" .}}

<section class="ticket-page">
  <div class="container">
    <div class="ticket-card">
      <div class="ticket-check" aria-hidden="true">
        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 12.5l4.5 4.5L19 7.5"></path>
        </svg>
      </div>
      <h1>
        Your ticket is ready for
        <span>{{.Event.Name}}</span>{{if .Event.City}} in {{.Event.City}}{{end}}.
      </h1>
      <a href="{{.BackURL}}" class="back-link">&larr; Back to event</a>
    </div>
  </div>
</section>

{{template "partials/footer.tpl" .}}