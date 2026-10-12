// The lock page (pair#495): every cast link opens here until its code is
// entered. The server issues the session as an HttpOnly cookie, so this
// script never holds it; it only posts the code and, once in, reloads the
// link, which then serves the viewer.
//
// The name is the viewer's, for their notes: kept in localStorage under the
// key the web viewer's notes already read (couch.viewerName), so neither page
// asks again.

export const NAME_KEY = 'couch.viewerName';
export const MAX_NAME = 40;

// codeFromFragment reads a code carried as URL#NNNN. Browsers never send the
// fragment, and link-preview fetchers don't run this script.
export function codeFromFragment(hash) {
  const m = /^#([0-9][0-9 -]{2,14})$/.exec(hash || '');
  return m ? m[1].replace(/[ -]/g, '') : '';
}

export function cleanName(name) {
  return String(name || '').replace(/\s+/g, ' ').trim().slice(0, MAX_NAME);
}

// unlock posts one attempt and reports {ok, error}.
export async function unlock(fetchFn, code) {
  let resp;
  try {
    resp = await fetchFn('unlock', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
      credentials: 'same-origin',
    });
  } catch {
    return { ok: false, error: 'Couldn’t reach the cast. Try again.' };
  }
  let answer = {};
  try { answer = await resp.json(); } catch { /* fixed text below */ }
  if (resp.ok && answer.ok) return { ok: true };
  return { ok: false, error: answer.error || 'That didn’t work. Try again.' };
}

function storage() {
  try { return window.localStorage; } catch { return null; }
}

function start() {
  const form = document.getElementById('lock');
  const code = document.getElementById('code');
  const name = document.getElementById('name');
  const message = document.getElementById('message');
  const button = form.querySelector('button');
  const store = storage();
  try { name.value = store?.getItem(NAME_KEY) || ''; } catch { /* private mode */ }

  const fromURL = codeFromFragment(location.hash);
  if (location.hash) {
    // Drop the fragment at once: it is a code, and history keeps URLs.
    history.replaceState(null, '', location.pathname + location.search);
  }
  if (fromURL) code.value = fromURL;

  async function submit() {
    const who = cleanName(name.value);
    if (!code.value.trim()) { message.textContent = 'Enter the code the host gave you.'; code.focus(); return; }
    if (!who) { message.textContent = 'Add your name.'; name.focus(); return; }
    button.disabled = true;
    message.textContent = '';
    const result = await unlock(fetch, code.value);
    button.disabled = false;
    if (!result.ok) {
      message.textContent = result.error;
      code.select();
      return;
    }
    try { store?.setItem(NAME_KEY, who); } catch { /* the cast still opens */ }
    location.replace('./');
  }

  form.addEventListener('submit', (e) => { e.preventDefault(); submit(); });
  if (fromURL && cleanName(name.value)) submit();
  else (fromURL ? name : code).focus();
}

if (typeof document !== 'undefined') start();
