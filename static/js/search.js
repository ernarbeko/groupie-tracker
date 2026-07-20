(() => {
  const input = document.getElementById("search-input");
  const results = document.getElementById("search-results");
  const status = document.getElementById("search-status");
  if (!input || !results || !status) return;

  let timer = null;
  let requestId = 0;

  const escapeHtml = (value) =>
    String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;");

  const render = (payload) => {
    const items = payload.results || [];
    if (!payload.query) {
      results.hidden = true;
      results.innerHTML = "";
      status.textContent = "";
      return;
    }

    status.textContent =
      items.length === 0
        ? "No matches"
        : `${items.length} match${items.length === 1 ? "" : "es"}`;

    if (items.length === 0) {
      results.hidden = false;
      results.innerHTML = `<p class="empty">Nothing found for “${escapeHtml(payload.query)}”.</p>`;
      return;
    }

    results.hidden = false;
    results.innerHTML = items
      .map(
        (hit, index) => `
      <a class="search-hit" href="/artist/${hit.id}" style="animation-delay:${index * 40}ms">
        <img src="${escapeHtml(hit.image)}" alt="${escapeHtml(hit.name)}" width="64" height="64" loading="lazy">
        <div>
          <h3>${escapeHtml(hit.name)}</h3>
          <p>${hit.creationDate} · ${escapeHtml(hit.firstAlbum)}</p>
        </div>
        <span class="match-tag">${escapeHtml(hit.match)}</span>
      </a>`
      )
      .join("");
  };

  const search = async (query) => {
    const id = ++requestId;
    status.textContent = "Searching…";
    try {
      const res = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      if (id !== requestId) return;
      render(data);
    } catch (err) {
      if (id !== requestId) return;
      status.textContent = "Search failed";
      results.hidden = false;
      results.innerHTML = `<p class="empty">Could not reach the server. Try again.</p>`;
      console.error(err);
    }
  };

  input.addEventListener("input", () => {
    const query = input.value.trim();
    clearTimeout(timer);
    if (!query) {
      requestId += 1;
      render({ query: "", results: [] });
      return;
    }
    timer = setTimeout(() => search(query), 220);
  });
})();
