import { chromium } from 'playwright'
import { mkdirSync } from 'node:fs'

const BASE = process.env.BASE_URL ?? 'http://localhost:5173'
const SHOTS = process.env.SHOTS_DIR ?? new URL('./shots', import.meta.url).pathname
mkdirSync(SHOTS, { recursive: true })

let pass = 0
let fail = 0
const check = (desc, ok, detail = '') => {
  if (ok) {
    console.log(`  PASS  ${desc}`)
    pass++
  } else {
    console.log(`  FAIL  ${desc}${detail ? ' -- ' + detail : ''}`)
    fail++
  }
}

const stamp = Date.now()
const A = { first: 'Ada', last: `Lovelace${stamp}`, pw: 'pw12345' }
const B = { first: 'Alan', last: `Turing${stamp}`, pw: 'pw12345' }

const browser = await chromium.launch()

// Console/page errors are the whole point of this test: a build can pass while
// the app throws on mount.
const errors = []
function watch(page, label) {
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(`[${label}] console: ${m.text()}`)
  })
  page.on('pageerror', (e) => errors.push(`[${label}] pageerror: ${e.message}`))
}

async function register(page, who) {
  await page.goto(`${BASE}/register`)
  await page.getByLabel('First name').fill(who.first)
  await page.getByLabel('Last name').fill(who.last)
  await page.getByLabel('Password').fill(who.pw)
  await page.getByLabel('City').fill('Cambridge')
  await page.getByRole('button', { name: 'Register' }).click()
  await page.waitForURL('**/login', { timeout: 15000 })
  const id = await page.getByLabel('User ID').inputValue()
  return id
}

async function login(page, id, pw) {
  await page.goto(`${BASE}/login`)
  await page.getByLabel('User ID').fill(id)
  await page.getByLabel('Password').fill(pw)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.waitForURL('**/feed', { timeout: 15000 })
}

// ---- Session A ----
const ctxA = await browser.newContext()
const pageA = await ctxA.newPage()
watch(pageA, 'A')

await pageA.goto(BASE)
await pageA.waitForURL('**/login', { timeout: 15000 })
check('unauthenticated visit redirects to /login', pageA.url().includes('/login'))
await pageA.getByRole('button', { name: 'Sign in' }).waitFor({ timeout: 15000 })
check('login form rendered', true)
await pageA.screenshot({ path: `${SHOTS}/01-login.png` })

const idA = await register(pageA, A)
check('registration returned a user id', /^[0-9a-f-]{36}$/.test(idA), idA)
check(
  'user id is prefilled on the login screen after registering',
  (await pageA.getByLabel('User ID').inputValue()) === idA,
)
await pageA.screenshot({ path: `${SHOTS}/02-registered.png` })

await login(pageA, idA, A.pw)
check('signed in and landed on the feed', pageA.url().includes('/feed'))
await pageA.getByText('Your feed is empty.').waitFor({ timeout: 15000 })
check('empty feed explains what a feed is', true)
await pageA.screenshot({ path: `${SHOTS}/03-feed-empty.png` })

// ---- Session B ----
const ctxB = await browser.newContext()
const pageB = await ctxB.newPage()
watch(pageB, 'B')
const idB = await register(pageB, B)
await login(pageB, idB, B.pw)
check('second account signed in', pageB.url().includes('/feed'))

// ---- A follows B via search ----
await pageA.goto(`${BASE}/search`)
await pageA.getByPlaceholder('First name').fill(B.first)
await pageA.getByPlaceholder('Last name').fill(B.last)
await pageA.getByRole('button', { name: 'Search' }).click()
await pageA.getByRole('button', { name: 'Follow' }).first().waitFor({ timeout: 15000 })
check('search found the other user', true)
await pageA.screenshot({ path: `${SHOTS}/04-search.png` })

await pageA.getByRole('button', { name: 'Follow' }).first().click()
await pageA.getByRole('button', { name: 'Unfollow' }).first().waitFor({ timeout: 15000 })
check('follow flips the button to Unfollow', true)

await pageA.goto(`${BASE}/friends`)
await pageA.getByText(`${B.first} ${B.last}`).first().waitFor({ timeout: 15000 })
check('friends list shows the followed user', true)
await pageA.screenshot({ path: `${SHOTS}/05-friends.png` })

// ---- B posts; A should see it arrive over the WebSocket ----
await pageA.goto(`${BASE}/feed`)
const postText = `websocket delivered this ${stamp}`
await pageB.goto(`${BASE}/feed`)
await pageB.getByPlaceholder("What's on your mind?").fill(postText)
await pageB.getByRole('button', { name: 'Post' }).click()

// The feed shows only posts from accounts you follow, so the author's own post
// deliberately does NOT appear there. Confirm the post was accepted, then check
// the post itself on the author's profile.
await pageB.getByText('Posted').first().waitFor({ timeout: 15000 })
check('author gets confirmation that the post was created', true)

await pageB.goto(`${BASE}/me`)
await pageB.getByText(postText).first().waitFor({ timeout: 15000 })
check("author's own post appears on their profile", true)

await pageB.goto(`${BASE}/feed`)
await pageB.waitForTimeout(1000)
check(
  'author feed correctly excludes their own post (feed = accounts you follow)',
  !(await pageB.getByText(postText).first().isVisible().catch(() => false)),
)

// A never reloads — arrival proves the socket pushed and the cache invalidated.
try {
  await pageA.getByText(postText).first().waitFor({ timeout: 20000 })
  check('post reached the other session live, without a reload', true)
} catch {
  check('post reached the other session live, without a reload', false, 'timed out')
}
await pageA.screenshot({ path: `${SHOTS}/06-feed-live.png` })

// ---- Messaging ----
await pageA.goto(`${BASE}/dialogs/${idB}`)
await pageA.getByPlaceholder('Write a message').fill('hello from the smoke test')
await pageA.getByRole('button', { name: 'Send' }).click()
await pageA.getByText('hello from the smoke test').first().waitFor({ timeout: 15000 })
check('message appears in the thread', true)
await pageA.screenshot({ path: `${SHOTS}/07-thread.png` })

await pageA.goto(`${BASE}/dialogs`)
await pageA.getByText(`${B.first} ${B.last}`).first().waitFor({ timeout: 15000 })
check('conversation list resolves the peer name', true)
await pageA.screenshot({ path: `${SHOTS}/08-dialogs.png` })

// B should see the message too
await pageB.goto(`${BASE}/dialogs/${idA}`)
try {
  await pageB.getByText('hello from the smoke test').first().waitFor({ timeout: 15000 })
  check('recipient sees the message', true)
} catch {
  check('recipient sees the message', false, 'timed out')
}

// ---- Own profile: post, edit, delete ----
await pageA.goto(`${BASE}/me`)
await pageA.getByText(idA).first().waitFor({ timeout: 15000 })
check('own profile shows the user id', true)
await pageA.screenshot({ path: `${SHOTS}/09-profile.png` })

await pageA.goto(`${BASE}/feed`)
await pageA.getByPlaceholder("What's on your mind?").fill('a post to edit')
await pageA.getByRole('button', { name: 'Post' }).click()
await pageA.waitForTimeout(1500)
await pageA.goto(`${BASE}/me`)
await pageA.getByText('a post to edit').first().waitFor({ timeout: 15000 })
check('own post listed on profile', true)

await pageA.getByRole('button', { name: 'Edit' }).first().click()
const box = pageA.locator('textarea:not([aria-hidden="true"])').first()
await box.fill('an edited post')
await pageA.getByRole('button', { name: 'Save' }).click()
try {
  await pageA.getByText('an edited post').first().waitFor({ timeout: 15000 })
  check('post edit persists', true)
} catch {
  check('post edit persists', false, 'timed out')
}

await pageA.getByRole('button', { name: 'Delete' }).first().click()
await pageA.getByRole('button', { name: 'Delete', exact: true }).last().click()
await pageA.waitForTimeout(2000)
try {
  await pageA.getByText('an edited post').first().waitFor({ state: 'detached', timeout: 15000 })
  check('post delete removed it', true)
} catch {
  check('post delete removed it', false, 'post still present')
}

// ---- Sign out ----
await pageA.getByRole('button', { name: 'Sign out' }).click()
await pageA.waitForURL('**/login', { timeout: 15000 })
check('sign out returns to login', pageA.url().includes('/login'))

// ---- Expired/invalid token is discarded on load ----
await ctxA.addInitScript(() => {
  localStorage.setItem('social-media.token', 'not.a.jwt')
})
const pageC = await ctxA.newPage()
watch(pageC, 'C')
await pageC.goto(`${BASE}/feed`)
await pageC.waitForURL('**/login', { timeout: 15000 })
check('a malformed stored token is discarded rather than hanging', pageC.url().includes('/login'))

await browser.close()

console.log('')
if (errors.length) {
  console.log('Browser errors:')
  for (const e of [...new Set(errors)]) console.log('  ' + e)
} else {
  console.log('No console or page errors.')
}
console.log(`\nPASS=${pass} FAIL=${fail} ERRORS=${new Set(errors).size}`)
process.exit(fail === 0 ? 0 : 1)
