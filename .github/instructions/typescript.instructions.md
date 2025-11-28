---
applyTo: "ccxt-service/**/*.{ts,js}"
---

# TypeScript/CCXT Service Instructions

## Overview

The `ccxt-service/` directory contains the Bun-based exchange integration service that interfaces with cryptocurrency exchanges via the CCXT library.

## Coding Standards

- Use TypeScript strict mode
- Run `bun run format` before committing
- Keep strict lint output clean
- Use async/await for asynchronous operations
- Define explicit types for all function parameters and return values

## Project Structure

- `index.ts` - Main service entry point
- `types.ts` - Type definitions
- `*.test.ts` - Test files (co-located with source)

## Testing

- Write tests in `*.test.ts` files alongside source files
- Use Bun's built-in test runner
- Test exchange connection handling and error cases

## Build and Test Commands

```bash
cd ccxt-service
bun install          # Install dependencies
bun run format       # Format code
bun test             # Run tests
```

## Security

- Never log API keys or exchange credentials
- Validate all exchange responses
- Implement proper rate limiting for exchange API calls
