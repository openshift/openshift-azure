from pathlib import Path
# from os.path import dirname
import json

with open('vault-secrets.json') as json_file:
    data = json.load(json_file)
    for key, value in data.items():
        filepath = Path('secrets/%s' % key)
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, 'w') as writer:
            writer.write(value)
