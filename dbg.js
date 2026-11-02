const { chromium } = require('playwright');
(async () => {
  const b = await chromium.launch(); const p = await b.newPage();
  p.on('pageerror', (e) => console.log('PAGEERROR', e.message));
  p.on('console', (m) => console.log('CONSOLE', m.type(), m.text()));
  p.on('response', (r) => { if (r.url().includes('/api/')) console.log(r.status(), r.url().split('?')[0]); });
  await p.goto(process.argv[2]); await p.waitForTimeout(2500);
  console.log('BODY', (await p.textContent('body')).slice(0, 300).replace(/\s+/g, ' '));
  await b.close();
})();
