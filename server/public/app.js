document.addEventListener('DOMContentLoaded', () => {
    const statusDot = document.getElementById('status-dot');
    const statusText = document.getElementById('status-text');
    const cpuVal = document.getElementById('cpu-val');
    const memVal = document.getElementById('mem-val');
    const btnLock = document.getElementById('btn-lock');
    const btnHardLock = document.getElementById('btn-hard-lock');
    const btnUnlock = document.getElementById('btn-unlock');
    const targetUser = document.getElementById('target-user');
    const adminPassword = document.getElementById('admin-password');
    const actionResult = document.getElementById('action-result');

    // Connect to WebSocket server
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}`;
    
    let ws;

    function connect() {
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('Connected to server');
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);

            switch (data.type) {
                case 'status':
                    updateAgentStatus(data.online);
                    break;
                case 'stats':
                    updateStats(data.cpu, data.memory);
                    break;
                case 'command_result':
                    showCommandResult(data.command, data.success, data.message);
                    break;
            }
        };

        ws.onclose = () => {
            console.log('Disconnected from server. Reconnecting...');
            updateAgentStatus(false);
            setTimeout(connect, 3000);
        };
        
        ws.onerror = (err) => {
            console.error('WebSocket error:', err);
            ws.close();
        };
    }

    function updateAgentStatus(isOnline) {
        if (isOnline) {
            statusDot.className = 'dot online';
            statusText.textContent = 'PC Online';
            btnLock.disabled = false;
            btnHardLock.disabled = false;
            btnUnlock.disabled = false;
        } else {
            statusDot.className = 'dot offline';
            statusText.textContent = 'PC Offline';
            btnLock.disabled = true;
            btnHardLock.disabled = true;
            btnUnlock.disabled = true;
            cpuVal.textContent = '--%';
            memVal.textContent = '--%';
        }
    }

    function updateStats(cpu, memory) {
        cpuVal.textContent = `${cpu.toFixed(1)}%`;
        memVal.textContent = `${memory.toFixed(1)}%`;
    }

    function showCommandResult(command, success, message) {
        actionResult.textContent = message || (success ? `${command} executed successfully` : `Failed to execute ${command}`);
        actionResult.className = `result-msg ${success ? 'success' : 'error'}`;
        
        setTimeout(() => {
            actionResult.textContent = '';
            actionResult.className = 'result-msg';
        }, 5000);
    }

    function sendCommand(action) {
        const username = targetUser.value.trim();
        const unlockPassword = adminPassword.value;

        if ((action === 'HARD_LOCK' || action === 'UNLOCK') && !unlockPassword) {
            actionResult.textContent = 'Admin unlock password is required for this action';
            actionResult.className = 'result-msg error';
            return;
        }

        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'command',
                action: action,
                username,
                adminPassword: unlockPassword
            }));
            actionResult.textContent = `Sending ${action} command...`;
            actionResult.className = 'result-msg';
        }
    }

    // Button event listeners
    btnLock.addEventListener('click', () => sendCommand('LOCK'));
    btnHardLock.addEventListener('click', () => sendCommand('HARD_LOCK'));
    btnUnlock.addEventListener('click', () => sendCommand('UNLOCK'));

    // Initial connection
    connect();
});
