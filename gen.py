import re,sys
sys.path.insert(0,'../dec')
from lib import key
open('key.bin','wb').write(key)
src=open('/tmp/claude-0/-home-claude/5331c26f-c9b7-5044-b81c-40cc47f7eb70/scratchpad/artifact-files/4a881b74-1c88-4d7f-a964-0b8a9c0d72b3/project/Main.dc.html').read()
b=src.index("const CLASSES = ["); e=src.index("const initial = ()")
d=src[b:e]
for name in ('UNLOCKED','WORN_RARITY','STARS'):
    d=re.sub(r"const %s = .*?;\n"%name,'',d,count=1,flags=re.S)
d=re.sub(r"const TREES = .*?const initialTalents = .*?\n\}\);\n",'',d,count=1,flags=re.S)
for name in ('TABARDS','TABARD_BG'):
    d=re.sub(r"const %s = .*?;\n"%name,'',d,count=1,flags=re.S)
assert all(x in d for x in ('const ACH_NAMES','const PETS','const SETS','const COL_ORDER','const DUAL_WIELD','const STAT_LABELS','const RACES'))
assert 'UNLOCKED' not in d and 'STARS' not in d and 'TDATA' not in d and 'initialTalents' not in d
open('index.html','w').write(open('ui.template.html').read().replace('/*DATA*/',d))
