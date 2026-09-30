(function () {
  "use strict";

  const MIN_CHARS = 3;
  const DEBOUNCE_MS = 300;

  const form = document.getElementById("search-form");
  const input = document.getElementById("city-input");
  const errorBox = document.getElementById("search-error");
  const cityField = document.getElementById("city-value");
  const countryField = document.getElementById("country-value");
  const list = document.getElementById("suggestions");

  let selected = null;        //  city and countryCode/ once the user picks one
  let sessionToken = newToken();
  let debounceTimer = null;
  let controller = null;      // cancels the previous in-flight request
  let activeIndex = -1;

  function newToken() {
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID(); // 36 chars
    return "t" + Date.now().toString(36) + Math.random().toString(36).slice(2, 10);
  }

  function showError(msg) {
    errorBox.textContent = msg;
    errorBox.hidden = false;
    input.setAttribute("aria-invalid", "true");
  }

  function clearError() {
    errorBox.textContent = "";
    errorBox.hidden = true;
    input.removeAttribute("aria-invalid");
  }

  function clearSelection() {
    selected = null;
    cityField.value = "";
    countryField.value = "";
  }

  function hideList() {
    list.hidden = true;
    list.innerHTML = "";
    activeIndex = -1;
    input.setAttribute("aria-expanded", "false");
  }

  function showList() {
    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
  }

  function renderMessage(text) {
    list.innerHTML = "";
    const li = document.createElement("li");
    li.textContent = text;
    li.style.cursor = "default";
    li.style.color = "#5b6b68";
    list.appendChild(li);
    showList();
  }

  function renderSuggestions(items) {
    list.innerHTML = "";
    activeIndex = -1;
    if (!items.length) {
      renderMessage("No cities found.");
      return;
    }
    items.forEach(function (s, i) {
      const li = document.createElement("li");
      li.setAttribute("role", "option");
      li.dataset.placeId = s.placeId;
      li.dataset.text = s.text;
      li.textContent = s.text; // textContent: safe against HTML injection
      li.addEventListener("mousedown", function (e) {
        e.preventDefault(); // keep focus so blur doesn't hide the list first
        choose(i);
      });
      list.appendChild(li);
    });
    showList();
  }

  function setActive(i) {
    const items = list.querySelectorAll('li[role="option"]');
    if (!items.length) return;
    activeIndex = (i + items.length) % items.length;
    items.forEach(function (li, idx) {
      li.setAttribute("aria-selected", idx === activeIndex ? "true" : "false");
    });
  }

  async function fetchSuggestions(text) {
    if (controller) controller.abort();
    controller = new AbortController();

    const url =
      "/api/locations/autocomplete?input=" + encodeURIComponent(text) +
      "&sessionToken=" + encodeURIComponent(sessionToken);

    try {
      const res = await fetch(url, { signal: controller.signal });
      if (!res.ok) throw new Error("status " + res.status);
      const data = await res.json();
      renderSuggestions(data.suggestions || []);
    } catch (err) {
      if (err.name === "AbortError") return; // a newer request will comes here
      console.error("autocomplete failed:", err);
      renderMessage("Suggestions are unavailable right now.");
    }
  }

  async function choose(index) {
    const items = list.querySelectorAll('li[role="option"]');
    const li = items[index];
    if (!li) return;

    const placeId = li.dataset.placeId;
    const label = li.dataset.text;
    input.value = label;
    hideList();
    clearError();

    try {
      const res = await fetch(
        "/api/locations/" + encodeURIComponent(placeId) +
        "?sessionToken=" + encodeURIComponent(sessionToken)
      );
      if (!res.ok) throw new Error("status " + res.status);
      const loc = await res.json();

      selected = { city: loc.city, countryCode: loc.countryCode };
      cityField.value = loc.city;
      countryField.value = loc.countryCode;
      input.value = loc.label || label;

      // Selection finished: the next search starts a new session
      sessionToken = newToken();
    } catch (err) {
      console.error("place lookup failed:", err);
      clearSelection();
      showError("Could not load that city. Please try another suggestion.");
    }
  }

  input.addEventListener("input", function () {
    clearSelection();
    clearError();
    clearTimeout(debounceTimer);

    const text = input.value.trim();
    if (text.length < MIN_CHARS) {
      if (controller) controller.abort();
      hideList();
      return;
    }
    debounceTimer = setTimeout(function () {
      fetchSuggestions(text);
    }, DEBOUNCE_MS);
  });

  input.addEventListener("keydown", function (e) {
    if (list.hidden) return;
    if (e.key === "ArrowDown") { e.preventDefault(); setActive(activeIndex + 1); }
    else if (e.key === "ArrowUp") { e.preventDefault(); setActive(activeIndex - 1); }
    else if (e.key === "Enter" && activeIndex >= 0) { e.preventDefault(); choose(activeIndex); }
    else if (e.key === "Escape") { hideList(); }
  });

  document.addEventListener("click", function (e) {
    if (!e.target.closest(".input-wrap")) hideList();
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

  });
})();