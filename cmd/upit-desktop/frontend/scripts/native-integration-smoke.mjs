// Run against a native Desktop built with -tags mcp and a disposable HOME:
// node scripts/native-integration-smoke.mjs http://127.0.0.1:19109 setup
import assert from 'node:assert/strict'

const [endpoint, mode] = process.argv.slice(2)
assert(endpoint && ['setup', 'repair'].includes(mode), 'Provide MCP endpoint and Setup/Repair mode')
const url = new URL(endpoint)
assert(['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname), 'Only loopback MCP endpoints are allowed')
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

assert.equal(await js(`return Boolean(document.querySelector('.${mode}-card'))`), 'true', `Desktop must start in ${mode}`)
assert.equal(await js(`const b = [...document.querySelectorAll('.nav-item')].find(e => e.querySelector('.nav-label')?.innerText === 'File Manager Integration'); if (!b || b.disabled) return false; b.focus(); b.click(); return true`), 'true', 'Integration must remain accessible during Configuration Set recovery')
assert.equal(await js(`const deadline = Date.now() + 3000; while (Date.now() < deadline && !document.querySelector('#integration-title')?.parentElement.querySelector('strong[role=status]')) { if (document.querySelector('#integration-title')?.parentElement.querySelector('[role=alert]')) break; await new Promise(r => setTimeout(r, 25)); } return Boolean(document.querySelector('#integration-title')?.parentElement.querySelector('strong[role=status]'))`), 'true', 'Integration must render a real status')
assert.equal(await js(`return [...document.querySelectorAll('.nav-item')].filter(e => e.querySelector('.nav-label')?.innerText !== 'File Manager Integration').every(e => e.disabled)`), 'true', 'Integration must not unlock configuration-dependent navigation')
const status = await js(`return document.querySelector('#integration-title')?.parentElement.querySelector('strong[role=status]')?.innerText ?? ''`)
assert(['Registered', 'Needs Repair', 'Reinstall Upit', 'Not supported on Linux'].includes(status), `Expected a real integration status, got ${status}`)
if (process.platform === 'linux') {
  assert.equal(status, 'Not supported on Linux')
  assert.equal(await js(`return document.querySelector('#integration-title')?.parentElement.querySelectorAll('.editor-actions button').length`), '0', 'Linux must expose no platform actions')
}
assert.equal(await js(`const b = document.querySelector('#integration-title')?.parentElement.querySelector('button'); if (!b || b.disabled) return false; b.click(); await new Promise(r => setTimeout(r, 50)); return Boolean(document.querySelector('.${mode}-card'))`), 'true', 'Return action must restore Configuration Set recovery')
console.log(`PASS native ${mode}: Integration accessible, configuration navigation remains locked, real status ${status}, return action works`)
