(function () {
  "use strict";

  const MIN_CHARS = 3;

  const form = document.getElementById("search-form");
  const input = document.getElementById("city-input");
  const errorBox = document.getElementById("search-error");
  const cityField = document.getElementById("city-value");
  const countryField = document.getElementById("country-value");
  const suggestions = document.getElementById("suggestions");

  // The city the user picked from a suggestion (untill select it will null)
  let selected = null;

  function showError(message) {
    errorBox.textContent = message;
    errorBox.hidden = false;
    input.setAttribute("aria-invalid", "true");
  }

  function clearError() {
    errorBox.textContent = "";
    errorBox.hidden = true;
    input.removeAttribute("aria-invalid");
  }

  function setSelection(city, countryCode) {
    selected = { city: city, countryCode: countryCode };
    cityField.value = city;
    countryField.value = countryCode;
    clearError();
  }

  function clearSelection() {
    selected = null;
    cityField.value = "";
    countryField.value = "";
  }

  // If the user edits the text after picking a city, the pick is no longer valid.
  input.addEventListener("input", function () {
    clearSelection();
    clearError();
    // fetch suggestions from /api/locations/autocomplete here
  });

  form.addEventListener("submit", function (e) {
    if (input.value.trim().length < MIN_CHARS) {
      e.preventDefault();
      showError("Type at least " + MIN_CHARS + " characters and select a suggestion.");
      input.focus();
      return;
    }
    if (!selected) {
      e.preventDefault();
      showError("Please select a city from the suggestions.");
      input.focus();
    }
    // Otherwise the form submits normally: GET /events?city=...&countryCode=...
  });

  // Hook for the autocomplete code you will add later
  window.HomeSearch = {
    MIN_CHARS: MIN_CHARS,
    input: input,
    suggestions: suggestions,
    selectCity: function (city, countryCode, label) {
      input.value = label || city;
      setSelection(city, countryCode);
    },
  };
})();