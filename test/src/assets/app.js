const app = document.getElementById("app");
const links = Array.from(document.querySelectorAll("a[data-nav]"));

const routes = {
  "/": `
    <section class="card">
      <h2>Home</h2>
      <p>This page is served as static assets from Spack registry.</p>
    </section>
  `,
  "/docs": `
    <section class="card">
      <h2>Docs</h2>
      <p>Use this route to verify SPA fallback to <code>index.html</code>.</p>
      <p class="mono">Open /docs or /about directly in browser.</p>
    </section>
  `,
  "/about": `
    <section class="card">
      <h2>About</h2>
      <p>Compression smoke test fetches a larger JSON file below.</p>
    </section>
  `,
};

function normalizePath(pathname) {
  return pathname === "/" ? "/" : pathname.replace(/\/+$/, "");
}

function highlight(path) {
  links.forEach((link) => {
    link.classList.toggle("active", link.dataset.nav === path);
  });
}

async function loadPayload() {
  const start = performance.now();
  const response = await fetch("/assets/payload.json", {
    headers: { Accept: "application/json" },
  });
  const elapsed = (performance.now() - start).toFixed(2);
  const etag = response.headers.get("etag") ?? "-";
  const encoding = response.headers.get("content-encoding") ?? "identity";
  const data = await response.json();

  return `
    <section class="card">
      <h3>Payload Fetch</h3>
      <div class="mono">status: ${response.status}</div>
      <div class="mono">rows: ${data.rows.length}</div>
      <div class="mono">content-encoding: ${encoding}</div>
      <div class="mono">etag: ${etag}</div>
      <div class="mono">time: ${elapsed} ms</div>
    </section>
  `;
}

async function render(path) {
  const view = routes[path] ?? routes["/"];
  highlight(path);
  app.innerHTML = `${view}<section class="card">Loading payload...</section>`;
  try {
    const payloadBlock = await loadPayload();
    app.innerHTML = `${view}${payloadBlock}`;
  } catch (error) {
    app.innerHTML = `${view}<section class="card mono">payload error: ${String(error)}</section>`;
  }
}

function navigate(path, replace = false) {
  if (replace) {
    history.replaceState(null, "", path);
  } else {
    history.pushState(null, "", path);
  }
  void render(path);
}

document.addEventListener("click", (event) => {
  const target = event.target.closest("a[data-nav]");
  if (!target) {
    return;
  }
  event.preventDefault();
  navigate(target.dataset.nav);
});

window.addEventListener("popstate", () => {
  void render(normalizePath(location.pathname));
});

navigate(normalizePath(location.pathname), true);
