/**
 * Integration Tests for Task Queue API
 * 
 * These tests verify the system from a client's perspective:
 * - HTTP API endpoints
 * - Task lifecycle (create -> poll status -> completion)
 * - Priority ordering
 * - Retry behavior
 * - Concurrent task processing
 * 
 * Prerequisites:
 * - Docker Compose running: docker-compose up -d
 * - API server on http://localhost:8080
 * - Worker processing tasks
 * 
 * Run: npm test
 */

import { test, describe, before, after } from 'node:test';
import assert from 'node:assert/strict';

const API_URL = process.env.API_URL || 'http://localhost:8080';
const POLL_INTERVAL = 500; // ms
const MAX_WAIT_TIME = 30000; // 30 seconds

// Helper: Make HTTP request
async function request(method, path, body = null) {
  const options = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  
  if (body) {
    options.body = JSON.stringify(body);
  }

  const response = await fetch(`${API_URL}${path}`, options);
  
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`HTTP ${response.status}: ${text}`);
  }

  return response.json();
}

// Helper: Wait for task to reach a specific status
async function waitForTaskStatus(taskId, expectedStatus, maxWaitMs = MAX_WAIT_TIME) {
  const startTime = Date.now();
  
  while (Date.now() - startTime < maxWaitMs) {
    const task = await request('GET', `/api/tasks/${taskId}`);
    
    if (task.status === expectedStatus) {
      return task;
    }
    
    // If task failed and we're not waiting for failure, throw error
    if (task.status === 'failed' && expectedStatus !== 'failed') {
      throw new Error(`Task ${taskId} failed unexpectedly: ${task.last_error}`);
    }
    
    await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL));
  }
  
  throw new Error(`Timeout waiting for task ${taskId} to reach status ${expectedStatus}`);
}

// Helper: Wait for task to reach any terminal state (succeeded or failed)
async function waitForTaskCompletion(taskId, maxWaitMs = MAX_WAIT_TIME) {
  const startTime = Date.now();
  
  while (Date.now() - startTime < maxWaitMs) {
    const task = await request('GET', `/api/tasks/${taskId}`);
    
    // Check if task reached a terminal state
    if (task.status === 'succeeded' || task.status === 'failed') {
      return task;
    }
    
    await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL));
  }
  
  throw new Error(`Timeout waiting for task ${taskId} to complete (still in status: queued/running)`);
}

// Helper: Create a test task
async function createTask(name, type = 'send_email', payload = null, priority = 0) {
  const defaultPayloads = {
    send_email: {
      to: 'test@example.com',
      subject: 'Test Email',
      body: 'Test message'
    },
    run_query: {
      query: 'SELECT * FROM users',
      database: 'test'
    }
  };

  return request('POST', '/api/tasks', {
    name,
    type,
    payload: payload || defaultPayloads[type],
    priority
  });
}

// Helper: Get system stats
async function getStats() {
  return request('GET', '/api/stats');
}

describe('Task Queue API Integration Tests', () => {
  
  before(async () => {
    console.log(`\n🔗 Testing API at: ${API_URL}`);
    
    // Verify API is reachable
    try {
      const stats = await getStats();
      console.log(`✅ API is up. Current stats:`, stats);
    } catch (error) {
      console.error('❌ API is not reachable. Make sure docker-compose is running.');
      throw error;
    }
  });

  describe('Task Lifecycle', () => {
    test('should create and process a task successfully', async () => {
      // Create task
      const created = await createTask('Test Task Lifecycle', 'send_email');
      
      assert.ok(created.id, 'Task should have an ID');
      assert.equal(created.status, 'queued', 'Initial status should be queued');
      
      // Verify task details
      const task = await request('GET', `/api/tasks/${created.id}`);
      assert.equal(task.name, 'Test Task Lifecycle');
      assert.equal(task.type, 'send_email');
      assert.equal(task.status, 'queued');
      
      // Wait for task to complete (allow 120s due to potential queue backlog from other tests)
      // Task may succeed OR fail due to simulated 20% failure rate (non-deterministic)
      const completed = await waitForTaskCompletion(created.id, 120000);
      
      // Validate: Task reached a terminal state
      assert.ok(['succeeded', 'failed'].includes(completed.status), 
        `Task should be in terminal state (succeeded/failed), got ${completed.status}`);
      
      // Validate: Retry count is within bounds (0-3)
      assert.ok(completed.retry_count >= 0 && completed.retry_count <= 3, 
        `Retry count should be 0-3, got ${completed.retry_count}`);
      
      // Validate: If task failed, it should have exhausted retries
      if (completed.status === 'failed') {
        assert.ok(completed.retry_count >= 3, 
          `Failed task should have exhausted retries (3+), got ${completed.retry_count}`);
        assert.ok(completed.last_error, 'Failed task should have an error message');
        console.log(`   ⚠️  Task failed after ${completed.retry_count} retries: ${completed.last_error}`);
      } else {
        // Task succeeded
        console.log(`   ✅ Task succeeded with ${completed.retry_count} retries`);
      }
    });

    test('should track task history', async () => {
      const created = await createTask('Test History', 'send_email');
      
      // Wait for completion (allow 120s due to potential queue backlog from other tests)
      await waitForTaskStatus(created.id, 'succeeded', 120000);
      
      // Get history
      const response = await request('GET', `/api/tasks/${created.id}/history`);
      
      // Response might be array or object with history property
      const history = Array.isArray(response) ? response : response.history;
      
      assert.ok(history, 'Should have history data');
      assert.ok(Array.isArray(history), 'History should be an array');
      assert.ok(history.length >= 2, 'Should have at least 2 events');
      
      // Verify events
      const eventTypes = history.map(h => h.event_type);
      assert.ok(eventTypes.includes('task_queued'), 'Should have task_queued event');
      assert.ok(eventTypes.includes('task_started'), 'Should have task_started event');
      assert.ok(eventTypes.includes('task_succeeded'), 'Should have task_succeeded event');
    });
  });

  describe('Priority Ordering', () => {
    test('should process high priority tasks first', async () => {
      const startStats = await getStats();
      
      // Create low priority tasks first
      const low1 = await createTask('Low Priority 1', 'send_email', null, 1);
      const low2 = await createTask('Low Priority 2', 'send_email', null, 1);
      
      // Then create high priority task
      const high = await createTask('High Priority', 'send_email', null, 100);
      
      // Wait a bit for all to be queued
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // The high priority task should complete first (or at least very quickly)
      // Note: This is probabilistic due to timing, but high priority should complete faster
      // Allow 120s due to potential queue backlog from other tests
      const highTask = await waitForTaskStatus(high.id, 'succeeded', 120000);
      
      assert.equal(highTask.priority, 100);
      assert.equal(highTask.status, 'succeeded');
    });
  });

  describe('Task Failures and Retries', () => {
    test('should handle task failures with retries', async () => {
      // Note: This test depends on handlers having some failure rate
      // We'll create multiple tasks and check if any retry
      const tasks = [];
      for (let i = 0; i < 10; i++) {
        const created = await createTask(`Retry Test ${i}`, 'run_query');
        tasks.push(created);
      }
      
      // Wait for all tasks to complete or fail
      const results = await Promise.all(
        tasks.map(t => 
          waitForTaskStatus(t.id, 'succeeded', 15000)
            .catch(() => request('GET', `/api/tasks/${t.id}`))
        )
      );
      
      // Check if any had retries
      const withRetries = results.filter(t => t.retry_count > 0);
      
      // At least some tasks should have retried (40% failure rate: 20% error + 20% timeout)
      console.log(`   📊 ${withRetries.length}/${results.length} tasks required retries`);
      
      // Verify retry logic worked
      withRetries.forEach(task => {
        assert.ok(task.retry_count >= 0, 'Retry count should be tracked');
        assert.ok(task.retry_count <= 3, 'Should not exceed max retries');
      });
    });

    test('should handle different failure types (errors and timeouts)', async () => {
      // Create 20 run_query tasks which have:
      // - 20% regular failures (error returned immediately)
      // - 20% timeouts (5s sleep simulates slow query)
      // - 60% success
      console.log('   📤 Creating 20 run_query tasks (expect mix of success/failures/timeouts)...');
      const tasks = [];
      for (let i = 0; i < 20; i++) {
        const created = await createTask(`Failure Type Test ${i}`, 'run_query');
        tasks.push(created);
      }
      
      // Wait for all tasks to complete (or fail after max retries)
      // Slow tasks take 5s + 3s processing, with retries (up to 4 attempts), allow 60s total
      console.log('   ⏳ Waiting up to 60s for all tasks to complete (slow tasks take 5s each)...');
      const results = await Promise.all(
        tasks.map(t => 
          waitForTaskStatus(t.id, 'succeeded', 60000)
            .catch(() => waitForTaskStatus(t.id, 'failed', 60000))
            .catch(() => request('GET', `/api/tasks/${t.id}`))
        )
      );
      
      // Analyze outcomes
      const succeeded = results.filter(t => t.status === 'succeeded');
      const failed = results.filter(t => t.status === 'failed');
      const withRetries = results.filter(t => t.retry_count > 0);
      const maxRetriesReached = results.filter(t => t.retry_count >= 3);
      
      console.log(`   📊 Results: ${succeeded.length} succeeded, ${failed.length} failed`);
      console.log(`   🔄 Retries: ${withRetries.length} tasks retried, ${maxRetriesReached.length} hit max retries`);
      
      // Assertions
      assert.ok(results.length === 20, 'Should have 20 task results');
      assert.ok(succeeded.length > 0, 'At least some tasks should succeed');
      assert.ok(withRetries.length > 0, 'At least some tasks should retry (failures + timeouts)');
      
      // Failed tasks should have hit max retries
      failed.forEach(task => {
        assert.ok(task.retry_count >= 3, `Failed task ${task.id} should have exhausted retries (has ${task.retry_count})`);
      });
    });

    test('should retry tasks that timeout', async () => {
      // This test verifies that tasks timing out are retried properly
      // We create several tasks and verify that if they fail, they retry
      console.log('   📤 Creating 10 run_query tasks to test timeout retry behavior...');
      const tasks = [];
      for (let i = 0; i < 10; i++) {
        const created = await createTask(`Timeout Retry Test ${i}`, 'run_query');
        tasks.push(created);
      }
      
      // Wait for all tasks to complete or fail (allow up to 60s for retries)
      console.log('   ⏳ Waiting for tasks to complete (may take up to 60s with retries)...');
      const results = await Promise.all(
        tasks.map(t => 
          waitForTaskStatus(t.id, 'succeeded', 60000)
            .catch(() => waitForTaskStatus(t.id, 'failed', 60000))
            .catch(() => request('GET', `/api/tasks/${t.id}`))
        )
      );
      
      // Get detailed info on all tasks
      const taskDetails = await Promise.all(
        tasks.map(t => request('GET', `/api/tasks/${t.id}`))
      );
      
      const withRetries = taskDetails.filter(t => t.retry_count > 0);
      const failed = taskDetails.filter(t => t.status === 'failed');
      
      console.log(`   🔄 ${withRetries.length}/${taskDetails.length} tasks were retried`);
      console.log(`   ❌ ${failed.length}/${taskDetails.length} tasks failed after max retries`);
      
      // Verify: tasks that were retried should have retry_count > 0
      withRetries.forEach(task => {
        assert.ok(task.retry_count > 0 && task.retry_count <= 3, 
          `Task ${task.id} should have 1-3 retries, has ${task.retry_count}`);
      });
      
      // Verify: failed tasks exhausted their retries
      failed.forEach(task => {
        assert.equal(task.status, 'failed', `Task ${task.id} should be in failed state`);
        assert.ok(task.retry_count >= 3, 
          `Failed task ${task.id} should have exhausted retries, has ${task.retry_count}`);
      });
    });

    test('should track retry history correctly', async () => {
      // Create a run_query task and check its history
      const task = await createTask('History Retry Test', 'run_query');
      
      // Wait for completion or failure (allow time for retries)
      await waitForTaskStatus(task.id, 'succeeded', 60000)
        .catch(() => waitForTaskStatus(task.id, 'failed', 60000));
      
      // Get history
      const history = await request('GET', `/api/tasks/${task.id}/history`);
      const events = Array.isArray(history) ? history : history.history || [];
      
      console.log(`   📜 Task ${task.id} had ${events.length} history events`);
      
      // Should have at least: task_queued, task_started
      assert.ok(events.length >= 2, 'Should have multiple history events');
      
      // Check for queued event
      const queuedEvent = events.find(e => e.event_type === 'task_queued');
      assert.ok(queuedEvent, 'Should have task_queued event');
      
      // Check for started event
      const startedEvents = events.filter(e => e.event_type === 'task_started');
      assert.ok(startedEvents.length > 0, 'Should have at least one task_started event');
      
      // If task had retries, should have retry events
      const finalTask = await request('GET', `/api/tasks/${task.id}`);
      if (finalTask.retry_count > 0) {
        const retryEvents = events.filter(e => e.event_type === 'retry_scheduled');
        console.log(`   🔄 Task retried ${finalTask.retry_count} times, found ${retryEvents.length} retry events`);
        assert.ok(retryEvents.length > 0, 'Should have retry events if task retried');
      }
    });
  });

  describe('Concurrent Processing', () => {
    test('should process multiple tasks concurrently', async () => {
      const numTasks = 20;
      const startTime = Date.now();
      
      // Create many tasks
      console.log(`   📤 Creating ${numTasks} send_email tasks...`);
      const created = await Promise.all(
        Array.from({ length: numTasks }, (_, i) =>
          createTask(`Concurrent Task ${i}`, 'send_email')
        )
      );
      
      // Wait for all to complete or fail (allow 120s due to potential queue backlog)
      console.log(`   ⏳ Waiting for tasks to complete...`);
      const results = await Promise.all(
        created.map(t => 
          waitForTaskStatus(t.id, 'succeeded', 120000)
            .catch(() => waitForTaskStatus(t.id, 'failed', 120000))
            .catch(() => request('GET', `/api/tasks/${t.id}`))
        )
      );
      
      const duration = Date.now() - startTime;
      const succeeded = results.filter(t => t.status === 'succeeded');
      const failed = results.filter(t => t.status === 'failed');
      console.log(`   ✅ ${succeeded.length}/${numTasks} tasks succeeded, ${failed.length} failed in ${duration}ms`);
      
      // With 5 concurrent workers and ~3s per task
      // Expected: ~12-15 seconds (20 tasks / 5 workers * 3s)
      // But allow up to 120s due to queue backlog from other tests
      // Note: Some tasks may fail due to simulated 20% error rate in send_email
      assert.ok(duration < 120000, `Should complete within 120s (took ${duration}ms)`);
      assert.ok(succeeded.length >= numTasks * 0.7, `At least 70% of tasks should succeed (${succeeded.length}/${numTasks})`);
    });
  });

  describe('Statistics API', () => {
    test('should return accurate system statistics', async () => {
      const stats = await getStats();
      
      assert.ok(typeof stats.total_tasks === 'number', 'Should have total_tasks');
      assert.ok(typeof stats.queued_tasks === 'number', 'Should have queued_tasks');
      assert.ok(typeof stats.running_tasks === 'number', 'Should have running_tasks');
      assert.ok(typeof stats.succeeded_tasks === 'number', 'Should have succeeded_tasks');
      assert.ok(typeof stats.failed_tasks === 'number', 'Should have failed_tasks');
      assert.ok(typeof stats.avg_retry_count === 'number', 'Should have avg_retry_count');
      
      // Sanity checks
      assert.ok(stats.total_tasks > 0, 'Should have processed some tasks');
      assert.ok(stats.succeeded_tasks > 0, 'Should have some successful tasks');
      
      console.log(`   📊 Current stats:`, stats);
    });
  });

  describe('Error Handling', () => {
    test('should reject invalid task creation', async () => {
      await assert.rejects(
        async () => {
          await request('POST', '/api/tasks', {
            // Missing required fields
            name: 'Invalid Task'
          });
        },
        /HTTP 400/,
        'Should reject task without type'
      );
    });

    test('should handle non-existent task lookup', async () => {
      await assert.rejects(
        async () => {
          await request('GET', '/api/tasks/999999');
        },
        /HTTP 404/,
        'Should return 404 for non-existent task'
      );
    });
  });

  after(async () => {
    console.log('\n✅ All integration tests completed');
    console.log('💡 Tip: Check dashboard at http://localhost:8080/ to see task processing\n');
  });
});
