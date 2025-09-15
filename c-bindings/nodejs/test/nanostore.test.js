const { expect } = require('chai');
const { NanoStore } = require('../lib');
const fs = require('fs');
const path = require('path');

describe('NanoStore Node.js Bindings', function() {
    let store;
    let config;

    beforeEach(function() {
        config = {
            dimensions: [
                {
                    name: "type",
                    type: 0, // ENUMERATED
                    values: ["user", "product", "task", "project"],
                    prefixes: { "user": "u", "product": "p" }
                },
                {
                    name: "status",
                    type: 0, // ENUMERATED
                    values: ["active", "archived", "deleted"],
                    default_value: "active"
                }
            ]
        };
        store = new NanoStore(':memory:', config);
    });

    afterEach(function() {
        if (store) {
            store.close();
        }
    });

    describe('Basic Operations', function() {
        it('should create and retrieve a document', function() {
            const id = store.add('John Doe - Test User', {
                type: 'user',
                status: 'active'
            });

            expect(id).to.be.a('string');
            expect(id).to.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/);

            const docs = store.list();
            const doc = docs.find(d => d.uuid === id);
            expect(doc).to.exist;
            expect(doc).to.have.property('uuid', id);
            expect(doc).to.have.property('title', 'John Doe - Test User');
            expect(doc.dimensions).to.deep.equal({
                type: 'user',
                status: 'active'
            });
        });

        it('should update a document', function() {
            const id = store.add('Jane Doe - Test User', {
                type: 'user'
            });

            store.update(id, {
                title: 'Jane Smith - Updated User',
                dimensions: {
                    type: 'user',
                    status: 'archived'
                }
            });

            const docs = store.list();
            const doc = docs.find(d => d.uuid === id);
            expect(doc).to.exist;
            expect(doc.title).to.equal('Jane Smith - Updated User');
            expect(doc.dimensions.status).to.equal('archived');
        });

        it('should delete a document', function() {
            const id = store.add('Delete Me', {
                type: 'user'
            });

            store.delete(id);

            const docs = store.list();
            const doc = docs.find(d => d.uuid === id);
            expect(doc).to.not.exist;
        });

        it('should list documents', function() {
            // Add some documents
            const ids = [];
            for (let i = 1; i <= 3; i++) {
                ids.push(store.add(`Product ${i}`, {
                    type: 'product',
                    status: 'active'
                }));
            }

            const docs = store.list();
            expect(docs).to.have.lengthOf(3);
            
            const uuids = docs.map(d => d.uuid);
            expect(uuids).to.include.members(ids);
        });

        it('should filter documents by dimensions', function() {
            // Clear any existing documents first
            const initialDocs = store.list();
            initialDocs.forEach(doc => store.delete(doc.uuid));
            
            // Add mixed documents
            store.add('User 1', { type: 'user', status: 'active' });
            store.add('User 2', { type: 'user', status: 'archived' });
            store.add('Product 1', { type: 'product', status: 'active' });

            // Filter by type - C API expects flat filters
            const users = store.list({ type: 'user' });
            expect(users).to.have.lengthOf(2);
            expect(users.every(d => d.dimensions.type === 'user')).to.be.true;

            // Filter by status
            const activeItems = store.list({ status: 'active' });
            expect(activeItems).to.have.lengthOf(2);
            expect(activeItems.every(d => d.dimensions.status === 'active')).to.be.true;

            // Filter by multiple dimensions
            const activeUsers = store.list({ type: 'user', status: 'active' });
            expect(activeUsers).to.have.lengthOf(1);
        });
    });

    describe('UUID Resolution', function() {
        it('should resolve user-facing ID to UUID', function() {
            const id = store.add('Test User', {
                type: 'user'
            });

            const docs = store.list();
            const doc = docs.find(d => d.uuid === id);
            expect(doc).to.exist;
            const userFacingId = doc.user_facing_id;

            // Should be able to resolve the user-facing ID
            const resolvedUuid = store.resolveUUID(userFacingId);
            expect(resolvedUuid).to.equal(id);
        });
    });

    describe('Error Handling', function() {

        it('should throw error when updating non-existent document', function() {
            const fakeUuid = '00000000-0000-0000-0000-000000000000';
            expect(() => store.update(fakeUuid, { title: 'New Title' }))
                .to.throw(/not found/);
        });

        it('should throw error when deleting non-existent document', function() {
            const fakeUuid = '00000000-0000-0000-0000-000000000000';
            expect(() => store.delete(fakeUuid))
                .to.throw(/not found/);
        });
    });

    describe('File-based Storage', function() {
        it('should persist data to file', function() {
            const dbPath = path.join(__dirname, 'test.db');
            
            // Clean up any existing test db
            if (fs.existsSync(dbPath)) {
                fs.unlinkSync(dbPath);
            }

            const fileStore = new NanoStore(dbPath, config);
            
            const id = fileStore.add('Persistent Data', {
                type: 'user',
                status: 'active'
            });

            fileStore.close();

            // Reopen the store
            const newStore = new NanoStore(dbPath, config);
            const docs = newStore.list();
            const doc = docs.find(d => d.uuid === id);
            expect(doc).to.exist;
            
            expect(doc.title).to.equal('Persistent Data');
            expect(doc.dimensions.type).to.equal('user');
            
            newStore.close();
            
            // Clean up
            fs.unlinkSync(dbPath);
        });
    });

    describe('Memory Stress Tests', function() {
        this.timeout(10000); // Increase timeout for stress tests

        it('should handle many documents without memory issues', function() {
            const testStore = new NanoStore(':memory:', config);
            const docCount = 100;
            const uuids = [];

            // Add many documents
            for (let i = 0; i < docCount; i++) {
                const uuid = testStore.add(`Document ${i}`, {
                    type: i % 2 === 0 ? 'user' : 'product',
                    status: i % 3 === 0 ? 'archived' : 'active'
                });
                uuids.push(uuid);
            }

            // List documents multiple times
            for (let i = 0; i < 50; i++) {
                const docs = testStore.list();
                if (i === 0) {
                    expect(docs).to.have.lengthOf(docCount);
                }
            }

            // Update many documents
            const docs = testStore.list();
            for (let i = 0; i < Math.min(50, docs.length); i++) {
                testStore.update(docs[i].uuid, {
                    title: `Updated ${i}`,
                    dimensions: {
                        status: 'archived'
                    }
                });
            }

            // Verify updates
            const updatedDocs = testStore.list({ status: 'archived' });
            expect(updatedDocs.length).to.be.at.least(50);

            testStore.close();
        });

        it('should handle rapid add/delete cycles', function() {
            const testStore = new NanoStore(':memory:', config);
            
            // Perform rapid add/delete cycles
            for (let cycle = 0; cycle < 10; cycle++) {
                const ids = [];
                
                // Add 10 documents
                for (let i = 0; i < 10; i++) {
                    const id = testStore.add(`Cycle ${cycle} Doc ${i}`, {
                        type: 'task',
                        status: 'active'
                    });
                    ids.push(id);
                }
                
                // Delete half of them
                for (let i = 0; i < 5; i++) {
                    testStore.delete(ids[i]);
                }
                
                // Verify count
                const remainingDocs = testStore.list();
                expect(remainingDocs.length).to.equal((cycle + 1) * 5);
            }
            
            testStore.close();
        });
    });

});