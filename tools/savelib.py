"""Read and write the save format. Needs key.bin in the repo root.

A save is "RSE1:" + base64(nonce[16] + ciphertext + tag[20]). The keystream is
HMAC-SHA1(key, "ks" + nonce + block_le64) and the tag is
HMAC-SHA1(key, "tag" + nonce + ciphertext).
"""
import base64, hashlib, hmac, os, struct

key = open(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'key.bin'), 'rb').read()


def xor(nonce, data):
    out = bytearray(data)
    for i in range(0, len(out), 20):
        ks = hmac.new(key, b'ks' + nonce + struct.pack('<Q', i // 20), hashlib.sha1).digest()
        for j in range(min(20, len(out) - i)):
            out[i + j] ^= ks[j]
    return bytes(out)


def unseal(text):
    raw = base64.b64decode(text.strip()[5:])
    nonce, ct, tag = raw[:16], raw[16:-20], raw[-20:]
    if not hmac.compare_digest(hmac.new(key, b'tag' + nonce + ct, hashlib.sha1).digest(), tag):
        raise ValueError('bad tag')
    return xor(nonce, ct)


def seal(plain):
    nonce = os.urandom(16)
    ct = xor(nonce, plain)
    tag = hmac.new(key, b'tag' + nonce + ct, hashlib.sha1).digest()
    return 'RSE1:' + base64.b64encode(nonce + ct + tag).decode()
