# Contributing

This repository is primarily a portfolio archive of the Tensa startup's engineering work. It is open for reading, learning, and reuse, but it is not maintained as an active product.

## Pull Requests

Small corrections are welcome:

- documentation fixes,
- build or test fixes,
- removal of accidentally committed private or generated artifacts,
- improvements that make the repo easier to understand.

Large product changes, new hardware integrations, or dataset/model additions are outside the intended scope of this archive.

## Local Checks

```bash
make test-go
make test-python
```

The AI pipeline requires external model weights and ZED SVO recordings that are not included in this repo. The Go robot stack builds without ZED hardware by using stub implementations unless the `zed_sdk` build tag is enabled.
