<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

# Castafiore Backend - Copilot Instructions

## Project Overview
This is a Go-based music streaming server with Subsonic API compatibility. The project focuses on:
- Music streaming with user management
- Subscription plans and concurrent connection limits
- Download tracking and limits
- Database-driven architecture with PostgreSQL

## Code Style Guidelines
- Follow standard Go conventions and idioms
- Use meaningful variable and function names
- Add comprehensive error handling
- Include relevant documentation comments
- Prefer dependency injection for services

## Architecture Patterns
- Clean architecture with separation of concerns
- Repository pattern for database operations
- Service layer for business logic
- Middleware for authentication and validation
- RESTful API design following Subsonic specification

## Key Technologies
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL with database/sql
- **Authentication**: JWT with bcrypt for passwords
- **Configuration**: Environment variables with Viper
- **API Compatibility**: Subsonic API v1.16.1

## Database Schema
- Users table with subscription plans and limits
- Music catalog (artists, albums, songs)
- Session tracking for concurrent connections
- Download history for daily limits

## Security Considerations
- Always hash passwords with bcrypt
- Validate all user inputs
- Implement rate limiting
- Use JWT for stateless authentication
- Support Subsonic salt/token authentication

## Testing Guidelines
- Write unit tests for business logic
- Integration tests for database operations
- Mock external dependencies
- Test error conditions and edge cases

## API Design
- Follow Subsonic API specification exactly
- Support both XML and JSON responses
- Consistent error handling with proper HTTP codes
- Authentication required for all endpoints except ping/license
