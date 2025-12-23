let eventSource = null;

function updateStats(stats) {
    // Update stat cards
    document.getElementById('total-tasks').textContent = stats.total_tasks;
    document.getElementById('queued-tasks').textContent = stats.queued_tasks;
    document.getElementById('running-tasks').textContent = stats.running_tasks;
    document.getElementById('succeeded-tasks').textContent = stats.succeeded_tasks;
    document.getElementById('failed-tasks').textContent = stats.failed_tasks;
    
    // Calculate success rate
    const completedTasks = stats.succeeded_tasks + stats.failed_tasks;
    const successRate = completedTasks > 0 
        ? Math.round((stats.succeeded_tasks / completedTasks) * 100) 
        : 0;
    
    document.getElementById('success-rate').style.width = successRate + '%';
    document.getElementById('success-percentage').textContent = successRate;
    
    // Update additional metrics
    document.getElementById('avg-retry').textContent = stats.avg_retry_count.toFixed(2);
    document.getElementById('tasks-with-retries').textContent = stats.tasks_with_retries;
    
    // Update timestamp
    const now = new Date();
    document.getElementById('last-updated').textContent = now.toLocaleTimeString();
    
    // Add pulse animation
    document.querySelector('.container').classList.add('updating');
    setTimeout(() => {
        document.querySelector('.container').classList.remove('updating');
    }, 300);
}

function connectSSE() {
    const statusEl = document.getElementById('connection-status');
    
    // Close existing connection if any
    if (eventSource) {
        eventSource.close();
    }
    
    // Connect to SSE endpoint
    eventSource = new EventSource('/api/tasks/stream');
    
    eventSource.addEventListener('stats', function(e) {
        try {
            const stats = JSON.parse(e.data);
            updateStats(stats);
        } catch (err) {
            console.error('Failed to parse stats:', err);
        }
    });
    
    eventSource.onopen = function() {
        statusEl.textContent = 'Connected';
        statusEl.className = 'status connected';
        console.log('SSE connection established');
    };
    
    eventSource.onerror = function(err) {
        statusEl.textContent = 'Disconnected';
        statusEl.className = 'status disconnected';
        console.error('SSE error:', err);
        
        // Reconnect after 5 seconds
        setTimeout(() => {
            console.log('Attempting to reconnect...');
            connectSSE();
        }, 5000);
    };
}

// Start connection when page loads
connectSSE();

// Cleanup on page unload
window.addEventListener('beforeunload', function() {
    if (eventSource) {
        eventSource.close();
    }
});
