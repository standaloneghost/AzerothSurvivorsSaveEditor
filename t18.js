const { chromium } = require('playwright');
const { execFileSync } = require('child_process');
const [url, save] = process.argv.slice(2);
(async () => {
  const b = await chromium.launch(); const p = await b.newPage();
  await p.goto(url); await p.waitForSelector('h2');
  await p.click('button[data-a=page][data-v=rewards]'); await p.click('button[data-a=rtab][data-v=Tabards]');
  await p.check('input[data-c=tabard][data-k="90011"]');
  await p.click('button[data-a=save]'); await p.waitForSelector('.banner.ok');
  const L = execFileSync('python3', ['-I', '../dec/show.py', save], { encoding: 'utf8' }).split('\n');
  const i = L.findIndex((l) => l.startsWith('tabard|'));
  console.log('no-line save:', L[i - 1].slice(0, 5), '|', L[i], '|', L[i + 1].slice(0, 6));
  await b.close();
})();
