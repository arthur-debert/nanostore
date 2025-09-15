#!/usr/bin/env python3
"""
Simple example usage of nanostore Python bindings
"""

import sys
import os
import json

# Add parent directory to path to import nanostore
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from nanostore import NanoStore

def main():
    """Demonstrate basic nanostore usage"""
    
    # Create an in-memory store
    print("Creating in-memory nanostore...")
    store = NanoStore(":memory:")
    
    try:
        # Example 1: Add documents
        print("\n1. Adding documents...")
        
        user_id = store.add("users", "user", json.dumps({
            "name": "Alice Johnson",
            "email": "alice@example.com",
            "role": "admin"
        }))
        print(f"   Created user: {user_id}")
        
        product_id = store.add("products", "product", json.dumps({
            "name": "Laptop",
            "price": 999.99,
            "stock": 50
        }))
        print(f"   Created product: {product_id}")
        
        # Example 2: Retrieve documents
        print("\n2. Retrieving documents...")
        
        user = store.get(user_id)
        print(f"   User ID: {user['id']}")
        print(f"   Content: {user['content']}")
        
        # Example 3: Update document
        print("\n3. Updating document...")
        
        store.update(user_id, json.dumps({
            "name": "Alice Johnson",
            "email": "alice.johnson@example.com",
            "role": "superadmin"
        }))
        
        updated_user = store.get(user_id)
        content = json.loads(updated_user['content'])
        print(f"   Updated role: {content['role']}")
        
        # Example 4: Hierarchical documents
        print("\n4. Creating hierarchical documents...")
        
        project_id = store.add("projects", "project", json.dumps({
            "name": "Website Redesign",
            "status": "active"
        }))
        print(f"   Created project: {project_id}")
        
        task_id = store.add("tasks", "task", json.dumps({
            "title": "Design mockups",
            "assignee": "Alice"
        }), parent=project_id)
        print(f"   Created task: {task_id}")
        
        # Example 5: List documents
        print("\n5. Listing documents...")
        
        users = store.list("users")
        print(f"   Found {len(users)} users")
        
        tasks = store.list("tasks")
        print(f"   Found {len(tasks)} tasks")
        
        # Example 6: UUID Resolution
        print("\n6. UUID Resolution...")
        
        project = store.get(project_id)
        print(f"   Project UUID: {project['uuid']}")
        
        resolved_id = store.resolve(project['uuid'])
        print(f"   Resolved ID: {resolved_id}")
        print(f"   IDs match: {resolved_id == project_id}")
        
        # Example 7: Delete document
        print("\n7. Deleting document...")
        
        store.delete(product_id)
        print(f"   Deleted product: {product_id}")
        
        try:
            store.get(product_id)
        except Exception as e:
            print(f"   Confirmed: product no longer exists")
        
    finally:
        store.close()
        print("\nStore closed.")

if __name__ == "__main__":
    main()