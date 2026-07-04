const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const path = require('path');

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

const PORT = process.env.PORT || 3000;

// Serve static files from the 'public' directory
app.use(express.static(path.join(__dirname, 'public')));

// Store connected clients
let webClients = new Set();
let pcAgents = new Set();

wss.on('connection', (ws, req) => {
    // Basic routing based on URL path
    const isAgent = req.url === '/agent';
    
    if (isAgent) {
        console.log('PC Agent connected');
        pcAgents.add(ws);
        
        // Notify web clients that an agent is online
        broadcastToWebClients(JSON.stringify({ type: 'status', online: true }));
        
        ws.on('close', () => {
            console.log('PC Agent disconnected');
            pcAgents.delete(ws);
            broadcastToWebClients(JSON.stringify({ type: 'status', online: false }));
        });
    } else {
        console.log('Web Client connected');
        webClients.add(ws);
        
        // Send initial status to new web client
        ws.send(JSON.stringify({ type: 'status', online: pcAgents.size > 0 }));
        
        ws.on('close', () => {
            console.log('Web Client disconnected');
            webClients.delete(ws);
        });
    }

    ws.on('message', (message) => {
        try {
            const data = JSON.parse(message);
            
            // If message is from agent (like stats), forward to web clients
            if (isAgent) {
                if (data.type === 'stats') {
                    broadcastToWebClients(message.toString());
                } else if (data.type === 'command_result') {
                    broadcastToWebClients(message.toString());
                }
            } 
            // If message is from web client (like a command), forward to agent
            else {
                if (data.type === 'command') {
                    broadcastToAgents(message.toString());
                }
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
        }
    });
});

function broadcastToWebClients(data) {
    for (let client of webClients) {
        if (client.readyState === WebSocket.OPEN) {
            client.send(data);
        }
    }
}

function broadcastToAgents(data) {
    for (let agent of pcAgents) {
        if (agent.readyState === WebSocket.OPEN) {
            agent.send(data);
        }
    }
}

server.listen(PORT, () => {
    console.log(`Server is running on http://localhost:${PORT}`);
});
