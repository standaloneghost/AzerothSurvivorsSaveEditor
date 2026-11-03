#!/usr/bin/env python3
"""Write key.bin from your own copy of AzerothSurvivors.exe.

Usage: python3 tools/extract_key.py path/to/AzerothSurvivors.exe

The key is built into the game, so it is never stored in this repo. The
addresses below match the build this editor was written against. If an update
moves them, the check below fails and they need finding again.
"""
import sys

RDATA_FILE_OFFSET = 103665152
RDATA_ADDRESS = 0x1462de000
PAD_ADDRESSES = (0x1471229a0, 0x1471229b0, 0x1471229c0, 0x146322e10)

data = open(sys.argv[1], 'rb').read()
offset = lambda a: a - RDATA_ADDRESS + RDATA_FILE_OFFSET
pad = b''.join(data[offset(a):offset(a) + 16] for a in PAD_ADDRESSES)
key = bytes(b ^ 0x36 for b in pad).rstrip(b'\0')
if len(key) != 42:
    sys.exit('Unexpected key size. A game update may have moved the key.')
open('key.bin', 'wb').write(key)
print('Wrote key.bin. Keep it out of git.')
