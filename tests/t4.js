const { chromium } = require('playwright');
const { execFileSync } = require('child_process');
const [url, save] = process.argv.slice(2);
const dec = (f) => execFileSync('python3', ['-I', 'tools/show_save.py', f], { encoding: 'utf8' });
const m = (t) => t.split('\n').filter((l) => l.startsWith('marks|')).join(' ');
(async () => {
  const b = await chromium.launch(); const p = await b.newPage({ viewport: { width: 1280, height: 900 } });
  await p.goto(url); await p.waitForSelector('h2');
  for (const [c, how] of [['2', 'fill+tab'], ['8', 'type+click save']]) {
    await p.click(`button[data-a=cls][data-v="${c}"]`);
    const i = p.locator('input[data-c=marks]');
    if (how === 'fill+tab') { await i.fill('8000'); await i.press('Tab'); await p.click('button[data-a=save]'); }
    else { await i.click({ clickCount: 3 }); await p.keyboard.type('8000', { delay: 40 }); await p.locator('h2').first().click(); await p.click('button[data-a=save]'); }
    await p.waitForSelector('.banner.ok'); console.log(how, '->', m(dec(save)));
  }
  await b.close();
})();
