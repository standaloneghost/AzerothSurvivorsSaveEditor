#!/usr/bin/env python3
"""Print a save as plain text. Usage: python3 tools/show_save.py characters.txt"""
import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from savelib import unseal
print(unseal(open(sys.argv[1]).read()).decode(), end='')
