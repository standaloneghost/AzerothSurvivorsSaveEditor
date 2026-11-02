const { chromium } = require('playwright');
const [url] = process.argv.slice(2);
let fails = 0; const ok = (c, m) => { console.log((c ? 'PASS ' : 'FAIL ') + m); if (!c) fails++; };
(async () => {
  const b = await chromium.launch(); const p = await b.newPage({ viewport: { width: 1280, height: 900 } });
  const errs = []; p.on('pageerror', (e) => errs.push(e.message));
  await p.goto(url); await p.waitForSelector('h2');
  ok(await p.locator('.tabs, [role=tablist]').count() === 0, 'tab bar removed');
  const names = await p.$$eval('nav .navl', (n) => [...n[0].querySelectorAll('b')].map((x) => x.textContent));
  ok(names.join() === 'Achievements,Rewards,Lifetime stats,Advanced,Backups', 'player section: ' + names.join(', '));
  await p.screenshot({ path: '../shots/nav.png' });
  const page = async (v, text) => { await p.click(`button[data-a=page][data-v=${v}]`); ok((await p.textContent('h2')) === text, v + ' opens ' + text); ok(await p.locator('.navb.sel').count() === 1, v + ' highlights one item'); ok(await p.locator('.armor').count() === 0, v + ' hides class view'); };
  await page('life', 'Lifetime stats'); await page('raw', 'Advanced'); await page('backups', 'Backups'); await page('ach', 'Achievements'); await page('rewards', 'Rewards');
  for (const [c, n] of [['8', 'Mage'], ['4', 'Rogue']]) {
    await p.click('button[data-a=page][data-v=backups]');
    await p.click(`button[data-a=cls][data-v="${c}"]`);
    ok((await p.textContent('h2')) === n && await p.locator('.zones').count() === 1 && await p.locator('.armor').count() === 1, n + ' opens zones and gear');
    ok((await p.textContent('.navb.sel b')) === n, n + ' highlighted');
  }
  await p.setViewportSize({ width: 390, height: 844 });
  ok(await p.evaluate(() => document.documentElement.scrollWidth - innerWidth) <= 0, 'no sideways scroll at phone width');
  ok(errs.length === 0, 'no page errors');
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
  await b.close();
})().catch((e) => { console.error(e); process.exit(1); });
