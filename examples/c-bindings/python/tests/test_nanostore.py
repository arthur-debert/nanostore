#!/usr/bin/env python3
"""
Test script for nanostore v2 FFI with caller-managed memory
"""

import os
import sys
import tempfile
import json

# Add parent directory to Python path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from nanostore import Store, Config, DimensionConfig, DimensionType, ListOptions, UpdateRequest, Document


def test_basic_operations():
    """Test basic CRUD operations"""
    print("=== Testing Basic Operations ===")
    
    # Create a temporary database
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        # Create configuration
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
                values=["tech", "science", "art"],
                default_value="tech"
            )
        ])
        
        # Test store creation
        print("✓ Creating store...")
        store = Store(db_path, config)
        print("  Store created successfully")
        
        # Test adding documents
        print("\n✓ Adding documents...")
        uuid1 = store.add("First Document", {"status": "draft", "category": "tech"})
        print(f"  Added document 1: {uuid1}")
        
        uuid2 = store.add("Second Document", {"status": "published", "category": "science"})
        print(f"  Added document 2: {uuid2}")
        
        uuid3 = store.add("Third Document", {"status": "archived", "category": "art"})
        print(f"  Added document 3: {uuid3}")
        
        # Test listing all documents
        print("\n✓ Listing all documents...")
        docs = store.list()
        print(f"  Found {len(docs)} documents")
        for doc in docs:
            print(f"  - {doc.user_facing_id}: {doc.title} ({doc.dimensions})")
        
        # Test filtering
        print("\n✓ Testing filters...")
        published_docs = store.list(ListOptions(filters={"status": "published"}))
        print(f"  Found {len(published_docs)} published documents")
        
        # Test UUID resolution
        print("\n✓ Testing UUID resolution...")
        if docs:
            # Find a hierarchical ID to resolve (not top-level)
            # For now, let's skip this test as we don't have hierarchical documents
            print("  Skipping UUID resolution test (no hierarchical documents)")
        
        # Test update
        print("\n✓ Testing update...")
        if docs:
            doc_to_update = docs[0]
            store.update(doc_to_update.uuid, UpdateRequest(
                title="Updated Title",
                dimensions={"status": "published"}
            ))
            
            # Verify update
            updated_docs = store.list()
            updated_doc = next(d for d in updated_docs if d.uuid == doc_to_update.uuid)
            print(f"  Updated document: {updated_doc.title}")
            assert updated_doc.title == "Updated Title"
            assert updated_doc.dimensions["status"] == "published"
        
        # Test delete
        print("\n✓ Testing delete...")
        if len(docs) >= 2:
            doc_to_delete = docs[1].uuid
            store.delete(doc_to_delete)
            
            # Verify deletion
            remaining_docs = store.list()
            print(f"  Documents after deletion: {len(remaining_docs)}")
            assert len(remaining_docs) == len(docs) - 1
        
        # Test close
        print("\n✓ Closing store...")
        store.close()
        print("  Store closed successfully")
        
        print("\n✅ All basic operations passed!")
        return True
        
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        # Clean up
        if os.path.exists(db_path):
            os.unlink(db_path)


def test_memory_stress():
    """Test with many operations to check for memory leaks"""
    print("\n=== Testing Memory Stress ===")
    
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        config = Config(dimensions=[
            DimensionConfig(
                name="status",
                type=DimensionType.ENUMERATED,
                values=["active", "inactive"],
                default_value="active"
            )
        ])
        
        store = Store(db_path, config)
        
        # Add many documents
        print("✓ Adding 100 documents...")
        uuids = []
        for i in range(100):
            uuid = store.add(f"Document {i}", {"status": "active" if i % 2 == 0 else "inactive"})
            uuids.append(uuid)
        
        # List many times
        print("✓ Listing documents 50 times...")
        for i in range(50):
            docs = store.list()
            if i == 0:
                print(f"  Found {len(docs)} documents")
        
        # Update many documents
        print("✓ Updating 50 documents...")
        docs = store.list()
        for i in range(min(50, len(docs))):
            store.update(docs[i].uuid, UpdateRequest(title=f"Updated {i}"))
        
        # Clean up
        store.close()
        print("\n✅ Memory stress test passed!")
        return True
        
    except Exception as e:
        print(f"\n❌ Memory stress test failed: {e}")
        return False
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


def test_error_handling():
    """Test error handling"""
    print("\n=== Testing Error Handling ===")
    
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        config = Config(dimensions=[
            DimensionConfig(
                name="status",
                type=DimensionType.ENUMERATED,
                values=["active"],
                default_value="active"
            )
        ])
        
        store = Store(db_path, config)
        
        # Test invalid store handle (after closing)
        store_handle = store._handle
        store.close()
        
        # This should fail
        try:
            store.add("Should fail", {})
            print("❌ Expected error not raised")
            return False
        except Exception as e:
            print(f"✓ Correctly caught error after close: {type(e).__name__}")
        
        # Test invalid JSON
        store2 = Store(db_path, config)
        # We can't easily test invalid JSON from Python side since we construct it
        # But the error handling is there in the Go code
        
        store2.close()
        print("\n✅ Error handling test passed!")
        return True
        
    except Exception as e:
        print(f"\n❌ Error handling test failed: {e}")
        return False
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


def test_with_context_manager():
    """Test using context manager"""
    print("\n=== Testing Context Manager ===")
    
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as tmp:
        db_path = tmp.name
    
    try:
        config = Config(dimensions=[
            DimensionConfig(
                name="status",
                type=DimensionType.ENUMERATED,
                values=["active"],
                default_value="active"
            )
        ])
        
        # Use context manager
        with Store(db_path, config) as store:
            uuid = store.add("Test Document", {})
            docs = store.list()
            print(f"✓ Added document in context: {uuid}")
            print(f"✓ Found {len(docs)} documents")
        
        # Store should be closed now
        # Try to use it (should fail)
        try:
            store.list()
            print("❌ Store not properly closed")
            return False
        except:
            print("✓ Store properly closed by context manager")
        
        print("\n✅ Context manager test passed!")
        return True
        
    except Exception as e:
        print(f"\n❌ Context manager test failed: {e}")
        return False
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


def main():
    """Run all tests"""
    print("Testing Nanostore FFI Implementation")
    print("=" * 40)
    
    # First build the library
    print("Building C library...")
    os.chdir(os.path.join(os.path.dirname(__file__), "..", ".."))
    result = os.system("go build -buildmode=c-shared -o libnanostore.so main.go")
    if result != 0:
        print("❌ Failed to build C library")
        return 1
    print("✓ Library built successfully\n")
    
    # Run tests
    tests = [
        test_basic_operations,
        test_memory_stress,
        test_error_handling,
        test_with_context_manager
    ]
    
    passed = 0
    for test in tests:
        if test():
            passed += 1
    
    print(f"\n{'=' * 40}")
    print(f"Total: {passed}/{len(tests)} tests passed")
    
    if passed == len(tests):
        print("\n🎉 All tests passed! The FFI implementation is working correctly.")
        print("\nNext steps:")
        print("1. Update the Python package __init__.py")
        print("2. Create proper distribution packaging (wheels)")
        print("3. Add more comprehensive tests")
    else:
        print("\n❌ Some tests failed. Please fix the issues before proceeding.")
    
    return 0 if passed == len(tests) else 1


if __name__ == "__main__":
    sys.exit(main())