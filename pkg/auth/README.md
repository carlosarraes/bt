# pkg/auth

Authentication management for Bitbucket CLI.

## Overview

This package handles all authentication methods supported by Bitbucket:
- App Passwords (username + password)
- OAuth 2.0 (browser flow)
- Access Tokens (repository/workspace/project scoped)

## Files (To Be Implemented)

- `manager.go` - Main authentication manager interface
- `app_password.go` - App password authentication
- `oauth.go` - OAuth 2.0 authentication flow
- `access_token.go` - Access token authentication
- `storage.go` - Secure credential storage

## Design

The authentication system is designed around the `AuthManager` interface that abstracts the specific authentication method. This allows commands to work with any authentication type seamlessly.

All credentials are stored securely using OS keyring integration and can be overridden by environment variables.

## Real API Integration

This package will integrate with real Bitbucket API endpoints from day one:
- `GET https://api.bitbucket.org/2.0/user` for auth verification
- `POST https://bitbucket.org/site/oauth2/access_token` for OAuth tokens
- `GET https://bitbucket.org/site/oauth2/authorize` for OAuth flow