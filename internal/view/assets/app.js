// The trama graph page. It only draws the model that /api/workspace
// returns: every relation rule lives in Go. Text from the model always
// goes in with textContent, never as HTML.
(() => {
  "use strict";

  const plural = (n, one, many) => (n === 1 ? one : many);

  const STRINGS = {
    en: {
      relations: "Relations",
      view: "View",
      repos: "Repositories",
      hideIsolated: "Hide repos without relations",
      onlyChanges: "Only repos with open changes",
      search: "Search a repo or an id (GET /orders…)",
      hint: "Click a repository to see who may be affected if you release something there. Click an arrow to see its proof.",
      counts: (r, e, p) => `${r} ${plural(r, "repo", "repos")} · ${e} ${plural(e, "relation", "relations")} · ${p} not proven`,
      warnings: (n) => `${n} ${plural(n, "warning", "warnings")}`,
      loadError: (m) => `Could not load the workspace: ${m}`,
      today: "today",
      daysAgo: (n) => `${n} ${plural(n, "day", "days")} ago`,
      noRelease: "no release",
      notCloned: "not cloned",
      commitsAfter: (n) => `${n} ${plural(n, "commit", "commits")} after it`,
      kind: { http: "HTTP", event: "Events", package: "Packages", declared: "Declared", pending: "Not proven (?)", unverified: "Not verified" },
      card: { stale: "card out of date: run /trama:map", missing: "no card yet: run /trama:map", unknown: "card not checked: repository not cloned" },
      run: "How to run",
      port: (p) => `port ${p}`,
      provides: "Provides",
      consumes: "Consumes",
      affected: "If you release something here, these may be affected",
      direct: "Direct",
      indirect: "Indirect",
      possible: "Possible",
      nobody: "Nobody, as far as the cards show.",
      via: (path) => `may be affected, via ${path.join(" → ")}`,
      reason: { unknown: "relation without proof (from: unknown)", unverified: "relation not verified: see the warnings" },
      openChanges: "Open changes",
      focus: "Focus",
      levels: (n) => `${n} ${plural(n, "level", "levels")}`,
      showAll: "Show all",
      status: { proven: "proven", declared: "declared in workspace.yaml", unverified: "not verified", pending: "not proven yet" },
      from: "from",
      proof: "Proof",
      candidates: "Candidates",
      problems: "Problems",
      evidence: "Evidence",
      provider: "Provider",
      consumers: "Consumers",
      noMatch: (q) => `No repository or id matches “${q}”.`,
      legend: { selected: "selected", direct: "direct", indirect: "indirect", possible: "possible" },
    },
    pt: {
      relations: "Relações",
      view: "Exibir",
      repos: "Repositórios",
      hideIsolated: "Esconder repos sem relação",
      onlyChanges: "Só repos com mudança aberta",
      search: "Buscar repo ou id (GET /orders…)",
      hint: "Clique num repositório para ver quem pode ser afetado se você publicar algo ali. Clique numa seta para ver a prova.",
      counts: (r, e, p) => `${r} ${plural(r, "repo", "repos")} · ${e} ${plural(e, "relação", "relações")} · ${p} sem prova`,
      warnings: (n) => `${n} ${plural(n, "aviso", "avisos")}`,
      loadError: (m) => `Não foi possível carregar o workspace: ${m}`,
      today: "hoje",
      daysAgo: (n) => `há ${n} ${plural(n, "dia", "dias")}`,
      noRelease: "sem release",
      notCloned: "não clonado",
      commitsAfter: (n) => `${n} ${plural(n, "commit depois", "commits depois")}`,
      kind: { http: "HTTP", event: "Eventos", package: "Pacotes", declared: "Declaradas", pending: "Sem prova (?)", unverified: "Não verificadas" },
      card: { stale: "ficha desatualizada: rode /trama:map", missing: "sem ficha ainda: rode /trama:map", unknown: "ficha não conferida: repositório não clonado" },
      run: "Como rodar",
      port: (p) => `porta ${p}`,
      provides: "Fornece",
      consumes: "Consome",
      affected: "Se você publicar algo aqui, podem ser afetados",
      direct: "Diretos",
      indirect: "Indiretos",
      possible: "Possíveis",
      nobody: "Ninguém, pelo que as fichas mostram.",
      via: (path) => `pode ser afetado, via ${path.join(" → ")}`,
      reason: { unknown: "relação sem prova (from: unknown)", unverified: "relação não verificada: veja os avisos" },
      openChanges: "Mudanças abertas",
      focus: "Focar",
      levels: (n) => `${n} ${plural(n, "nível", "níveis")}`,
      showAll: "Mostrar tudo",
      status: { proven: "provada", declared: "declarada no workspace.yaml", unverified: "não verificada", pending: "ainda sem prova" },
      from: "de",
      proof: "Prova",
      candidates: "Candidatos",
      problems: "Problemas",
      evidence: "Evidência",
      provider: "Fornecedor",
      consumers: "Consumidores",
      noMatch: (q) => `Nenhum repositório ou id corresponde a “${q}”.`,
      legend: { selected: "selecionado", direct: "direto", indirect: "indireto", possible: "possível" },
    },
  };

  const params = new URLSearchParams(location.search);
  const lang = (params.get("lang") || navigator.language || "en").toLowerCase().startsWith("pt") ? "pt" : "en";
  const T = STRINGS[lang];
  document.documentElement.lang = lang === "pt" ? "pt-BR" : "en";

  const UNKNOWN = "u:?";
  const LABEL_LIMIT = 40;
  const trusted = (e) => e.status === "proven" || e.status === "declared";
  const $ = (id) => document.getElementById(id);

  function el(tag, text, cls) {
    const e = document.createElement(tag);
    if (text != null) e.textContent = text;
    if (cls) e.className = cls;
    return e;
  }

  function evidenceLink(l) {
    if (typeof l.url === "string" && l.url.startsWith("https://")) {
      const a = el("a", l.text);
      a.href = l.url;
      a.target = "_blank";
      a.rel = "noopener noreferrer";
      return a;
    }
    return el("code", l.text);
  }

  function evidenceList(list) {
    const box = el("div", null, "evidence");
    (list || []).forEach((l) => box.append(evidenceLink(l)));
    return box;
  }

  function ago(iso) {
    const days = Math.floor((Date.now() - Date.parse(iso)) / 86400000);
    if (Number.isNaN(days)) return "";
    return days <= 0 ? T.today : T.daysAgo(days);
  }

  function releaseText(r) {
    if (r.release) return `${r.release.version} · ${ago(r.release.date)}`;
    return r.cloned ? T.noRelease : T.notCloned;
  }

  const cssVar = (name) => getComputedStyle(document.documentElement).getPropertyValue(name).trim();

  let model = null;
  let cy = null;
  let repos = new Map();
  let selected = null; // the repo name or edge index the panel shows
  let focus = null; // the element ids kept by Focus, or null
  let focusLevels = 1;
  const repoShown = new Map();
  const kindShown = new Map();

  function edgeGroup(e) {
    if (e.status === "pending" || e.status === "unverified") return e.status;
    return e.kind || "declared";
  }

  function graphStyle() {
    const c = (v) => cssVar(v);
    const ring = (color, bg) => ({ "border-width": 2.5, "border-color": c(color), "background-color": c(bg) });
    return [
      { selector: "node", style: { shape: "round-rectangle", width: "data(w)", height: 44, label: "data(label)", "text-wrap": "wrap", "text-valign": "center", "text-halign": "center", "font-size": 11, color: c("--fg"), "background-color": c("--node"), "border-width": 1, "border-color": c("--node-border") } },
      { selector: "node.stale", style: { "border-style": "dashed", "border-color": c("--warn") } },
      { selector: "node.change", style: { "background-color": c("--change-bg") } },
      { selector: "node.notcloned", style: { opacity: 0.6 } },
      { selector: "node.unknown", style: { shape: "ellipse", width: 32, height: 32, "font-size": 14 } },
      { selector: "edge", style: { "curve-style": "bezier", width: 1.5, "line-color": c("--edge"), "target-arrow-color": c("--edge"), "target-arrow-shape": "triangle", label: "data(label)", "font-size": 9, color: c("--muted"), "text-rotation": "autorotate", "text-background-color": c("--bg"), "text-background-opacity": 0.85, "text-background-padding": 2 } },
      { selector: "edge.declared", style: { "line-color": c("--declared"), "target-arrow-color": c("--declared") } },
      { selector: "edge.unverified", style: { "line-style": "dotted", "line-color": c("--bad"), "target-arrow-color": c("--bad") } },
      { selector: "edge.pending", style: { "line-style": "dashed" } },
      { selector: "edge.nolabel", style: { label: "" } },
      { selector: ".faded", style: { opacity: 0.15 } },
      { selector: ".hide", style: { display: "none" } },
      { selector: "node.selected", style: { ...ring("--sel", "--sel-bg"), "border-width": 3.5, opacity: 1 } },
      { selector: "node.direct", style: { ...ring("--direct", "--direct-bg"), opacity: 1 } },
      { selector: "node.indirect", style: { ...ring("--indirect", "--indirect-bg"), opacity: 1 } },
      { selector: "node.possible", style: { ...ring("--possible", "--possible-bg"), "border-style": "dashed", opacity: 1 } },
      { selector: "edge.hl-direct", style: { width: 3, "line-color": c("--direct"), "target-arrow-color": c("--direct"), label: "data(label)", opacity: 1, "z-index": 10 } },
      { selector: "edge.hl-indirect", style: { width: 2.5, "line-color": c("--indirect"), "target-arrow-color": c("--indirect"), label: "data(label)", opacity: 1, "z-index": 9 } },
      { selector: "edge.hl-possible", style: { width: 2.5, "line-color": c("--possible"), "target-arrow-color": c("--possible"), label: "data(label)", opacity: 1, "z-index": 9 } },
      { selector: "edge.hl-edge", style: { width: 3.5, "line-color": c("--sel"), "target-arrow-color": c("--sel"), label: "data(label)", opacity: 1, "z-index": 11 } },
    ];
  }

  function elements() {
    const els = [];
    model.repos.forEach((r) => {
      const label = `${r.name}\n${releaseText(r)}`;
      const width = Math.max(r.name.length, releaseText(r).length) * 6.6 + 28;
      const cls = [];
      if (r.card === "stale" || r.card === "missing") cls.push("stale");
      if ((r.openChanges || []).length) cls.push("change");
      if (!r.cloned) cls.push("notcloned");
      els.push({ group: "nodes", data: { id: `r:${r.name}`, label, w: width }, classes: cls.join(" ") });
    });
    let unknown = false;
    model.edges.forEach((e, i) => {
      let target = e.provider && repos.has(e.provider) ? `r:${e.provider}` : UNKNOWN;
      if (target === UNKNOWN) unknown = true;
      if (!repos.has(e.consumer)) return;
      els.push({ group: "edges", data: { id: `e:${i}`, source: `r:${e.consumer}`, target, label: e.id, idx: i }, classes: e.status });
    });
    if (unknown) els.push({ group: "nodes", data: { id: UNKNOWN, label: "?", w: 32 }, classes: "unknown" });
    return els;
  }

  function clearMarks() {
    cy.elements().removeClass("faded selected direct indirect possible hl-direct hl-indirect hl-possible hl-edge");
  }

  function modelEdge(ele) {
    return model.edges[ele.data("idx")];
  }

  function edgesWhere(pred) {
    return cy.edges().filter((ele) => pred(modelEdge(ele)));
  }

  function selectRepo(name) {
    const repo = repos.get(name);
    if (!repo) return;
    selected = name;
    clearMarks();
    cy.elements().addClass("faded");
    cy.$id(`r:${name}`).removeClass("faded").addClass("selected");
    const a = repo.affected;
    a.direct.forEach((h) => {
      cy.$id(`r:${h.repo}`).removeClass("faded").addClass("direct");
      edgesWhere((e) => trusted(e) && e.consumer === h.repo && e.provider === name && e.id === h.id).removeClass("faded").addClass("hl-direct");
    });
    a.indirect.forEach((h) => {
      cy.$id(`r:${h.repo}`).removeClass("faded").addClass("indirect");
      const chain = [h.repo, ...(h.path || []), name];
      for (let i = 0; i + 1 < chain.length; i++) {
        edgesWhere((e) => trusted(e) && e.consumer === chain[i] && e.provider === chain[i + 1]).removeClass("faded").addClass("hl-indirect");
      }
    });
    a.possible.forEach((h) => {
      const node = cy.$id(`r:${h.repo}`);
      if (!node.hasClass("direct") && !node.hasClass("indirect")) node.removeClass("faded").addClass("possible");
      edgesWhere((e) => e.consumer === h.repo && e.id === h.id && (e.provider === name || (e.candidates || []).includes(name))).removeClass("faded").addClass("hl-possible");
    });
    if (a.possible.some((h) => model.edges.some((e) => e.status === "pending" && e.consumer === h.repo))) cy.$id(UNKNOWN).removeClass("faded");
    renderRepo(repo);
    updateLabels();
  }

  function selectEdge(idx) {
    selected = idx;
    clearMarks();
    cy.elements().addClass("faded");
    const ele = cy.$id(`e:${idx}`);
    ele.removeClass("faded").addClass("hl-edge");
    ele.connectedNodes().removeClass("faded");
    renderEdge(model.edges[idx]);
    updateLabels();
  }

  function clearSelection() {
    selected = null;
    clearMarks();
    renderHint();
    updateLabels();
  }

  function section(title, body) {
    const frag = document.createDocumentFragment();
    frag.append(el("h3", title), body);
    return frag;
  }

  function hitItem(h, kind) {
    const li = el("li");
    const head = el("div");
    head.append(el("span", T[kind], `badge ${kind}`), " ", el("strong", h.repo));
    if (h.id) head.append(" · ", el("code", h.id));
    li.append(head);
    if (kind === "indirect") li.append(el("div", T.via(h.path || []), "muted"));
    if (kind === "possible") li.append(el("div", T.reason[h.reason] || h.reason, "muted"));
    if (h.evidence && h.evidence.length) li.append(evidenceList(h.evidence));
    return li;
  }

  function renderHint() {
    const panel = $("panel");
    panel.replaceChildren(el("p", T.hint, "muted"));
  }

  function renderRepo(r) {
    const panel = $("panel");
    panel.replaceChildren();
    const title = el("h2");
    if (typeof r.url === "string" && r.url.startsWith("https://")) {
      title.append(evidenceLink({ text: r.name, url: r.url }));
    } else {
      title.textContent = r.name;
    }
    panel.append(title);
    if (r.description) panel.append(el("p", r.description));
    if ((r.stack || []).length) {
      const chips = el("div", null, "chips");
      r.stack.forEach((s) => chips.append(el("span", s, "chip")));
      panel.append(chips);
    }
    const rel = el("p", releaseText(r), "muted");
    if (r.release) rel.textContent += ` · ${T.commitsAfter(r.release.commitsAfter)}`;
    panel.append(rel);
    if (T.card[r.card]) panel.append(el("p", T.card[r.card], "badge warn"));

    const controls = el("div", null, "controls");
    const button = el("button", T.focus);
    button.type = "button";
    const levels = el("select");
    [1, 2, 3].forEach((n) => {
      const o = el("option", T.levels(n));
      o.value = String(n);
      levels.append(o);
    });
    levels.value = String(focusLevels);
    button.addEventListener("click", () => setFocus(r.name, Number(levels.value)));
    const all = el("button", T.showAll);
    all.type = "button";
    all.hidden = !focus;
    all.addEventListener("click", () => setFocus(null));
    controls.append(button, levels, all);
    panel.append(controls);

    const a = r.affected;
    const affected = el("ul");
    a.direct.forEach((h) => affected.append(hitItem(h, "direct")));
    a.indirect.forEach((h) => affected.append(hitItem(h, "indirect")));
    a.possible.forEach((h) => affected.append(hitItem(h, "possible")));
    if (!affected.children.length) affected.append(el("li", T.nobody, "muted"));
    panel.append(section(T.affected, affected));

    if ((r.openChanges || []).length) {
      const list = el("ul");
      r.openChanges.forEach((c) => {
        const li = el("li");
        li.append(evidenceLink({ text: c.id, url: c.url }), el("div", c.title));
        list.append(li);
      });
      panel.append(section(T.openChanges, list));
    }
    if ((r.run || []).length) {
      const list = el("ul");
      r.run.forEach((c) => {
        const li = el("li");
        li.append(el("strong", c.kind), " ", el("code", c.cmd));
        if (c.port) li.append(el("span", ` · ${T.port(c.port)}`, "muted"));
        list.append(li);
      });
      panel.append(section(T.run, list));
    }
    if ((r.provides || []).length) {
      const list = el("ul");
      r.provides.forEach((p) => {
        const li = el("li");
        li.append(el("code", p.id), el("span", ` · ${p.kind}`, "muted"), evidenceList(p.evidence));
        list.append(li);
      });
      panel.append(section(T.provides, list));
    }
    const consumed = model.edges.filter((e) => e.consumer === r.name);
    if (consumed.length) {
      const list = el("ul");
      consumed.forEach((e) => {
        const li = el("li");
        const head = el("div");
        head.append(el("code", e.id), " ", el("span", `${T.from} ${e.provider || "?"}`, "muted"), " ", el("span", T.status[e.status], e.status === "proven" || e.status === "declared" ? "badge" : "badge bad"));
        li.append(head, evidenceList(e.evidence));
        list.append(li);
      });
      panel.append(section(T.consumes, list));
    }
  }

  function renderEdge(e) {
    const panel = $("panel");
    panel.replaceChildren(el("h2", e.id));
    const meta = el("p", null, "muted");
    meta.textContent = [e.kind, T.status[e.status]].filter(Boolean).join(" · ");
    panel.append(meta);
    const flow = el("p");
    flow.append(el("strong", e.consumer), " → ", el("strong", e.provider || "?"));
    panel.append(flow);
    if (e.via) panel.append(section(T.proof, el("p", e.via)));
    if ((e.candidates || []).length) panel.append(section(T.candidates, el("p", e.candidates.join(", "))));
    if ((e.evidence || []).length) panel.append(section(T.evidence, evidenceList(e.evidence)));
    if ((e.problems || []).length) {
      const list = el("ul");
      e.problems.forEach((p) => list.append(el("li", p, "badge bad")));
      panel.append(section(T.problems, list));
    }
  }

  function renderUses(id, providers, consumers) {
    const panel = $("panel");
    panel.replaceChildren(el("h2", id));
    panel.append(section(T.provider, el("p", providers.join(", ") || "?")));
    const list = el("ul");
    consumers.forEach((e) => {
      const li = el("li");
      li.append(el("strong", e.consumer), " ", el("span", T.status[e.status], "muted"), evidenceList(e.evidence));
      list.append(li);
    });
    panel.append(section(T.consumers, list));
  }

  function search(q) {
    const query = q.trim();
    if (!query) {
      clearSelection();
      return;
    }
    const lower = query.toLowerCase();
    const users = model.edges.filter((e) => e.id.toLowerCase() === lower);
    const providers = model.repos.filter((r) => (r.provides || []).some((p) => p.id.toLowerCase() === lower)).map((r) => r.name);
    if (users.length || providers.length) {
      selected = null;
      clearMarks();
      cy.elements().addClass("faded");
      providers.forEach((n) => cy.$id(`r:${n}`).removeClass("faded").addClass("selected"));
      users.forEach((e) => {
        cy.$id(`r:${e.consumer}`).removeClass("faded").addClass(trusted(e) ? "direct" : "possible");
        if (e.provider) cy.$id(`r:${e.provider}`).removeClass("faded").addClass("selected");
        edgesWhere((m) => m === e).removeClass("faded").addClass(trusted(e) ? "hl-direct" : "hl-possible").connectedNodes().removeClass("faded");
      });
      renderUses(users[0] ? users[0].id : query, [...new Set(providers.concat(users.map((e) => e.provider).filter(Boolean)))], users);
      updateLabels();
      return;
    }
    const match = model.repos.find((r) => r.name.toLowerCase().includes(lower));
    if (match) {
      selectRepo(match.name);
      cy.center(cy.$id(`r:${match.name}`));
      return;
    }
    clearSelection();
    $("panel").replaceChildren(el("p", T.noMatch(query), "muted"));
  }

  function setFocus(name, levels) {
    if (!name) {
      focus = null;
    } else {
      focusLevels = levels;
      let kept = cy.$id(`r:${name}`);
      for (let i = 0; i < levels; i++) kept = kept.union(kept.neighborhood());
      focus = new Set(kept.map((ele) => ele.id()));
    }
    applyVisibility();
    cy.fit(cy.elements(":visible"), 40);
    if (selected !== null && typeof selected === "string") selectRepo(selected);
  }

  function applyVisibility() {
    const keep = (id) => !focus || focus.has(id);
    const shown = new Set(
      model.repos
        .filter((r) => repoShown.get(r.name) && keep(`r:${r.name}`) && (!$("only-changes").checked || (r.openChanges || []).length))
        .map((r) => `r:${r.name}`),
    );
    cy.batch(() => {
      cy.edges().forEach((ele) => {
        const e = modelEdge(ele);
        const target = ele.data("target");
        const ends = shown.has(ele.data("source")) && (target === UNKNOWN ? keep(UNKNOWN) : shown.has(target));
        ele.toggleClass("hide", !(ends && kindShown.get(edgeGroup(e))));
      });
      cy.nodes().forEach((n) => {
        const linked = n.connectedEdges().some((ele) => !ele.hasClass("hide"));
        if (n.id() === UNKNOWN) {
          n.toggleClass("hide", !linked);
          return;
        }
        n.toggleClass("hide", !shown.has(n.id()) || ($("hide-isolated").checked && !linked));
      });
    });
    updateLabels();
  }

  function updateLabels() {
    const many = cy.edges(":visible").length > LABEL_LIMIT && cy.zoom() < 1.2;
    cy.batch(() => {
      cy.edges().forEach((ele) => {
        const marked = ele.hasClass("hl-direct") || ele.hasClass("hl-indirect") || ele.hasClass("hl-possible") || ele.hasClass("hl-edge") || ele.scratch("hover");
        ele.toggleClass("nolabel", many && !marked);
      });
    });
  }

  function checkbox(container, label, checked, onChange) {
    const wrap = el("label", null, "check");
    const input = el("input");
    input.type = "checkbox";
    input.checked = checked;
    input.addEventListener("change", () => onChange(input.checked));
    wrap.append(input, el("span", label));
    container.append(wrap);
  }

  function renderFilters() {
    const kinds = $("kind-filters");
    kinds.replaceChildren();
    ["http", "event", "package", "declared", "unverified", "pending"].forEach((k) => {
      kindShown.set(k, true);
      checkbox(kinds, T.kind[k], true, (on) => {
        kindShown.set(k, on);
        applyVisibility();
      });
    });
    const list = $("repo-filters");
    list.replaceChildren();
    model.repos.forEach((r) => {
      repoShown.set(r.name, true);
      checkbox(list, r.name, true, (on) => {
        repoShown.set(r.name, on);
        applyVisibility();
      });
    });
    ["hide-isolated", "only-changes"].forEach((id) => $(id).addEventListener("change", applyVisibility));
  }

  function renderHeader() {
    $("ws-name").textContent = model.workspace;
    const proven = model.edges.filter(trusted).length;
    $("counts").textContent = T.counts(model.repos.length, proven, model.edges.length - proven);
    const warnings = model.warnings || [];
    if (warnings.length) showBanner(T.warnings(warnings.length), warnings);
    const legend = $("legend");
    legend.replaceChildren();
    ["selected", "direct", "indirect", "possible"].forEach((k) => {
      const item = el("span");
      item.append(el("span", null, `swatch ${k}`), T.legend[k]);
      legend.append(item);
    });
  }

  function showBanner(title, lines) {
    const toggle = $("banner-toggle");
    const list = $("banner-list");
    toggle.textContent = title;
    list.replaceChildren(...lines.map((l) => el("li", l)));
    toggle.onclick = () => {
      list.hidden = !list.hidden;
    };
    $("banner").hidden = false;
  }

  function translateStatic() {
    document.querySelectorAll("[data-i18n]").forEach((node) => {
      const text = T[node.getAttribute("data-i18n")];
      if (typeof text === "string") node.textContent = text;
    });
    $("search").placeholder = T.search;
  }

  function start(data) {
    model = data;
    repos = new Map(model.repos.map((r) => [r.name, r]));
    renderHeader();
    renderFilters();
    renderHint();
    cy = cytoscape({
      container: $("cy"),
      elements: elements(),
      style: graphStyle(),
      layout: { name: "cose", animate: false, nodeDimensionsIncludeLabels: true, idealEdgeLength: () => 90, nodeRepulsion: () => 9000, padding: 30 },
      minZoom: 0.1,
      maxZoom: 3,
    });
    cy.on("tap", "node", (evt) => {
      if (evt.target.id() !== UNKNOWN) selectRepo(evt.target.id().slice(2));
    });
    cy.on("tap", "edge", (evt) => selectEdge(evt.target.data("idx")));
    cy.on("tap", (evt) => {
      if (evt.target === cy) clearSelection();
    });
    cy.on("mouseover", "edge", (evt) => {
      evt.target.scratch("hover", true);
      updateLabels();
    });
    cy.on("mouseout", "edge", (evt) => {
      evt.target.scratch("hover", false);
      updateLabels();
    });
    cy.on("zoom", updateLabels);
    applyVisibility();
    if (cy.zoom() > 1.3) {
      cy.zoom(1.3);
      cy.center();
    }
    let timer = null;
    $("search").addEventListener("input", (ev) => {
      clearTimeout(timer);
      timer = setTimeout(() => search(ev.target.value), 200);
    });
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", () => cy.style(graphStyle()));
  }

  translateStatic();
  fetch("/api/workspace", { cache: "no-store" })
    .then((res) => res.json().then((body) => {
      if (!res.ok) throw new Error(body.error || res.statusText);
      return body;
    }))
    .then(start)
    .catch((err) => showBanner(T.loadError(err.message), []));
})();
