# Changelog

## [Unreleased]

### Added

#### Authentication & Authorization
- JWT-based authentication system with configurable token expiry
- Role-based access control (RBAC) with three user roles:
  - **Admin**: Full system access including user management
  - **Operator**: Machine and group management capabilities
  - **Viewer**: Read-only access to all resources
- User management API endpoints:
  - POST `/api/v1/login` - User login
  - POST `/api/v1/users` - Create new users (Admin only)
  - GET `/api/v1/users` - List all users (Admin only)
  - GET `/api/v1/users/{id}` - Get user details (Admin only)
  - PUT `/api/v1/users/{id}` - Update user (Admin only)
  - DELETE `/api/v1/users/{id}` - Delete user (Admin only)
  - GET `/api/v1/auth/me` - Get current user info
  - POST `/api/v1/auth/refresh` - Refresh JWT token
- Authentication middleware with role-based route protection
- Password hashing using bcrypt
- Command-line flag `--create-admin` to create default admin user
- Optional authentication mode via `--enable-auth` flag
- Environment variables for JWT configuration (`JWT_SECRET`, `ENABLE_AUTH`)

#### PostgreSQL Support
- Full PostgreSQL database support alongside SQLite
- Database driver abstraction with automatic query adaptation
- JSONB field types for PostgreSQL (performance optimization)
- Connection pooling with configurable limits
- All database operations support both SQLite and PostgreSQL

#### Machine Grouping
- Machine group model with tags and descriptions
- Group management API endpoints:
  - POST `/api/v1/groups` - Create group
  - GET `/api/v1/groups` - List all groups
  - GET `/api/v1/groups/{id}` - Get group details
  - PUT `/api/v1/groups/{id}` - Update group
  - DELETE `/api/v1/groups/{id}` - Delete group
  - GET `/api/v1/groups/{id}/machines` - List machines in group
  - PUT `/api/v1/groups/{id}/machines/{machine_id}` - Add machine to group
  - DELETE `/api/v1/groups/{id}/machines/{machine_id}` - Remove machine from group
- GET `/api/v1/machines/{id}/groups` - List groups a machine belongs to
- Many-to-many relationship between machines and groups
- Cascade deletion of group memberships

#### Bulk Operations
- Bulk operations API endpoint: POST `/api/v1/bulk`
- Supported operations:
  - `update` - Update multiple machines at once
  - `build` - Trigger builds for multiple machines
  - `delete` - Delete multiple machines
- Operations can target:
  - Specific machine IDs
  - All machines in a group
- Detailed operation results with success/failure counts and error messages

### Changed
- API routes now have role-based access control
- Machine enrollment endpoint remains public (no authentication required)
- Updated README with comprehensive authentication documentation
- Updated README with new API endpoint examples
- Enhanced security considerations section in README
- Roadmap updated to reflect completed features

### Security
- Default JWT secret must be changed in production
- Default admin credentials (admin/admin) should be changed immediately
- PostgreSQL recommended for production use
- SQLite suitable for development/testing only

## Dependencies Added
- `github.com/golang-jwt/jwt/v5` v5.2.0 - JWT token generation and validation
- `golang.org/x/crypto` v0.18.0 - Password hashing with bcrypt

## Database Schema Changes

### New Tables
- `users` - User accounts with authentication credentials
- `api_keys` - API keys for programmatic access (future use)
- `groups` - Machine groups
- `group_memberships` - Many-to-many relationship between machines and groups

## Migration Notes

### From Previous Version
1. Run database migrations to create new tables
2. Create an admin user: `./server --create-admin`
3. Login and change the default password
4. Create additional users as needed
5. Update `JWT_SECRET` environment variable in production

### Disabling Authentication (Not Recommended)
For backward compatibility or testing, authentication can be disabled:
```bash
./server --enable-auth=false
```

## API Changes

### Breaking Changes
- Most API endpoints now require authentication (when `ENABLE_AUTH=true`)
- Requests must include `Authorization: Bearer <token>` header

### Backward Compatibility
- Authentication is enabled by default but can be disabled
- When disabled, all endpoints work as before
- Machine enrollment endpoint is always public

## Performance Improvements
- PostgreSQL JSONB fields for better JSON query performance
- Database connection pooling
- Optimized group membership queries with proper indexing

## Future Enhancements
- API key authentication for programmatic access
- Webhook notifications for machine events
- Advanced filtering and search capabilities
- Machine templates for common configurations
- Integration with configuration management tools
- IPMI/BMC integration for remote power control
