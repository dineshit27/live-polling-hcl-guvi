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
        port: 8080,
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
    name: 'backend-runner',
    configureServer(server) {
      const binPath = process.platform === 'win32'
        ? path.resolve(__dirname, 'backend/server-bin.exe')
        : path.resolve(__dirname, 'backend/server-bin');

      if (fs.existsSync(binPath)) {
        console.log('[go-backend] Spawning Go backend server on port 8080...');
        backendProcess = spawn(binPath, [], {
          cwd: path.resolve(__dirname, 'backend'),
          env: {
            ...process.env,
            PORT: '8080',
            GIN_MODE: process.env.GIN_MODE || 'debug',
            MONGO_URI: process.env.MONGO_URI || 'mongodb://127.0.0.1:27017',
            MONGO_DB_NAME: process.env.MONGO_DB_NAME || 'polling_db',
            REDIS_ADDR: process.env.REDIS_ADDR || '127.0.0.1:6379',
            JWT_SECRET:
              process.env.JWT_SECRET ||
              'dev_jwt_secret_key_change_in_production_12345',
          },
          stdio: ['ignore', 'pipe', 'pipe'],
        });
      } else {
        console.log('[go-backend] Note: Go backend executable not found at', binPath, '- Go server can be run directly or via docker on port 8080.');
      }

      if (backendProcess) {
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
      }

      // Periodically check readiness until backend binds to 8080
      const checkInterval = setInterval(async () => {
        if (!isBackendReady) {
          const ready = await checkBackendReady();
          if (ready) {
            isBackendReady = true;
            console.log('[go-backend] Go backend is ready on port 8080');
            clearInterval(checkInterval);
          }
        } else {
          clearInterval(checkInterval);
        }
      }, 500);

      // Middleware: intercept /api requests before the proxy while backend is spinning up
      server.middlewares.use(async (req, res, next) => {
        if (req.url && req.url.startsWith('/api')) {
          if (!isBackendReady) {
            const ready = await checkBackendReady();
            if (ready) {
              isBackendReady = true;
              return next();
            }

            res.writeHead(503, {
              'Content-Type': 'application/json',
              'Cache-Control': 'no-cache',
            });
            res.end(
              JSON.stringify({
                status: 'starting',
                timestamp: new Date().toISOString(),
                message: 'Go/Gin backend server is initializing on port 8080 with MongoDB and Redis...',
                services: {
                  mongodb: 'connecting',
                  redis: 'connecting',
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
            if (process.platform === 'win32' && backendProcess.pid) {
              execSync(`taskkill /pid ${backendProcess.pid} /T /F 2>nul`);
            } else {
              backendProcess.kill('SIGTERM');
            }
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
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
          ws: true,
        },
      },
    },
  };
});
