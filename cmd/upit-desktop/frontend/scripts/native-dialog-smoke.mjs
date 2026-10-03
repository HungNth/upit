// Run against a native `go build -tags mcp` desktop with disposable definitions:
// node scripts/native-dialog-smoke.mjs http://127.0.0.1:19099 upit-smoke-uploader upit-smoke-shortener
import assert from 'node:assert/strict'

const [endpoint, uploader, shortener] = process.argv.slice(2)
assert(endpoint && uploader && shortener, 'Provide MCP endpoint and two disposable definition names')
const url = new URL(endpoint)
assert(['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname), 'Only loopback MCP endpoints are allowed')
for (const name of [uploader, shortener]) {
  assert(name.startsWith('upit-smoke-'), 'Only upit-smoke-* disposable definitions may be renamed/deleted')
}
url.pathname = '/mcp'

async function js(source) {
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name: 'js_eval', arguments: { js: source } } }),
    signal: AbortSignal.timeout(10000),
  })
  assert(response.ok, `MCP HTTP ${response.status}`)
  const result = await response.json()
  assert(!result.error && !result.result.isError, JSON.stringify(result))
  return result.result.content[0].text
}

async function click(label, selector = 'button') {
  await js(`const e = [...document.querySelectorAll(${JSON.stringify(selector)})].find(e => e.innerText.trim() === ${JSON.stringify(label)}); if (!e || e.disabled) throw new Error('Missing enabled button: ' + ${JSON.stringify(label)}); e.focus(); e.click(); await new Promise(r => setTimeout(r, 100)); return true`)
}

async function waitFor(predicate, message) {
  const deadline = Date.now() + 3000
  while (Date.now() < deadline) {
    if (await js(`return Boolean(${predicate})`) === 'true') return
    await new Promise(resolve => setTimeout(resolve, 50))
  }
  const error = await js('return document.querySelector("[role=dialog] [role=alert], .uploader-form [role=alert]")?.innerText ?? "no validation error"')
  assert.fail(`${message}: ${error}`)
}

for (const [area, name] of [['Uploaders', uploader], ['Shorteners', shortener]]) {
  await js(`const e = [...document.querySelectorAll('.nav-item')].find(e => e.innerText.startsWith(${JSON.stringify(area)})); e.click(); await new Promise(r => setTimeout(r, 100)); return true`)
  await click('Refresh')
  await click(name)
  assert.equal(await js('return [...document.querySelectorAll("input,select,textarea")].filter(e => e.getClientRects().length && !e.labels?.length && !e.getAttribute("aria-label") && !e.getAttribute("aria-labelledby")).length'), '0', `${area}: every visible control must have an accessible name`)
  if (area === 'Uploaders') {
    await js('const e = document.querySelectorAll(".uploader-form select")[1]; e.value = "body"; e.dispatchEvent(new Event("change", { bubbles: true })); await new Promise(r => setTimeout(r, 50)); return true')
    await click('Save Uploader')
    await waitFor('[...document.querySelectorAll("button")].some(e => e.innerText.trim() === "Save Uploader" && e.disabled) && !document.querySelector(".uploader-form [role=alert]")', 'JSON-to-body URL transition must save without hidden path fields')
    await js('document.querySelector(".uploader-form input[type=checkbox]").click(); await new Promise(r => setTimeout(r, 50)); const e = document.querySelectorAll(".uploader-form select")[2]; e.value = "json"; e.dispatchEvent(new Event("change", { bubbles: true })); await new Promise(r => setTimeout(r, 50)); const p = document.querySelector("input[placeholder=JSONPath]"); p.value = "$.error"; p.dispatchEvent(new Event("input", { bubbles: true })); e.value = "body"; e.dispatchEvent(new Event("change", { bubbles: true })); await new Promise(r => setTimeout(r, 50)); return true')
    await click('Save Uploader')
    await waitFor('[...document.querySelectorAll("button")].some(e => e.innerText.trim() === "Save Uploader" && e.disabled) && !document.querySelector(".uploader-form [role=alert]")', 'JSON-to-body error transition must save without hidden path fields')
  }
  await click('Rename')
  await waitFor('document.querySelector("[role=dialog] #rename-input") && document.activeElement.id === "rename-input"', `${area}: Rename dialog focus`)
  assert.equal(await js('return !!document.querySelector("[role=dialog] #rename-input")'), 'true', `${area}: Rename must present a usable dialog`)
  assert.equal(await js('return document.activeElement.id'), 'rename-input', `${area}: rename input must receive focus`)
  assert.equal(await js('const buttons = document.querySelectorAll("[role=dialog] button"); const last = buttons[buttons.length - 1]; last.focus(); last.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true })); return document.activeElement.id'), 'rename-input', `${area}: Tab must wrap inside Rename`)
  assert.equal(await js('const e = document.getElementById("rename-input"); e.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", shiftKey: true, bubbles: true, cancelable: true })); return document.activeElement.innerText.trim()'), 'Cancel', `${area}: Shift+Tab must wrap inside Rename`)
  await js(`const e = document.getElementById('rename-input'); e.value = ${JSON.stringify(` ${name} `)}; e.dispatchEvent(new Event('input', { bubbles: true })); await new Promise(r => setTimeout(r, 50)); return true`)
  await click('Rename', '[role=dialog] button')
  await waitFor('document.querySelector("[role=dialog] [role=alert]")', `${area}: whitespace-bearing names must be rejected`)
  assert.equal(await js('const form = document.querySelector("[role=dialog] form"); const id = form.getAttribute("aria-describedby"); return !!id && !!document.getElementById(id)?.innerText'), 'true', `${area}: validation error must be associated with Rename form`)
  assert.equal(await js(`return document.getElementById(${JSON.stringify(area === 'Uploaders' ? 'uploader-name' : 'shortener-name')}).value`), name, `${area}: invalid Rename must preserve the original definition`)
  assert.equal(await js('return document.getElementById("rename-input").value'), ` ${name} `, `${area}: invalid input must not be silently normalized`)
  const renamed = `${name}-renamed`
  await js(`const e = document.getElementById('rename-input'); e.value = ${JSON.stringify(renamed)}; e.dispatchEvent(new Event('input', { bubbles: true })); await new Promise(r => setTimeout(r, 50)); return true`)
  await click('Rename', '[role=dialog] button')
  await waitFor('!document.querySelector("[role=dialog]")', `${area}: successful Rename must close dialog`)
  assert.equal(await js('return !!document.querySelector("[role=dialog]")'), 'false', `${area}: successful Rename must close dialog`)
  assert.equal(await js(`return document.getElementById(${JSON.stringify(area === 'Uploaders' ? 'uploader-name' : 'shortener-name')}).value`), renamed, `${area}: persisted editor must select renamed definition`)
  await click('Delete')
  await waitFor('document.querySelector("[role=dialog]") && document.activeElement.innerText.trim() === "Cancel"', `${area}: safe Delete focus`)
  assert.equal(await js('return document.activeElement.innerText.trim()'), 'Cancel', `${area}: Delete must focus safe action`)
  await click('Cancel', '[role=dialog] button')
  await waitFor('!document.querySelector("[role=dialog]") && document.activeElement.innerText.trim() === "Delete"', `${area}: Delete focus restoration`)
  assert.equal(await js('return document.activeElement.innerText.trim()'), 'Delete', `${area}: dismissing dialog must restore focus`)
  await click('Delete')
  await click('Delete', '[role=dialog] button')
  await waitFor('!document.querySelector("[role=dialog]")', `${area}: confirmed Delete must close dialog`)
  assert.equal(await js('return !!document.querySelector("[role=dialog]")'), 'false', `${area}: confirmed Delete must close dialog`)
  assert.equal(await js(`return [...document.querySelectorAll('button')].some(e => e.innerText.trim() === ${JSON.stringify(renamed)})`), 'false', `${area}: deleted definition must disappear`)
  console.log(`PASS native ${area}: Rename, Delete, confirmation, initial/restored focus`)
}
