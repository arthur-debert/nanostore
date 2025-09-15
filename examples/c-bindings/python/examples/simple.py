#!/usr/bin/env python3
"""
Example usage of nanostore Python bindings
"""

import sys
import os
import tempfile

# Add python package to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'python'))

from nanostore import Store, Config, DimensionConfig, DimensionType, ListOptions, UpdateRequest


def main():
    """Demonstrate nanostore usage"""
    
    # Create a temporary database
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        # Configure dimensions for a blog-like system
        config = Config(dimensions=[
            DimensionConfig(
                name="status",
                type=DimensionType.ENUMERATED,
                values=["draft", "published", "archived"],
                prefixes={"published": "p", "archived": "a"},
                default_value="draft"
            ),
            DimensionConfig(
                name="category",
                type=DimensionType.ENUMERATED,
                values=["tech", "science", "art", "general"],
                default_value="general"
            )
        ])
        
        # Create store
        print("Creating nanostore...")
        with Store(db_path, config) as store:
            
            # Add some documents
            print("\nAdding documents...")
            doc1 = store.add("Introduction to Nanostore", {
                "status": "published",
                "category": "tech"
            })
            print(f"  Added: {doc1}")
            
            doc2 = store.add("Quantum Computing Basics", {
                "status": "draft",
                "category": "science"
            })
            print(f"  Added: {doc2}")
            
            doc3 = store.add("Digital Art Techniques", {
                "status": "published",
                "category": "art"
            })
            print(f"  Added: {doc3}")
            
            # List all documents
            print("\nAll documents:")
            for doc in store.list():
                print(f"  [{doc.user_facing_id}] {doc.title}")
                print(f"    Status: {doc.dimensions['status']}, Category: {doc.dimensions['category']}")
            
            # Filter by status
            print("\nPublished documents only:")
            published = store.list(ListOptions(filters={"status": "published"}))
            for doc in published:
                print(f"  [{doc.user_facing_id}] {doc.title}")
            
            # Update a document
            print("\nUpdating first document...")
            store.update(doc1, UpdateRequest(
                title="Getting Started with Nanostore",
                dimensions={"category": "tech"}
            ))
            
            # Verify update
            updated_docs = store.list()
            updated = next(d for d in updated_docs if d.uuid == doc1)
            print(f"  Updated title: {updated.title}")
            
            # Delete a document
            print("\nDeleting draft document...")
            store.delete(doc2)
            
            # Final list
            print("\nRemaining documents:")
            for doc in store.list():
                print(f"  [{doc.user_facing_id}] {doc.title}")
        
        print("\n✅ Example completed successfully!")
        
    finally:
        # Clean up
        if os.path.exists(db_path):
            os.unlink(db_path)


if __name__ == "__main__":
    # First ensure the library is built
    print("Building nanostore library...")
    os.chdir(os.path.dirname(__file__))
    result = os.system("go build -buildmode=c-shared -o libnanostore.so main.go")
    if result != 0:
        print("❌ Failed to build library")
        sys.exit(1)
    print("✓ Library built\n")
    
    main()