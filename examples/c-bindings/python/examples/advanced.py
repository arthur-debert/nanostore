#!/usr/bin/env python3
"""
Example usage of nanostore Python FFI bindings
"""

import os
import sys
import tempfile

# Add the nanostore module to path
sys.path.insert(0, os.path.dirname(__file__))

from nanostore import Store, todo_config, ListOptions, UpdateRequest, DimensionType, DimensionConfig, Config


def basic_todo_example():
    """Basic todo example using FFI bindings"""
    print("=== Basic Todo Example ===")
    
    # Create temporary database
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name
    
    try:
        # Create store with todo configuration
        config = todo_config()
        
        with Store(db_path, config) as store:
            # Add some tasks
            print("Adding tasks...")
            task1_uuid = store.add("Buy groceries")
            task2_uuid = store.add("Write report", {"status": "pending"})
            task3_uuid = store.add("Review code", {"parent_uuid": task1_uuid})
            
            print(f"Created tasks:")
            print(f"  - Task 1: {task1_uuid}")
            print(f"  - Task 2: {task2_uuid}")
            print(f"  - Task 3: {task3_uuid}")
            
            # List all tasks
            print("\nAll tasks:")
            all_tasks = store.list()
            for doc in all_tasks:
                print(f"  {doc.user_facing_id}: {doc.title}")
            
            # Complete a task
            print(f"\nCompleting task: {all_tasks[0].user_facing_id}")
            store.update(all_tasks[0].user_facing_id, UpdateRequest(
                dimensions={"status": "completed"}
            ))
            
            # List again to see updated IDs
            print("\nAfter completion:")
            updated_tasks = store.list()
            for doc in updated_tasks:
                status = doc.dimensions.get("status", "pending")
                print(f"  {doc.user_facing_id}: {doc.title} [{status}]")
            
            # Test ID resolution
            print(f"\nResolving user-facing ID '{updated_tasks[0].user_facing_id}' to UUID:")
            resolved_uuid = store.resolve_uuid(updated_tasks[0].user_facing_id)
            print(f"  UUID: {resolved_uuid}")
            
            # Filter by status
            print("\nCompleted tasks only:")
            completed = store.list(ListOptions(filters={"status": "completed"}))
            for doc in completed:
                print(f"  {doc.user_facing_id}: {doc.title}")
                
    finally:
        # Cleanup
        if os.path.exists(db_path):
            os.unlink(db_path)


def custom_config_example():
    """Example with custom project management configuration"""
    print("\n=== Custom Project Management Example ===")
    
    # Create custom configuration
    config = Config(dimensions=[
        DimensionConfig(
            name="priority",
            type=DimensionType.ENUMERATED,
            values=["low", "normal", "high", "urgent"],
            prefixes={"high": "h", "urgent": "u"},
            default_value="normal"
        ),
        DimensionConfig(
            name="status",
            type=DimensionType.ENUMERATED,
            values=["backlog", "todo", "in_progress", "done"],
            prefixes={"in_progress": "p", "done": "d"},
            default_value="backlog"
        ),
        DimensionConfig(
            name="parent",
            type=DimensionType.HIERARCHICAL,
            ref_field="parent_task_id"
        )
    ])
    
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name
    
    try:
        with Store(db_path, config) as store:
            # Add project and tasks
            project_uuid = store.add("Q1 Product Launch")
            task1_uuid = store.add("Design mockups", {
                "parent_uuid": project_uuid,
                "priority": "high"
            })
            task2_uuid = store.add("Implement backend", {
                "parent_uuid": project_uuid,
                "priority": "urgent",
                "status": "in_progress"
            })
            
            print("Project structure:")
            all_docs = store.list()
            for doc in all_docs:
                indent = "  " if doc.dimensions.get("parent_uuid") else ""
                priority = doc.dimensions.get("priority", "normal")
                status = doc.dimensions.get("status", "backlog")
                print(f"  {indent}{doc.user_facing_id}: {doc.title} [p:{priority}, s:{status}]")
            
            # Update task status
            print(f"\nMarking task {task1_uuid} as done...")
            store.update(task1_uuid, UpdateRequest(
                dimensions={"status": "done"}
            ))
            
            print("\nAfter status update:")
            updated_docs = store.list()
            for doc in updated_docs:
                indent = "  " if doc.dimensions.get("parent_uuid") else ""
                priority = doc.dimensions.get("priority", "normal")
                status = doc.dimensions.get("status", "backlog")
                print(f"  {indent}{doc.user_facing_id}: {doc.title} [p:{priority}, s:{status}]")
                
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


def performance_test():
    """Basic performance test"""
    print("\n=== Performance Test ===")
    
    import time
    
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name
    
    try:
        config = todo_config()
        
        with Store(db_path, config) as store:
            # Add many documents
            print("Adding 1000 documents...")
            start_time = time.time()
            
            for i in range(1000):
                store.add(f"Task {i+1}")
            
            add_time = time.time() - start_time
            print(f"  Added 1000 documents in {add_time:.2f} seconds ({1000/add_time:.0f} docs/sec)")
            
            # List all documents
            start_time = time.time()
            docs = store.list()
            list_time = time.time() - start_time
            
            print(f"  Listed {len(docs)} documents in {list_time:.3f} seconds")
            
            # Test ID resolution
            start_time = time.time()
            for i in range(100):
                uuid = store.resolve_uuid(str(i+1))
            resolve_time = time.time() - start_time
            
            print(f"  Resolved 100 IDs in {resolve_time:.3f} seconds ({100/resolve_time:.0f} resolves/sec)")
            
    finally:
        if os.path.exists(db_path):
            os.unlink(db_path)


if __name__ == "__main__":
    # Check if library exists
    library_path = os.path.join(os.path.dirname(__file__), "..", "libnanostore.so")
    if not os.path.exists(library_path):
        print("Error: libnanostore.so not found!")
        print("Please run the build script first:")
        print("  cd examples/c-bindings && ./build.sh")
        sys.exit(1)
    
    basic_todo_example()
    custom_config_example()
    performance_test()
    
    print("\n=== All examples completed successfully! ===")