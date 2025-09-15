const { NanoStore } = require('../lib');

console.log('NanoStore Node.js Example\n');

// Configuration with dimensions
const config = {
    dimensions: [
        {
            name: "type",
            type: 0, // ENUMERATED
            values: ["user", "product", "order", "project", "task"],
            prefixes: { "user": "u", "product": "p", "order": "o" }
        },
        {
            name: "status",
            type: 0, // ENUMERATED  
            values: ["active", "archived", "deleted"],
            default_value: "active"
        }
    ]
};

// Create a new store (in-memory for this example)
const store = new NanoStore(':memory:', config);

try {
    // Example 1: Basic document operations
    console.log('1. Creating documents...');
    
    const userId = store.add('Alice Johnson - Admin User', {
        type: 'user',
        status: 'active'
    });
    console.log(`   Created user: ${userId}`);

    const productId = store.add('Laptop - High Performance', {
        type: 'product',
        status: 'active'
    });
    console.log(`   Created product: ${productId}`);

    // Example 2: Listing documents
    console.log('\n2. Listing all documents...');
    
    const allDocs = store.list();
    console.log(`   Found ${allDocs.length} documents:`);
    allDocs.forEach(doc => {
        console.log(`   - ${doc.uuid}: ${doc.title} (${doc.dimensions.type})`);
    });

    // Example 3: Filtering documents
    console.log('\n3. Filtering by dimension...');
    
    const users = store.list({ type: 'user' });
    console.log(`   Found ${users.length} users`);

    const activeItems = store.list({ status: 'active' });
    console.log(`   Found ${activeItems.length} active items`);

    // Example 4: Getting a specific document
    console.log('\n4. Getting a specific document...');
    
    const user = store.get(userId);
    console.log(`   User UUID: ${user.uuid}`);
    console.log(`   Title: ${user.title}`);
    console.log(`   Type: ${user.dimensions.type}`);
    console.log(`   Status: ${user.dimensions.status}`);

    // Example 5: Updating documents
    console.log('\n5. Updating document...');
    
    store.update(userId, {
        title: 'Alice Johnson - Super Admin',
        dimensions: {
            type: 'user',
            status: 'active'
        }
    });
    
    const updatedUser = store.get(userId);
    console.log(`   Updated title: ${updatedUser.title}`);

    // Example 6: UUID Resolution
    console.log('\n6. User-facing IDs...');
    
    // The C API returns UUIDs, but also provides user-facing IDs
    const resolvedUuid = store.resolveUUID('u-1'); // Try to resolve user-facing ID
    console.log(`   Resolved u-1 to: ${resolvedUuid}`);

    // Example 7: Creating more documents
    console.log('\n7. Creating multiple documents...');
    
    for (let i = 1; i <= 5; i++) {
        store.add(`Order #${1000 + i}`, {
            type: 'order',
            status: 'active'
        });
    }

    const orders = store.list({ dimensions: { type: 'order' } });
    console.log(`   Created ${orders.length} orders`);

    // Example 8: Deleting documents
    console.log('\n8. Deleting documents...');
    
    store.delete(productId);
    console.log(`   Deleted product: ${productId}`);
    
    try {
        store.get(productId);
    } catch (e) {
        console.log(`   Confirmed: product no longer exists`);
    }

    // Example 9: Document hierarchy (if supported by dimensions)
    console.log('\n9. Creating hierarchical data...');
    
    const projectId = store.add('Website Redesign Project', {
        type: 'project',
        status: 'active'
    });
    console.log(`   Created project: ${projectId}`);

    // Tasks would need a parent_id dimension configured
    const taskId = store.add('Design Homepage Mockup', {
        type: 'task',
        status: 'active'
    });
    console.log(`   Created task: ${taskId}`);

} catch (error) {
    console.error('Error:', error.message);
} finally {
    // Always close the store when done
    store.close();
    console.log('\nStore closed.');
}