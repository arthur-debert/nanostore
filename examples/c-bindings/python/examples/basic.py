#!/usr/bin/env python3
"""
Basic example of using nanostore Python bindings
"""

import sys
import os
import tempfile

# Add the nanostore module to path (for development)
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from nanostore import Store, Config, DimensionConfig, DimensionType


def main():
    """Basic usage example"""
    # Create a temporary database
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        # Configure dimensions
        config = Config(dimensions=[
            DimensionConfig(
                name="status",
                type=DimensionType.ENUMERATED,
                values=["draft", "published", "archived"],
                prefixes={"published": "p", "archived": "a"},
                default_value="draft"
            )
        ])
        
        # Create store and add documents
        with Store(db_path, config) as store:
            # Add documents
            doc1_uuid = store.add("My First Document", {"status": "draft"})
            doc2_uuid = store.add("Published Article", {"status": "published"})
            
            print(f"Created documents:")
            print(f"  Doc 1: {doc1_uuid}")
            print(f"  Doc 2: {doc2_uuid}")
            
            # List all documents
            print("\nAll documents:")
            for doc in store.list():
                print(f"  [{doc.user_facing_id}] {doc.title} - {doc.dimensions['status']}")
            
            # Filter documents
            print("\nPublished documents:")
            from nanostore import ListOptions
            published = store.list(ListOptions(filters={"status": "published"}))
            for doc in published:
                print(f"  [{doc.user_facing_id}] {doc.title}")
        
        print("\n✅ Example completed successfully!")
        
    finally:
        # Clean up
        if os.path.exists(db_path):
            os.unlink(db_path)


if __name__ == "__main__":
    main()