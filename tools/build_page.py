#!/usr/bin/env python3
"""Build index.html from ui.template.html and data.js."""
template = open('ui.template.html', encoding='utf-8').read()
open('index.html', 'w', encoding='utf-8').write(template.replace('/*DATA*/', open('data.js', encoding='utf-8').read().rstrip('\n')))
