const statsEl = document.getElementById("stats");
const accountsEl = document.getElementById("accounts");
const packagesEl = document.getElementById("packages");
const subscriptionsEl = document.getElementById("subscriptions");
const keysEl = document.getElementById("keys");
const modelAliasEl = document.getElementById("model-alias");
const dispatchResultEl = document.getElementById("dispatch-result");
const proxyCoreOriginEl = document.getElementById("proxy-core-origin");

async function getJSON(path) {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }
  return response.json();
}

function statCard(label, value, hint = "") {
  return `
    <article class="stat">
      <span class="muted">${label}</span>
      <strong>${value}</strong>
      <span class="muted">${hint}</span>
    </article>
  `;
}

function accountCard(account) {
  return `
    <article class="card">
      <header>
        <div>
          <div class="provider">${account.provider}</div>
          <h3>${account.display_name}</h3>
        </div>
        <div class="status">${account.status}</div>
      </header>
      <div class="muted">${account.email || "shared pool account"}</div>
      <div class="muted">mode: ${account.auth_mode} · tier: ${account.tier}</div>
      <div class="chips">${(account.supports_models || []).map((m) => `<span class="chip">${m}</span>`).join("")}</div>
    </article>
  `;
}

function packageCard(pkg) {
  return `
    <article class="card">
      <header>
        <div>
          <h3>${pkg.name}</h3>
          <div class="muted">${pkg.description}</div>
        </div>
        <div class="tag">${pkg.tier}</div>
      </header>
      ${pkg.provider_access.map((access) => `
        <div class="muted">${access.provider}: ${access.models.join(", ")}</div>
      `).join("")}
      <div class="chips">
        <span class="chip">fallback: ${pkg.allow_cross_provider_fallback ? "on" : "off"}</span>
        <span class="chip">concurrency: ${pkg.default_concurrency}</span>
      </div>
    </article>
  `;
}

function subscriptionCard(sub) {
  return `
    <article class="card">
      <header>
        <div>
          <h3>${sub.id}</h3>
          <div class="muted">user: ${sub.user_id} · package: ${sub.package_id}</div>
        </div>
        <div class="tag">${sub.status}</div>
      </header>
      <div class="muted">expires: ${new Date(sub.expires_at).toLocaleString()}</div>
    </article>
  `;
}

function keyCard(key) {
  return `
    <article class="card">
      <header>
        <div>
          <h3>${key.id}</h3>
          <div class="muted">${key.key}</div>
        </div>
        <div class="tag">${key.status}</div>
      </header>
      <div class="muted">user: ${key.user_id} · package: ${key.package_id}</div>
    </article>
  `;
}

async function bootstrap() {
  const [overview, data, catalog] = await Promise.all([
    getJSON("/api/admin/overview"),
    getJSON("/api/admin/data"),
    getJSON("/api/public/catalog"),
  ]);

  statsEl.innerHTML = [
    statCard("Active Accounts", overview.active_accounts, "shared provider pools"),
    statCard("Active API Keys", overview.active_keys, "unified external credentials"),
    statCard("Active Subscriptions", overview.active_subscriptions, "service access control"),
    statCard("Packages", overview.packages, "basic / advanced / hybrid"),
    statCard("Users", overview.users, "control-plane users"),
  ].join("");

  accountsEl.innerHTML = data.upstream_accounts.map(accountCard).join("");
  packagesEl.innerHTML = data.service_packages.map(packageCard).join("");
  subscriptionsEl.innerHTML = data.subscriptions.map(subscriptionCard).join("");
  keysEl.innerHTML = data.api_keys.map(keyCard).join("");
  proxyCoreOriginEl.textContent = catalog.proxy_core_origin;

  modelAliasEl.innerHTML = catalog.model_alias_policies
    .map((item) => `<option value="${item.alias}">${item.alias}</option>`)
    .join("");
}

document.getElementById("dispatch-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  dispatchResultEl.textContent = "Loading...";

  const apiKey = document.getElementById("api-key").value;
  const modelAlias = document.getElementById("model-alias").value;
  const input = document.getElementById("input").value;

  try {
    const catalog = await getJSON("/api/public/catalog");
    const response = await fetch(`${catalog.proxy_core_origin}/api/v1/dispatch`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${apiKey}`,
      },
      body: JSON.stringify({ model_alias: modelAlias, input }),
    });
    const data = await response.json();
    dispatchResultEl.textContent = JSON.stringify(data, null, 2);
  } catch (error) {
    dispatchResultEl.textContent = error.message;
  }
});

bootstrap().catch((error) => {
  dispatchResultEl.textContent = error.message;
});
