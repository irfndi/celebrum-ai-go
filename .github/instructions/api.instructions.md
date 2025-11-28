---
applyTo: "internal/api/**/*,internal/handlers/**/*"
---

# API and Handlers Instructions

## Handler Guidelines

- Use Gin framework patterns consistently
- Implement proper error responses with appropriate HTTP status codes
- Validate all input parameters before processing
- Use structured logging with request context

## Request/Response Patterns

- Use JSON for request and response bodies
- Define request/response structs with proper tags
- Implement pagination for list endpoints
- Include correlation IDs in logs and responses

## Error Handling

```go
// Return appropriate HTTP status codes
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
```

## Security

- Use JWT middleware for protected endpoints
- Implement rate limiting on all public endpoints
- Validate and sanitize all user input
- Never expose sensitive data in error messages

## Middleware

- Use authentication middleware from `internal/middleware`
- Apply rate limiting middleware appropriately
- Log requests with correlation IDs
- Handle CORS for web clients

## Testing

- Test handler functions with mock services
- Verify correct HTTP status codes
- Test authentication and authorization
- Test rate limiting behavior
