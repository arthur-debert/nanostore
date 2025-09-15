"""
Nanostore Python bindings using FFI

A generic document store library with dynamic ID generation,
configurable dimensions, and hierarchical document structure.
"""

from .nanostore import (
    Store,
    Config,
    DimensionConfig,
    DimensionType,
    Document,
    ListOptions,
    UpdateRequest,
    NanoStoreError,
    todo_config,
    example_config,
)

__version__ = "0.3.0"
__all__ = [
    "Store",
    "Config", 
    "DimensionConfig",
    "DimensionType",
    "Document",
    "ListOptions",
    "UpdateRequest",
    "NanoStoreError",
    "todo_config",
    "example_config",
]