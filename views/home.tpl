{{template "partials/header.tpl" .}}
<section class="hero">
  <div class="container hero-inner">
    <div class="hero-text">
      <p class="eyebrow">Less scrolling. More going.</p>
      <h1>A city of possibilities.<span>Find your next one.</span></h1>
      <p class="lead">
        Discover music and sports in one place.<br>
        Choose your city. Find something worth heading out for.
      </p>
      <ul class="chips">
        <li>Live music</li>
        <li>Sports &amp; matchdays</li>
        <li>One simple search</li>
      </ul>
    </div>

    <div class="hero-visual">
      <img src="/static/img/explore-icon.png" alt="" width="420" height="420">
    </div>
  </div>
</section>

<section class="search-section">
  <div class="container">
    <div class="search-card">
      <div class="search-head">
        <h2>Where are we going?</h2>
        <p>Start with a city, then explore what is on.</p>
      </div>

      <form id="search-form" action="/events" method="get" novalidate>
        <label for="city-input" class="field-label">Choose a city</label>

        <div class="search-row">
          <div class="input-wrap">
            <svg class="input-icon" width="20" height="20" viewBox="0 0 24 24" fill="none"
                 stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
              <circle cx="11" cy="11" r="7"></circle>
              <path d="M20 20l-3.5-3.5"></path>
            </svg>
            <input id="city-input" type="text" autocomplete="off"
                   placeholder="Search a city, e.g. Toronto"
                   role="combobox" aria-expanded="false" aria-controls="suggestions"
                   aria-describedby="search-hint search-error">
            <!-- Suggestions will be filled by JS later -->
            <ul id="suggestions" class="suggestions" role="listbox" hidden></ul>
          </div>

          <button type="submit" class="btn-primary">
            Explore events
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                 stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M5 12h14M13 6l6 6-6 6"></path>
            </svg>
          </button>
        </div>

        <!-- Filled by JS when the user picks a suggestion -->
        <input type="hidden" name="city" id="city-value">
        <input type="hidden" name="countryCode" id="country-value">

        <p id="search-error" class="error" role="alert" hidden></p>

        <div class="search-foot">
          <p id="search-hint" class="hint">Type at least 3 characters and select a suggestion.</p>
          <p class="hint">Suggestions powered by Google</p>
        </div>
      </form>
    </div>
  </div>
</section>

{{template "partials/footer.tpl" .}}