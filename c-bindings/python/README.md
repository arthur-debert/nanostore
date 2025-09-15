# Nanostore Python Bindings

Python bindings for nanostore using ctypes FFI.

## Installation

```bash
pip install nanostore
```

The package will automatically download the appropriate shared library for your platform from GitHub releases.

## Quick Start

```python
from nanostore import Store, Config, DimensionConfig, DimensionType

# Configure dimensions
config = Config(dimensions=[
    DimensionConfig(
        name="status",
        type=DimensionType.ENUMERATED,
        values=["draft", "published"],
        default_value="draft"
    )
])

# Create store and use it
with Store("mydb.db", config) as store:
    # Add a document
    uuid = store.add("My Document", {"status": "draft"})
    
    # List documents
    for doc in store.list():
        print(f"{doc.user_facing_id}: {doc.title}")
```

## Development

To run tests:
```bash
python -m pytest tests/
```

To build from source:
```bash
# Build the shared library
cd .. && go build -buildmode=c-shared -o libnanostore.so main.go
```

## API Reference

See the [main documentation](../README.md) for detailed API reference.