const { chromium } = require('playwright');
const fs = require('fs'), { execFileSync } = require('child_process');
const [url, dir, shots] = process.argv.slice(2);
const save = dir + '/characters.txt', flag = dir + '/game.flag';
const dec = (f) => execFileSync('python3', ['-I', 'tools/show_save.py', f], { encoding: 'utf8' });
let fails = 0;
const ok = (c, m) => { console.log((c ? 'PASS ' : 'FAIL ') + m); if (!c) fails++; };
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage({ viewport: { width: 1280, height: 900 } });
  const errs = [];
  p.on('pageerror', (e) => errs.push(e.message));
  p.on('console', (m) => { if (m.type() === 'error') errs.push(m.text()); });
  await p.goto(url);
  await p.waitForSelector('h2');
  const orig = dec(save);
  ok(await p.locator('button[data-a=save]').isDisabled(), 'save disabled with no changes');
  ok((await p.textContent('.pending')).includes('Nothing changed'), 'no phantom changes on open');
  await p.screenshot({ path: shots + '/zones.png', fullPage: true });

  // no-op edits must return to identical text
  await p.click('button[data-a=tier][data-z=sm][data-v="2"]');
  ok((await p.textContent('.pending')).includes('Scarlet Monastery: Heroic None → II'), 'heroic pending');
  await p.click('button[data-a=tier][data-z=sm][data-v="0"]');
  ok((await p.textContent('.pending')).includes('Nothing changed'), 'undo restores identical text');

  // zones, marks, armor
  await p.click('button[data-a=tier][data-z=naxx][data-v="4"]');
  await p.fill('input[data-c=marks]', '2000'); await p.press('input[data-c=marks]', 'Tab');
  const lockCell = p.locator('.armor button[data-a=item]').first();
  await lockCell.click();
  // class with nothing yet: new lines
  await p.click('button[data-a=cls][data-v="11"]');
  await p.click('button[data-a=tier][data-z=elwynn][data-v="1"]');
  await p.fill('input[data-c=marks]', '7'); await p.press('input[data-c=marks]', 'Tab');
  ok((await p.textContent('.armor .count')).includes('of 54'), 'druid armor count');
  await p.click('button[data-a=cls][data-v="4"]');
  ok((await p.textContent('.armor .count')).includes('of 48'), 'rogue armor total 48');
  ok(await p.locator('.cell.na').count() === 12, 'rogue has 12 N/A cells');
  await p.screenshot({ path: shots + '/rogue.png', fullPage: true });

  // achievements
  await p.click('button[data-a=page][data-v=ach]');
  await p.screenshot({ path: shots + '/ach.png', fullPage: true });
  const t0 = Math.floor(Date.now() / 1000);
  await p.click('button[data-a=achcat][data-v="Feats"]');
  await p.check('input[data-c=ach][data-id="feat:stitches"]');
  await p.uncheck('input[data-c=ach][data-id="feat:lairs"]');

  // rewards
  await p.click('button[data-a=page][data-v=rewards]');
  await p.click('button[data-a=perk][data-k=skip][data-v="0"]');
  await p.click('button[data-a=rtab][data-v=Companions]');
  await p.selectOption('select[data-c=pet][data-i="1"]', 'whelp');
  await p.click('button[data-a=earn][data-id="feat:mosh"]');
  await p.screenshot({ path: shots + '/pets.png', fullPage: true });
  await p.click('button[data-a=rtab][data-v=Races]');
  await p.check('input[data-c=race][data-k="8"]');
  await p.click('button[data-a=rtab][data-v=Tabards]');
  ok((await p.textContent('main')).includes("can't be edited yet"), 'tabards read-only');

  // stats
  await p.click('button[data-a=page][data-v=life]');
  await p.fill('input[data-k=gold]', '999999'); await p.press('input[data-k=gold]', 'Tab');

  // game-running block
  fs.writeFileSync(flag, '1');
  await p.waitForFunction(() => document.querySelector('.game').textContent.includes('running'), null, { timeout: 7000 });
  ok(await p.locator('button[data-a=save]').isDisabled(), 'save blocked while game runs');
  await p.screenshot({ path: shots + '/running.png', fullPage: true });
  fs.unlinkSync(flag);
  await p.waitForFunction(() => document.querySelector('.game').textContent.includes('closed'), null, { timeout: 7000 });

  await p.click('button[data-a=save]');
  await p.waitForSelector('.banner.ok');
  ok((await p.textContent('.banner.ok')).includes('backed up'), 'save reports backup');
  const after = dec(save);
  const L0 = orig.split('\n'), L1 = after.split('\n');
  const changed = L1.filter((l) => !L0.includes(l)).map((l) => l.split('|').slice(0, 2).join('|'));
  console.log('changed lines:', changed.join('  '));
  const line = (k) => L1.find((l) => l.startsWith(k + '|')) || '';
  ok(line('hbeat|2').endsWith('naxx:4'), 'paladin naxx 4');
  ok(line('hbeat|11') === 'hbeat|11|elwynn:1', 'druid heroic line created');
  ok(line('beat|11') === 'beat|11|elwynn', 'druid zone auto-cleared');
  ok(line('marks|2') === 'marks|2|2000' && line('marks|11') === 'marks|11|7', 'marks written');
  ok(line('ach').includes('feat:stitches') && line('ach').includes('feat:mosh') && !line('ach').includes('feat:lairs'), 'achievements added and removed');
  const at = Object.fromEntries(line('achat').slice(6).split(',').map((x) => x.split('=')));
  ok(Math.abs(Number(at['feat:stitches']) - t0) < 120 && !at['feat:lairs'], 'new achievement dated at click time, removed one undated');
  ok(!line('perk').includes('skip:') && line('perk').includes('second:1'), 'perk removed, others kept');
  ok(line('pets').endsWith(',whelp') && line('pets').startsWith('pets|doomguard'), 'second companion slot written in order');
  ok(line('races') === 'races|2,5,6,7,8', 'race added');
  ok(line('stat').includes('gold=999999'), 'stat written');
  ok(L1.filter((l) => l.startsWith('stars|'))[0] === L0.filter((l) => l.startsWith('stars|'))[0], 'stars untouched');
  ok(L0.filter((l) => !['hbeat', 'beat', 'marks', 'ach', 'achat', 'perk', 'pets', 'races', 'stat', 'items'].includes(l.split('|')[0])).every((l) => L1.includes(l)), 'all other lines preserved');
  const items0 = new Set(L0.find((l) => l.startsWith('items|')).slice(6).split(',')), items1 = line('items').slice(6).split(',');
  ok(items1.length === items0.size + 1 && [...items0].every((x) => items1.includes(x)), 'exactly one item added');
  console.log('added item:', items1.filter((x) => !items0.has(x)).join());

  // conflict
  await p.click('button[data-a=page][data-v=life]');
  await p.fill('input[data-k=kills]', '1'); await p.press('input[data-k=kills]', 'Tab');
  const s = fs.readFileSync(save); fs.writeFileSync(save, s);
  fs.utimesSync(save, new Date(), new Date(Date.now() + 5000));
  await p.waitForSelector('.banner.err', { timeout: 7000 });
  ok((await p.textContent('.banner.err')).includes('changed your save'), 'conflict detected');
  ok(await p.locator('button[data-a=save]').isDisabled(), 'save blocked on conflict');
  await p.click('.banner.err button[data-a=reload]');
  await p.waitForFunction(() => !document.querySelector('.banner.err'));

  // advanced
  await p.click('button[data-a=page][data-v=raw]');
  const raw = await p.inputValue('#raw');
  await p.fill('#raw', raw.replace(/^glow\|8\|0$/m, 'glow|8|1')); await p.click('h2');
  ok((await p.textContent('.pending')).includes('Advanced: 1 other line'), 'raw edit shows as pending');
  await p.click('button[data-a=discard]');

  // restore
  await p.click('button[data-a=page][data-v=backups]');
  await p.waitForSelector('.row button');
  await p.screenshot({ path: shots + '/backups.png', fullPage: true });
  const idx = await p.$$eval('.row', (r) => r.findIndex((x) => x.textContent.includes('Editor backup')));
  const btn = p.locator('.row').nth(idx).locator('button');
  await btn.click();
  ok((await btn.textContent()).includes('Click again'), 'restore asks for confirmation');
  await btn.click();
  await p.waitForSelector('.banner.ok');
  ok(dec(save) === orig, 'restore returned the exact original save');

  // access
  const hdr = await p.evaluate(async () => (await fetch('/api/load')).status);
  ok(hdr === 403, 'API refuses calls without token');
  const trav = await p.evaluate(async (t) => (await fetch('/api/restore', { method: 'POST', headers: { 'X-Token': t }, body: JSON.stringify({ name: '../../etc/passwd', source: 'Game backup' }) })).status, new URL(url).searchParams.get('t'));
  ok(trav === 404, 'restore refuses paths outside the backup list');
  ok(errs.length === 0, 'no page errors ' + errs.join(' | '));

  await p.setViewportSize({ width: 390, height: 844 });
  await p.click('button[data-a=cls][data-v="2"]');
  const over = await p.evaluate(() => document.documentElement.scrollWidth - innerWidth);
  ok(over <= 0, 'no sideways scroll at phone width ' + over);
  await b.close();
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
})().catch((e) => { console.error(e); process.exit(1); });
