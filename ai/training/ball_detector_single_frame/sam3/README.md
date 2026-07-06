# TENSA SAM3 labeling

```sh
uv python install 3.12
uv python pin 3.12
uv venv
source .venv/bin/activate
cd sam3/
uv pip install -e .
cd ..
uv pip install einops decord pycocotools psutil
```

Download the data to ./data.
```sh
....
```

Log in to HF
```sh
hf auth login
```

```sh
python eval.py --data-dir data --prompt 'tennis ball' --prompt 'small tennis ball' --device cuda
```

Optional overlays (off by default):
```sh
python eval.py --data-dir data --prompt 'tennis ball' --device cuda --overlays
```

Outputs are written to `output/predictions.jsonl` by default (override with `--output`).
