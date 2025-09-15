"""
Python wrapper for nanostore using ctypes/FFI with caller-managed memory
"""
import json
import ctypes
from ctypes import POINTER, c_char_p, c_int, c_char, create_string_buffer
from typing import Dict, List, Optional, Union, Any
from dataclasses import dataclass
from datetime import datetime
from enum import IntEnum
import os


# Default buffer size for most operations
DEFAULT_BUFFER_SIZE = 4096
# Larger buffer for list operations which might return many documents
LIST_BUFFER_SIZE = 65536


class DimensionType(IntEnum):
    ENUMERATED = 0
    HIERARCHICAL = 1


@dataclass
class DimensionConfig:
    name: str
    type: DimensionType
    values: Optional[List[str]] = None
    prefixes: Optional[Dict[str, str]] = None
    ref_field: Optional[str] = None
    default_value: Optional[str] = None

    def to_dict(self):
        result = {
            "name": self.name,
            "type": int(self.type)
        }
        if self.values:
            result["values"] = self.values
        if self.prefixes:
            result["prefixes"] = self.prefixes
        if self.ref_field:
            result["ref_field"] = self.ref_field
        if self.default_value:
            result["default_value"] = self.default_value
        return result


@dataclass
class Config:
    dimensions: List[DimensionConfig]

    def to_json(self) -> str:
        return json.dumps({
            "dimensions": [dim.to_dict() for dim in self.dimensions]
        })


@dataclass
class Document:
    uuid: str
    user_facing_id: str
    title: str
    body: str
    dimensions: Dict[str, Any]
    created_at: datetime
    updated_at: datetime

    @classmethod
    def from_dict(cls, data: Dict) -> 'Document':
        return cls(
            uuid=data["uuid"],
            user_facing_id=data["user_facing_id"],
            title=data["title"],
            body=data["body"],
            dimensions=data["dimensions"],
            created_at=datetime.fromtimestamp(data["created_at"]),
            updated_at=datetime.fromtimestamp(data["updated_at"])
        )


@dataclass
class ListOptions:
    filters: Optional[Dict[str, Any]] = None
    filter_by_search: Optional[str] = None

    def to_json(self) -> str:
        result = {}
        if self.filters:
            result.update(self.filters)
        if self.filter_by_search:
            result["search"] = self.filter_by_search
        return json.dumps(result) if result else ""


@dataclass
class UpdateRequest:
    title: Optional[str] = None
    body: Optional[str] = None
    dimensions: Optional[Dict[str, str]] = None

    def to_json(self) -> str:
        result = {}
        if self.title is not None:
            result["title"] = self.title
        if self.body is not None:
            result["body"] = self.body
        if self.dimensions:
            result["dimensions"] = self.dimensions
        return json.dumps(result)


class NanoStoreError(Exception):
    """Exception raised for nanostore errors"""
    pass


class Store:
    """Python wrapper for nanostore C library with caller-managed memory"""
    
    def __init__(self, db_path: str, config: Config, library_path: str = None):
        # Load the C library
        if library_path is None:
            # Try to find library in same directory as this module
            current_dir = os.path.dirname(__file__)
            
            # Try different library names based on platform
            import platform
            system = platform.system()
            
            if system == "Darwin":
                lib_names = ["libnanostore.dylib", "libnanostore.so"]
            elif system == "Windows":
                lib_names = ["nanostore.dll", "libnanostore.dll"]
            else:  # Linux and others
                lib_names = ["libnanostore.so"]
            
            # First try in the package directory
            for lib_name in lib_names:
                lib_path = os.path.join(current_dir, lib_name)
                if os.path.exists(lib_path):
                    library_path = lib_path
                    break
            
            # If not found, try parent directory (for development)
            if library_path is None:
                parent_dir = os.path.join(current_dir, "..", "..")
                for lib_name in lib_names:
                    lib_path = os.path.join(parent_dir, lib_name)
                    if os.path.exists(lib_path):
                        library_path = lib_path
                        break
            
            if library_path is None:
                raise RuntimeError(
                    "Could not find nanostore library. "
                    "Please ensure it's installed or provide explicit path."
                )
        
        self._lib = ctypes.CDLL(library_path)
        
        # Define function signatures
        self._setup_function_signatures()
        
        # Create store
        config_json = config.to_json().encode('utf-8')
        db_path_bytes = db_path.encode('utf-8')
        
        buffer = create_string_buffer(DEFAULT_BUFFER_SIZE)
        result_len = self._lib.nanostore_new(
            db_path_bytes, config_json, buffer, DEFAULT_BUFFER_SIZE
        )
        
        if result_len < 0:
            raise NanoStoreError("Response buffer too small")
        
        result_str = buffer.value[:result_len].decode('utf-8')
        result = json.loads(result_str)
        
        if "error" in result:
            raise NanoStoreError(result["error"])
        
        self._handle = result["handle"].encode('utf-8')

    def _setup_function_signatures(self):
        """Setup ctypes function signatures for type safety"""
        # All functions return int (bytes written) and take output buffer
        
        # nanostore_new
        self._lib.nanostore_new.argtypes = [c_char_p, c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_new.restype = c_int
        
        # nanostore_add
        self._lib.nanostore_add.argtypes = [c_char_p, c_char_p, c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_add.restype = c_int
        
        # nanostore_list
        self._lib.nanostore_list.argtypes = [c_char_p, c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_list.restype = c_int
        
        # nanostore_update
        self._lib.nanostore_update.argtypes = [c_char_p, c_char_p, c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_update.restype = c_int
        
        # nanostore_delete
        self._lib.nanostore_delete.argtypes = [c_char_p, c_char_p, c_int, POINTER(c_char), c_int]
        self._lib.nanostore_delete.restype = c_int
        
        # nanostore_resolve_uuid
        self._lib.nanostore_resolve_uuid.argtypes = [c_char_p, c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_resolve_uuid.restype = c_int
        
        # nanostore_close
        self._lib.nanostore_close.argtypes = [c_char_p, POINTER(c_char), c_int]
        self._lib.nanostore_close.restype = c_int

    def _call_and_parse(self, func, *args, buffer_size=DEFAULT_BUFFER_SIZE) -> Union[Dict, List]:
        """Helper to call C function with pre-allocated buffer and parse JSON result"""
        buffer = create_string_buffer(buffer_size)
        
        # Add buffer and buffer size to arguments
        args_with_buffer = args + (buffer, buffer_size)
        result_len = func(*args_with_buffer)
        
        if result_len < 0:
            # Buffer too small, try with larger buffer
            larger_size = buffer_size * 4
            buffer = create_string_buffer(larger_size)
            args_with_buffer = args + (buffer, larger_size)
            result_len = func(*args_with_buffer)
            
            if result_len < 0:
                raise NanoStoreError("Response too large for buffer")
        
        result_str = buffer.value[:result_len].decode('utf-8')
        result = json.loads(result_str)
        
        if isinstance(result, dict) and "error" in result:
            raise NanoStoreError(result["error"])
        
        return result

    def add(self, title: str, dimensions: Optional[Dict[str, Any]] = None) -> str:
        """Add a new document"""
        title_bytes = title.encode('utf-8')
        dimensions_json = json.dumps(dimensions or {}).encode('utf-8')
        
        result = self._call_and_parse(
            self._lib.nanostore_add,
            self._handle, title_bytes, dimensions_json
        )
        return result["uuid"]

    def list(self, options: Optional[ListOptions] = None) -> List[Document]:
        """List documents with optional filtering"""
        options = options or ListOptions()
        filters_json = options.to_json().encode('utf-8')
        
        result = self._call_and_parse(
            self._lib.nanostore_list,
            self._handle, filters_json,
            buffer_size=LIST_BUFFER_SIZE
        )
        
        return [Document.from_dict(doc_data) for doc_data in result]

    def update(self, id: str, updates: UpdateRequest) -> None:
        """Update an existing document"""
        id_bytes = id.encode('utf-8')
        updates_json = updates.to_json().encode('utf-8')
        
        self._call_and_parse(
            self._lib.nanostore_update,
            self._handle, id_bytes, updates_json
        )

    def delete(self, id: str, cascade: bool = False) -> None:
        """Delete a document"""
        id_bytes = id.encode('utf-8')
        cascade_int = 1 if cascade else 0
        
        self._call_and_parse(
            self._lib.nanostore_delete,
            self._handle, id_bytes, cascade_int
        )

    def resolve_uuid(self, user_facing_id: str) -> str:
        """Convert user-facing ID to UUID"""
        id_bytes = user_facing_id.encode('utf-8')
        
        result = self._call_and_parse(
            self._lib.nanostore_resolve_uuid,
            self._handle, id_bytes
        )
        return result["uuid"]

    def close(self) -> None:
        """Close the store"""
        if hasattr(self, '_handle'):
            self._call_and_parse(self._lib.nanostore_close, self._handle)
            del self._handle

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


# Convenience functions for common configurations
def todo_config() -> Config:
    """Create a todo-app compatible configuration"""
    return Config(dimensions=[
        DimensionConfig(
            name="status",
            type=DimensionType.ENUMERATED,
            values=["pending", "completed"],
            prefixes={"completed": "c"},
            default_value="pending"
        ),
        DimensionConfig(
            name="parent",
            type=DimensionType.HIERARCHICAL,
            ref_field="parent_uuid"
        )
    ])


def example_config() -> Config:
    """Create an example configuration"""
    return Config(dimensions=[
        DimensionConfig(
            name="category",
            type=DimensionType.ENUMERATED,
            values=["default", "archived"],
            prefixes={"archived": "a"},
            default_value="default"
        ),
        DimensionConfig(
            name="parent",
            type=DimensionType.HIERARCHICAL,
            ref_field="parent_uuid"
        )
    ])