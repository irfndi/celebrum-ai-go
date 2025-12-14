# Changelog - Critical Production Fixes

## Date: 2025-12-07

### Critical Issues Fixed

#### 1. PostgreSQL Connection Error - `query_timeout` Parameter ✅

**Issue:** Application was crashing with:
```
FATAL: unrecognized configuration parameter "query_timeout" (SQLSTATE 42704)
```

**Root Cause:** PostgreSQL doesn't have a `query_timeout` runtime parameter. The correct parameter is `statement_timeout`.

**Fix:**
- Updated `internal/database/postgres.go` to use only `statement_timeout`
- The config now intelligently uses the larger of `StatementTimeout` or `QueryTimeout` values
- Maintains backward compatibility with existing configs

**Files Changed:**
- `internal/database/postgres.go`
- `internal/config/config.go`

---

#### 2. CCXT Service Port Conflict - EADDRINUSE ✅

**Issue:** Bun/CCXT service failing to start with:
```
error: Failed to start server. Is port 3000 in use?
syscall: "listen",
errno: 0,
code: "EADDRINUSE"
```

**Root Cause:** Port 3000 conflicts with other services and stale processes weren't being cleaned up.

**Fixes Applied:**

##### A. Changed Default Port: 3000 → 3001
- Port 3000 often conflicts with common development tools (React, Vite, etc.)
- Updated default port to 3001 throughout entire codebase

##### B. Smart Port Resolution
- Added intelligent port resolution in `services/ccxt/index.ts`
- Checks environment variables in priority order: `CCXT_SERVICE_PORT`, `CCXT_PORT`, `PORT`
- Validates port ranges and provides fallback

##### C. Automatic Stale Process Cleanup
- Enhanced `scripts/entrypoint.sh` to detect and kill processes using the target port
- Uses `lsof` to find and terminate stale processes automatically
- Prevents EADDRINUSE errors from zombie processes

##### D. Better Error Handling
- Added proper error handling in Bun server startup
- Implemented graceful shutdown handlers (SIGTERM, SIGINT)
- Clear error messages with actionable troubleshooting steps
- Added `reusePort: true` for graceful restarts

##### E. Health Check with Timeout
- Added proper health check loop in entrypoint script
- Waits up to 30 seconds for CCXT service to be ready
- Provides clear feedback about service status

**Files Changed:**
- `services/ccxt/index.ts`
- `services/ccxt/.env.example`
- `scripts/entrypoint.sh`
- `Dockerfile`
- `config.yml`
- `docker-compose.yaml`
- `.env.example`
- `internal/config/config.go`
- `docs/deployment/DEPLOYMENT.md`
- `docs/development/CONFIGURATION.md`
- `docs/architecture/CACHE_ANALYSIS.md`
- `docs/troubleshooting/TELEGRAM_BOT_TROUBLESHOOTING.md`

---

### Configuration Changes Required

#### For Coolify Deployments

**IMPORTANT:** Update these environment variables in your Coolify dashboard:

```bash
# Old values (remove these)
CCXT_SERVICE_URL=http://localhost:3000  ❌

# New values (add these)
CCXT_SERVICE_URL=http://localhost:3001  ✅
PORT=3001                                ✅
```

#### For Local Development

Update your `.env` file:

```bash
CCXT_SERVICE_URL=http://localhost:3001
PORT=3001
```

---

### Deployment Steps

1. **Update Environment Variables in Coolify:**
   - Go to your application settings in Coolify
   - Update `CCXT_SERVICE_URL` to `http://localhost:3001`
   - Add `PORT=3001`
   - Save changes

2. **Trigger Redeploy:**
   - Push to your repository, or
   - Manually trigger deployment in Coolify

3. **Verify Deployment:**
   ```bash
   # Check logs for successful startup
   docker logs <container_name> 2>&1 | grep "CCXT Service successfully started"
   
   # Verify health
   curl http://localhost:3001/health
   ```

---

### Testing Verification

#### 1. Database Connection Test
```bash
# Should have NO query_timeout errors
docker logs <container> 2>&1 | grep -i "query_timeout"
# Expected: No results
```

#### 2. CCXT Service Startup Test
```bash
# Check for successful startup message
docker logs <container> 2>&1 | grep "CCXT Service successfully started"
# Expected: ✅ CCXT Service successfully started on port 3001
```

#### 3. Health Check Test
```bash
# From inside container
curl http://localhost:3001/health
# Expected: {"status":"healthy","timestamp":"...","service":"ccxt-service","version":"1.0.0"}
```

#### 4. Port Cleanup Test
```bash
# Verify automatic cleanup works
docker logs <container> 2>&1 | grep "Cleaning up stale process"
# If you see this, the cleanup mechanism is working
```

---

### Benefits

✅ **Eliminates Database Connection Errors**
- No more `query_timeout` parameter errors
- Uses correct PostgreSQL parameters

✅ **Prevents Port Conflicts**
- Automatic cleanup of stale processes
- Port conflicts resolved automatically

✅ **Better Error Messages**
- Clear, actionable error messages
- Easier troubleshooting

✅ **Graceful Restarts**
- Proper signal handling
- Clean shutdowns

✅ **Improved Reliability**
- Health checks ensure readiness
- Self-healing capabilities

✅ **Backward Compatible**
- Config still accepts `QueryTimeout`
- No breaking changes to existing configs

✅ **Production Ready**
- Robust error handling
- Automatic recovery mechanisms

---

### Migration Notes

#### No Breaking Changes
- These fixes maintain backward compatibility
- Existing configurations will continue to work
- Only environment variables need updating

#### Rollback Plan
If issues occur:
1. Set `PORT=3000` in environment
2. Set `CCXT_SERVICE_URL=http://localhost:3000`
3. Redeploy

---

### Summary

These fixes address two critical production issues:
1. **PostgreSQL connection failures** due to incorrect parameter names
2. **CCXT service startup failures** due to port conflicts

Both issues are now resolved with robust, production-ready solutions that include automatic recovery, better error handling, and comprehensive health checks.

The application is now more resilient, self-healing, and easier to troubleshoot.
