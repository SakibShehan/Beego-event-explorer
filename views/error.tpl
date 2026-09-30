{{template "partials/header.tpl" .}}

<section class="error-page">
  <div class="container">
    <h1>{{.Heading}}</h1>
    <p class="lead">{{.Message}}</p>
    <p><a href="/" class="btn-outline">Back to search</a></p>
  </div>
</section>

{{template "partials/footer.tpl" .}}