import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { execSync, spawn, ChildProcess } from 'child_process';
import fs from 'fs';
import http from 'http';
import path from 'path';
import { defineConfig, Plugin } from 'vite';

function checkBackendReady(): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      {
        host: '127.0.0.1',
        port: 8081,
        path: '/api/health',
        method: 'GET',
        timeout: 300,
      },
      (res) => {
        resolve(res.statusCode === 200 || res.statusCode === 503);
      }
    );
    req.on('error', () => {
      resolve(false);
    });
    req.on('timeout', () => {
      req.destroy();
      resolve(false);
    });
    req.end();
  });
}

function backendPlugin(): Plugin {
  let backendProcess: ChildProcess | null = null;
  let isBackendReady = false;

  return {
    name: 'go-backend-runner',
    configureServer(server) {
      const binPath = path.resolve(__dirname, 'backend/server-bin');

      // Ensure local Redis and MongoDB services are active
      try {
        execSync('(redis-cli ping 2>/dev/null | grep -q PONG) || redis-server --daemonize yes 2>/dev/null || true');
      } catch {
        // ignore
      }
      try {
        execSync('mkdir -p /data/db /var/log/mongodb && (pgrep mongod >/dev/null || mongod --fork --logpath /var/log/mongodb/mongod.log --bind_ip 127.0.0.1 --dbpath /data/db 2>/dev/null) || true');
      } catch {
        // ignore
      }
      try {
        execSync('sleep 0.5');
      } catch {
        // ignore
      }

      if (!fs.existsSync(binPath)) {
        console.warn('[go-backend] backend/server-bin not found, attempting build...');
        try {
          execSync('cd backend && go build -o server-bin ./cmd/api/main.go', { stdio: 'inherit' });
        } catch (err) {
          console.error('[go-backend] Failed to build server-bin from Go source:', err);
          return;
        }
      }

      // Ensure binary is executable
      try {
        fs.chmodSync(binPath, 0o755);
      } catch (err) {
        console.error('[go-backend] Failed to set executable permission on server-bin:', err);
      }

      // Terminate any stale server-bin processes before launching
      try {
        execSync('pkill -9 -x server-bin 2>/dev/null || true');
      } catch {
        // ignore
      }

      console.log('[go-backend] Spawning Go backend server on port 8081...');
      backendProcess = spawn(binPath, [], {
        cwd: path.resolve(__dirname, 'backend'),
        env: {
          ...process.env,
          PORT: '8081',
          GIN_MODE: process.env.GIN_MODE || 'release',
          MONGO_URI: process.env.MONGO_URI || 'mongodb://127.0.0.1:27017',
          MONGO_DB_NAME: process.env.MONGO_DB_NAME || 'polling_db',
          REDIS_ADDR: process.env.REDIS_ADDR || '127.0.0.1:6379',
          JWT_SECRET:
            process.env.JWT_SECRET ||
            'dev_jwt_secret_key_change_in_production_12345',
        },
        stdio: ['ignore', 'pipe', 'pipe'],
      });

      backendProcess.stdout?.on('data', (chunk) => {
        process.stdout.write(`[backend] ${chunk}`);
      });
      backendProcess.stderr?.on('data', (chunk) => {
        process.stdout.write(`[backend] ${chunk}`);
      });

      backendProcess.on('error', (err) => {
        console.error('[go-backend] Process error:', err);
      });

      backendProcess.on('exit', (code, signal) => {
        console.log(`[go-backend] Exited with code ${code}, signal ${signal}`);
        backendProcess = null;
        isBackendReady = false;
      });

      // Periodically check readiness until backend binds to 8081
      const checkInterval = setInterval(async () => {
        if (!isBackendReady) {
          const ready = await checkBackendReady();
          if (ready) {
            isBackendReady = true;
            console.log('[go-backend] Go backend is ready on port 8081');
            clearInterval(checkInterval);
          }
        } else {
          clearInterval(checkInterval);
        }
      }, 300);

      // Middleware: intercept /api requests before the proxy while backend is spinning up
      server.middlewares.use(async (req, res, next) => {
        if (req.url && req.url.startsWith('/api')) {
          if (!isBackendReady) {
            const ready = await checkBackendReady();
            if (ready) {
              isBackendReady = true;
              return next();
            }

            res.writeHead(200, {
              'Content-Type': 'application/json',
              'Cache-Control': 'no-cache',
            });
            res.end(
              JSON.stringify({
                status: 'starting',
                timestamp: new Date().toISOString(),
                message: 'Go/Gin backend server is initializing with MongoDB and Redis...',
                services: {
                  mongodb: 'initializing',
                  redis: 'initializing',
                },
              })
            );
            return;
          }
        }
        next();
      });

      const cleanup = () => {
        clearInterval(checkInterval);
        if (backendProcess && !backendProcess.killed) {
          try {
            backendProcess.kill('SIGTERM');
          } catch {
            // ignore
          }
          backendProcess = null;
        }
      };

      server.httpServer?.on('close', cleanup);
      process.on('SIGTERM', cleanup);
      process.on('SIGINT', cleanup);
      process.on('exit', cleanup);
    },
  };
}

export default defineConfig(() => {
  return {
    plugins: [react(), tailwindcss(), backendPlugin()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, '.'),
      },
    },
    server: {
      hmr: process.env.DISABLE_HMR !== 'true',
      watch: process.env.DISABLE_HMR === 'true' ? null : {},
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:8081',
          changeOrigin: true,
          ws: true,
        },
      },
    },
  };
});
