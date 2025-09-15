#!/usr/bin/env python3
"""Test actual Python API"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from nanostore import Store, Config, DimensionConfig, DimensionType

# Create a store with a type dimension
config = Config(dimensions=[
    DimensionConfig(
        name="type",
        type=DimensionType.ENUMERATED,
        values=["task", "note", "project"]
    )
])

store = Store(":memory:", config)

# Add a document
doc_id = store.add("My first task", {"type": "task"})
print(f"Created document: {doc_id}")

# List documents
docs = store.list()
print(f"Found {len(docs)} documents")
for doc in docs:
    print(f"  - {doc.title} (type: {doc.dimensions.get('type')})")

store.close()
print("Test passed!")