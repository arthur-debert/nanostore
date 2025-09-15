#!/usr/bin/env python3
"""Test basic functionality of Python bindings"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from nanostore import Store, Config

# Create an in-memory store with a simple dimension
from nanostore import DimensionConfig, DimensionType
config = Config(dimensions=[
    DimensionConfig(
        name="type",
        type=DimensionType.ENUMERATED,
        values=["doc", "test"]
    )
])
store = Store(":memory:", config)

# Add a document
doc_id = store.add("test", "doc", '{"name": "test"}')
print(f"Created document: {doc_id}")

# Get the document
doc = store.get(doc_id)
print(f"Retrieved: {doc}")

# Close
store.close()
print("Test passed!")