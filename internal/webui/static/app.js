'use strict';

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

const els = {
  grid: $('#grid'),
  empty: $('#empty'),
  emptyTitle: $('#emptyTitle'),
  emptyText: $('#emptyText'),
  emptyAddText: $('#emptyAddText'),
  metaText: $('#metaText'),
  footCount: $('#footCount'),
  catNav: $('#catNav'),
  mobileCats: $('#mobileCats'),
  manageCatsBtn: $('#manageCatsBtn'),
  searchInput: $('#searchInput'),
  sortSelect: $('#sortSelect'),
  densityBtn: $('#densityBtn'),
  densityMenu: $('#densityMenu'),
  checkBtn: $('#checkBtn'),
  clearFilter: $('#clearFilter'),
  themeBtn: $('#themeBtn'),
  importBtn: $('#importBtn'),
  exportBtn: $('#exportBtn'),
  fileInput: $('#fileInput'),
  toasts: $('#toasts'),
  greeting: $('#greeting'),
  today: $('#today'),
  editDialog: $('#editDialog'),
  linkForm: $('#linkForm'),
  dialogTitle: $('#dialogTitle'),
  saveBtn: $('#saveBtn'),
  formError: $('#formError'),
  fetchBtn: $('#fetchBtn'),
  fUrl: $('#fUrl'),
  fTitle: $('#fTitle'),
  fCatInput: $('#fCatInput'),
  catPicker: $('#catPicker'),
  catPickerField: $('#catPickerField'),
  catPickerChips: $('#catPickerChips'),
  catMenu: $('#catMenu'),
  fDesc: $('#fDesc'),
  fIcon: $('#fIcon'),
  fPinned: $('#fPinned'),
  catDialog: $('#catDialog'),
  catCreateForm: $('#catCreateForm'),
  catNameInput: $('#catNameInput'),
  catList: $('#catList'),
  catEmpty: $('#catEmpty'),
  catError: $('#catError'),
  confirmDialog: $('#confirmDialog'),
  confirmTitle: $('#confirmTitle'),
  confirmText: $('#confirmText'),
  okConfirm: $('#okConfirm'),
  cancelConfirm: $('#cancelConfirm'),
};

const SORTS = ['manual', 'recent', 'visits', 'az'];
const DENSITIES = ['default', 'compact', 'mini'];
const UNCATEGORIZED = 'Chưa phân loại';

const state = {
  links: [],
  categories: [],
  filter: 'all',
  query: '',
  sort: SORTS.includes(localStorage.getItem('lh-sort') || '') ? localStorage.getItem('lh-sort') : 'manual',
  density: DENSITIES.includes(localStorage.getItem('lh-density') || '') ? localStorage.getItem('lh-density') : 'default',
  health: {},
};

/* ---------- Helpers ---------- */
const ESC_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
const esc = (value) => String(value ?? '').replace(/[&<>"']/g, (c) => ESC_MAP[c]);

const normalizeText = (value) =>
  String(value ?? '')
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toLowerCase();

const fmtNumber = (n) => new Intl.NumberFormat('vi-VN').format(n || 0);

function hostOf(rawURL) {
  try {
    return new URL(rawURL).hostname.replace(/^www\./, '');
  } catch {
    return rawURL;
  }
}

function hueOf(text) {
  let hash = 0;
  const s = String(text || '');
  for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) % 360;
  return hash;
}

function faviconOf(rawURL) {
  const host = hostOf(rawURL);
  if (!host) return '';
  return 'https://www.google.com/s2/favicons?domain=' + encodeURIComponent(host) + '&sz=64';
}

function relativeTime(iso) {
  if (!iso) return '';
  const then = new Date(iso).getTime();
  if (!then) return '';
  const diff = (Date.now() - then) / 1000;
  if (diff < 60) return 'vừa xong';
  if (diff < 3600) return Math.floor(diff / 60) + ' phút trước';
  if (diff < 86400) return Math.floor(diff / 3600) + ' giờ trước';
  if (diff < 86400 * 30) return Math.floor(diff / 86400) + ' ngày trước';
  return new Date(iso).toLocaleDateString('vi-VN');
}

async function api(path, options = {}) {
  const init = { ...options };
  init.headers = { ...(options.headers || {}) };
  if (init.body && typeof init.body !== 'string') {
    init.body = JSON.stringify(init.body);
  }
  if (init.body) init.headers['Content-Type'] = 'application/json';

  const res = await fetch(path, init);
  if (!res.ok) {
    let message = 'Có lỗi xảy ra (' + res.status + ')';
    try {
      const data = await res.json();
      if (data && data.error) message = data.error;
    } catch {}
    throw new Error(message);
  }
  if (res.status === 204) return null;
  const type = res.headers.get('content-type') || '';
  return type.includes('application/json') ? res.json() : res.text();
}

function toast(message, type = 'success') {
  const node = document.createElement('div');
  node.className = 'toast ' + type;
  const dot = document.createElement('span');
  dot.className = 'dot';
  const text = document.createElement('span');
  text.textContent = message;
  node.append(dot, text);
  els.toasts.append(node);
  setTimeout(() => {
    node.classList.add('out');
    setTimeout(() => node.remove(), 240);
  }, 3400);
}

/* ---------- Theme & greeting ---------- */
function applyThemeUI(theme) {
  document.documentElement.dataset.theme = theme;
  localStorage.setItem('lh-theme', theme);
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.setAttribute('content', theme === 'light' ? '#f3f5fb' : '#080c16');
  const use = $('#themeBtn use');
  if (use) use.setAttribute('href', theme === 'light' ? '#i-sun' : '#i-moon');
  els.themeBtn.title = theme === 'light' ? 'Chuyển sang giao diện tối' : 'Chuyển sang giao diện sáng';
}

/* ---------- Display density ---------- */
function applyDensityUI(density) {
  document.documentElement.dataset.density = density;
  localStorage.setItem('lh-density', density);
  $$('.density-item').forEach((item) => {
    const active = item.dataset.density === density;
    item.classList.toggle('active', active);
    item.setAttribute('aria-checked', active ? 'true' : 'false');
  });
}

function closeDensityMenu() {
  if (els.densityMenu.hidden) return;
  els.densityMenu.hidden = true;
  els.densityBtn.setAttribute('aria-expanded', 'false');
}

function toggleDensityMenu() {
  const open = els.densityMenu.hidden;
  els.densityMenu.hidden = !open;
  els.densityBtn.setAttribute('aria-expanded', open ? 'true' : 'false');
  if (open) {
    const active = els.densityMenu.querySelector('.density-item.active');
    if (active) active.focus();
  }
}

function setGreeting() {
  const now = new Date();
  const hour = now.getHours();
  let greeting = 'Chào buổi tối';
  if (hour < 11) greeting = 'Chào buổi sáng';
  else if (hour < 18) greeting = 'Chào buổi chiều';

  els.greeting.textContent = greeting;

  const dateText = new Intl.DateTimeFormat('vi-VN', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
  }).format(now);
  els.today.textContent = dateText.charAt(0).toUpperCase() + dateText.slice(1);
}

/* ---------- Data helpers ---------- */
const catById = (id) => state.categories.find((c) => c.id === id) || null;

function categoriesOf(link) {
  return (link.categoryIds || []).map(catById).filter(Boolean);
}

function categoryNames(link) {
  return categoriesOf(link).map((c) => c.name);
}

function categoryCounts() {
  const counts = new Map();
  const byId = new Map(state.categories.map((c) => [c.id, c]));
  for (const link of state.links) {
    for (const id of link.categoryIds || []) {
      if (byId.has(id)) counts.set(id, (counts.get(id) || 0) + 1);
    }
  }
  return counts;
}

function uncategorizedCount() {
  const known = new Set(state.categories.map((c) => c.id));
  return state.links.filter((l) => !(l.categoryIds || []).some((id) => known.has(id))).length;
}

function categoryKey(name) {
  return normalizeText(name).trim().replace(/\s+/g, ' ');
}

function matchesFilter(link) {
  if (state.filter === 'fav') return !!link.pinned;
  if (state.filter === 'none') return categoriesOf(link).length === 0;
  if (state.filter.startsWith('cat:')) return (link.categoryIds || []).includes(state.filter.slice(4));
  return true;
}

function matchesSearch(link) {
  const q = normalizeText(state.query).trim();
  if (!q) return true;
  const haystack = normalizeText([
    link.title, link.description, categoryNames(link).join(' '), link.url, hostOf(link.url),
  ].join(' '));
  return haystack.includes(q);
}

function sortLinks(list) {
  const arr = [...list];
  switch (state.sort) {
    case 'recent':
      arr.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
      break;
    case 'visits':
      arr.sort((a, b) => (b.visits || 0) - (a.visits || 0) || a.title.localeCompare(b.title, 'vi'));
      break;
    case 'az':
      arr.sort((a, b) => a.title.localeCompare(b.title, 'vi'));
      break;
    default:
      arr.sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
  }
  return arr;
}

function visibleLinks() {
  return sortLinks(state.links.filter((l) => matchesFilter(l) && matchesSearch(l)));
}

function canDrag() {
  return state.sort === 'manual' && state.filter === 'all' && !state.query.trim();
}

/* ---------- Rendering ---------- */
function render() {
  const counts = categoryCounts();
  renderSidebarCounts();
  renderCatNav(counts);
  renderMobileCats(counts);
  if (els.catDialog.open) renderCatManager();

  const links = visibleLinks();
  renderCards(links);
  renderMeta(links);
  renderFooter();
  syncControls(links);
}

function renderSidebarCounts() {
  const set = (key, value) => {
    const el = $('[data-count="' + key + '"]');
    if (el) el.textContent = value;
  };
  set('all', fmtNumber(state.links.length));
  set('fav', fmtNumber(state.links.filter((l) => l.pinned).length));
  set('cats', fmtNumber(state.categories.length));

  $$('.sidebar [data-filter]').forEach((btn) => {
    btn.classList.toggle('active', btn.dataset.filter === state.filter);
  });
}

function makeNavButton(filterValue, label, count, opts = {}) {
  const btn = document.createElement('button');
  btn.type = 'button';
  btn.className = 'nav-item';
  btn.dataset.filter = filterValue;
  btn.innerHTML =
    (opts.hue !== undefined ? '<span class="cat-dot" style="--h:' + opts.hue + '"></span>' : '') +
    (opts.withIcon ? '<svg class="ic"><use href="#i-folder"/></svg>' : '') +
    '<span></span><span class="pill"></span>';
  const spans = btn.querySelectorAll('span:not(.cat-dot)');
  spans[0].textContent = label;
  spans[1].textContent = fmtNumber(count);
  return btn;
}

function renderCatNav(counts) {
  els.catNav.replaceChildren();
  for (const cat of state.categories) {
    const btn = makeNavButton('cat:' + cat.id, cat.name, counts.get(cat.id) || 0, { hue: hueOf(cat.name) });
    if (state.filter === 'cat:' + cat.id) btn.classList.add('active');
    els.catNav.append(btn);
  }
  const rest = uncategorizedCount();
  if (rest > 0) {
    const btn = makeNavButton('none', UNCATEGORIZED, rest, { withIcon: true });
    if (state.filter === 'none') btn.classList.add('active');
    els.catNav.append(btn);
  }
}

function renderMobileCats(counts) {
  els.mobileCats.replaceChildren();

  const addChip = (filterValue, label, count, hue) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'mobile-cat' + (state.filter === filterValue ? ' active' : '');
    btn.dataset.filter = filterValue;
    if (hue !== undefined) {
      const dot = document.createElement('span');
      dot.className = 'cat-dot';
      dot.style.setProperty('--h', hue);
      btn.append(dot);
    }
    btn.append(document.createTextNode(label));
    const badge = document.createElement('span');
    badge.className = 'count';
    badge.textContent = fmtNumber(count);
    btn.append(badge);
    els.mobileCats.append(btn);
  };

  addChip('all', 'Tất cả', state.links.length);
  addChip('fav', 'Yêu thích', state.links.filter((l) => l.pinned).length);
  for (const cat of state.categories) {
    addChip('cat:' + cat.id, cat.name, counts.get(cat.id) || 0, hueOf(cat.name));
  }
  const rest = uncategorizedCount();
  if (rest > 0) addChip('none', UNCATEGORIZED, rest);

  const manage = document.createElement('button');
  manage.type = 'button';
  manage.className = 'mobile-cat mobile-cat-manage';
  manage.dataset.action = 'manage-cats';
  manage.innerHTML = '<svg class="ic"><use href="#i-sliders"/></svg><span>Quản lý</span>';
  els.mobileCats.append(manage);
}

function iconHTML(link, initial) {
  if (link.icon) {
    return '<span class="emoji">' + esc(link.icon) + '</span>';
  }
  return '<span class="letter">' + esc(initial) + '</span>' +
    '<img src="' + esc(faviconOf(link.url)) + '" alt="" loading="lazy" onerror="this.remove()">';
}

function visitsTitle(link) {
  if (link.lastVisited) return 'Truy cập lần cuối: ' + relativeTime(link.lastVisited);
  return 'Số lượt truy cập';
}

function actionButtons(link) {
  return [
    '<button type="button" data-act="fav"' + (link.pinned ? ' class="active"' : '') +
      ' title="' + (link.pinned ? 'Bỏ yêu thích' : 'Yêu thích') + '" aria-label="' + (link.pinned ? 'Bỏ yêu thích' : 'Yêu thích') + '">' +
      '<svg class="ic"><use href="#i-star"/></svg></button>',
    '<button type="button" data-act="copy" title="Sao chép URL" aria-label="Sao chép URL">' +
      '<svg class="ic"><use href="#i-copy"/></svg></button>',
    '<button type="button" data-act="edit" title="Chỉnh sửa" aria-label="Chỉnh sửa">' +
      '<svg class="ic"><use href="#i-edit"/></svg></button>',
    '<button type="button" data-act="del" title="Xóa" aria-label="Xóa">' +
      '<svg class="ic"><use href="#i-trash"/></svg></button>',
  ].join('');
}

function tagsHTML(link, limit = 2) {
  const cats = categoriesOf(link);
  if (!cats.length) return '';
  let html = cats.slice(0, limit).map((cat) =>
    '<span class="tag" style="--h:' + hueOf(cat.name) + '">' + esc(cat.name) + '</span>'
  ).join('');
  if (cats.length > limit) {
    html += '<span class="tag tag-more" title="' + esc(cats.map((c) => c.name).join(', ')) + '">+' +
      (cats.length - limit) + '</span>';
  }
  return html;
}

function cardHTML(link) {
  const domain = hostOf(link.url);
  const initial = (link.title || domain || '?').trim().charAt(0).toUpperCase();
  const hue = hueOf(link.title || domain);
  const status = state.health[link.id];
  const statusHTML = status
    ? '<span class="status ' + status + '" title="' + (status === 'ok' ? 'Liên kết hoạt động' : 'Không phản hồi') + '"></span>'
    : '';
  const desc = link.description ? '<p class="card-desc">' + esc(link.description) + '</p>' : '';
  const visits = link.visits > 0
    ? '<span class="visits" title="' + esc(visitsTitle(link)) + '"><b>' + fmtNumber(link.visits) + '</b> lượt</span>'
    : '';
  const pin = link.pinned
    ? '<span class="pin" title="Yêu thích"><svg class="ic"><use href="#i-star-fill"/></svg></span>'
    : '';
  const dragAttr = canDrag() ? ' draggable="true"' : '';

  return '<article class="card" data-id="' + esc(link.id) + '"' + dragAttr + '>' +
    '<a class="card-hit" href="' + esc(link.url) + '" target="_blank" rel="noopener noreferrer" draggable="false" aria-label="Mở ' + esc(link.title) + ' trong thẻ mới"></a>' +
    '<div class="card-head">' +
      '<span class="fav" style="--h:' + hue + '">' + iconHTML(link, initial) + statusHTML + '</span>' +
      '<div class="card-title-wrap">' +
        '<h3 class="card-title">' + esc(link.title) + '</h3>' +
        '<span class="card-domain"><svg class="ic"><use href="#i-link"/></svg>' + esc(domain) + '</span>' +
      '</div>' +
    '</div>' +
    desc +
    '<div class="card-foot">' + tagsHTML(link) + visits + '</div>' +
    pin +
    '<div class="card-actions">' + actionButtons(link) + '</div>' +
  '</article>';
}

function renderCards(links) {
  els.grid.innerHTML = links.map(cardHTML).join('');
  els.grid.hidden = links.length === 0;
  els.empty.hidden = links.length !== 0;

  if (state.links.length === 0) {
    els.emptyTitle.textContent = 'Chưa có liên kết nào';
    els.emptyText.textContent = 'Thêm những trang web bạn hay dùng để truy cập nhanh hơn.';
    els.emptyAddText.textContent = 'Thêm liên kết đầu tiên';
  } else {
    els.emptyTitle.textContent = 'Không tìm thấy liên kết';
    els.emptyText.textContent = 'Thử từ khóa khác, đổi danh mục hoặc xóa bộ lọc.';
    els.emptyAddText.textContent = 'Thêm liên kết mới';
  }
}

function renderMeta(links) {
  const totalVisits = state.links.reduce((sum, l) => sum + (l.visits || 0), 0);
  let html = '<span><b>' + fmtNumber(state.links.length) + '</b> liên kết</span>' +
    '<span class="dot-sep"></span><span><b>' + fmtNumber(state.categories.length) + '</b> danh mục</span>' +
    '<span class="dot-sep"></span><span><b>' + fmtNumber(totalVisits) + '</b> lượt truy cập</span>';

  if (state.query || state.filter !== 'all') {
    html += '<span class="dot-sep"></span><span>Hiển thị <b>' + fmtNumber(links.length) + '</b></span>';
  } else if (canDrag() && state.links.length > 1) {
    html += '<span class="dot-sep"></span><span>Kéo thả để sắp xếp</span>';
  }
  els.metaText.innerHTML = html;
}

function renderFooter() {
  els.footCount.textContent = fmtNumber(state.links.length) + ' liên kết';
}

function syncControls() {
  els.sortSelect.value = state.sort;
  els.clearFilter.hidden = !(state.query || state.filter !== 'all');
}

/* ---------- Actions ---------- */
function linkPayload(link, overrides = {}) {
  return {
    title: link.title,
    url: link.url,
    description: link.description || '',
    categoryIds: [...(link.categoryIds || [])],
    icon: link.icon || '',
    pinned: !!link.pinned,
    ...overrides,
  };
}

function recordVisit(id) {
  const link = state.links.find((l) => l.id === id);
  if (!link) return;

  link.visits = (link.visits || 0) + 1;
  link.lastVisited = new Date().toISOString();
  fetch('/api/links/' + encodeURIComponent(id) + '/visit', { method: 'POST', keepalive: true }).catch(() => {});

  const card = els.grid.querySelector('.card[data-id="' + CSS.escape(id) + '"]');
  if (card) {
    let foot = card.querySelector('.card-foot');
    if (!foot) {
      foot = document.createElement('div');
      foot.className = 'card-foot';
      const anchor = card.querySelector('.pin') || card.querySelector('.card-actions');
      card.insertBefore(foot, anchor);
    }
    foot.innerHTML = tagsHTML(link) + '<span class="visits" title="' + esc(visitsTitle(link)) + '"><b>' +
      fmtNumber(link.visits) + '</b> lượt</span>';
  }
  renderMeta(visibleLinks());
}

async function toggleFavorite(id) {
  const link = state.links.find((l) => l.id === id);
  if (!link) return;
  try {
    const updated = await api('/api/links/' + encodeURIComponent(id), {
      method: 'PUT',
      body: linkPayload(link, { pinned: !link.pinned }),
    });
    Object.assign(link, updated);
    render();
    toast(updated.pinned ? 'Đã thêm vào Yêu thích' : 'Đã bỏ khỏi Yêu thích', 'info');
  } catch (err) {
    toast(err.message, 'error');
  }
}

function fallbackCopy(text) {
  const area = document.createElement('textarea');
  area.value = text;
  area.style.position = 'fixed';
  area.style.top = '-1000px';
  area.style.opacity = '0';
  document.body.append(area);
  area.select();
  document.execCommand('copy');
  area.remove();
}

async function copyLink(id) {
  const link = state.links.find((l) => l.id === id);
  if (!link) return;
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(link.url);
    } else {
      fallbackCopy(link.url);
    }
    toast('Đã sao chép URL', 'info');
  } catch {
    toast('Không thể sao chép URL', 'error');
  }
}

async function deleteLink(id) {
  const link = state.links.find((l) => l.id === id);
  if (!link) return;

  const ok = await askConfirm(
    'Xóa liên kết?',
    '“' + link.title + '” sẽ bị xóa khỏi dashboard. Hành động này không thể hoàn tác.',
    'Xóa'
  );
  if (!ok) return;

  try {
    await api('/api/links/' + encodeURIComponent(id), { method: 'DELETE' });
    state.links = state.links.filter((l) => l.id !== id);
    delete state.health[id];
    render();
    toast('Đã xóa liên kết');
  } catch (err) {
    toast(err.message, 'error');
  }
}

/* ---------- Confirm dialog ---------- */
let confirmResolver = null;

function askConfirm(title, text, label = 'Xóa') {
  els.confirmTitle.textContent = title;
  els.confirmText.textContent = text;
  els.okConfirm.textContent = label;
  els.confirmDialog.showModal();
  return new Promise((resolve) => {
    confirmResolver = resolve;
  });
}

function settleConfirm(value) {
  if (els.confirmDialog.open) els.confirmDialog.close();
  if (confirmResolver) {
    confirmResolver(value);
    confirmResolver = null;
  }
}

/* ---------- Editor ---------- */
let editingId = null;
let catDraft = [];
let catActive = 0;

function findCategoryByName(name) {
  const key = categoryKey(name);
  if (!key) return null;
  return state.categories.find((cat) => categoryKey(cat.name) === key) || null;
}

async function ensureCategory(name) {
  const trimmed = name.trim().slice(0, 40);
  const existing = findCategoryByName(trimmed);
  if (existing) return existing;
  const cat = await api('/api/categories', { method: 'POST', body: { name: trimmed } });
  if (!state.categories.some((c) => c.id === cat.id)) {
    state.categories.push(cat);
    render();
  }
  return cat;
}

function renderCatChips() {
  const selected = state.categories.filter((cat) => catDraft.includes(cat.id));
  els.catPickerChips.replaceChildren();
  for (const cat of selected) {
    const chip = document.createElement('span');
    chip.className = 'cat-chip';
    chip.style.setProperty('--h', hueOf(cat.name));

    const dot = document.createElement('span');
    dot.className = 'cat-dot';

    const label = document.createElement('span');
    label.className = 'cat-chip-label';
    label.textContent = cat.name;

    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'cat-chip-x';
    remove.setAttribute('aria-label', 'Bỏ danh mục ' + cat.name);
    remove.innerHTML = '<svg class="ic"><use href="#i-x"/></svg>';
    remove.addEventListener('click', (event) => {
      event.stopPropagation();
      toggleCatDraft(cat.id);
      els.fCatInput.focus();
    });

    chip.append(dot, label, remove);
    els.catPickerChips.append(chip);
  }
  els.fCatInput.placeholder = selected.length ? 'Thêm danh mục...' : 'Tìm hoặc thêm danh mục...';
}

function catMenuItems() {
  const query = els.fCatInput.value.trim();
  const key = categoryKey(query);
  const matched = state.categories.filter((cat) => !key || categoryKey(cat.name).includes(key));
  const exact = query ? state.categories.find((cat) => categoryKey(cat.name) === key) : null;
  const items = [];
  if (query && !exact) items.push({ type: 'create', name: query });
  for (const cat of matched) items.push({ type: 'cat', cat });
  return items;
}

function openCatMenu() {
  els.catMenu.hidden = false;
  els.catPicker.classList.add('open');
  els.fCatInput.setAttribute('aria-expanded', 'true');
}

function closeCatMenu() {
  if (els.catMenu.hidden) return;
  els.catMenu.hidden = true;
  els.catPicker.classList.remove('open');
  els.fCatInput.setAttribute('aria-expanded', 'false');
  catActive = 0;
}

function renderCatMenu() {
  const items = catMenuItems();
  if (catActive >= items.length) catActive = Math.max(0, items.length - 1);
  els.catMenu.replaceChildren();

  if (!items.length) {
    const empty = document.createElement('p');
    empty.className = 'cat-menu-empty';
    empty.textContent = 'Chưa có danh mục. Nhập tên để tạo mới.';
    els.catMenu.append(empty);
  }

  items.forEach((item, index) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'cat-option';
    btn.setAttribute('role', 'option');
    if (index === catActive) btn.classList.add('active');

    if (item.type === 'create') {
      btn.classList.add('create');
      const icon = document.createElement('span');
      icon.className = 'cat-option-icon';
      icon.innerHTML = '<svg class="ic"><use href="#i-plus"/></svg>';
      const label = document.createElement('span');
      label.className = 'cat-option-name';
      label.textContent = 'Tạo danh mục “' + item.name + '”';
      btn.append(icon, label);
    } else {
      const dot = document.createElement('span');
      dot.className = 'cat-dot';
      dot.style.setProperty('--h', hueOf(item.cat.name));
      const label = document.createElement('span');
      label.className = 'cat-option-name';
      label.textContent = item.cat.name;
      btn.append(dot, label);
      if (catDraft.includes(item.cat.id)) {
        btn.classList.add('selected');
        btn.setAttribute('aria-selected', 'true');
        const check = document.createElement('span');
        check.className = 'cat-option-check';
        check.innerHTML = '<svg class="ic"><use href="#i-check"/></svg>';
        btn.append(check);
      }
    }

    btn.addEventListener('pointerdown', (event) => {
      event.preventDefault();
      chooseCatItem(index);
    });
    els.catMenu.append(btn);
  });

  openCatMenu();
}

function toggleCatDraft(id) {
  if (catDraft.includes(id)) catDraft = catDraft.filter((x) => x !== id);
  else catDraft.push(id);
  renderCatChips();
  renderCatMenu();
}

async function chooseCatItem(index) {
  const item = catMenuItems()[index];
  if (!item) return;
  if (item.type === 'create') {
    try {
      const cat = await ensureCategory(item.name);
      if (!catDraft.includes(cat.id)) catDraft.push(cat.id);
      els.fCatInput.value = '';
      renderCatChips();
      renderCatMenu();
    } catch (err) {
      toast(err.message, 'error');
    }
    els.fCatInput.focus();
    return;
  }
  els.fCatInput.value = '';
  toggleCatDraft(item.cat.id);
  els.fCatInput.focus();
}

function openEditor(link) {
  editingId = link ? link.id : null;
  els.linkForm.reset();
  els.formError.hidden = true;

  catDraft = link
    ? [...new Set(link.categoryIds || [])].filter((id) => catById(id))
    : state.filter.startsWith('cat:')
      ? [state.filter.slice(4)]
      : [];
  els.fCatInput.value = '';
  closeCatMenu();
  renderCatChips();

  els.fUrl.value = link ? link.url : '';
  els.fTitle.value = link ? link.title : '';
  els.fDesc.value = link ? link.description || '' : '';
  els.fIcon.value = link ? link.icon || '' : '';
  els.fPinned.checked = link ? !!link.pinned : false;

  els.dialogTitle.textContent = link ? 'Chỉnh sửa liên kết' : 'Thêm liên kết';
  els.saveBtn.textContent = link ? 'Lưu thay đổi' : 'Lưu liên kết';

  els.editDialog.showModal();
  setTimeout(() => (link ? els.fTitle : els.fUrl).focus(), 60);
}

function showFormError(message) {
  els.formError.textContent = message;
  els.formError.hidden = false;
}

async function submitEditor(event) {
  event.preventDefault();
  els.formError.hidden = true;

  const rawURL = els.fUrl.value.trim();
  if (!rawURL) {
    showFormError('Vui lòng nhập URL cho liên kết.');
    return;
  }

  let candidate = rawURL;
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(candidate)) candidate = 'https://' + candidate;
  try {
    const parsed = new URL(candidate);
    if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') || !parsed.hostname) {
      throw new Error('invalid');
    }
  } catch {
    showFormError('URL không hợp lệ. Hãy nhập địa chỉ bắt đầu bằng http:// hoặc https://.');
    return;
  }

  els.saveBtn.classList.add('loading');
  els.saveBtn.disabled = true;

  try {
    const categoryIDs = [...catDraft];
    const typedCategory = els.fCatInput.value.trim();
    if (typedCategory) {
      const cat = await ensureCategory(typedCategory);
      if (!categoryIDs.includes(cat.id)) categoryIDs.push(cat.id);
    }

    const payload = {
      url: rawURL,
      title: els.fTitle.value.trim(),
      description: els.fDesc.value.trim(),
      categoryIds: categoryIDs,
      icon: els.fIcon.value.trim(),
      pinned: els.fPinned.checked,
    };

    if (editingId) {
      const saved = await api('/api/links/' + encodeURIComponent(editingId), { method: 'PUT', body: payload });
      const index = state.links.findIndex((l) => l.id === editingId);
      if (index >= 0) state.links[index] = saved;
      delete state.health[saved.id];
      toast('Đã cập nhật liên kết');
    } else {
      const saved = await api('/api/links', { method: 'POST', body: payload });
      state.links.push(saved);
      state.filter = 'all';
      state.query = '';
      els.searchInput.value = '';
      toast('Đã thêm liên kết mới');
    }
    els.editDialog.close();
    render();
  } catch (err) {
    showFormError(err.message);
  } finally {
    els.saveBtn.classList.remove('loading');
    els.saveBtn.disabled = false;
  }
}

/* ---------- Category manager ---------- */
function openCatManager() {
  els.catError.hidden = true;
  els.catNameInput.value = '';
  renderCatManager();
  els.catDialog.showModal();
  setTimeout(() => els.catNameInput.focus(), 60);
}

function renderCatManager() {
  const counts = categoryCounts();
  els.catList.replaceChildren();

  for (const cat of state.categories) {
    const row = document.createElement('li');
    row.className = 'cat-row';
    row.dataset.id = cat.id;

    const dot = document.createElement('span');
    dot.className = 'cat-dot';
    dot.style.setProperty('--h', hueOf(cat.name));

    const input = document.createElement('input');
    input.className = 'cat-name';
    input.type = 'text';
    input.maxLength = 40;
    input.value = cat.name;
    input.setAttribute('aria-label', 'Tên danh mục ' + cat.name);

    const count = document.createElement('span');
    count.className = 'cat-count';
    count.textContent = fmtNumber(counts.get(cat.id) || 0);
    count.title = (counts.get(cat.id) || 0) + ' liên kết';

    const del = document.createElement('button');
    del.type = 'button';
    del.className = 'icon-btn cat-del';
    del.dataset.catDel = cat.id;
    del.title = 'Xóa danh mục';
    del.setAttribute('aria-label', 'Xóa danh mục ' + cat.name);
    del.innerHTML = '<svg class="ic"><use href="#i-trash"/></svg>';

    row.append(dot, input, count, del);
    els.catList.append(row);
  }

  els.catEmpty.hidden = state.categories.length > 0;
}

async function createCategoryFromManager(event) {
  event.preventDefault();
  const name = els.catNameInput.value.trim();
  if (!name) {
    els.catNameInput.focus();
    return;
  }

  els.catError.hidden = true;
  const existing = findCategoryByName(name);
  try {
    const cat = await ensureCategory(name);
    els.catNameInput.value = '';
    render();
    els.catNameInput.focus();
    toast(existing ? 'Danh mục đã tồn tại' : 'Đã thêm danh mục “' + cat.name + '”', existing ? 'info' : 'success');
  } catch (err) {
    els.catError.textContent = err.message;
    els.catError.hidden = false;
  }
}

async function renameCategory(input) {
  const row = input.closest('.cat-row');
  const cat = row ? catById(row.dataset.id) : null;
  if (!cat) return;

  const name = input.value.trim();
  if (!name) {
    input.value = cat.name;
    toast('Tên danh mục không được để trống', 'error');
    return;
  }
  if (name === cat.name) {
    input.value = cat.name;
    return;
  }

  try {
    const updated = await api('/api/categories/' + encodeURIComponent(cat.id), {
      method: 'PUT',
      body: { name },
    });
    Object.assign(cat, updated);
    input.value = cat.name;
    render();
    toast('Đã đổi tên danh mục', 'info');
  } catch (err) {
    input.value = cat.name;
    toast(err.message, 'error');
  }
}

async function deleteCategory(id) {
  const cat = catById(id);
  if (!cat) return;

  const count = categoryCounts().get(id) || 0;
  const ok = await askConfirm(
    'Xóa danh mục?',
    'Danh mục “' + cat.name + '” sẽ bị gỡ khỏi ' + fmtNumber(count) +
      ' liên kết. Các liên kết vẫn được giữ lại.',
    'Xóa'
  );
  if (!ok) return;

  try {
    await api('/api/categories/' + encodeURIComponent(id), { method: 'DELETE' });
    state.categories = state.categories.filter((c) => c.id !== id);
    for (const link of state.links) {
      link.categoryIds = (link.categoryIds || []).filter((cid) => cid !== id);
    }
    if (state.filter === 'cat:' + id) state.filter = 'all';
    render();
    toast('Đã xóa danh mục');
  } catch (err) {
    toast(err.message, 'error');
  }
}

async function fetchMetadata({ silent = false, overwrite = false } = {}) {
  const rawURL = els.fUrl.value.trim();
  if (!rawURL) {
    if (!silent) toast('Hãy nhập URL trước', 'info');
    return;
  }
  if (els.fetchBtn.disabled) return;

  els.fetchBtn.classList.add('loading');
  els.fetchBtn.disabled = true;
  try {
    const data = await api('/api/metadata?url=' + encodeURIComponent(rawURL));
    let filled = false;
    if (data.title && (overwrite || !els.fTitle.value.trim())) {
      els.fTitle.value = data.title;
      filled = true;
    }
    if (data.description && (overwrite || !els.fDesc.value.trim())) {
      els.fDesc.value = data.description;
      filled = true;
    }
    if (!silent) {
      toast(filled ? 'Đã lấy thông tin từ trang web' : 'Không tìm thấy thông tin bổ sung', filled ? 'success' : 'info');
    }
  } catch (err) {
    if (!silent) toast(err.message, 'error');
  } finally {
    els.fetchBtn.classList.remove('loading');
    els.fetchBtn.disabled = false;
  }
}

/* ---------- Health check ---------- */
async function checkLinks() {
  if (!state.links.length) {
    toast('Chưa có liên kết để kiểm tra', 'info');
    return;
  }
  els.checkBtn.classList.add('loading');
  els.checkBtn.disabled = true;

  try {
    const urls = Array.from(new Set(state.links.map((l) => l.url)));
    const results = await api('/api/check', { method: 'POST', body: { urls } });
    const byURL = new Map(results.map((r) => [r.url, r.ok]));

    state.health = {};
    for (const link of state.links) {
      state.health[link.id] = byURL.get(link.url) ? 'ok' : 'fail';
    }
    render();

    const okCount = results.filter((r) => r.ok).length;
    toast(
      okCount + '/' + results.length + ' liên kết phản hồi tốt',
      okCount === results.length ? 'success' : 'info'
    );
  } catch (err) {
    toast(err.message, 'error');
  } finally {
    els.checkBtn.classList.remove('loading');
    els.checkBtn.disabled = false;
  }
}

/* ---------- Import / export ---------- */
async function importFile(file) {
  if (!file) return;

  const text = await file.text();
  let payload;
  try {
    payload = JSON.parse(text);
  } catch {
    toast('Tệp không phải JSON hợp lệ', 'error');
    return;
  }

  const items = Array.isArray(payload) ? payload : Array.isArray(payload.links) ? payload.links : null;
  if (!items) {
    toast('Tệp không chứa danh sách liên kết', 'error');
    return;
  }

  const ok = await askConfirm(
    'Nhập dữ liệu?',
    'Thêm ' + fmtNumber(items.length) + ' liên kết từ tệp “' + file.name + '” vào dashboard?',
    'Nhập'
  );
  if (!ok) return;

  try {
    const result = await api('/api/import?mode=merge', { method: 'POST', body: text });
    toast('Đã nhập ' + fmtNumber(result.imported) + ' liên kết' +
      (result.skipped ? ', bỏ qua ' + fmtNumber(result.skipped) : ''));
    await refresh();
  } catch (err) {
    toast(err.message, 'error');
  }
}

/* ---------- Drag & drop ---------- */
let dragId = null;

function clearDragState() {
  dragId = null;
  $$('.card.dragging, .card.drag-over', els.grid).forEach((card) => {
    card.classList.remove('dragging', 'drag-over');
  });
}

function bindDragAndDrop() {
  els.grid.addEventListener('dragstart', (event) => {
    const card = event.target.closest('.card');
    if (!card || !card.draggable) {
      event.preventDefault();
      return;
    }
    dragId = card.dataset.id;
    card.classList.add('dragging');
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', dragId);
  });

  els.grid.addEventListener('dragover', (event) => {
    if (!dragId) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
    const card = event.target.closest('.card');
    $$('.card.drag-over', els.grid).forEach((c) => c.classList.remove('drag-over'));
    if (card && card.dataset.id !== dragId) card.classList.add('drag-over');
  });

  els.grid.addEventListener('drop', (event) => {
    if (!dragId) return;
    event.preventDefault();
    const target = event.target.closest('.card');
    const source = dragId;
    clearDragState();
    if (!target || target.dataset.id === source) return;

    const ids = sortLinks(state.links).map((l) => l.id);
    const from = ids.indexOf(source);
    const to = ids.indexOf(target.dataset.id);
    if (from < 0 || to < 0) return;

    ids.splice(to, 0, ids.splice(from, 1)[0]);
    state.links.forEach((link) => { link.order = ids.indexOf(link.id); });
    render();

    api('/api/reorder', { method: 'POST', body: { ids } }).catch((err) => {
      toast(err.message, 'error');
      refresh();
    });
  });

  els.grid.addEventListener('dragend', clearDragState);
}

/* ---------- Initialisation ---------- */
async function refresh() {
  const [links, categories] = await Promise.all([api('/api/links'), api('/api/categories')]);
  state.links = links || [];
  state.categories = categories || [];
  if (state.filter.startsWith('cat:') && !catById(state.filter.slice(4))) state.filter = 'all';
  render();
}

function bindEvents() {
  document.addEventListener('click', (event) => {
    if (!event.target.closest('.cat-picker')) closeCatMenu();

    if (event.target.closest('.density-wrap')) {
      if (event.target.closest('#densityBtn')) {
        toggleDensityMenu();
        return;
      }
      const option = event.target.closest('[data-density]');
      if (option) {
        state.density = option.dataset.density;
        applyDensityUI(state.density);
        closeDensityMenu();
        els.densityBtn.focus();
      }
      return;
    }
    closeDensityMenu();

    const manageTrigger = event.target.closest('#manageCatsBtn, [data-action="manage-cats"]');
    if (manageTrigger) {
      openCatManager();
      return;
    }

    const filterBtn = event.target.closest('[data-filter]');
    if (filterBtn) {
      state.filter = filterBtn.dataset.filter;
      if (window.innerWidth <= 1020) window.scrollTo({ top: 0, behavior: 'smooth' });
      render();
      return;
    }
    const addBtn = event.target.closest('[data-action="add"]');
    if (addBtn) openEditor(null);
  });

  els.grid.addEventListener('click', (event) => {
    const card = event.target.closest('.card');
    if (!card) return;
    const id = card.dataset.id;
    const actionBtn = event.target.closest('button[data-act]');

    if (actionBtn) {
      event.preventDefault();
      event.stopPropagation();
      const act = actionBtn.dataset.act;
      if (act === 'fav') toggleFavorite(id);
      else if (act === 'copy') copyLink(id);
      else if (act === 'edit') openEditor(state.links.find((l) => l.id === id));
      else if (act === 'del') deleteLink(id);
      return;
    }

    if (event.target.closest('.card-hit')) recordVisit(id);
  });

  els.grid.addEventListener('auxclick', (event) => {
    if (event.button !== 1) return;
    const card = event.target.closest('.card');
    if (card && event.target.closest('.card-hit')) recordVisit(card.dataset.id);
  });

  bindDragAndDrop();

  let searchTimer;
  els.searchInput.addEventListener('input', () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      state.query = els.searchInput.value;
      render();
    }, 80);
  });

  els.sortSelect.addEventListener('change', () => {
    state.sort = SORTS.includes(els.sortSelect.value) ? els.sortSelect.value : 'manual';
    localStorage.setItem('lh-sort', state.sort);
    render();
  });

  els.clearFilter.addEventListener('click', () => {
    state.filter = 'all';
    state.query = '';
    els.searchInput.value = '';
    render();
    els.searchInput.focus();
  });

  els.themeBtn.addEventListener('click', () => {
    const next = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
    applyThemeUI(next);
  });

  els.checkBtn.addEventListener('click', checkLinks);

  els.exportBtn.addEventListener('click', () => {
    const anchor = document.createElement('a');
    anchor.href = '/api/export';
    anchor.download = '';
    document.body.append(anchor);
    anchor.click();
    anchor.remove();
  });

  els.importBtn.addEventListener('click', () => els.fileInput.click());
  els.fileInput.addEventListener('change', async () => {
    const file = els.fileInput.files && els.fileInput.files[0];
    els.fileInput.value = '';
    if (file) await importFile(file);
  });

  els.linkForm.addEventListener('submit', submitEditor);

  els.catPickerField.addEventListener('click', (event) => {
    if (event.target.closest('.cat-chip-x')) return;
    renderCatMenu();
    els.fCatInput.focus();
  });
  els.fCatInput.addEventListener('focus', () => renderCatMenu());
  els.fCatInput.addEventListener('input', () => {
    catActive = 0;
    renderCatMenu();
  });
  els.fCatInput.addEventListener('keydown', (event) => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      const items = catMenuItems();
      if (!items.length) return;
      event.preventDefault();
      catActive = (catActive + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length;
      renderCatMenu();
      return;
    }
    if (event.key === 'Enter' && !els.catMenu.hidden) {
      event.preventDefault();
      chooseCatItem(catActive);
      return;
    }
    if (event.key === 'Escape' && !els.catMenu.hidden) {
      event.preventDefault();
      closeCatMenu();
      return;
    }
    if (event.key === 'Backspace' && !els.fCatInput.value && catDraft.length) {
      toggleCatDraft(catDraft[catDraft.length - 1]);
    }
  });
  els.editDialog.addEventListener('close', closeCatMenu);

  els.catCreateForm.addEventListener('submit', createCategoryFromManager);
  els.catList.addEventListener('change', (event) => {
    const input = event.target.closest('.cat-name');
    if (input) renameCategory(input);
  });
  els.catList.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && event.target.classList.contains('cat-name')) {
      event.preventDefault();
      event.target.blur();
    }
  });
  els.catList.addEventListener('click', (event) => {
    const del = event.target.closest('[data-cat-del]');
    if (del) deleteCategory(del.dataset.catDel);
  });
  els.catDialog.addEventListener('click', (event) => {
    if (event.target === els.catDialog) els.catDialog.close();
  });
  els.catDialog.addEventListener('close', () => {
    els.catError.hidden = true;
  });

  els.fetchBtn.addEventListener('click', () => fetchMetadata({ overwrite: true }));
  els.fUrl.addEventListener('blur', (event) => {
    if (event.relatedTarget === els.fetchBtn) return;
    if (!els.editDialog.open) return;
    if (!els.fUrl.value.trim() || els.fTitle.value.trim()) return;
    setTimeout(() => fetchMetadata({ silent: true }), 90);
  });

  $$('[data-close]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const dialog = btn.closest('dialog');
      if (dialog) dialog.close();
    });
  });

  els.editDialog.addEventListener('click', (event) => {
    if (event.target === els.editDialog) els.editDialog.close();
  });

  els.okConfirm.addEventListener('click', () => settleConfirm(true));
  els.cancelConfirm.addEventListener('click', () => settleConfirm(false));
  els.confirmDialog.addEventListener('cancel', (event) => {
    event.preventDefault();
    settleConfirm(false);
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !els.densityMenu.hidden) {
      event.preventDefault();
      closeDensityMenu();
      els.densityBtn.focus();
      return;
    }
    if (event.key === 'Escape' && !els.catMenu.hidden) {
      event.preventDefault();
      closeCatMenu();
      els.fCatInput.focus();
      return;
    }

    const typing = event.target.closest('input, textarea, select, [contenteditable="true"]');
    const modalOpen = els.editDialog.open || els.confirmDialog.open || els.catDialog.open;

    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      els.searchInput.focus();
      els.searchInput.select();
      return;
    }
    if (modalOpen) return;

    if (event.key === '/' && !typing) {
      event.preventDefault();
      els.searchInput.focus();
      els.searchInput.select();
    } else if ((event.key === 'n' || event.key === 'N') && !typing && !event.ctrlKey && !event.metaKey && !event.altKey) {
      event.preventDefault();
      openEditor(null);
    } else if (event.key === 'Escape' && document.activeElement === els.searchInput && els.searchInput.value) {
      els.searchInput.value = '';
      state.query = '';
      render();
    }
  });
}

async function init() {
  applyThemeUI(document.documentElement.dataset.theme || 'dark');
  applyDensityUI(state.density);
  setGreeting();
  els.sortSelect.value = state.sort;
  bindEvents();

  try {
    await refresh();
  } catch (err) {
    render();
    toast('Không thể tải dữ liệu: ' + err.message, 'error');
  }
}

init();
